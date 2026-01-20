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

// PostgresTransactionRepository is the PostgreSQL implementation of TransactionRepository.
type PostgresTransactionRepository struct {
	db *sqlx.DB
}

// NewPostgresTransactionRepository creates a new PostgreSQL transaction repository.
func NewPostgresTransactionRepository(db *sqlx.DB) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{db: db}
}

// Create creates a new bank transaction.
func (r *PostgresTransactionRepository) Create(ctx context.Context, tx *domain.BankTransaction) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	if tx.ImportedAt.IsZero() {
		tx.ImportedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.bank_transactions (id, booking_date, value_date, payer_name, payer_iban,
		                                    description, amount, currency, import_batch_id, imported_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, tx.ID, tx.BookingDate, tx.ValueDate, tx.PayerName, tx.PayerIBAN,
		tx.Description, tx.Amount, tx.Currency, tx.ImportBatchID, tx.ImportedAt)
	return err
}

// GetByID retrieves a bank transaction by ID.
func (r *PostgresTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BankTransaction, error) {
	var tx domain.BankTransaction
	err := r.db.GetContext(ctx, &tx, `
		SELECT id, booking_date, value_date, payer_name, payer_iban,
		       description, amount, currency, import_batch_id, imported_at
		FROM fees.bank_transactions
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}
	return &tx, nil
}

// Exists checks if a similar transaction already exists (for deduplication).
func (r *PostgresTransactionRepository) Exists(ctx context.Context, bookingDate time.Time, payerIBAN *string, amount float64, description *string) (bool, error) {
	var count int

	// Build query based on available fields
	query := `
		SELECT COUNT(*)
		FROM fees.bank_transactions
		WHERE booking_date = $1 AND amount = $2
	`
	args := []interface{}{bookingDate, amount}
	argIdx := 3

	if payerIBAN != nil {
		query += " AND payer_iban = $" + string(rune('0'+argIdx))
		args = append(args, *payerIBAN)
		argIdx++
	}

	if description != nil {
		query += " AND description = $" + string(rune('0'+argIdx))
		args = append(args, *description)
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListUnmatched retrieves transactions that haven't been matched to any fee.
func (r *PostgresTransactionRepository) ListUnmatched(ctx context.Context, offset, limit int) ([]domain.BankTransaction, int64, error) {
	var transactions []domain.BankTransaction
	var total int64

	// Count total unmatched
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*)
		FROM fees.bank_transactions bt
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE pm.id IS NULL AND bt.amount > 0
	`)
	if err != nil {
		return nil, 0, err
	}

	// Fetch unmatched transactions
	err = r.db.SelectContext(ctx, &transactions, `
		SELECT bt.id, bt.booking_date, bt.value_date, bt.payer_name, bt.payer_iban,
		       bt.description, bt.amount, bt.currency, bt.import_batch_id, bt.imported_at
		FROM fees.bank_transactions bt
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE pm.id IS NULL AND bt.amount > 0
		ORDER BY bt.booking_date DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// GetBatches retrieves import batch history.
func (r *PostgresTransactionRepository) GetBatches(ctx context.Context, offset, limit int) ([]domain.ImportBatch, int64, error) {
	var batches []domain.ImportBatch
	var total int64

	// Count total batches
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(DISTINCT import_batch_id)
		FROM fees.bank_transactions
		WHERE import_batch_id IS NOT NULL
	`)
	if err != nil {
		return nil, 0, err
	}

	// Fetch batch summaries
	err = r.db.SelectContext(ctx, &batches, `
		SELECT 
			bt.import_batch_id as id,
			MIN(bt.imported_at) as imported_at,
			COUNT(*) as transaction_count,
			COUNT(pm.id) as matched_count,
			'' as file_name
		FROM fees.bank_transactions bt
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE bt.import_batch_id IS NOT NULL
		GROUP BY bt.import_batch_id
		ORDER BY MIN(bt.imported_at) DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return batches, total, nil
}
