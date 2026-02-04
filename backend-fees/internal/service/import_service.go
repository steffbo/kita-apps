package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

const (
	// Fee combination amounts
	foodWithReminderAmount       = domain.FoodFeeAmount + domain.ReminderFeeAmount                 // 55.40
	membershipWithReminderAmount = domain.MembershipFeeAmount + domain.MembershipReminderFeeAmount // 35.00

	// Late payment threshold: 15th day of the month
	latePaymentDayThreshold = 15

	// Confidence thresholds
	autoMatchConfidenceThreshold = 0.95
	memberNumberConfidence       = 0.95
	trustedIBANConfidence        = 0.99
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
	AutoMatched int                         `json:"autoMatched"`
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

// HideResult represents the result of hiding a transaction.
type HideResult struct {
	TransactionID uuid.UUID `json:"transactionId"`
}

// UnmatchResult represents the result of unmatching a transaction.
type UnmatchResult struct {
	TransactionID      uuid.UUID `json:"transactionId"`
	MatchesRemoved     int64     `json:"matchesRemoved"`
	TransactionDeleted bool      `json:"transactionDeleted"`
}

// ChildUnmatchedSuggestionsResult represents likely unmatched transactions for a child.
type ChildUnmatchedSuggestionsResult struct {
	ChildID     uuid.UUID               `json:"childId"`
	Scanned     int                     `json:"scanned"`
	Suggestions []domain.MatchSuggestion `json:"suggestions"`
}

// AllocationInput represents a manual allocation for a transaction.
type AllocationInput struct {
	ExpectationID uuid.UUID
	Amount        float64
}

// AllocateResult represents the result of allocating a transaction.
type AllocateResult struct {
	TransactionID      uuid.UUID `json:"transactionId"`
	AllocationsCreated int       `json:"allocationsCreated"`
	TotalAllocated     float64   `json:"totalAllocated"`
	Overpayment        float64   `json:"overpayment"`
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
	children, _, _ := s.childRepo.List(ctx, true, false, false, false, "", "", "", 0, 1000)
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
		suggestion, warning := s.matchTransaction(ctx, tx, children)
		if suggestion != nil {
			// High confidence with matching fee expectation(s) -> auto-confirm
			if suggestion.Confidence >= autoMatchConfidenceThreshold && (suggestion.Expectation != nil || len(suggestion.Expectations) > 0) {
				if s.autoConfirmMatch(ctx, suggestion) {
					result.AutoMatched++
					continue
				}
			}

			result.Suggestions = append(result.Suggestions, *suggestion)
		} else if warning != nil {
			s.saveWarning(ctx, warning, result)
		} else {
			// No match - check if trusted IBAN needs a warning
			if w := s.checkForWarning(ctx, tx); w != nil {
				s.saveWarning(ctx, w, result)
			}
		}
	}

	return result, nil
}

func (s *ImportService) saveWarning(ctx context.Context, warning *domain.TransactionWarning, result *ImportResult) {
	// Add to result list for frontend display during import
	result.WarningList = append(result.WarningList, *warning)

	// Persist to database except for MULTIPLE_OPEN_FEES (computed on-the-fly)
	if s.warningRepo != nil && warning.WarningType != domain.WarningTypeMultipleOpenFees {
		if err := s.warningRepo.Create(ctx, warning); err == nil {
			result.Warnings++
		}
	}
}

func (s *ImportService) matchTransaction(ctx context.Context, tx domain.BankTransaction, children []domain.Child) (*domain.MatchSuggestion, *domain.TransactionWarning) {
	suggestion := &domain.MatchSuggestion{
		Transaction:  tx,
		DetectedType: s.detectFeeType(tx.Amount),
	}

	s.matchTrustedIBAN(ctx, tx, children, suggestion)
	if suggestion.Child == nil {
		s.matchChild(buildMatchText(tx), children, suggestion)
	}

	if suggestion.Child != nil && suggestion.DetectedType != nil {
		if warning := s.matchFeeExpectation(ctx, tx, suggestion); warning != nil {
			return nil, warning
		}
	}

	// Boost confidence for name-based matches with fee expectations
	if suggestion.Expectation != nil && (suggestion.MatchedBy == "name" || suggestion.MatchedBy == "parent_name") {
		suggestion.Confidence = min(suggestion.Confidence+confidenceBoostNameMatch, maxConfidenceNameMatch)
	}

	if suggestion.Confidence > 0 {
		return suggestion, nil
	}
	return nil, nil
}

