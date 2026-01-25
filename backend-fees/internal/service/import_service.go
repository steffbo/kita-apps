package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
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

	// Get blacklisted IBANs for efficient filtering
	blacklistedIBANs, err := s.knownIBANRepo.GetBlacklistedIBANs(ctx)
	if err != nil {
		// Log but don't fail - continue without blacklist filtering
		blacklistedIBANs = make(map[string]bool)
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, "", "", "", 0, 1000)

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
	// Note: Combined amounts (fee + reminder) are handled separately below
	switch tx.Amount {
	case domain.FoodFeeAmount:
		feeType := domain.FeeTypeFood
		suggestion.DetectedType = &feeType
	case domain.MembershipFeeAmount:
		feeType := domain.FeeTypeMembership
		suggestion.DetectedType = &feeType
	case domain.FoodFeeAmount + domain.ReminderFeeAmount: // 55.40 = food + reminder
		feeType := domain.FeeTypeFood
		suggestion.DetectedType = &feeType
	case domain.MembershipFeeAmount + domain.ReminderFeeAmount: // 40.00 = membership + reminder
		feeType := domain.FeeTypeMembership
		suggestion.DetectedType = &feeType
	default:
		// Could be childcare fee (or childcare + reminder)
		feeType := domain.FeeTypeChildcare
		suggestion.DetectedType = &feeType
	}

	description := ""
	if tx.Description != nil {
		description = *tx.Description
	}

	// Try to match by member number
	memberNumber := csvparser.ExtractMemberNumber(description)
	if memberNumber != "" {
		for i := range children {
			if children[i].MemberNumber == memberNumber {
				suggestion.Child = &children[i]
				suggestion.MatchedBy = "member_number"
				suggestion.Confidence = 0.95
				break
			}
		}
	}

	// If no match by member number, try by name
	if suggestion.Child == nil {
		matchedChild, confidence := csvparser.MatchChildByName(description, children)
		if matchedChild != nil {
			suggestion.Child = matchedChild
			suggestion.MatchedBy = "name"
			suggestion.Confidence = confidence
		}
	}

	// If we found a child, try to find the corresponding fee expectation
	if suggestion.Child != nil && suggestion.DetectedType != nil {
		// First try exact amount match (oldest unpaid)
		fee, err := s.feeRepo.FindOldestUnpaid(ctx, suggestion.Child.ID, *suggestion.DetectedType, tx.Amount)
		if err == nil && fee != nil {
			suggestion.Expectation = fee
		} else {
			// No exact match found - try combined fee + reminder match
			// This handles cases like 55.40 EUR = 45.40 food + 10.00 reminder
			fees, err := s.feeRepo.FindOldestUnpaidWithReminder(ctx, suggestion.Child.ID, *suggestion.DetectedType, tx.Amount)
			if err == nil && len(fees) == 2 {
				suggestion.Expectations = fees
				suggestion.Expectation = &fees[0] // Primary fee for backward compatibility
				suggestion.MatchedBy = "combined"
				// Boost confidence slightly for combined matches since amount is precise
				if suggestion.Confidence > 0 {
					suggestion.Confidence = min(suggestion.Confidence+0.02, 0.99)
				}
			}
		}
	}

	// Only return if we have some confidence
	if suggestion.Confidence > 0 {
		return suggestion
	}

	return nil
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

		// Mark IBAN as trusted when match is confirmed
		s.markIBANAsTrusted(ctx, m.TransactionID)

		// Auto-resolve any warning for this transaction
		if s.warningRepo != nil {
			s.warningRepo.ResolveByTransactionID(ctx, m.TransactionID, domain.ResolutionTypeMatched, "Auto-resolved: Zahlung wurde zugeordnet")
		}
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

	// Mark IBAN as trusted
	s.markIBANAsTrusted(ctx, transactionID)

	// Auto-resolve any warning for this transaction
	if s.warningRepo != nil {
		s.warningRepo.ResolveByTransactionID(ctx, transactionID, domain.ResolutionTypeMatched, "Auto-resolved: Zahlung wurde manuell zugeordnet")
	}

	return match, nil
}

// Rescan re-scans all unmatched transactions for potential matches.
func (s *ImportService) Rescan(ctx context.Context) (*RescanResult, error) {
	result := &RescanResult{}

	// Get all unmatched transactions
	transactions, _, err := s.transactionRepo.ListUnmatched(ctx, 0, 10000)
	if err != nil {
		return nil, err
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, "", "", "", 0, 1000)

	// Re-scan each transaction
	for _, tx := range transactions {
		result.Scanned++
		suggestion := s.matchTransaction(ctx, tx, children)
		if suggestion != nil {
			result.Suggestions = append(result.Suggestions, *suggestion)
		}
	}

	return result, nil
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

// GetWarnings returns all unresolved transaction warnings.
func (s *ImportService) GetWarnings(ctx context.Context, offset, limit int) ([]domain.TransactionWarning, int64, error) {
	if s.warningRepo == nil {
		return nil, 0, nil
	}
	return s.warningRepo.ListUnresolved(ctx, offset, limit)
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
