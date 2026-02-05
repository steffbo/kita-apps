package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/email"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

type passwordResetToken struct {
	userID    uuid.UUID
	email     string
	expiresAt time.Time
}

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	SendPasswordResetEmail(to, token, baseURL string) error
	IsEnabled() bool
}

// AuthService handles authentication logic.
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	emailService     EmailSender
	emailLogRepo     repository.EmailLogRepository
	baseURL          string
	refreshExpiry    time.Duration
	resetExpiry      time.Duration
	resetTokens      map[string]passwordResetToken
	resetMutex       sync.Mutex
}

// NewAuthService creates a new auth service.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	emailService EmailSender,
	emailLogRepo repository.EmailLogRepository,
	baseURL string,
	refreshExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		emailService:     emailService,
		emailLogRepo:     emailLogRepo,
		baseURL:          baseURL,
		refreshExpiry:    refreshExpiry,
		resetExpiry:      time.Hour,
		resetTokens:      make(map[string]passwordResetToken),
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

// ChangePassword changes the password for a user after verifying the current password.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrNotFound
	}

	// Verify current password
	if !auth.CheckPassword(currentPassword, user.PasswordHash) {
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

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
		return err
	}

	// Revoke all refresh tokens to force re-login on other devices
	return s.refreshTokenRepo.DeleteByUserID(ctx, userID)
}

// RequestPasswordReset creates a password reset token and sends an email.
func (s *AuthService) RequestPasswordReset(ctx context.Context, emailAddr string) {
	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil || user == nil {
		// Don't reveal if user exists
		return
	}

	token, err := auth.GenerateRandomToken(32)
	if err != nil {
		return
	}

	s.resetMutex.Lock()
	s.resetTokens[token] = passwordResetToken{
		userID:    user.ID,
		email:     user.Email,
		expiresAt: time.Now().Add(s.resetExpiry),
	}
	s.resetMutex.Unlock()

	// Send password reset email
	if s.emailService != nil && s.emailService.IsEnabled() {
		if err := s.emailService.SendPasswordResetEmail(user.Email, token, s.baseURL); err != nil {
			log.Error().Err(err).Str("email", user.Email).Msg("failed to send password reset email")
		} else {
			subject, body := email.BuildPasswordResetEmail(token, s.baseURL)
			s.logPasswordResetEmail(ctx, user.Email, subject, body)
		}
	} else {
		// Fallback to logging when email is not configured
		log.Info().Str("email", user.Email).Str("token", token).Msg("password reset token generated (email not configured)")
	}
}

// ConfirmPasswordReset confirms a reset token and sets a new password.
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	s.resetMutex.Lock()
	resetToken, ok := s.resetTokens[token]
	s.resetMutex.Unlock()

	if !ok {
		return ErrInvalidInput
	}

	if time.Now().After(resetToken.expiresAt) {
		s.resetMutex.Lock()
		delete(s.resetTokens, token)
		s.resetMutex.Unlock()
		return ErrInvalidInput
	}

	// Validate new password
	if len(newPassword) < 8 {
		return ErrInvalidInput
	}

	// Hash new password
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, resetToken.userID, hash); err != nil {
		return err
	}

	// Revoke all refresh tokens
	s.refreshTokenRepo.DeleteByUserID(ctx, resetToken.userID)

	// Delete the reset token
	s.resetMutex.Lock()
	delete(s.resetTokens, token)
	s.resetMutex.Unlock()

	return nil
}

func (s *AuthService) logPasswordResetEmail(ctx context.Context, recipient, subject, body string) {
	if s.emailLogRepo == nil {
		return
	}

	bodyCopy := body
	logEntry := &domain.EmailLog{
		ID:        uuid.New(),
		SentAt:    time.Now().UTC(),
		ToEmail:   recipient,
		Subject:   subject,
		Body:      &bodyCopy,
		EmailType: domain.EmailLogTypePasswordReset,
	}
	if err := s.emailLogRepo.Create(ctx, logEntry); err != nil {
		log.Error().Err(err).Str("email", recipient).Msg("failed to log password reset email")
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
