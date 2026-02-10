package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// EinstufungService handles Einstufung-related business logic.
type EinstufungService struct {
	einstufungRepo repository.EinstufungRepository
	householdRepo  repository.HouseholdRepository
	childRepo      repository.ChildRepository
	feeService     *FeeService
}

// NewEinstufungService creates a new Einstufung service.
func NewEinstufungService(
	einstufungRepo repository.EinstufungRepository,
	householdRepo repository.HouseholdRepository,
	childRepo repository.ChildRepository,
	feeService *FeeService,
) *EinstufungService {
	return &EinstufungService{
		einstufungRepo: einstufungRepo,
		householdRepo:  householdRepo,
		childRepo:      childRepo,
		feeService:     feeService,
	}
}

// CreateInput defines input for creating an Einstufung.
type CreateEinstufungInput struct {
	ChildID              uuid.UUID                          `json:"childId"`
	Year                 int                                `json:"year"`
	ValidFrom            time.Time                          `json:"validFrom"`
	IncomeCalculation    domain.HouseholdIncomeCalculation  `json:"incomeCalculation"`
	HighestRateVoluntary bool                               `json:"highestRateVoluntary"`
	CareHoursPerWeek     int                                `json:"careHoursPerWeek"`
	ChildrenCount        int                                `json:"childrenCount"`
	Notes                string                             `json:"notes"`
}

