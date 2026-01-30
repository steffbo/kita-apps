package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

const (
	// Fee combination amounts
	foodWithReminderAmount       = domain.FoodFeeAmount + domain.ReminderFeeAmount             // 55.40
	membershipWithReminderAmount = domain.MembershipFeeAmount + domain.MembershipReminderFeeAmount // 35.00

	// Late payment threshold: 15th day of the month
	latePaymentDayThreshold = 15

	// Confidence thresholds
	autoMatchConfidenceThreshold = 0.95
	memberNumberConfidence       = 0.95
	confidenceBoostCombined      = 0.02
	confidenceBoostNameMatch     = 0.05
	maxConfidenceNameMatch       = 0.93
	maxConfidenceCombined        = 0.99
)

// ImportService handles CSV import and matching logic.
type ImportService struct {
	transactionRepo repository.TransactionRepository
	feeRepo         repository.FeeRepository
	childRepo       repository.ChildRepository
	matchRepo       repository.MatchRepository
	knownIBANRepo   repository.KnownIBANRepository
	warningRepo     repository.WarningRepository
}

// NewImportService creates a new import service.
func NewImportService(
	transactionRepo repository.TransactionRepository,
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	matchRepo repository.MatchRepository,
	knownIBANRepo repository.KnownIBANRepository,
	warningRepo repository.WarningRepository,
) *ImportService {
	return &ImportService{
		transactionRepo: transactionRepo,
		feeRepo:         feeRepo,
		childRepo:       childRepo,
		matchRepo:       matchRepo,
		knownIBANRepo:   knownIBANRepo,
		warningRepo:     warningRepo,
	}
}

// ImportResult represents the result of a CSV import.
type ImportResult struct {
	BatchID     uuid.UUID                   `json:"batchId"`
	FileName    string                      `json:"fileName"`
	TotalRows   int                         `json:"totalRows"`
	Imported    int                         `json:"imported"`
	Skipped     int                         `json:"skipped"`
	Blacklisted int                         `json:"blacklisted"`
	Warnings    int                         `json:"warnings"`
	Suggestions []domain.MatchSuggestion    `json:"suggestions"`
	WarningList []domain.TransactionWarning `json:"warningList,omitempty"`
}

// MatchConfirmation represents a match to confirm.
type MatchConfirmation struct {
	TransactionID uuid.UUID
	ExpectationID uuid.UUID
}

// ConfirmResult represents the result of confirming matches.
type ConfirmResult struct {
	Confirmed int `json:"confirmed"`
	Failed    int `json:"failed"`
}

// RescanResult represents the result of rescanning unmatched transactions.
type RescanResult struct {
	Scanned     int                      `json:"scanned"`
	AutoMatched int                      `json:"autoMatched"`
	Suggestions []domain.MatchSuggestion `json:"suggestions"`
}

// DismissResult represents the result of dismissing a transaction.
type DismissResult struct {
	IBAN                string `json:"iban"`
	TransactionsRemoved int64  `json:"transactionsRemoved"`
}

