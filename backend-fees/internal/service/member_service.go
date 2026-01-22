package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// MemberService handles member-related business logic.
type MemberService struct {
	memberRepo    repository.MemberRepository
	householdRepo repository.HouseholdRepository
}

// NewMemberService creates a new member service.
func NewMemberService(
	memberRepo repository.MemberRepository,
	householdRepo repository.HouseholdRepository,
) *MemberService {
	return &MemberService{
		memberRepo:    memberRepo,
		householdRepo: householdRepo,
	}
}

// List returns members with optional filtering.
func (s *MemberService) List(ctx context.Context, activeOnly bool, search string, sortBy, sortDir string, offset, limit int) ([]domain.Member, int64, error) {
	return s.memberRepo.List(ctx, activeOnly, search, sortBy, sortDir, offset, limit)
}

// GetByID returns a member by ID.
func (s *MemberService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Member, error) {
	member, err := s.memberRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return member, nil
}

// CreateMemberInput defines input for creating a member.
type CreateMemberInput struct {
	MemberNumber    *string // If nil, auto-generate
	FirstName       string
	LastName        string
	Email           *string
	Phone           *string
	Street          *string
	StreetNo        *string
	PostalCode      *string
	City            *string
	HouseholdID     *uuid.UUID
	MembershipStart time.Time
	MembershipEnd   *time.Time
}

// Create creates a new member.
func (s *MemberService) Create(ctx context.Context, input CreateMemberInput) (*domain.Member, error) {
	// Generate member number if not provided
	memberNumber := ""
	if input.MemberNumber != nil {
		memberNumber = *input.MemberNumber
	} else {
		var err error
		memberNumber, err = s.memberRepo.GetNextMemberNumber(ctx)
		if err != nil {
			return nil, err
		}
	}

	member := &domain.Member{
		ID:              uuid.New(),
		MemberNumber:    memberNumber,
		FirstName:       input.FirstName,
		LastName:        input.LastName,
		Email:           input.Email,
		Phone:           input.Phone,
		Street:          input.Street,
		StreetNo:        input.StreetNo,
		PostalCode:      input.PostalCode,
		City:            input.City,
		HouseholdID:     input.HouseholdID,
		MembershipStart: input.MembershipStart,
		MembershipEnd:   input.MembershipEnd,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

// UpdateMemberInput defines input for updating a member.
type UpdateMemberInput struct {
	FirstName       *string
	LastName        *string
	Email           *string
	Phone           *string
	Street          *string
	StreetNo        *string
	PostalCode      *string
	City            *string
	HouseholdID     *uuid.UUID
	MembershipStart *time.Time
	MembershipEnd   *time.Time
	IsActive        *bool
}

// Update updates a member.
func (s *MemberService) Update(ctx context.Context, id uuid.UUID, input UpdateMemberInput) (*domain.Member, error) {
	member, err := s.memberRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if input.FirstName != nil {
		member.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		member.LastName = *input.LastName
	}
	if input.Email != nil {
		member.Email = input.Email
	}
	if input.Phone != nil {
		member.Phone = input.Phone
	}
	if input.Street != nil {
		member.Street = input.Street
	}
	if input.StreetNo != nil {
		member.StreetNo = input.StreetNo
	}
	if input.PostalCode != nil {
		member.PostalCode = input.PostalCode
	}
	if input.City != nil {
		member.City = input.City
	}
	if input.HouseholdID != nil {
		member.HouseholdID = input.HouseholdID
	}
	if input.MembershipStart != nil {
		member.MembershipStart = *input.MembershipStart
	}
	if input.MembershipEnd != nil {
		member.MembershipEnd = input.MembershipEnd
	}
	if input.IsActive != nil {
		member.IsActive = *input.IsActive
	}

	if err := s.memberRepo.Update(ctx, member); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// Delete deletes a member (soft delete by setting IsActive = false).
func (s *MemberService) Delete(ctx context.Context, id uuid.UUID) error {
	member, err := s.memberRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	member.IsActive = false
	return s.memberRepo.Update(ctx, member)
}

// HardDelete permanently deletes a member.
func (s *MemberService) HardDelete(ctx context.Context, id uuid.UUID) error {
	_, err := s.memberRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	return s.memberRepo.Delete(ctx, id)
}

// ListActiveAt returns all members active at a given date.
func (s *MemberService) ListActiveAt(ctx context.Context, date time.Time) ([]domain.Member, error) {
	return s.memberRepo.ListActiveAt(ctx, date)
}
