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
	childRepo     repository.ChildRepository
	parentRepo    repository.ParentRepository
	householdRepo repository.HouseholdRepository
}

// NewChildService creates a new child service.
func NewChildService(childRepo repository.ChildRepository, parentRepo repository.ParentRepository, householdRepo repository.HouseholdRepository) *ChildService {
	return &ChildService{
		childRepo:     childRepo,
		parentRepo:    parentRepo,
		householdRepo: householdRepo,
	}
}

// ChildFilter defines filters for listing children.
type ChildFilter struct {
	ActiveOnly  bool
	U3Only      bool
	HasWarnings bool
	HasOpenFees bool
	Search      string
	SortBy      string
	SortDir     string
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

// List returns children matching the filter with parents and households loaded.
func (s *ChildService) List(ctx context.Context, filter ChildFilter, offset, limit int) ([]domain.Child, int64, error) {
	children, total, err := s.childRepo.List(ctx, filter.ActiveOnly, filter.U3Only, filter.HasWarnings, filter.HasOpenFees, filter.Search, filter.SortBy, filter.SortDir, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	if len(children) > 0 {
		childIDs := make([]uuid.UUID, len(children))
		householdIDs := make([]uuid.UUID, 0, len(children))
		householdIDSet := make(map[uuid.UUID]bool)

		for i, child := range children {
			childIDs[i] = child.ID
			if child.HouseholdID != nil && !householdIDSet[*child.HouseholdID] {
				householdIDs = append(householdIDs, *child.HouseholdID)
				householdIDSet[*child.HouseholdID] = true
			}
		}

		// Batch-load parents for all children
		parentsMap, err := s.childRepo.GetParentsForChildren(ctx, childIDs)
		if err == nil {
			for i := range children {
				if parents, ok := parentsMap[children[i].ID]; ok {
					children[i].Parents = parents
				}
			}
		}

		// Batch-load households for all children
		if len(householdIDs) > 0 && s.householdRepo != nil {
			householdsMap, err := s.householdRepo.GetByIDs(ctx, householdIDs)
			if err == nil {
				for i := range children {
					if children[i].HouseholdID != nil {
						if household, ok := householdsMap[*children[i].HouseholdID]; ok {
							children[i].Household = household
						}
					}
				}
			}
		}
	}

	return children, total, nil
}

// GetByID returns a child by ID with parents and household.
func (s *ChildService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	child, err := s.childRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	parents, err := s.childRepo.GetParents(ctx, id)
	if err == nil {
		child.Parents = parents
	}

	// Load household with siblings if child has a household
	if child.HouseholdID != nil && s.householdRepo != nil {
		household, err := s.householdRepo.GetWithMembers(ctx, *child.HouseholdID)
		if err == nil {
			child.Household = household
		}
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

// LinkParent links a parent to a child, creating a household if needed.
func (s *ChildService) LinkParent(ctx context.Context, childID, parentID uuid.UUID, isPrimary bool) error {
	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return ErrNotFound
	}

	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		return ErrNotFound
	}

	// Create or assign household
	if s.householdRepo != nil {
		if child.HouseholdID == nil {
			// Create a new household for this family
			householdName := child.LastName
			if parent.LastName != child.LastName {
				householdName = parent.LastName + "/" + child.LastName
			}
			household := &domain.Household{
				ID:        uuid.New(),
				Name:      "Familie " + householdName,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Migrate income from parent to household if available
			if parent.IncomeStatus != "" && parent.IncomeStatus != domain.IncomeStatusUnknown {
				household.IncomeStatus = parent.IncomeStatus
				household.AnnualHouseholdIncome = parent.AnnualHouseholdIncome
			}

			if err := s.householdRepo.Create(ctx, household); err != nil {
				return err
			}

			// Assign child to household
			child.HouseholdID = &household.ID
			if err := s.childRepo.Update(ctx, child); err != nil {
				return err
			}

			// Assign parent to household
			parent.HouseholdID = &household.ID
			if err := s.parentRepo.Update(ctx, parent); err != nil {
				return err
			}
		} else if parent.HouseholdID == nil {
			// Child has a household but parent doesn't - add parent to child's household
			parent.HouseholdID = child.HouseholdID
			if err := s.parentRepo.Update(ctx, parent); err != nil {
				return err
			}
		}
	}

	return s.childRepo.LinkParent(ctx, childID, parentID, isPrimary)
}

// UnlinkParent unlinks a parent from a child.
func (s *ChildService) UnlinkParent(ctx context.Context, childID, parentID uuid.UUID) error {
	return s.childRepo.UnlinkParent(ctx, childID, parentID)
}

// GetAll returns all active children (for matching purposes).
func (s *ChildService) GetAll(ctx context.Context) ([]domain.Child, error) {
	children, _, err := s.childRepo.List(ctx, true, false, false, false, "", "", "", 0, 1000)
	return children, err
}