// ProcessCSV processes a CSV file and returns match suggestions.
func (s *ImportService) ProcessCSV(ctx context.Context, file io.Reader, fileName string, userID uuid.UUID) (*ImportResult, error) {
	// Parse CSV
	transactions, err := csvparser.ParseBankCSV(file)
	if err != nil {
		return nil, err
	}

	batchID := uuid.New()
	result := &ImportResult{
		BatchID:   batchID,
		FileName:  fileName,
		TotalRows: len(transactions),
	}

	// Create batch record first (so we have the metadata stored)
	if err := s.transactionRepo.CreateBatch(ctx, batchID, fileName, userID); err != nil {
		// Log but don't fail - we can still process without batch metadata
		// In production, you might want to handle this differently
	}

	// Get blacklisted IBANs for efficient filtering
	blacklistedIBANs, err := s.knownIBANRepo.GetBlacklistedIBANs(ctx)
	if err != nil {
		// Log but don't fail - continue without blacklist filtering
		blacklistedIBANs = make(map[string]bool)
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, "", "", "", 0, 1000)
	s.enrichChildrenWithParents(ctx, children)

	// Process each transaction
	for _, tx := range transactions {
		// Only process incoming payments
		if tx.Amount <= 0 {
			result.Skipped++
			continue
		}

		// Skip blacklisted IBANs
		if tx.PayerIBAN != nil && blacklistedIBANs[*tx.PayerIBAN] {
			result.Blacklisted++
			continue
		}

		tx.ImportBatchID = &batchID

		// Check if transaction already exists
		exists, _ := s.transactionRepo.Exists(ctx, tx.BookingDate, tx.PayerIBAN, tx.Amount, tx.Description)
		if exists {
			result.Skipped++
			continue
		}

		// Save transaction
		if err := s.transactionRepo.Create(ctx, &tx); err != nil {
			result.Skipped++
			continue
		}
		result.Imported++

		// Try to match
		suggestion := s.matchTransaction(ctx, tx, children)
		if suggestion != nil {
			result.Suggestions = append(result.Suggestions, *suggestion)
		} else {
			// No match found - check if this is from a trusted IBAN
			warning := s.checkForWarning(ctx, tx)
			if warning != nil {
				// Save warning to database
				if s.warningRepo != nil {
					if err := s.warningRepo.Create(ctx, warning); err == nil {
						result.Warnings++
						result.WarningList = append(result.WarningList, *warning)
					}
				}
			}
		}
	}

	return result, nil
}

func (s *ImportService) matchTransaction(ctx context.Context, tx domain.BankTransaction, children []domain.Child) *domain.MatchSuggestion {
	suggestion := &domain.MatchSuggestion{
		Transaction: tx,
		Confidence:  0,
	}

	// Detect fee type from amount
	suggestion.DetectedType = s.detectFeeType(tx.Amount)

	// Match child by member number, name, or parent name
	matchText := buildMatchText(tx)
	s.matchChild(matchText, children, suggestion)

	// Find the corresponding fee expectation
	if suggestion.Child != nil && suggestion.DetectedType != nil {
		s.matchFeeExpectation(ctx, tx, suggestion)
	}

	// Boost confidence for name-based matches with fee expectations
	if suggestion.Expectation != nil && (suggestion.MatchedBy == "name" || suggestion.MatchedBy == "parent_name") {
		suggestion.Confidence = min(suggestion.Confidence+confidenceBoostNameMatch, maxConfidenceNameMatch)
	}

	// Only return if we have some confidence
	if suggestion.Confidence > 0 {
		return suggestion
	}

	return nil
}

func (s *ImportService) detectFeeType(amount float64) *domain.FeeType {
	switch amount {
	case domain.FoodFeeAmount:
		feeType := domain.FeeTypeFood
		return &feeType
	case domain.MembershipFeeAmount:
		feeType := domain.FeeTypeMembership
		return &feeType
	case foodWithReminderAmount:
		feeType := domain.FeeTypeFood
		return &feeType
	case membershipWithReminderAmount:
		feeType := domain.FeeTypeMembership
		return &feeType
	default:
		feeType := domain.FeeTypeChildcare
		return &feeType
	}
}

func (s *ImportService) matchChild(matchText string, children []domain.Child, suggestion *domain.MatchSuggestion) {
	// Try to match by member number first
	memberNumber := csvparser.ExtractMemberNumber(matchText)
	if memberNumber != "" {
		for i := range children {
			if children[i].MemberNumber == memberNumber {
				suggestion.Child = &children[i]
				suggestion.MatchedBy = "member_number"
				suggestion.Confidence = memberNumberConfidence
				return
			}
		}
	}

	// Try by child name
	matchedChild, confidence := csvparser.MatchChildByName(matchText, children)
	if matchedChild != nil {
		suggestion.Child = matchedChild
		suggestion.MatchedBy = "name"
		suggestion.Confidence = confidence
		return
	}

	// Try by parent name
	matchedChild, confidence = csvparser.MatchChildByParentName(matchText, children)
	if matchedChild != nil {
		suggestion.Child = matchedChild
		suggestion.MatchedBy = "parent_name"
		suggestion.Confidence = confidence
	}
}

