package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/repository"
)

type UserStore interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type RefreshTokenStore interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	RevokeActiveByHash(ctx context.Context, tokenHash string) (*repository.RefreshToken, error)
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type AuthService struct {
	users         UserStore
	refreshTokens RefreshTokenStore
	jwtService    *auth.JWTService
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         *domain.User
}

func NewAuthService(users UserStore, refreshTokens RefreshTokenStore, jwtService *auth.JWTService) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		jwtService:    jwtService,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, ErrBadRequest
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if !canAuthenticate(user) || !auth.CheckPassword(password, *user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	result, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrBadRequest
	}

	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrUnauthorized
	}

	tokenHash := auth.TokenHash(refreshToken)
	storedToken, err := s.refreshTokens.RevokeActiveByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if storedToken == nil || storedToken.UserID != claims.UserID {
		return nil, ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if !canAuthenticate(user) {
		return nil, ErrUnauthorized
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !canAuthenticate(user) {
		return nil, ErrUnauthorized
	}
	return user, nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokens.RevokeAllForUser(ctx, userID)
}

func (s *AuthService) issueTokens(ctx context.Context, user *domain.User) (*AuthResult, error) {
	pair, err := s.jwtService.GenerateTokenPair(user.ID, user.Email, roleStrings(user.Roles))
	if err != nil {
		return nil, err
	}

	if err := s.refreshTokens.Create(ctx, user.ID, auth.TokenHash(pair.RefreshToken), pair.RefreshExpiresAt); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.AccessExpiresAt,
		User:         user,
	}, nil
}

func canAuthenticate(user *domain.User) bool {
	return user != nil &&
		user.Status == domain.UserStatusActive &&
		user.PasswordHash != nil &&
		*user.PasswordHash != ""
}

func roleStrings(roles []domain.UserRole) []string {
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		result = append(result, string(role))
	}
	return result
}
