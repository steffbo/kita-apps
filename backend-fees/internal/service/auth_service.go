package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// AuthService handles authentication logic using environment configuration.
type AuthService struct {
	username         string
	passwordHash     string
	refreshExpiry    time.Duration
	refreshTokenRepo repository.RefreshTokenRepository
}

// NewAuthService creates a new auth service.
func NewAuthService(username, password string, refreshExpiry time.Duration, refreshTokenRepo repository.RefreshTokenRepository) *AuthService {
	var passwordHash string
	if password != "" {
		// Hash the password for storage/comparison
		hash, _ := auth.HashPassword(password)
		passwordHash = hash
	}

	return &AuthService{
		username:         username,
		passwordHash:     passwordHash,
		refreshExpiry:    refreshExpiry,
		refreshTokenRepo: refreshTokenRepo,
	}
}

// Authenticate validates credentials and returns the user.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*domain.User, error) {
	if s.username == "" || s.passwordHash == "" {
		return nil, ErrUnauthorized
	}

	if email != s.username {
		return nil, ErrUnauthorized
	}

	if !auth.CheckPassword(password, s.passwordHash) {
		return nil, ErrUnauthorized
	}

	// Return a static user
	admin := "Admin"
	knirps := "Knirpsenstadt"
	return &domain.User{
		ID:           uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
		Email:        s.username,
		FirstName:    &admin,
		LastName:     &knirps,
		Role:         domain.UserRoleAdmin,
		IsActive:     true,
		PasswordHash: s.passwordHash,
	}, nil
}

// GetUserByID retrieves a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if id.String() != "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" {
		return nil, ErrNotFound
	}

	admin := "Admin"
	knirps := "Knirpsenstadt"
	return &domain.User{
		ID:           id,
		Email:        s.username,
		FirstName:    &admin,
		LastName:     &knirps,
		Role:         domain.UserRoleAdmin,
		IsActive:     true,
		PasswordHash: s.passwordHash,
	}, nil
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
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if userID.String() != "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" {
		return ErrNotFound
	}

	// Verify current password
	if !auth.CheckPassword(currentPassword, s.passwordHash) {
		return ErrUnauthorized
	}

	// Validate new password
	if len(newPassword) < 8 {
		return ErrInvalidInput
	}

	// Hash new password
	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update stored password hash
	s.passwordHash = newHash

	// Revoke all refresh tokens to force re-login on other devices
	return s.RevokeAllUserTokens(ctx, userID)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