func (s *ImportService) matchFeeExpectation(ctx context.Context, tx domain.BankTransaction, suggestion *domain.MatchSuggestion) {
	// First try exact amount match - prefer fee for same month as payment date
	fee, err := s.feeRepo.FindBestUnpaid(ctx, suggestion.Child.ID, *suggestion.DetectedType, tx.Amount, tx.BookingDate)
	if err == nil && fee != nil {
		suggestion.Expectation = fee
		return
	}

	// No exact match found - try combined fee + reminder match
	fees, err := s.feeRepo.FindOldestUnpaidWithReminder(ctx, suggestion.Child.ID, *suggestion.DetectedType, tx.Amount)
	if err == nil && len(fees) == 2 {
		suggestion.Expectations = fees
		suggestion.Expectation = &fees[0]
		suggestion.MatchedBy = "combined"
		// Boost confidence slightly for combined matches since amount is precise
		if suggestion.Confidence > 0 {
			suggestion.Confidence = min(suggestion.Confidence+confidenceBoostCombined, maxConfidenceCombined)
		}
	}
}

func (s *ImportService) enrichChildrenWithParents(ctx context.Context, children []domain.Child) {
	if len(children) == 0 {
		return
	}

	childIDs := make([]uuid.UUID, len(children))
	for i := range children {
		childIDs[i] = children[i].ID
	}

	parentsMap, err := s.childRepo.GetParentsForChildren(ctx, childIDs)
	if err != nil {
		return
	}

	for i := range children {
		if parents, ok := parentsMap[children[i].ID]; ok {
			children[i].Parents = parents
		}
	}
}

func buildMatchText(tx domain.BankTransaction) string {
	var parts []string
	if tx.PayerName != nil && strings.TrimSpace(*tx.PayerName) != "" {
		parts = append(parts, *tx.PayerName)
	}
	if tx.Description != nil && strings.TrimSpace(*tx.Description) != "" {
		parts = append(parts, *tx.Description)
	}
	return strings.Join(parts, " ")
}

// checkForWarning checks if an unmatched transaction from a trusted IBAN should generate a warning.
func (s *ImportService) checkForWarning(ctx context.Context, tx domain.BankTransaction) *domain.TransactionWarning {
	if tx.PayerIBAN == nil {
		return nil
	}

	// Check if IBAN is trusted
	knownIBAN, err := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
	if err != nil || knownIBAN == nil || knownIBAN.Status != domain.KnownIBANStatusTrusted {
		return nil
	}

	warning := &domain.TransactionWarning{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ActualAmount:  &tx.Amount,
		ChildID:       knownIBAN.ChildID,
		CreatedAt:     time.Now(),
	}

	// If we have a linked child, check what fees are open
	if knownIBAN.ChildID != nil {
		childID := *knownIBAN.ChildID

		// Check for possible bulk payment (amount is multiple of known fee amounts)
		bulkCount := s.checkBulkPayment(tx.Amount)
		if bulkCount > 1 {
			warning.WarningType = domain.WarningTypePossibleBulk
			warning.Message = fmt.Sprintf("Betrag %.2f EUR könnte eine Sammelzahlung sein (%d Zahlungen)", tx.Amount, bulkCount)
			return warning
		}

		// Check open fees for this child to determine warning type
		filter := repository.FeeFilter{ChildID: &childID}
		fees, _, err := s.feeRepo.List(ctx, filter, 0, 100)
		if err == nil && len(fees) > 0 {
			// Find unpaid fees
			for _, fee := range fees {
				// Check if this fee is already paid
				isPaid, _ := s.matchRepo.ExistsForExpectation(ctx, fee.ID)
				if isPaid {
					continue
				}

				// Compare amounts
				if tx.Amount < fee.Amount {
					warning.WarningType = domain.WarningTypePartialPayment
					warning.ExpectedAmount = &fee.Amount
					warning.Message = fmt.Sprintf("Teilzahlung: %.2f EUR erhalten, %.2f EUR erwartet", tx.Amount, fee.Amount)
					return warning
				} else if tx.Amount > fee.Amount {
					warning.WarningType = domain.WarningTypeOverpayment
					warning.ExpectedAmount = &fee.Amount
					warning.Message = fmt.Sprintf("Überzahlung: %.2f EUR erhalten, %.2f EUR erwartet", tx.Amount, fee.Amount)
					return warning
				}
			}
		}

		// No open fees found for this child
		warning.WarningType = domain.WarningTypeNoMatchingFee
		warning.Message = fmt.Sprintf("Keine offene Beitragsforderung für dieses Kind gefunden (%.2f EUR)", tx.Amount)
		return warning
	}

	// Trusted IBAN without linked child - just note the unexpected payment
	warning.WarningType = domain.WarningTypeUnexpectedAmount
	warning.Message = fmt.Sprintf("Zahlung von %.2f EUR von vertrauter IBAN ohne zugeordnetes Kind", tx.Amount)
	return warning
}

