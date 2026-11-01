package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

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
	Roles     []domain.UserRole
	IsActive  bool
}

func (in *UserInput) normalize() error {
	in.Email = strings.TrimSpace(in.Email)
	if !strings.Contains(in.Email, "@") || strings.ContainsAny(in.Email, " \t") {
		return fmt.Errorf("%w: ungültige E-Mail-Adresse", ErrInvalidInput)
	}
	if len(in.Roles) == 0 {
		return fmt.Errorf("%w: Mindestens eine Rolle auswählen", ErrInvalidInput)
	}
	for _, role := range in.Roles {
		if !slices.Contains(domain.AssignableRoles, role) {
			return fmt.Errorf("%w: Ungültige Rolle", ErrInvalidInput)
		}
	}
	// Most privileged first and without duplicates, whatever the client sent.
	in.Roles = slices.DeleteFunc(slices.Clone(domain.AssignableRoles), func(role domain.UserRole) bool {
		return !slices.Contains(in.Roles, role)
	})
	in.FirstName = trimmedOrNil(in.FirstName)
	in.LastName = trimmedOrNil(in.LastName)
	return nil
}

func (in *UserInput) storedRoles() pq.StringArray {
	roles := make(pq.StringArray, len(in.Roles))
	for i, role := range in.Roles {
		roles[i] = string(role)
	}
	return roles
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

// ImpersonationTarget returns the account an admin may act as. Admins (also the
// admin themselves) and inactive accounts are excluded, so impersonation never
// escalates privileges or chains.
func (s *UserService) ImpersonationTarget(ctx context.Context, actorID, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, mapUserRepoError(err)
	}
	switch {
	case id == actorID || user.HasRole(domain.UserRoleAdmin):
		return nil, fmt.Errorf("%w: Administratoren können nicht übernommen werden", ErrInvalidInput)
	case !user.IsActive:
		return nil, fmt.Errorf("%w: Deaktivierte Benutzer können nicht übernommen werden", ErrInvalidInput)
	}
	return user, nil
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
		Role:         in.Roles[0],
		Roles:        in.storedRoles(),
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

// Update changes an account. Admins cannot deactivate themselves or give up the
// admin role, which also guarantees at least one active admin remains. The active
// role stays while it is assigned. Deactivating an account revokes its refresh tokens.
func (s *UserService) Update(ctx context.Context, actorID, id uuid.UUID, in UserInput) (*domain.User, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, mapUserRepoError(err)
	}
	if id == actorID && (!in.IsActive || !slices.Contains(in.Roles, domain.UserRoleAdmin)) {
		return nil, fmt.Errorf("%w: Das eigene Konto kann nicht deaktiviert oder herabgestuft werden", ErrInvalidInput)
	}
	deactivated := user.IsActive && !in.IsActive
	wasLinked := user.HasRole(domain.UserRolePARENT) && user.ParentID != nil
	staysParent := slices.Contains(in.Roles, domain.UserRolePARENT)
	oldEmail := user.Email

	user.Email = in.Email
	user.FirstName = in.FirstName
	user.LastName = in.LastName
	user.Roles = in.storedRoles()
	if !slices.Contains(in.Roles, user.Role) {
		user.Role = in.Roles[0]
	}
	user.IsActive = in.IsActive
	if !wasLinked || !staysParent {
		if err := s.linkParent(ctx, user); err != nil {
			return nil, err
		}
	}
	if wasLinked && staysParent && oldEmail != in.Email {
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
	if !user.HasRole(domain.UserRolePARENT) {
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
