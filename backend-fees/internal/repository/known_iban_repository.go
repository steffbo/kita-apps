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

// PostgresKnownIBANRepository is the PostgreSQL implementation of KnownIBANRepository.
type PostgresKnownIBANRepository struct {
	db *sqlx.DB
}

// NewPostgresKnownIBANRepository creates a new PostgreSQL known IBAN repository.
func NewPostgresKnownIBANRepository(db *sqlx.DB) *PostgresKnownIBANRepository {
	return &PostgresKnownIBANRepository{db: db}
}

// Create creates or updates a known IBAN entry.
func (r *PostgresKnownIBANRepository) Create(ctx context.Context, iban *domain.KnownIBAN) error {
	now := time.Now()
	iban.CreatedAt = now
	iban.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.known_ibans (iban, payer_name, status, child_id, reason, 
		                              original_transaction_id, original_description, original_amount, 
		                              created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (iban) DO UPDATE SET
			payer_name = EXCLUDED.payer_name,
			status = EXCLUDED.status,
			child_id = EXCLUDED.child_id,
			reason = EXCLUDED.reason,
			original_transaction_id = EXCLUDED.original_transaction_id,
			original_description = EXCLUDED.original_description,
			original_amount = EXCLUDED.original_amount,
			updated_at = EXCLUDED.updated_at
	`, iban.IBAN, iban.PayerName, iban.Status, iban.ChildID, iban.Reason,
		iban.OriginalTransactionID, iban.OriginalDescription, iban.OriginalAmount,
		iban.CreatedAt, iban.UpdatedAt)
	return err
}

// GetByIBAN retrieves a known IBAN entry by IBAN.
func (r *PostgresKnownIBANRepository) GetByIBAN(ctx context.Context, iban string) (*domain.KnownIBAN, error) {
	var entry domain.KnownIBAN
	err := r.db.GetContext(ctx, &entry, `
		SELECT iban, payer_name, status, child_id, reason, 
		       original_transaction_id, original_description, original_amount,
		       created_at, updated_at
		FROM fees.known_ibans
		WHERE iban = $1
	`, iban)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &entry, nil
}

// IsBlacklisted checks if an IBAN is blacklisted.
func (r *PostgresKnownIBANRepository) IsBlacklisted(ctx context.Context, iban string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM fees.known_ibans
		WHERE iban = $1 AND status = 'blacklisted'
	`, iban)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsTrusted checks if an IBAN is trusted.
func (r *PostgresKnownIBANRepository) IsTrusted(ctx context.Context, iban string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM fees.known_ibans
		WHERE iban = $1 AND status = 'trusted'
	`, iban)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByStatus lists known IBANs by status.
func (r *PostgresKnownIBANRepository) ListByStatus(ctx context.Context, status domain.KnownIBANStatus, offset, limit int) ([]domain.KnownIBAN, int64, error) {
	var entries []domain.KnownIBAN
	var total int64

	// Count total
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*)
		FROM fees.known_ibans
		WHERE status = $1
	`, status)
	if err != nil {
		return nil, 0, err
	}

	// Fetch entries
	err = r.db.SelectContext(ctx, &entries, `
		SELECT iban, payer_name, status, child_id, reason, 
		       original_transaction_id, original_description, original_amount,
		       created_at, updated_at
		FROM fees.known_ibans
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// ListTrustedByChildWithCounts returns trusted IBANs for a child with transaction counts.
func (r *PostgresKnownIBANRepository) ListTrustedByChildWithCounts(ctx context.Context, childID uuid.UUID) ([]domain.KnownIBANSummary, error) {
	var entries []domain.KnownIBANSummary
	err := r.db.SelectContext(ctx, &entries, `
		SELECT
			ki.iban,
			ki.payer_name,
			COALESCE(COUNT(DISTINCT pm.transaction_id) FILTER (WHERE fe.child_id = $1), 0) AS transaction_count
		FROM fees.known_ibans ki
		LEFT JOIN fees.bank_transactions bt ON bt.payer_iban = ki.iban
		LEFT JOIN fees.payment_matches pm ON pm.transaction_id = bt.id
		LEFT JOIN fees.fee_expectations fe ON fe.id = pm.expectation_id
		WHERE ki.child_id = $1 AND ki.status = 'trusted'
		GROUP BY ki.iban, ki.payer_name
		ORDER BY MAX(ki.updated_at) DESC
	`, childID)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// Delete removes a known IBAN entry.
func (r *PostgresKnownIBANRepository) Delete(ctx context.Context, iban string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.known_ibans
		WHERE iban = $1
	`, iban)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("iban not found")
	}

	return nil
}

// UpdateChildLink updates the child linkage for a known IBAN.
func (r *PostgresKnownIBANRepository) UpdateChildLink(ctx context.Context, iban string, childID *uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE fees.known_ibans
		SET child_id = $2, updated_at = NOW()
		WHERE iban = $1
	`, iban, childID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("iban not found")
	}

	return nil
}

// GetBlacklistedIBANs returns all blacklisted IBANs as a set for efficient lookup.
func (r *PostgresKnownIBANRepository) GetBlacklistedIBANs(ctx context.Context) (map[string]bool, error) {
	var ibans []string
	err := r.db.SelectContext(ctx, &ibans, `
		SELECT iban FROM fees.known_ibans WHERE status = 'blacklisted'
	`)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, iban := range ibans {
		result[iban] = true
	}
	return result, nil
}