// checkBulkPayment checks if an amount could be a bulk payment of multiple fees.
func (s *ImportService) checkBulkPayment(amount float64) int {
	// Common fee amounts
	foodFee := domain.FoodFeeAmount             // 45.40
	membershipFee := domain.MembershipFeeAmount // 30.00
	reminderFee := domain.ReminderFeeAmount     // 10.00

	// Check for exact multiples of food fee
	if amount >= foodFee*2 {
		count := int(amount / foodFee)
		if amount == foodFee*float64(count) {
			return count
		}
	}

	// Check for exact multiples of membership fee
	if amount >= membershipFee*2 {
		count := int(amount / membershipFee)
		if amount == membershipFee*float64(count) {
			return count
		}
	}

	// Check for food + reminder combinations
	combinedFood := foodFee + reminderFee // 55.40
	if amount >= combinedFood*2 {
		count := int(amount / combinedFood)
		if amount == combinedFood*float64(count) {
			return count
		}
	}

	return 1
}

// ConfirmMatches confirms a list of matches.
func (s *ImportService) ConfirmMatches(ctx context.Context, matches []MatchConfirmation, userID uuid.UUID) (*ConfirmResult, error) {
	result := &ConfirmResult{}

	for _, m := range matches {
		match := &domain.PaymentMatch{
			ID:            uuid.New(),
			TransactionID: m.TransactionID,
			ExpectationID: m.ExpectationID,
			MatchType:     domain.MatchTypeManual,
			MatchedAt:     time.Now(),
			MatchedBy:     &userID,
		}

		if err := s.matchRepo.Create(ctx, match); err != nil {
			result.Failed++
			continue
		}
		result.Confirmed++

		s.postMatchActions(ctx, m.TransactionID, m.ExpectationID, "Auto-resolved: Zahlung wurde zugeordnet")
	}

	return result, nil
}

// markIBANAsTrusted marks the IBAN from a transaction as trusted.
func (s *ImportService) markIBANAsTrusted(ctx context.Context, transactionID uuid.UUID) {
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil || tx.PayerIBAN == nil {
		return
	}

	// Check if already known
	existing, _ := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
	if existing != nil {
		// Already known, don't overwrite
		return
	}

	knownIBAN := &domain.KnownIBAN{
		IBAN:                  *tx.PayerIBAN,
		PayerName:             tx.PayerName,
		Status:                domain.KnownIBANStatusTrusted,
		Reason:                stringPtr("Automatically marked as trusted after successful match"),
		OriginalTransactionID: &tx.ID,
		OriginalDescription:   tx.Description,
		OriginalAmount:        &tx.Amount,
	}

	s.knownIBANRepo.Create(ctx, knownIBAN)
}

// GetHistory returns import batch history.
func (s *ImportService) GetHistory(ctx context.Context, offset, limit int) ([]domain.ImportBatch, int64, error) {
	return s.transactionRepo.GetBatches(ctx, offset, limit)
}

// GetUnmatchedTransactions returns transactions without matches.
func (s *ImportService) GetUnmatchedTransactions(ctx context.Context, offset, limit int) ([]domain.BankTransaction, int64, error) {
	return s.transactionRepo.ListUnmatched(ctx, offset, limit)
}

