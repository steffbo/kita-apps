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
	if in.Role != domain.UserRoleAdmin && in.Role != domain.UserRoleUser {
		return fmt.Errorf("%w: Rolle muss ADMIN oder USER sein", ErrInvalidInput)
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
	if err := s.users.Create(ctx, user); err != nil {
		return nil, mapUserRepoError(err)
	}
	return user, nil
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

	user.Email = in.Email
	user.FirstName = in.FirstName
	user.LastName = in.LastName
	user.Role = in.Role
	user.IsActive = in.IsActive
	if err := s.users.Update(ctx, user); err != nil {
		return nil, mapUserRepoError(err)
	}
	if deactivated {
		if err := s.refreshTokenRepo.DeleteByUserID(ctx, id); err != nil {
			return nil, err
		}
	}
	return user, nil
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
