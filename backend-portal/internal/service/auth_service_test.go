package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/repository"
)

func TestAuthServiceLoginIssuesTokensForActiveUser(t *testing.T) {
	passwordHash, err := auth.HashPassword("Secret123!")
	require.NoError(t, err)

	user := testUser(passwordHash, domain.UserStatusActive)
	users := &fakeUserStore{byEmail: map[string]*domain.User{user.Email: user}, byID: map[uuid.UUID]*domain.User{user.ID: user}}
	refreshTokens := newFakeRefreshTokenStore()
	service := NewAuthService(users, refreshTokens, auth.NewJWTService("test-secret", time.Minute, time.Hour, "test-portal"))

	result, err := service.Login(context.Background(), user.Email, "Secret123!")

	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Equal(t, user.ID, result.User.ID)
	require.True(t, users.lastLoginUpdated)
	require.Len(t, refreshTokens.tokens, 1)
}

func TestAuthServiceLoginRejectsInvalidCredentials(t *testing.T) {
	passwordHash, err := auth.HashPassword("Secret123!")
	require.NoError(t, err)

	user := testUser(passwordHash, domain.UserStatusActive)
	users := &fakeUserStore{byEmail: map[string]*domain.User{user.Email: user}, byID: map[uuid.UUID]*domain.User{user.ID: user}}
	service := NewAuthService(users, newFakeRefreshTokenStore(), auth.NewJWTService("test-secret", time.Minute, time.Hour, "test-portal"))

	result, err := service.Login(context.Background(), user.Email, "wrong-password")

	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.Nil(t, result)
}

func TestAuthServiceRefreshRotatesStoredToken(t *testing.T) {
	passwordHash, err := auth.HashPassword("Secret123!")
	require.NoError(t, err)

	user := testUser(passwordHash, domain.UserStatusActive)
	users := &fakeUserStore{byEmail: map[string]*domain.User{user.Email: user}, byID: map[uuid.UUID]*domain.User{user.ID: user}}
	refreshTokens := newFakeRefreshTokenStore()
	service := NewAuthService(users, refreshTokens, auth.NewJWTService("test-secret", time.Minute, time.Hour, "test-portal"))

	loginResult, err := service.Login(context.Background(), user.Email, "Secret123!")
	require.NoError(t, err)

	refreshResult, err := service.Refresh(context.Background(), loginResult.RefreshToken)

	require.NoError(t, err)
	require.NotEqual(t, loginResult.RefreshToken, refreshResult.RefreshToken)
	require.True(t, refreshTokens.revoked[auth.TokenHash(loginResult.RefreshToken)])
	require.Len(t, refreshTokens.tokens, 2)
}

func TestAuthServiceRefreshRejectsReusedToken(t *testing.T) {
	passwordHash, err := auth.HashPassword("Secret123!")
	require.NoError(t, err)

	user := testUser(passwordHash, domain.UserStatusActive)
	users := &fakeUserStore{byEmail: map[string]*domain.User{user.Email: user}, byID: map[uuid.UUID]*domain.User{user.ID: user}}
	refreshTokens := newFakeRefreshTokenStore()
	service := NewAuthService(users, refreshTokens, auth.NewJWTService("test-secret", time.Minute, time.Hour, "test-portal"))

	loginResult, err := service.Login(context.Background(), user.Email, "Secret123!")
	require.NoError(t, err)

	_, err = service.Refresh(context.Background(), loginResult.RefreshToken)
	require.NoError(t, err)

	reusedResult, err := service.Refresh(context.Background(), loginResult.RefreshToken)

	require.ErrorIs(t, err, ErrUnauthorized)
	require.Nil(t, reusedResult)
	require.Len(t, refreshTokens.tokens, 2)
}

func TestAuthServiceLogoutAllRevokesUserTokens(t *testing.T) {
	passwordHash, err := auth.HashPassword("Secret123!")
	require.NoError(t, err)

	user := testUser(passwordHash, domain.UserStatusActive)
	otherUser := testUser(passwordHash, domain.UserStatusActive)
	otherUser.Email = "other@example.test"
	users := &fakeUserStore{
		byEmail: map[string]*domain.User{user.Email: user, otherUser.Email: otherUser},
		byID:    map[uuid.UUID]*domain.User{user.ID: user, otherUser.ID: otherUser},
	}
	refreshTokens := newFakeRefreshTokenStore()
	service := NewAuthService(users, refreshTokens, auth.NewJWTService("test-secret", time.Minute, time.Hour, "test-portal"))

	firstResult, err := service.Login(context.Background(), user.Email, "Secret123!")
	require.NoError(t, err)
	secondResult, err := service.Login(context.Background(), user.Email, "Secret123!")
	require.NoError(t, err)
	otherResult, err := service.Login(context.Background(), otherUser.Email, "Secret123!")
	require.NoError(t, err)

	err = service.LogoutAll(context.Background(), user.ID)

	require.NoError(t, err)
	require.True(t, refreshTokens.revoked[auth.TokenHash(firstResult.RefreshToken)])
	require.True(t, refreshTokens.revoked[auth.TokenHash(secondResult.RefreshToken)])
	require.False(t, refreshTokens.revoked[auth.TokenHash(otherResult.RefreshToken)])
}

func testUser(passwordHash string, status domain.UserStatus) *domain.User {
	return &domain.User{
		ID:           uuid.New(),
		Email:        "admin@example.test",
		PasswordHash: &passwordHash,
		Status:       status,
		Roles:        []domain.UserRole{domain.UserRoleAdmin},
	}
}

type fakeUserStore struct {
	byEmail          map[string]*domain.User
	byID             map[uuid.UUID]*domain.User
	lastLoginUpdated bool
}

func (s *fakeUserStore) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	return s.byEmail[email], nil
}

func (s *fakeUserStore) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	return s.byID[id], nil
}

func (s *fakeUserStore) UpdateLastLogin(_ context.Context, _ uuid.UUID) error {
	s.lastLoginUpdated = true
	return nil
}

type fakeRefreshTokenStore struct {
	tokens  map[string]*repository.RefreshToken
	revoked map[string]bool
}

func newFakeRefreshTokenStore() *fakeRefreshTokenStore {
	return &fakeRefreshTokenStore{
		tokens:  make(map[string]*repository.RefreshToken),
		revoked: make(map[string]bool),
	}
}

func (s *fakeRefreshTokenStore) Create(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	s.tokens[tokenHash] = &repository.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (s *fakeRefreshTokenStore) RevokeActiveByHash(_ context.Context, tokenHash string) (*repository.RefreshToken, error) {
	token := s.tokens[tokenHash]
	if token == nil || s.revoked[tokenHash] || !token.ExpiresAt.After(time.Now()) {
		return nil, nil
	}
	s.revoked[tokenHash] = true
	now := time.Now()
	token.RevokedAt = &now
	return token, nil
}

func (s *fakeRefreshTokenStore) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	for tokenHash, token := range s.tokens {
		if token.UserID == userID {
			s.revoked[tokenHash] = true
		}
	}
	return nil
}