// CreateManualMatch creates a manual match between transaction and fee.
func (s *ImportService) CreateManualMatch(ctx context.Context, transactionID, expectationID, userID uuid.UUID) (*domain.PaymentMatch, error) {
	// Verify transaction exists
	_, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Verify expectation exists
	_, err = s.feeRepo.GetByID(ctx, expectationID)
	if err != nil {
		return nil, ErrNotFound
	}

	match := &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: transactionID,
		ExpectationID: expectationID,
		MatchType:     domain.MatchTypeManual,
		MatchedAt:     time.Now(),
		MatchedBy:     &userID,
	}

	if err := s.matchRepo.Create(ctx, match); err != nil {
		return nil, err
	}

	s.postMatchActions(ctx, transactionID, expectationID, "Auto-resolved: Zahlung wurde manuell zugeordnet")

	return match, nil
}

// Rescan re-scans all unmatched transactions for potential matches.
// High-confidence matches (95%+) are automatically confirmed.
func (s *ImportService) Rescan(ctx context.Context) (*RescanResult, error) {
	result := &RescanResult{}

	// Get all unmatched transactions
	transactions, _, err := s.transactionRepo.ListUnmatched(ctx, 0, 10000)
	if err != nil {
		return nil, err
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, "", "", "", 0, 1000)
	s.enrichChildrenWithParents(ctx, children)

	// Re-scan each transaction
	for _, tx := range transactions {
		result.Scanned++
		suggestion := s.matchTransaction(ctx, tx, children)
		if suggestion == nil {
			continue
		}

		// High confidence with matching fee expectation(s) -> auto-confirm
		if suggestion.Confidence >= autoMatchConfidenceThreshold && (suggestion.Expectation != nil || len(suggestion.Expectations) > 0) {
			autoMatched := s.autoConfirmMatch(ctx, suggestion)
			if autoMatched {
				result.AutoMatched++
				continue
			}
		}

		// Lower confidence or no auto-match -> add to suggestions for manual review
		result.Suggestions = append(result.Suggestions, *suggestion)
	}

	return result, nil
}

// autoConfirmMatch automatically confirms a high-confidence match.
// Returns true if the match was successfully confirmed.
func (s *ImportService) autoConfirmMatch(ctx context.Context, suggestion *domain.MatchSuggestion) bool {
	// Handle combined matches (fee + reminder)
	if len(suggestion.Expectations) > 0 {
		for _, fee := range suggestion.Expectations {
			match := &domain.PaymentMatch{
				ID:            uuid.New(),
				TransactionID: suggestion.Transaction.ID,
				ExpectationID: fee.ID,
				MatchType:     domain.MatchTypeAuto,
				Confidence:    &suggestion.Confidence,
				MatchedAt:     time.Now(),
				MatchedBy:     nil,
			}
			if err := s.matchRepo.Create(ctx, match); err != nil {
				return false
			}
		}
		s.postMatchActions(ctx, suggestion.Transaction.ID, uuid.Nil, "Auto-matched: Hohe Übereinstimmung (95%+)")
		return true
	}

	// Single fee match
	if suggestion.Expectation != nil {
		match := &domain.PaymentMatch{
			ID:            uuid.New(),
			TransactionID: suggestion.Transaction.ID,
			ExpectationID: suggestion.Expectation.ID,
			MatchType:     domain.MatchTypeAuto,
			Confidence:    &suggestion.Confidence,
			MatchedAt:     time.Now(),
			MatchedBy:     nil,
		}
		if err := s.matchRepo.Create(ctx, match); err != nil {
			return false
		}
		s.postMatchActions(ctx, suggestion.Transaction.ID, suggestion.Expectation.ID, "Auto-matched: Hohe Übereinstimmung (95%+)")
		return true
	}

	return false
}

// postMatchActions performs common actions after a match is created.
func (s *ImportService) postMatchActions(ctx context.Context, transactionID, feeID uuid.UUID, warningResolutionNote string) {
	s.markIBANAsTrusted(ctx, transactionID)

	if s.warningRepo != nil {
		s.warningRepo.ResolveByTransactionID(ctx, transactionID, domain.ResolutionTypeMatched, warningResolutionNote)
	}

	if feeID != uuid.Nil {
		s.checkLatePaymentAndCreateWarning(ctx, transactionID, feeID)
	}
}

