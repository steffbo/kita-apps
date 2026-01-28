package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// HouseholdService handles household-related business logic.
type HouseholdService struct {
	householdRepo repository.HouseholdRepository
	parentRepo    repository.ParentRepository
	childRepo     repository.ChildRepository
}

// NewHouseholdService creates a new household service.
func NewHouseholdService(
	householdRepo repository.HouseholdRepository,
	parentRepo repository.ParentRepository,
	childRepo repository.ChildRepository,
) *HouseholdService {
	return &HouseholdService{
		householdRepo: householdRepo,
		parentRepo:    parentRepo,
		childRepo:     childRepo,
	}
}

// List returns households with optional filtering.
func (s *HouseholdService) List(ctx context.Context, search string, sortBy, sortDir string, offset, limit int) ([]domain.Household, int64, error) {
	households, total, err := s.householdRepo.List(ctx, search, sortBy, sortDir, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Load members for each household
	for i := range households {
		parents, _ := s.householdRepo.GetParents(ctx, households[i].ID)
		households[i].Parents = parents
		children, _ := s.householdRepo.GetChildren(ctx, households[i].ID)
		households[i].Children = children
	}

	return households, total, nil
}

// GetByID returns a household by ID with all members loaded.
func (s *HouseholdService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Household, error) {
	household, err := s.householdRepo.GetWithMembers(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return household, nil
}

// CreateInput defines input for creating a household.
type CreateHouseholdInput struct {
	Name                  string
	AnnualHouseholdIncome *float64
	IncomeStatus          domain.IncomeStatus
}

// Create creates a new household.
func (s *HouseholdService) Create(ctx context.Context, input CreateHouseholdInput) (*domain.Household, error) {
	household := &domain.Household{
		ID:                    uuid.New(),
		Name:                  input.Name,
		AnnualHouseholdIncome: input.AnnualHouseholdIncome,
		IncomeStatus:          input.IncomeStatus,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.householdRepo.Create(ctx, household); err != nil {
		return nil, err
	}

	return household, nil
}

// UpdateInput defines input for updating a household.
type UpdateHouseholdInput struct {
	Name                  *string
	AnnualHouseholdIncome *float64
	IncomeStatus          *domain.IncomeStatus
	ChildrenCountForFees  *int
}

// Update updates a household.
func (s *HouseholdService) Update(ctx context.Context, id uuid.UUID, input UpdateHouseholdInput) (*domain.Household, error) {
	household, err := s.householdRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if input.Name != nil {
		household.Name = *input.Name
	}
	if input.AnnualHouseholdIncome != nil {
		household.AnnualHouseholdIncome = input.AnnualHouseholdIncome
	}
	if input.IncomeStatus != nil {
		household.IncomeStatus = *input.IncomeStatus
	}
	if input.ChildrenCountForFees != nil {
		household.ChildrenCountForFees = input.ChildrenCountForFees
	}

	if err := s.householdRepo.Update(ctx, household); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// Delete deletes a household.
func (s *HouseholdService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.householdRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	return s.householdRepo.Delete(ctx, id)
}

// LinkParent links a parent to a household.
func (s *HouseholdService) LinkParent(ctx context.Context, householdID, parentID uuid.UUID) error {
	// Verify household exists
	_, err := s.householdRepo.GetByID(ctx, householdID)
	if err != nil {
		return ErrNotFound
	}

	// Get parent and update their household_id
	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		return ErrNotFound
	}

	parent.HouseholdID = &householdID
	return s.parentRepo.Update(ctx, parent)
}

// LinkChild links a child to a household.
func (s *HouseholdService) LinkChild(ctx context.Context, householdID, childID uuid.UUID) error {
	// Verify household exists
	_, err := s.householdRepo.GetByID(ctx, householdID)
	if err != nil {
		return ErrNotFound
	}

	// Get child and update their household_id
	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return ErrNotFound
	}

	child.HouseholdID = &householdID
	return s.childRepo.Update(ctx, child)
}
