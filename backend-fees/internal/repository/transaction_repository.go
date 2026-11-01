package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
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
		tx.ImportedAt = util.Now()
	}

	result, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.bank_transactions (id, booking_date, value_date, payer_name, payer_iban,
		                                    description, amount, currency, import_batch_id, imported_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
	`, tx.ID, tx.BookingDate, tx.ValueDate, tx.PayerName, tx.PayerIBAN,
		tx.Description, tx.Amount, tx.Currency, tx.ImportBatchID, tx.ImportedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDuplicate
	}
	return nil
}

// GetByID retrieves a bank transaction by ID.
func (r *PostgresTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BankTransaction, error) {
	var tx domain.BankTransaction
	err := conn(ctx, r.db).GetContext(ctx, &tx, `
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
	rows := []domain.BankTransaction{tx}
	if err := r.enrich(ctx, rows); err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// GetByIDs retrieves multiple transactions by their IDs.
func (r *PostgresTransactionRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.BankTransaction, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.BankTransaction), nil
	}

	var transactions []domain.BankTransaction
	err := conn(ctx, r.db).SelectContext(ctx, &transactions, `
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

	err := conn(ctx, r.db).GetContext(ctx, &count, query, args...)
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
		whereClause += fmt.Sprintf(" AND (bt.payer_name ILIKE $%d OR bt.description ILIKE $%d OR bt.payer_iban ILIKE $%d)",
			argIdx, argIdx+1, argIdx+2)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIdx += 3
	}

	return whereClause, args, argIdx
}

// literalTransactionSearch preserves substring semantics for UI searches.
func literalTransactionSearch(search string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(search)
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

	return fmt.Sprintf("%s %s, bt.id ASC", orderColumn, orderDirection)
}

// inOtherLedger is the SQL condition "the transaction belongs to a plan or has a purpose", the two
// assignment kinds that always take the whole transaction. txID is a column or parameter reference.
func inOtherLedger(txID string) string {
	return fmt.Sprintf(`(EXISTS(SELECT 1 FROM fees.installment_payments ip WHERE ip.transaction_id=%[1]s)
	OR EXISTS(SELECT 1 FROM fees.transaction_assignments ta WHERE ta.transaction_id=%[1]s))`, txID)
}

// transactionOccupiedSQL is true when a transaction with ID $1 already has fee matches, a plan payment or a purpose.
// Assigning to a plan or purpose locks the bank row first and checks this; fee matches check inOtherLedger only,
// because a transaction can be split across several fees.
var transactionOccupiedSQL = `EXISTS(SELECT 1 FROM fees.payment_matches WHERE transaction_id=$1) OR ` + inOtherLedger("$1")

// listUnmatchedCondition matches transactions that are not fully allocated yet.
// Partially allocated transactions stay listed until their remaining amount is zero.
var listUnmatchedCondition = `NOT ` + inOtherLedger("bt.id") + ` AND bt.amount > 0 AND bt.is_hidden = false AND
	COALESCE((SELECT SUM(pm.amount) FROM fees.payment_matches pm WHERE pm.transaction_id = bt.id), 0) < bt.amount - 0.005`

// listMatchedAmountExpr computes the total amount already allocated for a transaction.
const listMatchedAmountExpr = `COALESCE((SELECT SUM(pm.amount) FROM fees.payment_matches pm WHERE pm.transaction_id = bt.id), 0)`

// ListUnmatched retrieves transactions that haven't been fully matched to fees.
// Transactions with partial allocations are included (with their matched amount),
// so the remainder can still be assigned.
func (r *PostgresTransactionRepository) ListUnmatched(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
	return r.listTransactions(ctx, listUnmatchedCondition, search, sortBy, sortDir, offset, limit)
}

var listHasMatches = `(` + inOtherLedger("bt.id") + ` OR EXISTS (
 SELECT 1 FROM fees.payment_matches pm WHERE pm.transaction_id = bt.id
))`

const listHasWarnings = `EXISTS (
 SELECT 1 FROM fees.transaction_warnings w WHERE w.transaction_id = bt.id
 AND w.resolved_at IS NULL AND w.warning_type != 'MULTIPLE_OPEN_FEES'
)`

// ListTransactions applies the unified bank reconciliation status before pagination.
// A partial allocation belongs to the matched tab, as in the merged UI list.
func (r *PostgresTransactionRepository) ListTransactions(
	ctx context.Context, status, search, sortBy, sortDir string, offset, limit int,
) ([]domain.BankTransaction, int64, error) {
	condition := "(" + listUnmatchedCondition + ") OR " + listHasMatches + " OR " + listHasWarnings
	switch status {
	case "offen":
		condition = "((" + listUnmatchedCondition + ") OR " + listHasWarnings + ") AND NOT " + listHasMatches
	case "zugeordnet":
		condition = listHasMatches
	case "warnungen":
		condition = listHasWarnings
	}
	rows, total, err := r.listTransactions(ctx, condition, literalTransactionSearch(search), sortBy, sortDir, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if err := r.enrichHouseholdSuggestions(ctx, rows); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *PostgresTransactionRepository) listTransactions(
	ctx context.Context, condition, search, sortBy, sortDir string, offset, limit int,
) ([]domain.BankTransaction, int64, error) {
	var transactions []domain.BankTransaction
	var total int64

	whereClause, args, argIdx := buildSearchFilter("("+condition+")", search, 1)

	// Count total unmatched with search filter
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM fees.bank_transactions bt
		WHERE %s
	`, whereClause)
	err := conn(ctx, r.db).GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildOrderByClause(sortBy, sortDir)

	// Fetch unmatched transactions
	selectQuery := fmt.Sprintf(`
		SELECT bt.id, bt.booking_date, bt.value_date, bt.payer_name, bt.payer_iban,
		       bt.description, bt.amount, bt.currency, bt.import_batch_id, bt.imported_at,
		       bt.is_hidden, bt.hidden_at, bt.hidden_by,
		       %s AS matched_amount
		FROM fees.bank_transactions bt
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, listMatchedAmountExpr, whereClause, orderBy, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = conn(ctx, r.db).SelectContext(ctx, &transactions, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	if err := r.enrich(ctx, transactions); err != nil {
		return nil, 0, err
	}
	return transactions, total, nil
}

// Hide marks a transaction as hidden.
func (r *PostgresTransactionRepository) Hide(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `
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
	err := conn(ctx, r.db).GetContext(ctx, &total, `
		SELECT COUNT(*)
		FROM fees.import_batches
	`)
	if err != nil {
		return nil, 0, err
	}

	// Fetch batch summaries with date range and user info
	// Start from import_batches to include batches with 0 transactions
	err = conn(ctx, r.db).SelectContext(ctx, &batches, `
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
			MAX(bt.booking_date) as date_to,
			ib.error_count as error_count,
			ib.errors as errors
		FROM fees.import_batches ib
		LEFT JOIN fees.bank_transactions bt ON bt.import_batch_id = ib.id
		LEFT JOIN fees.payment_matches pm ON bt.id = pm.transaction_id
		GROUP BY ib.id, ib.file_name, ib.imported_by, ib.imported_at, ib.error_count, ib.errors
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
	result, err := conn(ctx, r.db).ExecContext(ctx, `
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
	result, err := conn(ctx, r.db).ExecContext(ctx, `
		DELETE FROM fees.bank_transactions
		WHERE payer_iban = $1
		AND id NOT IN (SELECT transaction_id FROM fees.payment_matches)
		AND NOT `+inOtherLedger("fees.bank_transactions.id"), iban)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// CreateBatch creates a new import batch record.
func (r *PostgresTransactionRepository) CreateBatch(ctx context.Context, id uuid.UUID, fileName string, importedBy uuid.UUID) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.import_batches (id, file_name, imported_by, imported_at)
		VALUES ($1, $2, $3, NOW())
	`, id, fileName, importedBy)
	return err
}

// SetBatchErrors stores the row errors of an import batch. Only the first
// domain.MaxStoredImportErrors details are kept; error_count stays exact.
func (r *PostgresTransactionRepository) SetBatchErrors(ctx context.Context, id uuid.UUID, errs []domain.ImportError) error {
	stored := errs
	if len(stored) > domain.MaxStoredImportErrors {
		stored = stored[:domain.MaxStoredImportErrors]
	}
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.import_batches SET error_count = $2, errors = $3 WHERE id = $1
	`, id, len(errs), domain.ImportErrors(stored))
	return err
}