// DismissTransaction dismisses a transaction and blacklists its IBAN.
func (s *ImportService) DismissTransaction(ctx context.Context, transactionID uuid.UUID) (*DismissResult, error) {
	// Get the transaction
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	if tx.PayerIBAN == nil {
		return nil, ErrInvalidInput
	}

	iban := *tx.PayerIBAN

	// Add to blacklist
	knownIBAN := &domain.KnownIBAN{
		IBAN:                  iban,
		PayerName:             tx.PayerName,
		Status:                domain.KnownIBANStatusBlacklisted,
		Reason:                stringPtr("User dismissed transaction"),
		OriginalTransactionID: &tx.ID,
		OriginalDescription:   tx.Description,
		OriginalAmount:        &tx.Amount,
	}

	if err := s.knownIBANRepo.Create(ctx, knownIBAN); err != nil {
		return nil, err
	}

	// Delete all unmatched transactions from this IBAN
	deleted, err := s.transactionRepo.DeleteUnmatchedByIBAN(ctx, iban)
	if err != nil {
		return nil, err
	}

	return &DismissResult{
		IBAN:                iban,
		TransactionsRemoved: deleted,
	}, nil
}

// GetBlacklist returns all blacklisted IBANs.
func (s *ImportService) GetBlacklist(ctx context.Context, offset, limit int) ([]domain.KnownIBAN, int64, error) {
	return s.knownIBANRepo.ListByStatus(ctx, domain.KnownIBANStatusBlacklisted, offset, limit)
}

// RemoveFromBlacklist removes an IBAN from the blacklist.
func (s *ImportService) RemoveFromBlacklist(ctx context.Context, iban string) error {
	existing, err := s.knownIBANRepo.GetByIBAN(ctx, iban)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	if existing.Status != domain.KnownIBANStatusBlacklisted {
		return ErrInvalidInput
	}

	return s.knownIBANRepo.Delete(ctx, iban)
}

// LinkIBANToChild links a trusted IBAN to a specific child.
func (s *ImportService) LinkIBANToChild(ctx context.Context, iban string, childID uuid.UUID) error {
	// Verify the IBAN exists and is trusted
	existing, err := s.knownIBANRepo.GetByIBAN(ctx, iban)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	if existing.Status != domain.KnownIBANStatusTrusted {
		return ErrInvalidInput
	}

	// Verify child exists
	_, err = s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return ErrNotFound
	}

	return s.knownIBANRepo.UpdateChildLink(ctx, iban, &childID)
}

// UnlinkIBANFromChild removes the child link from a trusted IBAN.
func (s *ImportService) UnlinkIBANFromChild(ctx context.Context, iban string) error {
	return s.knownIBANRepo.UpdateChildLink(ctx, iban, nil)
}

// GetTrustedIBANs returns all trusted IBANs.
func (s *ImportService) GetTrustedIBANs(ctx context.Context, offset, limit int) ([]domain.KnownIBAN, int64, error) {
	return s.knownIBANRepo.ListByStatus(ctx, domain.KnownIBANStatusTrusted, offset, limit)
}

