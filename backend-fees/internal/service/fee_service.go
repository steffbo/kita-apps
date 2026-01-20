package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// FeeService handles fee-related business logic.
type FeeService struct {
	feeRepo   repository.FeeRepository
	childRepo repository.ChildRepository
	matchRepo repository.MatchRepository
}

// NewFeeService creates a new fee service.
func NewFeeService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	matchRepo repository.MatchRepository,
) *FeeService {
	return &FeeService{
		feeRepo:   feeRepo,
		childRepo: childRepo,
		matchRepo: matchRepo,
	}
}

// FeeFilter defines filters for listing fees.
type FeeFilter struct {
	Year    *int
	Month   *int
	FeeType string
	Status  string
	ChildID *uuid.UUID
}



// GenerateResult represents the result of fee generation.
type GenerateResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

// ChildcareFeeResult represents the childcare fee calculation result.
type ChildcareFeeResult struct {
	Amount  float64 `json:"amount"`
	Bracket string  `json:"bracket"`
}

// List returns fees matching the filter.
func (s *FeeService) List(ctx context.Context, filter FeeFilter, offset, limit int) ([]domain.FeeExpectation, int64, error) {
	fees, total, err := s.feeRepo.List(ctx, repository.FeeFilter{
		Year:    filter.Year,
		Month:   filter.Month,
		FeeType: filter.FeeType,
		ChildID: filter.ChildID,
	}, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Enrich with payment status
	for i := range fees {
		matched, _ := s.matchRepo.ExistsForExpectation(ctx, fees[i].ID)
		fees[i].IsPaid = matched
	}

	return fees, total, nil
}

// GetByID returns a fee by ID.
func (s *FeeService) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeExpectation, error) {
	fee, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Check if paid
	matched, _ := s.matchRepo.ExistsForExpectation(ctx, id)
	fee.IsPaid = matched

	// Get child info
	child, _ := s.childRepo.GetByID(ctx, fee.ChildID)
	fee.Child = child

	return fee, nil
}

// GetOverview returns fee overview statistics.
func (s *FeeService) GetOverview(ctx context.Context, year *int) (*domain.FeeOverview, error) {
	targetYear := time.Now().Year()
	if year != nil {
		targetYear = *year
	}

	overview, err := s.feeRepo.GetOverview(ctx, targetYear)
	if err != nil {
		return nil, err
	}

	return overview, nil
}

// Generate creates fee expectations for the given period.
func (s *FeeService) Generate(ctx context.Context, year int, month *int) (*GenerateResult, error) {
	children, _, err := s.childRepo.List(ctx, true, "", 0, 1000)
	if err != nil {
		return nil, err
	}

	result := &GenerateResult{}
	now := time.Now()

	for _, child := range children {
		// Generate monthly fees
		if month != nil {
			dueDate := time.Date(year, time.Month(*month), 15, 0, 0, 0, 0, time.UTC)
			checkDate := time.Date(year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)

			// Food fee (all children)
			created, err := s.createFeeIfNotExists(ctx, child.ID, domain.FeeTypeFood, year, month, domain.FoodFeeAmount, dueDate)
			if err != nil {
				return nil, err
			}
			if created {
				result.Created++
			} else {
				result.Skipped++
			}

			// Childcare fee (only U3)
			if child.IsUnderThree(checkDate) {
				amount := s.CalculateChildcareFee(0).Amount // Default for now
				created, err := s.createFeeIfNotExists(ctx, child.ID, domain.FeeTypeChildcare, year, month, amount, dueDate)
				if err != nil {
					return nil, err
				}
				if created {
					result.Created++
				} else {
					result.Skipped++
				}
			}
		} else {
			// Generate yearly membership fee
			dueDate := time.Date(year, 1, 31, 0, 0, 0, 0, time.UTC)

			// Only generate if child was enrolled before the year ends
			if child.EntryDate.Year() <= year {
				created, err := s.createFeeIfNotExists(ctx, child.ID, domain.FeeTypeMembership, year, nil, domain.MembershipFeeAmount, dueDate)
				if err != nil {
					return nil, err
				}
				if created {
					result.Created++
				} else {
					result.Skipped++
				}
			}
		}
	}

	_ = now // suppress unused warning

	return result, nil
}

func (s *FeeService) createFeeIfNotExists(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int, amount float64, dueDate time.Time) (bool, error) {
	// Check if fee already exists
	exists, err := s.feeRepo.Exists(ctx, childID, feeType, year, month)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   childID,
		FeeType:   feeType,
		Year:      year,
		Month:     month,
		Amount:    amount,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}

	if err := s.feeRepo.Create(ctx, fee); err != nil {
		return false, err
	}

	return true, nil
}

// Update updates a fee's amount.
func (s *FeeService) Update(ctx context.Context, id uuid.UUID, amount *float64) (*domain.FeeExpectation, error) {
	fee, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if amount != nil {
		fee.Amount = *amount
	}

	if err := s.feeRepo.Update(ctx, fee); err != nil {
		return nil, err
	}

	return fee, nil
}

// Delete deletes a fee.
func (s *FeeService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	return s.feeRepo.Delete(ctx, id)
}

// CalculateChildcareFee calculates the childcare fee based on income.
// For now, returns a fixed amount of 100 EUR.
func (s *FeeService) CalculateChildcareFee(income float64) *ChildcareFeeResult {
	// TODO: Implement income-based calculation
	// For now, always return 100 EUR as per requirements
	return &ChildcareFeeResult{
		Amount:  domain.DefaultChildcareFee,
		Bracket: "default",
	}
}
