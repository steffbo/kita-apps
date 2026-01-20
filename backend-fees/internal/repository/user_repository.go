package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresUserRepository is the PostgreSQL implementation of UserRepository.
type PostgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository.
func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// GetByID retrieves a user by ID.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM fees.users
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM fees.users
		WHERE email = $1
	`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Create creates a new user.
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt)
	return err
}

// Update updates an existing user.
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.users
		SET email = $2, password_hash = $3, first_name = $4, last_name = $5, role = $6, is_active = $7, updated_at = $8
		WHERE id = $1
	`, user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName, user.Role, user.IsActive, user.UpdatedAt)
	return err
}

// PostgresRefreshTokenRepository is the PostgreSQL implementation of RefreshTokenRepository.
type PostgresRefreshTokenRepository struct {
	db *sqlx.DB
}

// NewPostgresRefreshTokenRepository creates a new PostgreSQL refresh token repository.
func NewPostgresRefreshTokenRepository(db *sqlx.DB) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{db: db}
}

// Create stores a new refresh token.
func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	return err
}

// Exists checks if a refresh token exists and is valid.
func (r *PostgresRefreshTokenRepository) Exists(ctx context.Context, userID uuid.UUID, tokenHash string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM fees.refresh_tokens
		WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW()
	`, userID, tokenHash)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteByHash deletes a refresh token by its hash.
func (r *PostgresRefreshTokenRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.refresh_tokens
		WHERE token_hash = $1
	`, tokenHash)
	return err
}

// DeleteByUserID deletes all refresh tokens for a user.
func (r *PostgresRefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.refresh_tokens
		WHERE user_id = $1
	`, userID)
	return err
}

// DeleteExpired deletes all expired refresh tokens.
func (r *PostgresRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.refresh_tokens
		WHERE expires_at <= NOW()
	`)
	return err
}