// GetWarnings returns all unresolved transaction warnings with related entities.
func (s *ImportService) GetWarnings(ctx context.Context, offset, limit int) ([]domain.TransactionWarning, int64, error) {
	if s.warningRepo == nil {
		return nil, 0, nil
	}

	warnings, total, err := s.warningRepo.ListUnresolved(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Collect all IDs for batch loading
	transactionIDs := make([]uuid.UUID, 0, len(warnings))
	childIDs := make([]uuid.UUID, 0, len(warnings))
	feeIDs := make([]uuid.UUID, 0, len(warnings))

	for _, w := range warnings {
		transactionIDs = append(transactionIDs, w.TransactionID)
		if w.ChildID != nil {
			childIDs = append(childIDs, *w.ChildID)
		}
		if w.MatchedFeeID != nil {
			feeIDs = append(feeIDs, *w.MatchedFeeID)
		}
	}

	// Batch load all related entities (3 queries instead of N*3)
	transactionsMap, err := s.transactionRepo.GetByIDs(ctx, transactionIDs)
	if err != nil {
		return nil, 0, err
	}

	childrenMap, err := s.childRepo.GetByIDs(ctx, childIDs)
	if err != nil {
		return nil, 0, err
	}

	feesMap, err := s.feeRepo.GetByIDs(ctx, feeIDs)
	if err != nil {
		return nil, 0, err
	}

	// Collect additional child IDs from fees (for warnings without direct child link)
	additionalChildIDs := make([]uuid.UUID, 0)
	for _, w := range warnings {
		if w.ChildID == nil && w.MatchedFeeID != nil {
			if fee, ok := feesMap[*w.MatchedFeeID]; ok && fee.ChildID != uuid.Nil {
				if _, exists := childrenMap[fee.ChildID]; !exists {
					additionalChildIDs = append(additionalChildIDs, fee.ChildID)
				}
			}
		}
	}

	// Load additional children if needed
	if len(additionalChildIDs) > 0 {
		additionalChildren, err := s.childRepo.GetByIDs(ctx, additionalChildIDs)
		if err != nil {
			return nil, 0, err
		}
		for id, child := range additionalChildren {
			childrenMap[id] = child
		}
	}

	// Enrich warnings with loaded entities
	for i := range warnings {
		if tx, ok := transactionsMap[warnings[i].TransactionID]; ok {
			warnings[i].Transaction = tx
		}

		if warnings[i].ChildID != nil {
			if child, ok := childrenMap[*warnings[i].ChildID]; ok {
				warnings[i].Child = child
			}
		}

		if warnings[i].MatchedFeeID != nil {
			if fee, ok := feesMap[*warnings[i].MatchedFeeID]; ok {
				warnings[i].MatchedFee = fee
				// Also set child from fee if not already set
				if warnings[i].Child == nil && fee.ChildID != uuid.Nil {
					if child, ok := childrenMap[fee.ChildID]; ok {
						warnings[i].Child = child
					}
				}
			}
		}
	}

	return warnings, total, nil
}

// GetWarningByID returns a warning by its ID.
func (s *ImportService) GetWarningByID(ctx context.Context, id uuid.UUID) (*domain.TransactionWarning, error) {
	if s.warningRepo == nil {
		return nil, ErrNotFound
	}
	return s.warningRepo.GetByID(ctx, id)
}

// DismissWarning dismisses a warning with a note.
func (s *ImportService) DismissWarning(ctx context.Context, id uuid.UUID, userID uuid.UUID, note string) error {
	if s.warningRepo == nil {
		return ErrNotFound
	}
	return s.warningRepo.Resolve(ctx, id, userID, domain.ResolutionTypeDismissed, note)
}

func stringPtr(s string) *string {
	return &s
}

// isLatePayment checks if a payment is late based on the fee's month and payment date.
// Returns true if the payment is late.
func isLatePayment(fee *domain.FeeExpectation, paymentDate time.Time) bool {
	// Only monthly fees (CHILDCARE, FOOD) can be late - not MEMBERSHIP
	if fee.FeeType == domain.FeeTypeMembership || fee.FeeType == domain.FeeTypeReminder {
		return false
	}

	// Fee must have a month (monthly fees)
	if fee.Month == nil {
		return false
	}

	feeMonth := *fee.Month
	feeYear := fee.Year

	deadline := time.Date(feeYear, time.Month(feeMonth), latePaymentDayThreshold, 23, 59, 59, 0, time.UTC)

	// Payment is late if it's after the deadline
	return paymentDate.After(deadline)
}

// checkLatePaymentAndCreateWarning checks if a match is late and creates a warning if so.
func (s *ImportService) checkLatePaymentAndCreateWarning(ctx context.Context, transactionID, feeID uuid.UUID) error {
	if s.warningRepo == nil {
		return nil
	}

	// Get the transaction
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil // Don't fail the match, just skip warning
	}

	// Get the fee
	fee, err := s.feeRepo.GetByID(ctx, feeID)
	if err != nil {
		return nil
	}

	// Check if late
	if !isLatePayment(fee, tx.BookingDate) {
		return nil
	}

	// Create LATE_PAYMENT warning
	warning := &domain.TransactionWarning{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		WarningType:   domain.WarningTypeLatePayment,
		Message:       fmt.Sprintf("Zahlung nach dem 15. des Monats (%s %d)", time.Month(*fee.Month).String(), fee.Year),
		ActualAmount:  &tx.Amount,
		ChildID:       &fee.ChildID,
		MatchedFeeID:  &fee.ID,
		CreatedAt:     time.Now(),
	}

	return s.warningRepo.Create(ctx, warning)
}

