package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

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

		err = s.txm.WithTx(ctx, func(ctx context.Context) error {
			if err := s.matchRepo.Create(ctx, match); err != nil {
				return err
			}
			return s.postMatchActions(ctx, m.TransactionID, m.ExpectationID, &fee.ChildID, "Auto-resolved: Zahlung wurde zugeordnet")
		})
		if err != nil {
			log.Warn().Err(err).Str("transactionId", m.TransactionID.String()).Str("expectationId", m.ExpectationID.String()).Msg("confirm match failed")
			result.Failed++
			continue
		}
		result.Confirmed++
	}

	return result, nil
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

	err = s.txm.WithTx(ctx, func(ctx context.Context) error {
		if err := s.matchRepo.Create(ctx, match); err != nil {
			return err
		}
		return s.postMatchActions(ctx, transactionID, expectationID, &fee.ChildID, "Auto-resolved: Zahlung wurde manuell zugeordnet")
	})
	if err != nil {
		return nil, err
	}

	return match, nil
}

// autoConfirmMatch automatically confirms a high-confidence match.
// All matches and follow-up actions are written atomically.
func (s *ImportService) autoConfirmMatch(ctx context.Context, suggestion *domain.MatchSuggestion) error {
	// Handle combined matches (fee + reminder)
	if len(suggestion.Expectations) > 0 {
		expectationIDs := make([]string, 0, len(suggestion.Expectations))
		err := s.txm.WithTx(ctx, func(ctx context.Context) error {
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
					return err
				}
			}
			childID := suggestion.Expectations[0].ChildID
			return s.postMatchActions(ctx, suggestion.Transaction.ID, uuid.Nil, &childID, "Auto-matched: Hohe Übereinstimmung (95%+)")
		})
		if err != nil {
			log.Warn().Err(err).Str("transactionId", suggestion.Transaction.ID.String()).Msg("auto-match failed, keeping as suggestion")
			return err
		}
		log.Info().
			Str("transactionId", suggestion.Transaction.ID.String()).
			Float64("confidence", suggestion.Confidence).
			Str("matchedBy", suggestion.MatchedBy).
			Int("expectationCount", len(suggestion.Expectations)).
			Strs("expectationIds", expectationIDs).
			Msg("auto-matched transaction (high confidence)")
		return nil
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
		childID := suggestion.Expectation.ChildID
		err := s.txm.WithTx(ctx, func(ctx context.Context) error {
			if err := s.matchRepo.Create(ctx, match); err != nil {
				return err
			}
			return s.postMatchActions(ctx, suggestion.Transaction.ID, suggestion.Expectation.ID, &childID, "Auto-matched: Hohe Übereinstimmung (95%+)")
		})
		if err != nil {
			log.Warn().Err(err).Str("transactionId", suggestion.Transaction.ID.String()).Msg("auto-match failed, keeping as suggestion")
			return err
		}
		log.Info().
			Str("transactionId", suggestion.Transaction.ID.String()).
			Float64("confidence", suggestion.Confidence).
			Str("matchedBy", suggestion.MatchedBy).
			Int("expectationCount", 1).
			Strs("expectationIds", []string{suggestion.Expectation.ID.String()}).
			Msg("auto-matched transaction (high confidence)")
		return nil
	}

	return errors.New("suggestion has no fee expectation")
}

