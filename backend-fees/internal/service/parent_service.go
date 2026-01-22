package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ParentService handles parent-related business logic.
type ParentService struct {
	parentRepo repository.ParentRepository
	childRepo  repository.ChildRepository
}

// NewParentService creates a new parent service.
func NewParentService(parentRepo repository.ParentRepository, childRepo repository.ChildRepository) *ParentService {
	return &ParentService{
		parentRepo: parentRepo,
		childRepo:  childRepo,
	}
}

// CreateParentInput defines input for creating a parent.
type CreateParentInput struct {
	FirstName             string
	LastName              string
	BirthDate             *string
	Email                 *string
	Phone                 *string
	Street                *string
	StreetNo              *string
	PostalCode            *string
	City                  *string
	AnnualHouseholdIncome *float64
	IncomeStatus          *string
}

// UpdateParentInput defines input for updating a parent.
type UpdateParentInput struct {
	FirstName             *string
	LastName              *string
	BirthDate             *string
	Email                 *string
	Phone                 *string
	Street                *string
	StreetNo              *string
	PostalCode            *string
	City                  *string
	AnnualHouseholdIncome *float64
	IncomeStatus          *string
}

// List returns parents matching the search term.
func (s *ParentService) List(ctx context.Context, search string, offset, limit int) ([]domain.Parent, int64, error) {
	return s.parentRepo.List(ctx, search, offset, limit)
}

// GetByID returns a parent by ID with children.
func (s *ParentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	parent, err := s.parentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	children, err := s.parentRepo.GetChildren(ctx, id)
	if err == nil {
		parent.Children = children
	}

	return parent, nil
}

// Create creates a new parent.
func (s *ParentService) Create(ctx context.Context, input CreateParentInput) (*domain.Parent, error) {
	var birthDate *time.Time
	if input.BirthDate != nil {
		t, err := time.Parse("2006-01-02", *input.BirthDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		birthDate = &t
	}

	parent := &domain.Parent{
		ID:                    uuid.New(),
		FirstName:             input.FirstName,
		LastName:              input.LastName,
		BirthDate:             birthDate,
		Email:                 input.Email,
		Phone:                 input.Phone,
		Street:                input.Street,
		StreetNo:              input.StreetNo,
		PostalCode:            input.PostalCode,
		City:                  input.City,
		AnnualHouseholdIncome: input.AnnualHouseholdIncome,
		IncomeStatus:          domain.IncomeStatus(stringOrEmpty(input.IncomeStatus)),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.parentRepo.Create(ctx, parent); err != nil {
		return nil, err
	}

	return parent, nil
}

// Update updates a parent.
func (s *ParentService) Update(ctx context.Context, id uuid.UUID, input UpdateParentInput) (*domain.Parent, error) {
	parent, err := s.parentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if input.FirstName != nil {
		parent.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		parent.LastName = *input.LastName
	}
	if input.BirthDate != nil {
		t, err := time.Parse("2006-01-02", *input.BirthDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		parent.BirthDate = &t
	}
	if input.Email != nil {
		parent.Email = input.Email
	}
	if input.Phone != nil {
		parent.Phone = input.Phone
	}
	if input.Street != nil {
		parent.Street = input.Street
	}
	if input.StreetNo != nil {
		parent.StreetNo = input.StreetNo
	}
	if input.PostalCode != nil {
		parent.PostalCode = input.PostalCode
	}
	if input.City != nil {
		parent.City = input.City
	}
	if input.AnnualHouseholdIncome != nil {
		parent.AnnualHouseholdIncome = input.AnnualHouseholdIncome
	}
	if input.IncomeStatus != nil {
		parent.IncomeStatus = domain.IncomeStatus(*input.IncomeStatus)
	}

	if err := s.parentRepo.Update(ctx, parent); err != nil {
		return nil, err
	}

	return parent, nil
}

// Delete deletes a parent.
func (s *ParentService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.parentRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	return s.parentRepo.Delete(ctx, id)
}

// stringOrEmpty returns the string value or empty string if nil.
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
