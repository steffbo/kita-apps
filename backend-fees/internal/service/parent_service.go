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
	memberRepo repository.MemberRepository
}

// NewParentService creates a new parent service.
func NewParentService(parentRepo repository.ParentRepository, childRepo repository.ChildRepository, memberRepo repository.MemberRepository) *ParentService {
	return &ParentService{
		parentRepo: parentRepo,
		childRepo:  childRepo,
		memberRepo: memberRepo,
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

// List returns parents matching the search term with sorting.
func (s *ParentService) List(ctx context.Context, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Parent, int64, error) {
	parents, total, err := s.parentRepo.List(ctx, search, sortBy, sortDir, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Batch load children for all parents
	if len(parents) > 0 {
		parentIDs := make([]uuid.UUID, len(parents))
		for i, p := range parents {
			parentIDs[i] = p.ID
		}

		childrenMap, err := s.parentRepo.GetChildrenForParents(ctx, parentIDs)
		if err == nil {
			for i := range parents {
				parents[i].Children = childrenMap[parents[i].ID]
			}
		}
	}

	return parents, total, nil
}

// GetByID returns a parent by ID with children and member.
func (s *ParentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	parent, err := s.parentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	children, err := s.parentRepo.GetChildren(ctx, id)
	if err == nil {
		parent.Children = children
	}

	// Load member if linked
	if parent.MemberID != nil {
		member, err := s.memberRepo.GetByID(ctx, *parent.MemberID)
		if err == nil {
			parent.Member = member
		}
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

// CreateMemberFromParent creates a new member from a parent's data and links them.
// If membershipStart is zero, uses the oldest child's entry date, or today if no children.
func (s *ParentService) CreateMemberFromParent(ctx context.Context, parentID uuid.UUID, membershipStart time.Time) (*domain.Parent, error) {
	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Check if already linked to a member
	if parent.MemberID != nil {
		return nil, ErrConflict
	}

	// Load children to find oldest entry date
	children, err := s.parentRepo.GetChildren(ctx, parentID)
	if err == nil {
		parent.Children = children
	}

	// If no membership start provided, use oldest child's entry date
	if membershipStart.IsZero() {
		membershipStart = s.getOldestChildEntryDate(children)
	}

	// Get next member number
	memberNumber, err := s.memberRepo.GetNextMemberNumber(ctx)
	if err != nil {
		return nil, err
	}

	// Create member from parent data
	member := &domain.Member{
		ID:              uuid.New(),
		MemberNumber:    memberNumber,
		FirstName:       parent.FirstName,
		LastName:        parent.LastName,
		Email:           parent.Email,
		Phone:           parent.Phone,
		Street:          parent.Street,
		StreetNo:        parent.StreetNo,
		PostalCode:      parent.PostalCode,
		City:            parent.City,
		HouseholdID:     parent.HouseholdID,
		MembershipStart: membershipStart,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, err
	}

	// Link member to parent
	parent.MemberID = &member.ID
	if err := s.parentRepo.Update(ctx, parent); err != nil {
		return nil, err
	}

	// Attach the member to parent response
	parent.Member = member

	return parent, nil
}

// getOldestChildEntryDate returns the oldest entry date among children, or today if no children.
func (s *ParentService) getOldestChildEntryDate(children []domain.Child) time.Time {
	if len(children) == 0 {
		return time.Now()
	}

	oldest := children[0].EntryDate
	for _, child := range children[1:] {
		if child.EntryDate.Before(oldest) {
			oldest = child.EntryDate
		}
	}
	return oldest
}

// UnlinkMember removes the member link from a parent (does not delete the member).
func (s *ParentService) UnlinkMember(ctx context.Context, parentID uuid.UUID) (*domain.Parent, error) {
	parent, err := s.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		return nil, ErrNotFound
	}

	if parent.MemberID == nil {
		return nil, ErrNotFound
	}

	parent.MemberID = nil
	if err := s.parentRepo.Update(ctx, parent); err != nil {
		return nil, err
	}

	// Load children and return
	children, err := s.parentRepo.GetChildren(ctx, parentID)
	if err == nil {
		parent.Children = children
	}

	return parent, nil
}

// stringOrEmpty returns the string value or empty string if nil.
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
