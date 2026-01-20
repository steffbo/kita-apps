package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresMatchRepository is the PostgreSQL implementation of MatchRepository.
type PostgresMatchRepository struct {
	db *sqlx.DB
}

// NewPostgresMatchRepository creates a new PostgreSQL match repository.
func NewPostgresMatchRepository(db *sqlx.DB) *PostgresMatchRepository {
	return &PostgresMatchRepository{db: db}
}

// Create creates a new payment match.
func (r *PostgresMatchRepository) Create(ctx context.Context, match *domain.PaymentMatch) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.payment_matches (id, transaction_id, expectation_id, match_type, confidence, matched_at, matched_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, match.ID, match.TransactionID, match.ExpectationID, match.MatchType, match.Confidence, match.MatchedAt, match.MatchedBy)
	return err
}

// GetByID retrieves a payment match by ID.
func (r *PostgresMatchRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PaymentMatch, error) {
	var match domain.PaymentMatch
	err := r.db.GetContext(ctx, &match, `
		SELECT id, transaction_id, expectation_id, match_type, confidence, matched_at, matched_by
		FROM fees.payment_matches
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("match not found")
		}
		return nil, err
	}
	return &match, nil
}

// ExistsForExpectation checks if a match exists for a fee expectation.
func (r *PostgresMatchRepository) ExistsForExpectation(ctx context.Context, expectationID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM fees.payment_matches
		WHERE expectation_id = $1
	`, expectationID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsForTransaction checks if a match exists for a transaction.
func (r *PostgresMatchRepository) ExistsForTransaction(ctx context.Context, transactionID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM fees.payment_matches
		WHERE transaction_id = $1
	`, transactionID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByExpectation retrieves a match by its fee expectation ID.
func (r *PostgresMatchRepository) GetByExpectation(ctx context.Context, expectationID uuid.UUID) (*domain.PaymentMatch, error) {
	var match domain.PaymentMatch
	err := r.db.GetContext(ctx, &match, `
		SELECT id, transaction_id, expectation_id, match_type, confidence, matched_at, matched_by
		FROM fees.payment_matches
		WHERE expectation_id = $1
	`, expectationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}

// Delete deletes a payment match.
func (r *PostgresMatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.payment_matches WHERE id = $1`, id)
	return err
}
