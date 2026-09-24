package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

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

	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("load transaction: %w", err)
	}

	fee, err := s.feeRepo.GetByID(ctx, feeID)
	if err != nil {
		return fmt.Errorf("load fee: %w", err)
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
// Creating the fee and resolving the warning happen atomically.
func (s *ImportService) ResolveWarningWithLateFee(ctx context.Context, warningID, userID uuid.UUID) (*LateFeeResolution, error) {
	var result *LateFeeResolution
	err := s.txm.WithTx(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.resolveWarningWithLateFee(ctx, warningID, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ImportService) resolveWarningWithLateFee(ctx context.Context, warningID, userID uuid.UUID) (*LateFeeResolution, error) {
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
		HouseholdID:   originalFee.HouseholdID,
		FeeType:       domain.FeeTypeReminder,
		Year:          originalFee.Year,
		Month:         originalFee.Month, // Same month as original fee
		Amount:        domain.ReminderFeeAmount,
		DueDate:       util.Today().AddDate(0, 0, 14), // Due in 14 days
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