// LateFeeResolution contains the result of resolving a late payment warning.
type LateFeeResolution struct {
	WarningID     uuid.UUID `json:"warningId"`
	LateFeeID     uuid.UUID `json:"lateFeeId"`
	LateFeeAmount float64   `json:"lateFeeAmount"`
}

// ResolveWarningWithLateFee resolves a LATE_PAYMENT warning by creating a REMINDER fee.
// The REMINDER fee is linked to the original fee and is for 10 EUR.
func (s *ImportService) ResolveWarningWithLateFee(ctx context.Context, warningID, userID uuid.UUID) (*LateFeeResolution, error) {
	if s.warningRepo == nil {
		return nil, ErrNotFound
	}

	// Get the warning
	warning, err := s.warningRepo.GetByID(ctx, warningID)
	if err != nil {
		return nil, err
	}

	// Must be a LATE_PAYMENT warning
	if warning.WarningType != domain.WarningTypeLatePayment {
		return nil, ErrInvalidInput
	}

	// Must not already be resolved
	if warning.ResolvedAt != nil {
		return nil, ErrInvalidInput
	}

	// Must have a matched fee
	if warning.MatchedFeeID == nil {
		return nil, ErrInvalidInput
	}

	// Get the original fee
	originalFee, err := s.feeRepo.GetByID(ctx, *warning.MatchedFeeID)
	if err != nil {
		return nil, err
	}

	// Create REMINDER fee (10 EUR late fee)
	reminderFee := &domain.FeeExpectation{
		ID:            uuid.New(),
		ChildID:       originalFee.ChildID,
		FeeType:       domain.FeeTypeReminder,
		Year:          originalFee.Year,
		Month:         originalFee.Month, // Same month as original fee
		Amount:        domain.ReminderFeeAmount,
		DueDate:       time.Now().AddDate(0, 0, 14), // Due in 14 days
		CreatedAt:     time.Now(),
		ReminderForID: &originalFee.ID, // Link to original fee
	}

	if err := s.feeRepo.Create(ctx, reminderFee); err != nil {
		return nil, err
	}

	// Resolve the warning
	note := fmt.Sprintf("Mahngebühr von %.2f EUR erstellt", domain.ReminderFeeAmount)
	if err := s.warningRepo.Resolve(ctx, warningID, userID, domain.ResolutionTypeMatched, note); err != nil {
		return nil, err
	}

	return &LateFeeResolution{
		WarningID:     warningID,
		LateFeeID:     reminderFee.ID,
		LateFeeAmount: domain.ReminderFeeAmount,
	}, nil
}

// GetMatchedTransactions returns transactions that have been matched to fees.
func (s *ImportService) GetMatchedTransactions(ctx context.Context, offset, limit int) ([]domain.BankTransaction, int64, error) {
	return s.transactionRepo.ListMatched(ctx, offset, limit)
}

// GetSuggestionsForTransaction returns match suggestions for a single transaction.
func (s *ImportService) GetSuggestionsForTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.MatchSuggestion, error) {
	// Get the transaction
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, "", "", "", 0, 1000)
	s.enrichChildrenWithParents(ctx, children)

	// Run matching algorithm
	suggestion := s.matchTransaction(ctx, *tx, children)
	if suggestion == nil {
		// No automatic match found, return transaction info only
		return &domain.MatchSuggestion{
			Transaction: *tx,
			Confidence:  0,
			MatchedBy:   "none",
		}, nil
	}

	return suggestion, nil
}
