package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// UserService manages user accounts (admin only).
type UserService struct {
	users            repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
}

// NewUserService creates a new user service.
func NewUserService(users repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository) *UserService {
	return &UserService{users: users, refreshTokenRepo: refreshTokenRepo}
}

// UserInput holds the editable fields of an account.
type UserInput struct {
	Email     string
	FirstName *string
	LastName  *string
	Role      domain.UserRole
	IsActive  bool
}

func (in *UserInput) normalize() error {
	in.Email = strings.TrimSpace(in.Email)
	if !strings.Contains(in.Email, "@") || strings.ContainsAny(in.Email, " \t") {
		return fmt.Errorf("%w: ungültige E-Mail-Adresse", ErrInvalidInput)
	}
	if in.Role != domain.UserRoleAdmin && in.Role != domain.UserRoleUser &&
		in.Role != domain.UserRoleParentWork && in.Role != domain.UserRolePARENT {
		return fmt.Errorf("%w: Ungültige Rolle", ErrInvalidInput)
	}
	in.FirstName = trimmedOrNil(in.FirstName)
	in.LastName = trimmedOrNil(in.LastName)
	return nil
}

func trimmedOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func validatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("%w: Passwort muss mindestens %d Zeichen haben", ErrInvalidInput, MinPasswordLength)
	}
	return nil
}

func mapUserRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrDuplicate):
		return fmt.Errorf("%w: E-Mail-Adresse wird bereits verwendet", ErrConflict)
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	default:
		return err
	}
}

// List returns all accounts.
func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

// Create adds an account with an initial password.
func (s *UserService) Create(ctx context.Context, in UserInput, password string) (*domain.User, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Email:        in.Email,
		PasswordHash: hash,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Role:         in.Role,
		IsActive:     in.IsActive,
	}
	if err := s.linkParent(ctx, user); err != nil {
		return nil, err
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, mapUserRepoError(err)
	}
	return s.users.GetByID(ctx, user.ID)
}

// Update changes an account. Admins cannot deactivate or demote themselves,
// which also guarantees at least one active admin remains. Deactivating an
// account revokes its refresh tokens.
func (s *UserService) Update(ctx context.Context, actorID, id uuid.UUID, in UserInput) (*domain.User, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, mapUserRepoError(err)
	}
	if id == actorID && (!in.IsActive || in.Role != domain.UserRoleAdmin) {
		return nil, fmt.Errorf("%w: Das eigene Konto kann nicht deaktiviert oder herabgestuft werden", ErrInvalidInput)
	}
	deactivated := user.IsActive && !in.IsActive
	wasLinked := user.Role == domain.UserRolePARENT && user.ParentID != nil
	oldEmail := user.Email

	user.Email = in.Email
	user.FirstName = in.FirstName
	user.LastName = in.LastName
	user.Role = in.Role
	user.IsActive = in.IsActive
	if !wasLinked || in.Role != domain.UserRolePARENT {
		if err := s.linkParent(ctx, user); err != nil {
			return nil, err
		}
	}
	if wasLinked && in.Role == domain.UserRolePARENT && oldEmail != in.Email {
		writer, ok := s.users.(interface {
			UpdateLinkedUser(context.Context, *domain.User, uuid.UUID) error
		})
		if !ok {
			return nil, fmt.Errorf("linked user update unavailable")
		}
		if err := writer.UpdateLinkedUser(ctx, user, actorID); err != nil {
			return nil, mapUserRepoError(err)
		}
	} else if err := s.users.Update(ctx, user); err != nil {
		return nil, mapUserRepoError(err)
	}
	if deactivated {
		if err := s.refreshTokenRepo.DeleteByUserID(ctx, id); err != nil {
			return nil, err
		}
	}
	return s.users.GetByID(ctx, user.ID)
}

func (s *UserService) linkParent(ctx context.Context, user *domain.User) error {
	if user.Role != domain.UserRolePARENT {
		user.ParentID, user.ParentName = nil, nil
		return nil
	}
	ids, err := s.users.FindParentMatches(ctx, user.Email)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("%w: Kein Elternteil mit dieser E-Mail-Adresse", ErrInvalidInput)
	}
	if len(ids) > 1 {
		return fmt.Errorf("%w: Mehrere Elternteile mit dieser E-Mail-Adresse", ErrConflict)
	}
	linked, err := s.users.ParentLinkedToOther(ctx, ids[0], user.ID)
	if err != nil {
		return err
	}
	if linked {
		return fmt.Errorf("%w: Elternteil ist bereits mit einem anderen Konto verknüpft", ErrConflict)
	}
	user.ParentID = &ids[0]
	return nil
}

// SetPassword sets a new password for an account (admin reset) and ends its sessions.
func (s *UserService) SetPassword(ctx context.Context, id uuid.UUID, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, id, hash); err != nil {
		return mapUserRepoError(err)
	}
	return s.refreshTokenRepo.DeleteByUserID(ctx, id)
}
