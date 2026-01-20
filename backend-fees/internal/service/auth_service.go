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

// AuthService handles authentication logic.
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	refreshExpiry    time.Duration
}

// NewAuthService creates a new auth service.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	refreshExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		refreshExpiry:    refreshExpiry,
	}
}

// Authenticate validates credentials and returns the user.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if !user.IsActive {
		return nil, ErrUnauthorized
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		return nil, ErrUnauthorized
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
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

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
