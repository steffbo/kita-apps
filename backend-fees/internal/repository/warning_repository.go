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

// PostgresWarningRepository is the PostgreSQL implementation of WarningRepository.
type PostgresWarningRepository struct {
	db *sqlx.DB
}

// NewPostgresWarningRepository creates a new PostgreSQL warning repository.
func NewPostgresWarningRepository(db *sqlx.DB) *PostgresWarningRepository {
	return &PostgresWarningRepository{db: db}
}

// Create creates a new transaction warning.
func (r *PostgresWarningRepository) Create(ctx context.Context, warning *domain.TransactionWarning) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.transaction_warnings (
			id, transaction_id, warning_type, message, expected_amount, actual_amount, 
			child_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, warning.ID, warning.TransactionID, warning.WarningType, warning.Message,
		warning.ExpectedAmount, warning.ActualAmount, warning.ChildID, warning.CreatedAt)
	return err
}

// GetByID retrieves a warning by its ID.
func (r *PostgresWarningRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TransactionWarning, error) {
	var warning domain.TransactionWarning
	err := r.db.GetContext(ctx, &warning, `
		SELECT id, transaction_id, warning_type, message, expected_amount, actual_amount,
			   child_id, resolved_at, resolved_by, resolution_type, resolution_note, created_at
		FROM fees.transaction_warnings
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("warning not found")
		}
		return nil, err
	}
	return &warning, nil
}

// GetByTransactionID retrieves a warning by its transaction ID.
func (r *PostgresWarningRepository) GetByTransactionID(ctx context.Context, transactionID uuid.UUID) (*domain.TransactionWarning, error) {
	var warning domain.TransactionWarning
	err := r.db.GetContext(ctx, &warning, `
		SELECT id, transaction_id, warning_type, message, expected_amount, actual_amount,
			   child_id, resolved_at, resolved_by, resolution_type, resolution_note, created_at
		FROM fees.transaction_warnings
		WHERE transaction_id = $1
	`, transactionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &warning, nil
}

// ListUnresolved retrieves all unresolved warnings with pagination.
func (r *PostgresWarningRepository) ListUnresolved(ctx context.Context, offset, limit int) ([]domain.TransactionWarning, int64, error) {
	var warnings []domain.TransactionWarning
	var total int64

	// Count total unresolved
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM fees.transaction_warnings WHERE resolved_at IS NULL
	`)
	if err != nil {
		return nil, 0, err
	}

	// Fetch with pagination
	err = r.db.SelectContext(ctx, &warnings, `
		SELECT w.id, w.transaction_id, w.warning_type, w.message, w.expected_amount, w.actual_amount,
			   w.child_id, w.resolved_at, w.resolved_by, w.resolution_type, w.resolution_note, w.created_at
		FROM fees.transaction_warnings w
		WHERE w.resolved_at IS NULL
		ORDER BY w.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return warnings, total, nil
}

// Resolve marks a warning as resolved.
func (r *PostgresWarningRepository) Resolve(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID, resolutionType domain.ResolutionType, note string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE fees.transaction_warnings
		SET resolved_at = $2, resolved_by = $3, resolution_type = $4, resolution_note = $5
		WHERE id = $1 AND resolved_at IS NULL
	`, id, time.Now(), resolvedBy, resolutionType, note)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("warning not found or already resolved")
	}
	return nil
}

// ResolveByTransactionID marks a warning as resolved by its transaction ID.
// This is used when a transaction is manually matched, auto-resolving any associated warning.
func (r *PostgresWarningRepository) ResolveByTransactionID(ctx context.Context, transactionID uuid.UUID, resolutionType domain.ResolutionType, note string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.transaction_warnings
		SET resolved_at = $2, resolution_type = $3, resolution_note = $4
		WHERE transaction_id = $1 AND resolved_at IS NULL
	`, transactionID, time.Now(), resolutionType, note)
	return err
}

// Delete deletes a warning.
func (r *PostgresWarningRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.transaction_warnings WHERE id = $1`, id)
	return err
}