func (s *ImportService) matchTrustedIBAN(ctx context.Context, tx domain.BankTransaction, children []domain.Child, suggestion *domain.MatchSuggestion) {
	if s.knownIBANRepo == nil || tx.PayerIBAN == nil {
		return
	}

	knownIBAN, err := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
	if err != nil || knownIBAN == nil || knownIBAN.Status != domain.KnownIBANStatusTrusted || knownIBAN.ChildID == nil {
		return
	}

	childID := *knownIBAN.ChildID
	for i := range children {
		if children[i].ID == childID {
			suggestion.Child = &children[i]
			suggestion.MatchedBy = "trusted_iban"
			suggestion.Confidence = trustedIBANConfidence
			return
		}
	}

	if s.childRepo != nil {
		child, err := s.childRepo.GetByID(ctx, childID)
		if err == nil && child != nil {
			suggestion.Child = child
			suggestion.MatchedBy = "trusted_iban"
			suggestion.Confidence = trustedIBANConfidence
		}
	}
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

func (s *ImportService) matchFeeExpectation(ctx context.Context, tx domain.BankTransaction, suggestion *domain.MatchSuggestion) *domain.TransactionWarning {
	childID := suggestion.Child.ID
	feeType := *suggestion.DetectedType

	// Check how many unpaid fees exist with this exact amount
	count, err := s.feeRepo.CountUnpaidByType(ctx, childID, feeType, tx.Amount)
	if err != nil {
		return nil
	}

	// Multiple fees with same amount -> manual review required
	if count > 1 {
		return &domain.TransactionWarning{
			ID:            uuid.New(),
			TransactionID: tx.ID,
			WarningType:   domain.WarningTypeMultipleOpenFees,
			Message:       fmt.Sprintf("Mehrere offene Beiträge (%d) für dieses Kind gefunden - manuelle Zuordnung erforderlich", count),
			ActualAmount:  &tx.Amount,
			ChildID:       &childID,
			CreatedAt:     time.Now(),
		}
	}

	// Exactly one fee -> auto-match
	if count == 1 {
		if fee, err := s.feeRepo.FindBestUnpaid(ctx, childID, feeType, tx.Amount, tx.BookingDate); err == nil && fee != nil {
			suggestion.Expectation = fee
		}
		return nil
	}

	// No exact match -> try combined fee + reminder (e.g., 55.40 = 45.40 + 10.00)
	if fees, err := s.feeRepo.FindOldestUnpaidWithReminder(ctx, childID, feeType, tx.Amount); err == nil && len(fees) == 2 {
		suggestion.Expectations = fees
		suggestion.Expectation = &fees[0]
		suggestion.MatchedBy = "combined"
		if suggestion.Confidence > 0 {
			suggestion.Confidence = min(suggestion.Confidence+confidenceBoostCombined, maxConfidenceCombined)
		}
	}
	return nil
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
		fee, err := s.feeRepo.GetByID(ctx, m.ExpectationID)
		if err != nil {
			result.Failed++
			continue
		}
		match := &domain.PaymentMatch{
			ID:            uuid.New(),
			TransactionID: m.TransactionID,
			ExpectationID: m.ExpectationID,
			Amount:        fee.Amount,
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
func (s *ImportService) GetUnmatchedTransactions(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
	return s.transactionRepo.ListUnmatched(ctx, search, sortBy, sortDir, offset, limit)
}

// CreateManualMatch creates a manual match between transaction and fee.
func (s *ImportService) CreateManualMatch(ctx context.Context, transactionID, expectationID, userID uuid.UUID) (*domain.PaymentMatch, error) {
	// Verify transaction exists
	_, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Verify expectation exists
	fee, err := s.feeRepo.GetByID(ctx, expectationID)
	if err != nil {
		return nil, ErrNotFound
	}

	match := &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: transactionID,
		ExpectationID: expectationID,
		Amount:        fee.Amount,
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

	// Get all unmatched transactions (no search/sort, just get all)
	transactions, _, err := s.transactionRepo.ListUnmatched(ctx, "", "date", "desc", 0, 10000)
	if err != nil {
		return nil, err
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, false, "", "", "", 0, 1000)
	s.enrichChildrenWithParents(ctx, children)

	// Re-scan each transaction
	for _, tx := range transactions {
		result.Scanned++
		suggestion, warning := s.matchTransaction(ctx, tx, children)

		if warning != nil {
			if s.warningRepo != nil {
				_ = s.warningRepo.Create(ctx, warning)
			}
			continue
		}

		if suggestion == nil {
			continue
		}

		// High confidence with matching fee expectation(s) -> auto-confirm
		if suggestion.Confidence >= autoMatchConfidenceThreshold && (suggestion.Expectation != nil || len(suggestion.Expectations) > 0) {
			if s.autoConfirmMatch(ctx, suggestion) {
				result.AutoMatched++
				continue
			}
		}

		result.Suggestions = append(result.Suggestions, *suggestion)
	}

	return result, nil
}

// autoConfirmMatch automatically confirms a high-confidence match.
// Returns true if the match was successfully confirmed.
func (s *ImportService) autoConfirmMatch(ctx context.Context, suggestion *domain.MatchSuggestion) bool {
	// Handle combined matches (fee + reminder)
	if len(suggestion.Expectations) > 0 {
		expectationIDs := make([]string, 0, len(suggestion.Expectations))
		for _, fee := range suggestion.Expectations {
			expectationIDs = append(expectationIDs, fee.ID.String())
			match := &domain.PaymentMatch{
				ID:            uuid.New(),
				TransactionID: suggestion.Transaction.ID,
				ExpectationID: fee.ID,
				Amount:        fee.Amount,
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
		log.Info().
			Str("transactionId", suggestion.Transaction.ID.String()).
			Float64("confidence", suggestion.Confidence).
			Str("matchedBy", suggestion.MatchedBy).
			Int("expectationCount", len(suggestion.Expectations)).
			Strs("expectationIds", expectationIDs).
			Msg("auto-matched transaction (high confidence)")
		return true
	}

	// Single fee match
	if suggestion.Expectation != nil {
		match := &domain.PaymentMatch{
			ID:            uuid.New(),
			TransactionID: suggestion.Transaction.ID,
			ExpectationID: suggestion.Expectation.ID,
			Amount:        suggestion.Expectation.Amount,
			MatchType:     domain.MatchTypeAuto,
			Confidence:    &suggestion.Confidence,
			MatchedAt:     time.Now(),
			MatchedBy:     nil,
		}
		if err := s.matchRepo.Create(ctx, match); err != nil {
			return false
		}
		s.postMatchActions(ctx, suggestion.Transaction.ID, suggestion.Expectation.ID, "Auto-matched: Hohe Übereinstimmung (95%+)")
		log.Info().
			Str("transactionId", suggestion.Transaction.ID.String()).
			Float64("confidence", suggestion.Confidence).
			Str("matchedBy", suggestion.MatchedBy).
			Int("expectationCount", 1).
			Strs("expectationIds", []string{suggestion.Expectation.ID.String()}).
			Msg("auto-matched transaction (high confidence)")
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

// HideTransaction marks a transaction as hidden (no blacklist).
func (s *ImportService) HideTransaction(ctx context.Context, transactionID uuid.UUID, userID uuid.UUID) (*HideResult, error) {
	// Ensure transaction exists
	if _, err := s.transactionRepo.GetByID(ctx, transactionID); err != nil {
		return nil, ErrNotFound
	}

	if err := s.transactionRepo.Hide(ctx, transactionID, userID); err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &HideResult{TransactionID: transactionID}, nil
}

// AllocateTransaction allocates a transaction across multiple fee expectations.
func (s *ImportService) AllocateTransaction(ctx context.Context, transactionID, userID uuid.UUID, allocations []AllocationInput) (*AllocateResult, error) {
	const epsilon = 0.01

	if len(allocations) == 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Ensure transaction has no existing matches
	exists, err := s.matchRepo.ExistsForTransaction(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrInvalidInput
	}

	var childID *uuid.UUID
	var totalAllocated float64
	fees := make(map[uuid.UUID]*domain.FeeExpectation, len(allocations))

	for _, alloc := range allocations {
		if alloc.Amount <= 0 {
			return nil, ErrInvalidInput
		}

		fee, err := s.feeRepo.GetByID(ctx, alloc.ExpectationID)
		if err != nil {
			return nil, ErrNotFound
		}
		fees[alloc.ExpectationID] = fee

		if childID == nil {
			id := fee.ChildID
			childID = &id
		} else if fee.ChildID != *childID {
			return nil, ErrInvalidInput
		}

		matchedAmount, err := s.matchRepo.GetTotalMatchedAmount(ctx, fee.ID)
		if err != nil {
			return nil, err
		}
		remaining := fee.Amount - matchedAmount
		if remaining <= epsilon {
			return nil, ErrInvalidInput
		}
		if alloc.Amount-remaining > epsilon {
			return nil, ErrInvalidInput
		}

		totalAllocated += alloc.Amount
	}

	if totalAllocated-tx.Amount > epsilon {
		return nil, ErrInvalidInput
	}

	overpayment := tx.Amount - totalAllocated
	if overpayment < 0 {
		overpayment = 0
	}

	result := &AllocateResult{
		TransactionID:      transactionID,
		TotalAllocated:     totalAllocated,
		Overpayment:        overpayment,
		AllocationsCreated: 0,
	}

	for _, alloc := range allocations {
		match := &domain.PaymentMatch{
			ID:            uuid.New(),
			TransactionID: transactionID,
			ExpectationID: alloc.ExpectationID,
			Amount:        alloc.Amount,
			MatchType:     domain.MatchTypeManual,
			MatchedAt:     time.Now(),
			MatchedBy:     &userID,
		}

		if err := s.matchRepo.Create(ctx, match); err != nil {
			return nil, err
		}
		result.AllocationsCreated++
	}

	// Post-match actions
	s.markIBANAsTrusted(ctx, transactionID)
	if s.warningRepo != nil {
		s.warningRepo.ResolveByTransactionID(ctx, transactionID, domain.ResolutionTypeMatched, "Zahlung wurde manuell verteilt")
	}
	for _, alloc := range allocations {
		s.checkLatePaymentAndCreateWarning(ctx, transactionID, alloc.ExpectationID)
	}

	// Create overpayment warning if any remainder exists
	if result.Overpayment > epsilon && s.warningRepo != nil && childID != nil {
		warning := &domain.TransactionWarning{
			ID:             uuid.New(),
			TransactionID:  tx.ID,
			WarningType:    domain.WarningTypeOverpayment,
			Message:        fmt.Sprintf("Überzahlung: %.2f EUR nicht zugeordnet", result.Overpayment),
			ExpectedAmount: &totalAllocated,
			ActualAmount:   &tx.Amount,
			ChildID:        childID,
			CreatedAt:      time.Now(),
		}
		_ = s.warningRepo.Create(ctx, warning)
	}

	return result, nil
}

// GetUnmatchedSuggestionsForChild returns likely unmatched transactions for a specific child.
func (s *ImportService) GetUnmatchedSuggestionsForChild(ctx context.Context, childID uuid.UUID, minConfidence float64, limit int) (*ChildUnmatchedSuggestionsResult, error) {
	if limit < 1 {
		limit = 10
	}
	if minConfidence < 0 {
		minConfidence = 0
	}
	if minConfidence > 1 {
		minConfidence = 1
	}

	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return nil, ErrNotFound
	}

	children := []domain.Child{*child}
	s.enrichChildrenWithParents(ctx, children)

	scanLimit := 500
	if limit > scanLimit {
		scanLimit = limit
	}

	transactions, _, err := s.transactionRepo.ListUnmatched(ctx, "", "date", "desc", 0, scanLimit)
	if err != nil {
		return nil, err
	}

	result := &ChildUnmatchedSuggestionsResult{
		ChildID: childID,
	}

	for _, tx := range transactions {
		result.Scanned++
		suggestion := &domain.MatchSuggestion{
			Transaction:  tx,
			DetectedType: s.detectFeeType(tx.Amount),
		}

		s.matchChild(buildMatchText(tx), children, suggestion)
		if suggestion.Child == nil || suggestion.Child.ID != childID {
			continue
		}

		if suggestion.DetectedType != nil {
			_ = s.matchFeeExpectation(ctx, tx, suggestion)
		}

		// Boost confidence for name-based matches with fee expectations
		if suggestion.Expectation != nil && (suggestion.MatchedBy == "name" || suggestion.MatchedBy == "parent_name") {
			suggestion.Confidence = min(suggestion.Confidence+confidenceBoostNameMatch, maxConfidenceNameMatch)
		}

		if suggestion.Confidence < minConfidence {
			continue
		}

		result.Suggestions = append(result.Suggestions, *suggestion)
		if len(result.Suggestions) >= limit {
			break
		}
	}

	return result, nil
}

// UnmatchTransaction removes matches for a transaction and optionally deletes the transaction itself.
func (s *ImportService) UnmatchTransaction(ctx context.Context, transactionID uuid.UUID, deleteTransaction bool) (*UnmatchResult, error) {
	// Ensure transaction exists
	if _, err := s.transactionRepo.GetByID(ctx, transactionID); err != nil {
		return nil, ErrNotFound
	}

	if deleteTransaction {
		matchesByTx, err := s.matchRepo.GetByTransactionIDs(ctx, []uuid.UUID{transactionID})
		if err != nil {
			return nil, err
		}
		matchesRemoved := int64(len(matchesByTx[transactionID]))

		if err := s.transactionRepo.Delete(ctx, transactionID); err != nil {
			if err == repository.ErrNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}

		return &UnmatchResult{
			TransactionID:      transactionID,
			MatchesRemoved:     matchesRemoved,
			TransactionDeleted: true,
		}, nil
	}

	matchesRemoved, err := s.matchRepo.DeleteByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if matchesRemoved == 0 {
		return nil, ErrInvalidInput
	}

	return &UnmatchResult{
		TransactionID:      transactionID,
		MatchesRemoved:     matchesRemoved,
		TransactionDeleted: false,
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

// GetTrustedIBANsForChild returns trusted IBANs for a child with usage counts.
func (s *ImportService) GetTrustedIBANsForChild(ctx context.Context, childID uuid.UUID) ([]domain.KnownIBANSummary, error) {
	if s.knownIBANRepo == nil {
		return []domain.KnownIBANSummary{}, nil
	}
	return s.knownIBANRepo.ListTrustedByChildWithCounts(ctx, childID)
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
func (s *ImportService) GetMatchedTransactions(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
	transactions, total, err := s.transactionRepo.ListMatched(ctx, search, sortBy, sortDir, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// If no transactions, return early
	if len(transactions) == 0 {
		return transactions, total, nil
	}

	// Get transaction IDs
	txIDs := make([]uuid.UUID, len(transactions))
	for i, tx := range transactions {
		txIDs[i] = tx.ID
	}

	// Fetch matches for all transactions
	matchesByTxID, err := s.matchRepo.GetByTransactionIDs(ctx, txIDs)
	if err != nil {
		return nil, 0, err
	}

	// Collect all expectation IDs from matches
	expectationIDs := make([]uuid.UUID, 0)
	for _, matches := range matchesByTxID {
		for _, m := range matches {
			expectationIDs = append(expectationIDs, m.ExpectationID)
		}
	}

	// Fetch all fee expectations
	var expectationsMap map[uuid.UUID]*domain.FeeExpectation
	if len(expectationIDs) > 0 {
		expectationsMap, err = s.feeRepo.GetByIDs(ctx, expectationIDs)
		if err != nil {
			return nil, 0, err
		}
	} else {
		expectationsMap = make(map[uuid.UUID]*domain.FeeExpectation)
	}

	// Attach matches with expectations to transactions
	for i := range transactions {
		if matches, ok := matchesByTxID[transactions[i].ID]; ok {
			for j := range matches {
				if exp, ok := expectationsMap[matches[j].ExpectationID]; ok {
					matches[j].Expectation = exp
				}
			}
			transactions[i].Matches = matches
		}
	}

	return transactions, total, nil
}

// GetSuggestionsForTransaction returns match suggestions for a single transaction.
func (s *ImportService) GetSuggestionsForTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.MatchSuggestion, error) {
	// Get the transaction
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, false, "", "", "", 0, 1000)
	s.enrichChildrenWithParents(ctx, children)

	// Run matching algorithm
	suggestion, warning := s.matchTransaction(ctx, *tx, children)
	if suggestion == nil && warning != nil && warning.WarningType == domain.WarningTypeMultipleOpenFees {
		// Provide child/type confidence even when multiple open fees exist,
		// so the manual matching UI can still surface high-confidence candidates.
		fallback := &domain.MatchSuggestion{
			Transaction:  *tx,
			DetectedType: s.detectFeeType(tx.Amount),
		}
		s.matchTrustedIBAN(ctx, *tx, children, fallback)
		if fallback.Child == nil {
			s.matchChild(buildMatchText(*tx), children, fallback)
		}
		if fallback.Confidence > 0 {
			return fallback, nil
		}
	}
	if suggestion == nil {
		return &domain.MatchSuggestion{
			Transaction: *tx,
			MatchedBy:   "none",
		}, nil
	}
	return suggestion, nil
}