// postMatchActions performs common actions after a match is created.
// Call it inside the same transaction as the match so a failure rolls back the match too.
func (s *ImportService) postMatchActions(ctx context.Context, transactionID, feeID uuid.UUID, childID *uuid.UUID, warningResolutionNote string) error {
	resolvedChildID := childID
	if resolvedChildID == nil && feeID != uuid.Nil {
		fee, err := s.feeRepo.GetByID(ctx, feeID)
		if err != nil {
			return fmt.Errorf("load fee: %w", err)
		}
		id := fee.ChildID
		resolvedChildID = &id
	}

	if err := s.markIBANAsTrusted(ctx, transactionID, resolvedChildID); err != nil {
		return fmt.Errorf("mark IBAN as trusted: %w", err)
	}

	if s.warningRepo != nil {
		if err := s.warningRepo.ResolveByTransactionID(ctx, transactionID, domain.ResolutionTypeMatched, warningResolutionNote); err != nil {
			return fmt.Errorf("resolve warnings: %w", err)
		}
	}

	if feeID != uuid.Nil {
		if err := s.checkLatePaymentAndCreateWarning(ctx, transactionID, feeID); err != nil {
			return fmt.Errorf("check late payment: %w", err)
		}
	}
	return nil
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

	var deleted int64
	err = s.txm.WithTx(ctx, func(ctx context.Context) error {
		if err := s.knownIBANRepo.Create(ctx, knownIBAN); err != nil {
			return err
		}

		// Delete all unmatched transactions from this IBAN
		var err error
		deleted, err = s.transactionRepo.DeleteUnmatchedByIBAN(ctx, iban)
		return err
	})
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
// Allocations may target fees of different children (e.g. a shared sibling payment)
// and can be repeated incrementally until the full transaction amount is allocated.
// The unallocated remainder stays visible in the unmatched lists.
// Validation and all writes run in one transaction.
func (s *ImportService) AllocateTransaction(ctx context.Context, transactionID, userID uuid.UUID, allocations []AllocationInput) (*AllocateResult, error) {
	var result *AllocateResult
	err := s.txm.WithTx(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.allocateTransaction(ctx, transactionID, userID, allocations)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Amounts are compared in cents with domain.PaymentToleranceCents slack.
func (s *ImportService) allocateTransaction(ctx context.Context, transactionID, userID uuid.UUID, allocations []AllocationInput) (*AllocateResult, error) {
	const tolerance = domain.PaymentToleranceCents

	if len(allocations) == 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Existing matches on this transaction stay untouched; further allocations are
	// allowed as long as the total stays within the transaction amount.
	existingMatches, err := s.matchRepo.GetByTransactionIDs(ctx, []uuid.UUID{transactionID})
	if err != nil {
		return nil, err
	}
	existingByFee := make(map[uuid.UUID]bool)
	var existingTotal int64
	for _, m := range existingMatches[transactionID] {
		existingByFee[m.ExpectationID] = true
		existingTotal += domain.Cents(m.Amount)
	}

	requestedByFee := make(map[uuid.UUID]bool, len(allocations))
	childCount := make(map[uuid.UUID]int)
	var totalAllocated int64

	for _, alloc := range allocations {
		if alloc.Amount <= 0 {
			return nil, ErrInvalidInput
		}

		fee, err := s.feeRepo.GetByID(ctx, alloc.ExpectationID)
		if err != nil {
			return nil, ErrNotFound
		}
		childCount[fee.ChildID]++

		// A fee can only receive one match per transaction (DB unique constraint).
		if _, alreadyMatched := existingByFee[alloc.ExpectationID]; alreadyMatched {
			return nil, ErrInvalidInput
		}
		if _, duplicate := requestedByFee[alloc.ExpectationID]; duplicate {
			return nil, ErrInvalidInput
		}
		requestedByFee[alloc.ExpectationID] = true

		matchedAmount, err := s.matchRepo.GetTotalMatchedAmount(ctx, fee.ID)
		if err != nil {
			return nil, err
		}
		remaining := domain.Cents(fee.Amount) - domain.Cents(matchedAmount)
		if remaining <= tolerance {
			return nil, ErrInvalidInput
		}
		if domain.Cents(alloc.Amount)-remaining > tolerance {
			return nil, ErrInvalidInput
		}

		totalAllocated += domain.Cents(alloc.Amount)
	}

	if existingTotal+totalAllocated-domain.Cents(tx.Amount) > tolerance {
		return nil, ErrInvalidInput
	}

	overpayment := domain.Cents(tx.Amount) - existingTotal - totalAllocated
	if overpayment < 0 {
		overpayment = 0
	}

	result := &AllocateResult{
		TransactionID:      transactionID,
		TotalAllocated:     domain.Euros(existingTotal + totalAllocated),
		Overpayment:        domain.Euros(overpayment),
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

	// Post-match actions. Only link the IBAN to a child when all allocations of this
	// call belong to a single child; shared payments stay unlinked.
	var childID *uuid.UUID
	if len(childCount) == 1 {
		for id := range childCount {
			id := id
			childID = &id
		}
	}
	if err := s.markIBANAsTrusted(ctx, transactionID, childID); err != nil {
		return nil, fmt.Errorf("mark IBAN as trusted: %w", err)
	}
	if s.warningRepo != nil {
		if err := s.warningRepo.ResolveByTransactionID(ctx, transactionID, domain.ResolutionTypeMatched, "Zahlung wurde manuell verteilt"); err != nil {
			return nil, fmt.Errorf("resolve warnings: %w", err)
		}
	}
	for _, alloc := range allocations {
		if err := s.checkLatePaymentAndCreateWarning(ctx, transactionID, alloc.ExpectationID); err != nil {
			return nil, fmt.Errorf("check late payment: %w", err)
		}
	}

	// No overpayment warning here: an unallocated remainder keeps the transaction in
	// the unmatched lists so the rest can be assigned later (e.g. to a sibling).

	return result, nil
}

// UnmatchTransaction removes matches for a transaction and optionally deletes the transaction itself.
func (s *ImportService) UnmatchTransaction(ctx context.Context, transactionID uuid.UUID, deleteTransaction bool) (*UnmatchResult, error) {
	var result *UnmatchResult
	err := s.txm.WithTx(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.unmatchTransaction(ctx, transactionID, deleteTransaction)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ImportService) unmatchTransaction(ctx context.Context, transactionID uuid.UUID, deleteTransaction bool) (*UnmatchResult, error) {
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
