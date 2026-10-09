package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
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
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.transaction_warnings (
			id, transaction_id, warning_type, message, expected_amount, actual_amount, 
			child_id, matched_fee_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, warning.ID, warning.TransactionID, warning.WarningType, warning.Message,
		warning.ExpectedAmount, warning.ActualAmount, warning.ChildID, warning.MatchedFeeID, warning.CreatedAt)
	return err
}

// GetByID retrieves a warning by its ID.
func (r *PostgresWarningRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TransactionWarning, error) {
	var warning domain.TransactionWarning
	err := conn(ctx, r.db).GetContext(ctx, &warning, `
		SELECT id, transaction_id, warning_type, message, expected_amount, actual_amount,
			   child_id, matched_fee_id, resolved_at, resolved_by, resolution_type, resolution_note, created_at
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
	err := conn(ctx, r.db).GetContext(ctx, &warning, `
		SELECT id, transaction_id, warning_type, message, expected_amount, actual_amount,
			   child_id, matched_fee_id, resolved_at, resolved_by, resolution_type, resolution_note, created_at
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

// WarningListOptions filters unresolved warnings by their bank transaction.
type WarningListOptions struct {
	Search         string
	SortBy         string
	SortDir        string
	TransactionIDs []uuid.UUID
}

// ListUnresolved excludes non-actionable MULTIPLE_OPEN_FEES warnings.
func (r *PostgresWarningRepository) ListUnresolved(
	ctx context.Context, offset, limit int, options ...WarningListOptions,
) ([]domain.TransactionWarning, int64, error) {
	var option WarningListOptions
	if len(options) > 0 {
		option = options[0]
	}
	condition := "w.resolved_at IS NULL AND w.warning_type != 'MULTIPLE_OPEN_FEES'"
	where, args, idx := buildSearchFilter(condition, literalTransactionSearch(option.Search), 1)
	if option.TransactionIDs != nil {
		where += fmt.Sprintf(" AND w.transaction_id = ANY($%d)", idx)
		args = append(args, pq.Array(option.TransactionIDs))
		idx++
	}
	from := " FROM fees.transaction_warnings w JOIN fees.bank_transactions bt ON bt.id = w.transaction_id"
	var total int64
	if err := conn(ctx, r.db).GetContext(ctx, &total, "SELECT COUNT(*)"+from+" WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	order := "w.created_at DESC, w.id ASC"
	if option.SortBy != "" {
		order = buildOrderByClause(option.SortBy, option.SortDir) + ", w.id ASC"
	}
	query := `SELECT w.id, w.transaction_id, w.warning_type, w.message, w.expected_amount, w.actual_amount,
  w.child_id, w.matched_fee_id, w.resolved_at, w.resolved_by, w.resolution_type,
  w.resolution_note, w.created_at` + from + " WHERE " + where + " ORDER BY " + order +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)
	var warnings []domain.TransactionWarning
	if err := conn(ctx, r.db).SelectContext(ctx, &warnings, query, args...); err != nil {
		return nil, 0, err
	}
	return warnings, total, nil
}

// Resolve marks a warning as resolved.
func (r *PostgresWarningRepository) Resolve(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID, resolutionType domain.ResolutionType, note string) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.transaction_warnings
		SET resolved_at = $2, resolved_by = $3, resolution_type = $4, resolution_note = $5
		WHERE id = $1 AND resolved_at IS NULL
	`, id, util.Now(), resolvedBy, resolutionType, note)
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
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.transaction_warnings
		SET resolved_at = $2, resolution_type = $3, resolution_note = $4
		WHERE transaction_id = $1 AND resolved_at IS NULL
	`, transactionID, util.Now(), resolutionType, note)
	return err
}

// Delete deletes a warning.
func (r *PostgresWarningRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.transaction_warnings WHERE id = $1`, id)
	return err
}