// Create creates a new Einstufung, computing the fee from the income calculation.
// This is the main entry point for the Einstufung process:
// 1. Calculate fee-relevant household income from parent line items
// 2. Determine care type (Krippe/Kindergarten) from child age
// 3. Calculate childcare fee using existing fee tables
// 4. Store the complete classification record
// 5. Update household income on the household record
func (s *EinstufungService) Create(ctx context.Context, input CreateEinstufungInput) (*domain.Einstufung, error) {
	// Validate child exists and get household
	child, err := s.childRepo.GetByID(ctx, input.ChildID)
	if err != nil {
		return nil, ErrNotFound
	}
	if child.HouseholdID == nil {
		return nil, ErrInvalidInput
	}

	household, err := s.householdRepo.GetWithMembers(ctx, *child.HouseholdID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Determine care type from child age at validFrom
	careType := domain.ChildAgeTypeKrippe
	if !child.IsUnderThree(input.ValidFrom) {
		careType = domain.ChildAgeTypeKindergarten
	}

	// Calculate fee-relevant household income
	annualNetIncome := input.IncomeCalculation.CalculateAnnualNetIncome()

	// Default care hours
	careHours := input.CareHoursPerWeek
	if careHours == 0 {
		careHours = 30 // Default
	}

	// Default children count
	childrenCount := input.ChildrenCount
	if childrenCount == 0 {
		// Count active children in household
		activeCount := 0
		for _, c := range household.Children {
			if c.IsActive {
				activeCount++
			}
		}
		if activeCount > 0 {
			childrenCount = activeCount
		} else {
			childrenCount = 1
		}
	}

	// Calculate childcare fee using existing fee calculation logic
	feeResult := s.feeService.CalculateChildcareFee(domain.ChildcareFeeInput{
		ChildAgeType:  careType,
		NetIncome:     annualNetIncome,
		SiblingsCount: childrenCount,
		CareHours:     careHours,
		HighestRate:   input.HighestRateVoluntary,
		FosterFamily:  household.IncomeStatus == domain.IncomeStatusFosterFamily,
	})

	// Build the Einstufung record
	einstufung := &domain.Einstufung{
		ID:                   uuid.New(),
		ChildID:              input.ChildID,
		HouseholdID:          *child.HouseholdID,
		Year:                 input.Year,
		ValidFrom:            input.ValidFrom,
		IncomeCalculation:    input.IncomeCalculation,
		AnnualNetIncome:      annualNetIncome,
		HighestRateVoluntary: input.HighestRateVoluntary,
		CareHoursPerWeek:     careHours,
		CareType:             careType,
		ChildrenCount:        childrenCount,
		MonthlyChildcareFee:  feeResult.Fee,
		MonthlyFoodFee:       domain.FoodFeeAmount,
		AnnualMembershipFee:  domain.MembershipFeeAmount,
		FeeRule:              feeResult.Rule,
		DiscountPercent:      feeResult.DiscountPercent,
		DiscountFactor:       feeResult.DiscountFactor,
		BaseFee:              feeResult.BaseFee,
		Notes:                input.Notes,
	}

	if err := s.einstufungRepo.Create(ctx, einstufung); err != nil {
		return nil, err
	}

	// Update household with calculated income
	income := annualNetIncome
	household.AnnualHouseholdIncome = &income
	if !input.HighestRateVoluntary {
		household.IncomeStatus = domain.IncomeStatusProvided
	} else {
		household.IncomeStatus = domain.IncomeStatusMaxAccepted
	}
	if input.ChildrenCount > 0 {
		household.ChildrenCountForFees = &input.ChildrenCount
	}
	if err := s.householdRepo.Update(ctx, household); err != nil {
		return nil, err
	}

	// Load relations for response
	einstufung.Child = child
	einstufung.Household = household

	return einstufung, nil
}

// UpdateInput defines input for updating an Einstufung.
type UpdateEinstufungInput struct {
	IncomeCalculation    *domain.HouseholdIncomeCalculation `json:"incomeCalculation,omitempty"`
	HighestRateVoluntary *bool                              `json:"highestRateVoluntary,omitempty"`
	CareHoursPerWeek     *int                               `json:"careHoursPerWeek,omitempty"`
	ChildrenCount        *int                               `json:"childrenCount,omitempty"`
	ValidFrom            *time.Time                         `json:"validFrom,omitempty"`
	Notes                *string                            `json:"notes,omitempty"`
}

// Update recalculates and updates an existing Einstufung.
func (s *EinstufungService) Update(ctx context.Context, id uuid.UUID, input UpdateEinstufungInput) (*domain.Einstufung, error) {
	existing, err := s.einstufungRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Apply updates
	if input.IncomeCalculation != nil {
		existing.IncomeCalculation = *input.IncomeCalculation
	}
	if input.HighestRateVoluntary != nil {
		existing.HighestRateVoluntary = *input.HighestRateVoluntary
	}
	if input.CareHoursPerWeek != nil {
		existing.CareHoursPerWeek = *input.CareHoursPerWeek
	}
	if input.ChildrenCount != nil {
		existing.ChildrenCount = *input.ChildrenCount
	}
	if input.ValidFrom != nil {
		existing.ValidFrom = *input.ValidFrom
	}
	if input.Notes != nil {
		existing.Notes = *input.Notes
	}

	// Recalculate
	existing.AnnualNetIncome = existing.IncomeCalculation.CalculateAnnualNetIncome()

	// Get child to check care type
	child, err := s.childRepo.GetByID(ctx, existing.ChildID)
	if err != nil {
		return nil, ErrNotFound
	}

	careType := domain.ChildAgeTypeKrippe
	if !child.IsUnderThree(existing.ValidFrom) {
		careType = domain.ChildAgeTypeKindergarten
	}
	existing.CareType = careType

	household, err := s.householdRepo.GetByID(ctx, existing.HouseholdID)
	if err != nil {
		return nil, ErrNotFound
	}

	feeResult := s.feeService.CalculateChildcareFee(domain.ChildcareFeeInput{
		ChildAgeType:  existing.CareType,
		NetIncome:     existing.AnnualNetIncome,
		SiblingsCount: existing.ChildrenCount,
		CareHours:     existing.CareHoursPerWeek,
		HighestRate:   existing.HighestRateVoluntary,
		FosterFamily:  household.IncomeStatus == domain.IncomeStatusFosterFamily,
	})

	existing.MonthlyChildcareFee = feeResult.Fee
	existing.FeeRule = feeResult.Rule
	existing.DiscountPercent = feeResult.DiscountPercent
	existing.DiscountFactor = feeResult.DiscountFactor
	existing.BaseFee = feeResult.BaseFee

	if err := s.einstufungRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	// Update household income and children count
	income := existing.AnnualNetIncome
	household.AnnualHouseholdIncome = &income
	household.ChildrenCountForFees = &existing.ChildrenCount
	_ = s.householdRepo.Update(ctx, household)

	existing.Child = child
	existing.Household = household

	return existing, nil
}

// GetByID returns an Einstufung by ID with loaded relations.
func (s *EinstufungService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Einstufung, error) {
	e, err := s.einstufungRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	s.loadRelations(ctx, e)
	return e, nil
}

// GetByChildAndYear returns the Einstufung for a child in a given year.
func (s *EinstufungService) GetByChildAndYear(ctx context.Context, childID uuid.UUID, year int) (*domain.Einstufung, error) {
	e, err := s.einstufungRepo.GetByChildAndYear(ctx, childID, year)
	if err != nil {
		return nil, ErrNotFound
	}
	s.loadRelations(ctx, e)
	return e, nil
}

// ListByHousehold returns all Einstufungen for a household.
func (s *EinstufungService) ListByHousehold(ctx context.Context, householdID uuid.UUID) ([]domain.Einstufung, error) {
	results, err := s.einstufungRepo.ListByHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	for i := range results {
		s.loadRelations(ctx, &results[i])
	}
	return results, nil
}

// ListByYear returns Einstufungen for a year with pagination.
func (s *EinstufungService) ListByYear(ctx context.Context, year int, offset, limit int) ([]domain.Einstufung, int64, error) {
	results, total, err := s.einstufungRepo.ListByYear(ctx, year, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	for i := range results {
		s.loadRelations(ctx, &results[i])
	}
	return results, total, nil
}

// Delete deletes an Einstufung.
func (s *EinstufungService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.einstufungRepo.Delete(ctx, id)
}

// GetLatestForChild returns the most recent Einstufung for a child.
func (s *EinstufungService) GetLatestForChild(ctx context.Context, childID uuid.UUID) (*domain.Einstufung, error) {
	e, err := s.einstufungRepo.GetLatestForChild(ctx, childID)
	if err != nil {
		return nil, ErrNotFound
	}
	s.loadRelations(ctx, e)
	return e, nil
}

// loadRelations loads child and household relations for an Einstufung.
func (s *EinstufungService) loadRelations(ctx context.Context, e *domain.Einstufung) {
	if child, err := s.childRepo.GetByID(ctx, e.ChildID); err == nil {
		e.Child = child
	}
	if household, err := s.householdRepo.GetWithMembers(ctx, e.HouseholdID); err == nil {
		e.Household = household
	}
}
