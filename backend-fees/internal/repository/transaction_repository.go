package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
		       description, amount, currency, import_batch_id, imported_at,
		       is_hidden, hidden_at, hidden_by
		FROM fees.bank_transactions
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &tx, nil
}

// GetByIDs retrieves multiple transactions by their IDs.
func (r *PostgresTransactionRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.BankTransaction, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.BankTransaction), nil
	}

	var transactions []domain.BankTransaction
	err := r.db.SelectContext(ctx, &transactions, `
		SELECT id, booking_date, value_date, payer_name, payer_iban,
		       description, amount, currency, import_batch_id, imported_at,
		       is_hidden, hidden_at, hidden_by
		FROM fees.bank_transactions
		WHERE id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*domain.BankTransaction, len(transactions))
	for i := range transactions {
		result[transactions[i].ID] = &transactions[i]
	}
	return result, nil
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
		query += fmt.Sprintf(" AND payer_iban = $%d", argIdx)
		args = append(args, *payerIBAN)
		argIdx++
	}

	if description != nil {
		query += fmt.Sprintf(" AND description = $%d", argIdx)
		args = append(args, *description)
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// buildSearchFilter builds the search WHERE clause and returns the clause, args, and next arg index.
func buildSearchFilter(baseWhereClause, search string, startArgIdx int) (string, []interface{}, int) {
	whereClause := baseWhereClause
	args := []interface{}{}
	argIdx := startArgIdx

	if search != "" {
		whereClause += fmt.Sprintf(" AND (bt.payer_name ILIKE $%d OR bt.description ILIKE $%d)", argIdx, argIdx+1)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
		argIdx += 2
	}

	return whereClause, args, argIdx
}

// buildOrderByClause builds the ORDER BY clause based on sortBy and sortDir parameters.
func buildOrderByClause(sortBy, sortDir string) string {
	orderColumn := "bt.booking_date"
	switch sortBy {
	case "payer":
		orderColumn = "bt.payer_name"
	case "description":
		orderColumn = "bt.description"
	case "amount":
		orderColumn = "bt.amount"
	case "date":
		orderColumn = "bt.booking_date"
	}

	orderDirection := "DESC"
	if sortDir == "asc" {
		orderDirection = "ASC"
	}

	return fmt.Sprintf("%s %s", orderColumn, orderDirection)
}

// ListUnmatched retrieves transactions that haven't been matched to any fee.
func (r *PostgresTransactionRepository) ListUnmatched(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
	var transactions []domain.BankTransaction
	var total int64

	whereClause, args, argIdx := buildSearchFilter("pm.id IS NULL AND bt.amount > 0 AND bt.is_hidden = false", search, 1)

	// Count total unmatched with search filter
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM fees.bank_transactions bt
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE %s
	`, whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildOrderByClause(sortBy, sortDir)

	// Fetch unmatched transactions
	selectQuery := fmt.Sprintf(`
		SELECT bt.id, bt.booking_date, bt.value_date, bt.payer_name, bt.payer_iban,
		       bt.description, bt.amount, bt.currency, bt.import_batch_id, bt.imported_at,
		       bt.is_hidden, bt.hidden_at, bt.hidden_by
		FROM fees.bank_transactions bt
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &transactions, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// Hide marks a transaction as hidden.
func (r *PostgresTransactionRepository) Hide(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE fees.bank_transactions
		SET is_hidden = true, hidden_at = NOW(), hidden_by = $2
		WHERE id = $1
	`, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetBatches retrieves import batch history.
func (r *PostgresTransactionRepository) GetBatches(ctx context.Context, offset, limit int) ([]domain.ImportBatch, int64, error) {
	var batches []domain.ImportBatch
	var total int64

	// Count total batches from import_batches table (includes batches with 0 transactions)
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*)
		FROM fees.import_batches
	`)
	if err != nil {
		return nil, 0, err
	}

	// Fetch batch summaries with date range and user info
	// Start from import_batches to include batches with 0 transactions
	err = r.db.SelectContext(ctx, &batches, `
		SELECT 
			ib.id as id,
			ib.imported_at as imported_at,
			COALESCE(COUNT(bt.id), 0) as transaction_count,
			COALESCE(COUNT(pm.id), 0) as matched_count,
			COALESCE(ib.file_name, '') as file_name,
			COALESCE(ib.imported_by, '00000000-0000-0000-0000-000000000000') as imported_by,
			CASE 
				WHEN ib.imported_by = '00000000-0000-0000-0000-000000000001' THEN 'Automated (Banking Sync)'
				ELSE 'Manual Upload'
			END as imported_by_email,
			MIN(bt.booking_date) as date_from,
			MAX(bt.booking_date) as date_to
		FROM fees.import_batches ib
		LEFT JOIN fees.bank_transactions bt ON bt.import_batch_id = ib.id
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		GROUP BY ib.id, ib.file_name, ib.imported_by, ib.imported_at
		ORDER BY ib.imported_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return batches, total, nil
}

// Delete deletes a transaction by ID.
func (r *PostgresTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.bank_transactions WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteUnmatchedByIBAN deletes all unmatched transactions with a specific IBAN.
func (r *PostgresTransactionRepository) DeleteUnmatchedByIBAN(ctx context.Context, iban string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.bank_transactions
		WHERE payer_iban = $1
		AND id NOT IN (SELECT transaction_id FROM fees.payment_matches)
	`, iban)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// CreateBatch creates a new import batch record.
func (r *PostgresTransactionRepository) CreateBatch(ctx context.Context, id uuid.UUID, fileName string, importedBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.import_batches (id, file_name, imported_by, imported_at)
		VALUES ($1, $2, $3, NOW())
	`, id, fileName, importedBy)
	return err
}

// ListMatched retrieves transactions that have been matched to fees.
func (r *PostgresTransactionRepository) ListMatched(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
	var transactions []domain.BankTransaction
	var total int64

	whereClause, args, argIdx := buildSearchFilter("1=1", search, 1)

	// Count total distinct matched transactions with search filter
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT bt.id)
		FROM fees.bank_transactions bt
		INNER JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE %s
	`, whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildOrderByClause(sortBy, sortDir)

	// Fetch matched transactions (distinct)
	selectQuery := fmt.Sprintf(`
		SELECT DISTINCT bt.id, bt.booking_date, bt.value_date, bt.payer_name, bt.payer_iban,
		       bt.description, bt.amount, bt.currency, bt.import_batch_id, bt.imported_at
		FROM fees.bank_transactions bt
		INNER JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &transactions, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