// ListMatched retrieves transactions that have been matched to fees.
func (r *PostgresTransactionRepository) ListMatched(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
	return r.listTransactions(ctx, listHasMatches, search, sortBy, sortDir, offset, limit)
}

func (r *PostgresTransactionRepository) enrichInstallments(ctx context.Context, rows []domain.BankTransaction) error {
	for i := range rows {
		t := &rows[i]
		if t.Assignment != nil {
			continue // Has a purpose, so neither a plan payment nor a plan suggestion.
		}
		link := &domain.InstallmentPaymentLink{}
		err := conn(ctx, r.db).GetContext(ctx, link, `SELECT ip.id AS payment_id,ip.plan_id,p.debtor_name FROM fees.installment_payments ip JOIN fees.installment_plans p ON p.id=ip.plan_id WHERE ip.transaction_id=$1`, t.ID)
		if err == nil {
			t.InstallmentPayment = link
			v := t.Amount
			t.MatchedAmount = &v
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if t.PayerIBAN == nil {
			continue
		}
		suggestion := &domain.InstallmentPlanSuggestion{}
		err = conn(ctx, r.db).GetContext(ctx, suggestion, `SELECT p.id AS plan_id,p.debtor_name FROM fees.installment_plans p WHERE p.status='ACTIVE' AND p.payer_iban=upper(replace($1,' ','')) AND p.total_amount+COALESCE((SELECT sum(c.amount) FROM fees.installment_charges c WHERE c.plan_id=p.id),0)>COALESCE((SELECT sum(ip.amount) FROM fees.installment_payments ip WHERE ip.plan_id=p.id),0) ORDER BY p.created_at,p.id LIMIT 1`, *t.PayerIBAN)
		if err == nil {
			t.InstallmentPlanSuggestion = suggestion
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	return nil
}

func (r *PostgresTransactionRepository) enrich(ctx context.Context, rows []domain.BankTransaction) error {
	if err := r.enrichAssignments(ctx, rows); err != nil {
		return err
	}
	return r.enrichInstallments(ctx, rows)
}

// isOpenIncoming reports an incoming list row without any allocation; it needs matched_amount and enrich.
func isOpenIncoming(t *domain.BankTransaction) bool {
	return t.Amount > 0 && !t.HasExclusiveAssignment() && (t.MatchedAmount == nil || *t.MatchedAmount == 0)
}

// enrichHouseholdSuggestions proposes a household for open incoming transactions from their payer IBAN:
// the household of the latest donation from that IBAN, else the household of a child the IBAN is trusted for.
func (r *PostgresTransactionRepository) enrichHouseholdSuggestions(ctx context.Context, rows []domain.BankTransaction) error {
	var ibans []string
	for i := range rows {
		t := &rows[i]
		if t.PayerIBAN != nil && isOpenIncoming(t) {
			ibans = append(ibans, *t.PayerIBAN)
		}
	}
	if len(ibans) == 0 {
		return nil
	}
	var found []struct {
		IBAN string `db:"iban"`
		domain.HouseholdSuggestion
	}
	err := conn(ctx, r.db).SelectContext(ctx, &found, `
		SELECT DISTINCT ON (s.iban) s.iban, h.id AS household_id, h.name
		FROM (
			SELECT bt.payer_iban AS iban, ta.household_id, 1 AS prio, ta.assigned_at AS ts
			FROM fees.transaction_assignments ta JOIN fees.bank_transactions bt ON bt.id = ta.transaction_id
			WHERE ta.household_id IS NOT NULL AND bt.payer_iban = ANY($1)
			UNION ALL
			SELECT ki.iban, c.household_id, 2, ki.updated_at
			FROM fees.known_ibans ki JOIN fees.children c ON c.id = ki.child_id
			WHERE ki.status = 'trusted' AND c.household_id IS NOT NULL AND ki.iban = ANY($1)
		) s JOIN fees.households h ON h.id = s.household_id
		ORDER BY s.iban, s.prio, s.ts DESC`, pq.Array(ibans))
	if err != nil {
		return err
	}
	byIBAN := make(map[string]domain.HouseholdSuggestion, len(found))
	for _, f := range found {
		byIBAN[f.IBAN] = f.HouseholdSuggestion
	}
	for i := range rows {
		if rows[i].PayerIBAN == nil {
			continue
		}
		if sug, ok := byIBAN[*rows[i].PayerIBAN]; ok && isOpenIncoming(&rows[i]) {
			rows[i].HouseholdSuggestion = &sug
		}
	}
	return nil
}

// enrichAssignments attaches the assignment and counts the full amount as allocated.
func (r *PostgresTransactionRepository) enrichAssignments(ctx context.Context, rows []domain.BankTransaction) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var found []struct {
		TransactionID uuid.UUID `db:"transaction_id"`
		domain.TransactionAssignment
	}
	if err := conn(ctx, r.db).SelectContext(ctx, &found, `SELECT ta.transaction_id, ta.kind, ta.note, ta.assigned_at, ta.household_id, COALESCE(h.name, '') AS household_name
		FROM fees.transaction_assignments ta LEFT JOIN fees.households h ON h.id = ta.household_id
		WHERE ta.transaction_id = ANY($1)`, pq.Array(ids)); err != nil {
		return err
	}
	byID := make(map[uuid.UUID]domain.TransactionAssignment, len(found))
	for _, f := range found {
		byID[f.TransactionID] = f.TransactionAssignment
	}
	for i := range rows {
		if a, ok := byID[rows[i].ID]; ok {
			rows[i].Assignment = &a
			v := rows[i].Amount
			rows[i].MatchedAmount = &v
		}
	}
	return nil
}

// Assign gives a transaction a purpose (e.g. donation). The row is locked first, like the
// other assignment paths; a transaction with fee matches, a plan payment or a purpose is refused.
func (r *PostgresTransactionRepository) Assign(ctx context.Context, transactionID uuid.UUID, in domain.AssignmentInput, userID uuid.UUID) error {
	tx, err := beginTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id uuid.UUID
	if err = tx.GetContext(ctx, &id, `SELECT id FROM fees.bank_transactions WHERE id=$1 FOR UPDATE`, transactionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var occupied bool
	if err = tx.GetContext(ctx, &occupied, `SELECT `+transactionOccupiedSQL, transactionID); err != nil {
		return err
	}
	if occupied {
		return ErrDuplicate
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO fees.transaction_assignments (transaction_id, kind, note, household_id, assigned_at, assigned_by) VALUES ($1,$2,$3,$4,$5,$6)`,
		transactionID, in.Kind, in.Note, in.HouseholdID, util.Now(), userID)
	if err = mapMissingHousehold(err); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateAssignment changes note and household of an existing assignment; the kind stays.
func (r *PostgresTransactionRepository) UpdateAssignment(ctx context.Context, transactionID uuid.UUID, in domain.AssignmentInput) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE fees.transaction_assignments SET note=$2, household_id=$3 WHERE transaction_id=$1 AND kind=$4`,
		transactionID, in.Note, in.HouseholdID, in.Kind)
	if err = mapMissingHousehold(err); err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// mapMissingHousehold turns a foreign-key violation on household_id into ErrNotFound.
func mapMissingHousehold(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23503" {
		return ErrNotFound
	}
	return err
}

// ListAssigned returns the transactions with the given purpose, newest first; householdID narrows them to one household.
func (r *PostgresTransactionRepository) ListAssigned(ctx context.Context, kind domain.AssignmentKind, householdID *uuid.UUID) ([]domain.BankTransaction, error) {
	var rows []domain.BankTransaction
	err := conn(ctx, r.db).SelectContext(ctx, &rows, `
		SELECT bt.id, bt.booking_date, bt.value_date, bt.payer_name, bt.payer_iban,
		       bt.description, bt.amount, bt.currency, bt.import_batch_id, bt.imported_at,
		       bt.is_hidden, bt.hidden_at, bt.hidden_by
		FROM fees.bank_transactions bt
		JOIN fees.transaction_assignments ta ON ta.transaction_id = bt.id
		WHERE ta.kind = $1 AND ($2::uuid IS NULL OR ta.household_id = $2)
		ORDER BY bt.booking_date DESC, bt.id ASC
	`, kind, householdID)
	if err != nil {
		return nil, err
	}
	if err := r.enrichAssignments(ctx, rows); err != nil {
		return nil, err
	}
	return rows, nil
}
