package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ChildService handles child-related business logic.
type ChildService struct {
	childRepo  repository.ChildRepository
	parentRepo repository.ParentRepository
}

// NewChildService creates a new child service.
func NewChildService(childRepo repository.ChildRepository, parentRepo repository.ParentRepository) *ChildService {
	return &ChildService{
		childRepo:  childRepo,
		parentRepo: parentRepo,
	}
}

// ChildFilter defines filters for listing children.
type ChildFilter struct {
	ActiveOnly bool
	U3Only     bool
	Search     string
	SortBy     string
	SortDir    string
}

// CreateChildInput defines input for creating a child.
type CreateChildInput struct {
	MemberNumber    string
	FirstName       string
	LastName        string
	BirthDate       string
	EntryDate       string
	ExitDate        *string
	Street          *string
	StreetNo        *string
	PostalCode      *string
	City            *string
	LegalHours      *int
	LegalHoursUntil *string
	CareHours       *int
}

// UpdateChildInput defines input for updating a child.
type UpdateChildInput struct {
	FirstName       *string
	LastName        *string
	BirthDate       *string
	EntryDate       *string
	ExitDate        *string
	Street          *string
	StreetNo        *string
	PostalCode      *string
	City            *string
	LegalHours      *int
	LegalHoursUntil *string
	CareHours       *int
	IsActive        *bool
}

// List returns children matching the filter.
func (s *ChildService) List(ctx context.Context, filter ChildFilter, offset, limit int) ([]domain.Child, int64, error) {
	return s.childRepo.List(ctx, filter.ActiveOnly, filter.U3Only, filter.Search, filter.SortBy, filter.SortDir, offset, limit)
}

// GetByID returns a child by ID with parents.
func (s *ChildService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	parents, err := s.childRepo.GetParents(ctx, id)
	if err == nil {
		child.Parents = parents
	}

	return child, nil
}

// Create creates a new child.
func (s *ChildService) Create(ctx context.Context, input CreateChildInput) (*domain.Child, error) {
	birthDate, err := time.Parse("2006-01-02", input.BirthDate)
	if err != nil {
		return nil, ErrInvalidInput
	}

	entryDate, err := time.Parse("2006-01-02", input.EntryDate)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// Parse legalHoursUntil if provided
	var legalHoursUntil *time.Time
	if input.LegalHoursUntil != nil && *input.LegalHoursUntil != "" {
		parsed, err := time.Parse("2006-01-02", *input.LegalHoursUntil)
		if err != nil {
			return nil, ErrInvalidInput
		}
		legalHoursUntil = &parsed
	}

	// Parse exitDate if provided
	var exitDate *time.Time
	if input.ExitDate != nil && *input.ExitDate != "" {
		parsed, err := time.Parse("2006-01-02", *input.ExitDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		exitDate = &parsed
	}

	// Check for duplicate member number
	existing, _ := s.childRepo.GetByMemberNumber(ctx, input.MemberNumber)
	if existing != nil {
		return nil, ErrDuplicateMemberNumber
	}

	child := &domain.Child{
		ID:              uuid.New(),
		MemberNumber:    input.MemberNumber,
		FirstName:       input.FirstName,
		LastName:        input.LastName,
		BirthDate:       birthDate,
		EntryDate:       entryDate,
		ExitDate:        exitDate,
		Street:          input.Street,
		StreetNo:        input.StreetNo,
		PostalCode:      input.PostalCode,
		City:            input.City,
		LegalHours:      input.LegalHours,
		LegalHoursUntil: legalHoursUntil,
		CareHours:       input.CareHours,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.childRepo.Create(ctx, child); err != nil {
		return nil, err
	}

	return child, nil
}

// Update updates a child.
func (s *ChildService) Update(ctx context.Context, id uuid.UUID, input UpdateChildInput) (*domain.Child, error) {
	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if input.FirstName != nil {
		child.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		child.LastName = *input.LastName
	}
	if input.BirthDate != nil {
		birthDate, err := time.Parse("2006-01-02", *input.BirthDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		child.BirthDate = birthDate
	}
	if input.EntryDate != nil {
		entryDate, err := time.Parse("2006-01-02", *input.EntryDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		child.EntryDate = entryDate
	}
	if input.ExitDate != nil {
		if *input.ExitDate == "" {
			child.ExitDate = nil
		} else {
			parsed, err := time.Parse("2006-01-02", *input.ExitDate)
			if err != nil {
				return nil, ErrInvalidInput
			}
			child.ExitDate = &parsed
		}
	}
	if input.Street != nil {
		child.Street = input.Street
	}
	if input.StreetNo != nil {
		child.StreetNo = input.StreetNo
	}
	if input.PostalCode != nil {
		child.PostalCode = input.PostalCode
	}
	if input.City != nil {
		child.City = input.City
	}
	if input.LegalHours != nil {
		child.LegalHours = input.LegalHours
	}
	if input.LegalHoursUntil != nil {
		if *input.LegalHoursUntil == "" {
			child.LegalHoursUntil = nil
		} else {
			parsed, err := time.Parse("2006-01-02", *input.LegalHoursUntil)
			if err != nil {
				return nil, ErrInvalidInput
			}
			child.LegalHoursUntil = &parsed
		}
	}
	if input.CareHours != nil {
		child.CareHours = input.CareHours
	}
	if input.IsActive != nil {
		child.IsActive = *input.IsActive
	}

	if err := s.childRepo.Update(ctx, child); err != nil {
		return nil, err
	}

	return child, nil
}

// Deactivate soft-deletes a child.
func (s *ChildService) Deactivate(ctx context.Context, id uuid.UUID) error {
	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	child.IsActive = false
	return s.childRepo.Update(ctx, child)
}

// LinkParent links a parent to a child.
func (s *ChildService) LinkParent(ctx context.Context, childID, parentID uuid.UUID, isPrimary bool) error {
	_, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return ErrNotFound
	}

	_, err = s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		return ErrNotFound
	}

	return s.childRepo.LinkParent(ctx, childID, parentID, isPrimary)
}

// UnlinkParent unlinks a parent from a child.
func (s *ChildService) UnlinkParent(ctx context.Context, childID, parentID uuid.UUID) error {
	return s.childRepo.UnlinkParent(ctx, childID, parentID)
}

// GetAll returns all active children (for matching purposes).
func (s *ChildService) GetAll(ctx context.Context) ([]domain.Child, error) {
	children, _, err := s.childRepo.List(ctx, true, false, "", "", "", 0, 1000)
	return children, err
}
