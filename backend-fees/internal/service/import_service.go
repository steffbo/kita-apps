package service

import (
	"context"
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
}

// NewImportService creates a new import service.
func NewImportService(
	transactionRepo repository.TransactionRepository,
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	matchRepo repository.MatchRepository,
	knownIBANRepo repository.KnownIBANRepository,
) *ImportService {
	return &ImportService{
		transactionRepo: transactionRepo,
		feeRepo:         feeRepo,
		childRepo:       childRepo,
		matchRepo:       matchRepo,
		knownIBANRepo:   knownIBANRepo,
	}
}

// ImportResult represents the result of a CSV import.
type ImportResult struct {
	BatchID     uuid.UUID                `json:"batchId"`
	FileName    string                   `json:"fileName"`
	TotalRows   int                      `json:"totalRows"`
	Imported    int                      `json:"imported"`
	Skipped     int                      `json:"skipped"`
	Blacklisted int                      `json:"blacklisted"`
	Suggestions []domain.MatchSuggestion `json:"suggestions"`
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
	children, _, _ := s.childRepo.List(ctx, true, "", 0, 1000)

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
	switch tx.Amount {
	case domain.FoodFeeAmount:
		feeType := domain.FeeTypeFood
		suggestion.DetectedType = &feeType
	case domain.MembershipFeeAmount:
		feeType := domain.FeeTypeMembership
		suggestion.DetectedType = &feeType
	default:
		// Could be childcare fee
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
		year := tx.BookingDate.Year()
		var month *int
		if *suggestion.DetectedType != domain.FeeTypeMembership {
			m := int(tx.BookingDate.Month())
			month = &m
		}

		fee, err := s.feeRepo.FindUnpaid(ctx, suggestion.Child.ID, *suggestion.DetectedType, year, month)
		if err == nil && fee != nil {
			suggestion.Expectation = fee
		}
	}

	// Only return if we have some confidence
	if suggestion.Confidence > 0 {
		return suggestion
	}

	return nil
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
	children, _, _ := s.childRepo.List(ctx, true, "", 0, 1000)

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

func stringPtr(s string) *string {
	return &s
}
