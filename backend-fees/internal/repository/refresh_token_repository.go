package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

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
