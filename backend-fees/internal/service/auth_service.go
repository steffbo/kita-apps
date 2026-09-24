package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// BootstrapAdminID is the ID of the admin created from USER_NAME/USER_PASSWORD.
// It equals the former static admin identity, so existing refresh tokens and
// stored actor IDs (imported_by, matched_by, …) stay attached to that account.
var BootstrapAdminID = uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

// MinPasswordLength is the minimum length for new passwords.
const MinPasswordLength = 8

// dummyPasswordHash keeps login timing similar for unknown emails.
var dummyPasswordHash, _ = auth.HashPassword("timing-equaliser")

// AuthService handles login, sessions and the own password of users stored in fees.users.
type AuthService struct {
	users            repository.UserRepository
	refreshExpiry    time.Duration
	refreshTokenRepo repository.RefreshTokenRepository
}

// NewAuthService creates a new auth service.
func NewAuthService(users repository.UserRepository, refreshExpiry time.Duration, refreshTokenRepo repository.RefreshTokenRepository) *AuthService {
	return &AuthService{
		users:            users,
		refreshExpiry:    refreshExpiry,
		refreshTokenRepo: refreshTokenRepo,
	}
}

// BootstrapAdmin creates the admin from USER_NAME/USER_PASSWORD when no user
// with that email exists yet. It never touches an existing account, so a
// password changed in the app survives restarts. Deleting the row and
// restarting re-creates it from the env (recovery path).
func (s *AuthService) BootstrapAdmin(ctx context.Context, email, password string) (bool, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return false, nil
	}
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return false, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return false, fmt.Errorf("look up bootstrap admin: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}
	admin := "Admin"
	knirps := "Knirpsenstadt"
	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    &admin,
		LastName:     &knirps,
		Role:         domain.UserRoleAdmin,
		IsActive:     true,
	}
	// Reuse the former static ID unless another account already holds it.
	if _, err := s.users.GetByID(ctx, BootstrapAdminID); errors.Is(err, repository.ErrNotFound) {
		user.ID = BootstrapAdminID
	}
	if err := s.users.Create(ctx, user); err != nil {
		return false, fmt.Errorf("create bootstrap admin: %w", err)
	}
	return true, nil
}

// Authenticate validates credentials of an active user.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.users.GetByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			auth.CheckPassword(password, dummyPasswordHash)
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if !auth.CheckPassword(password, user.PasswordHash) || !user.IsActive {
		return nil, ErrUnauthorized
	}
	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return user, err
}

// StoreRefreshToken stores a refresh token hash.
func (s *AuthService) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	hash := hashToken(token)
	expiresAt := time.Now().Add(s.refreshExpiry)

	return s.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	})
}

// ValidateRefreshToken checks if a refresh token is valid.
func (s *AuthService) ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	hash := hashToken(token)
	return s.refreshTokenRepo.Exists(ctx, userID, hash)
}

// RevokeRefreshToken invalidates a refresh token.
func (s *AuthService) RevokeRefreshToken(ctx context.Context, token string) error {
	hash := hashToken(token)
	return s.refreshTokenRepo.DeleteByHash(ctx, hash)
}

// RevokeAllUserTokens invalidates all refresh tokens for a user.
func (s *AuthService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.DeleteByUserID(ctx, userID)
}

// ChangePassword changes the password for a user after verifying the current password.
// All refresh tokens of the user are revoked (other devices must log in again).
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(currentPassword, user.PasswordHash) {
		return ErrUnauthorized
	}
	if len(newPassword) < MinPasswordLength {
		return ErrInvalidInput
	}
	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, userID, newHash); err != nil {
		return err
	}
	return s.RevokeAllUserTokens(ctx, userID)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
