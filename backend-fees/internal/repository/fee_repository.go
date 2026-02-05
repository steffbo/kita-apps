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
)

// PostgresFeeRepository is the PostgreSQL implementation of FeeRepository.
type PostgresFeeRepository struct {
	db *sqlx.DB
}

// NewPostgresFeeRepository creates a new PostgreSQL fee repository.
func NewPostgresFeeRepository(db *sqlx.DB) *PostgresFeeRepository {
	return &PostgresFeeRepository{db: db}
}

// List retrieves fee expectations with optional filtering.
func (r *PostgresFeeRepository) List(ctx context.Context, filter FeeFilter, offset, limit int) ([]domain.FeeExpectation, int64, error) {
	var fees []domain.FeeExpectation
	var total int64

	// Determine sort order and whether we need to join children
	orderClause, needsChildJoin := getFeeSortOrder(filter.SortBy, filter.SortDir)

	statusFilter := strings.ToLower(strings.TrimSpace(filter.Status))
	needsStatusJoin := statusFilter == "paid" || statusFilter == "open" || statusFilter == "overdue"

	// Base query with optional JOIN for search/sorting by child
	baseQuery := `FROM fees.fee_expectations fe`
	if needsStatusJoin {
		baseQuery += ` LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id`
	}
	if filter.Search != "" || needsChildJoin {
		baseQuery += ` JOIN fees.children c ON fe.child_id = c.id`
	}
	baseQuery += ` WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if filter.Year != nil {
		baseQuery += fmt.Sprintf(" AND fe.year = $%d", argIdx)
		args = append(args, *filter.Year)
		argIdx++
	}

	if filter.Month != nil {
		baseQuery += fmt.Sprintf(" AND fe.month = $%d", argIdx)
		args = append(args, *filter.Month)
		argIdx++
	}

	if filter.FeeType != "" {
		baseQuery += fmt.Sprintf(" AND fe.fee_type = $%d", argIdx)
		args = append(args, filter.FeeType)
		argIdx++
	}

	if needsStatusJoin {
		switch statusFilter {
		case "paid":
			baseQuery += " AND COALESCE(pm_sum.matched_amount, 0) >= fe.amount - 0.01"
		case "open":
			baseQuery += " AND COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01"
		case "overdue":
			baseQuery += " AND COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01 AND fe.due_date < NOW()"
		}
	}

	if filter.ChildID != nil {
		baseQuery += fmt.Sprintf(" AND fe.child_id = $%d", argIdx)
		args = append(args, *filter.ChildID)
		argIdx++
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		baseQuery += fmt.Sprintf(" AND (c.member_number ILIKE $%d OR c.first_name ILIKE $%d OR c.last_name ILIKE $%d OR CONCAT(c.first_name, ' ', c.last_name) ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, searchPattern)
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &fees, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return fees, total, nil
}

// getFeeSortOrder returns a safe ORDER BY clause for fees and whether a child join is required.
func getFeeSortOrder(sortBy, sortDir string) (string, bool) {
	direction := "ASC"
	if sortDir == "desc" {
		direction = "DESC"
	}

	switch sortBy {
	case "memberNumber":
		return fmt.Sprintf("c.member_number %s, fe.year DESC, fe.month DESC NULLS LAST, fe.created_at DESC, fe.id DESC", direction), true
	case "childName":
		return fmt.Sprintf("c.last_name %s, c.first_name %s, fe.year DESC, fe.month DESC NULLS LAST, fe.created_at DESC, fe.id DESC", direction, direction), true
	case "feeType":
		return fmt.Sprintf("fe.fee_type %s, fe.year DESC, fe.month DESC NULLS LAST, fe.created_at DESC, fe.id DESC", direction), false
	case "period":
		return fmt.Sprintf("fe.year %s, fe.month %s NULLS LAST, fe.created_at DESC, fe.id DESC", direction, direction), false
	case "amount":
		return fmt.Sprintf("fe.amount %s, fe.year DESC, fe.month DESC NULLS LAST, fe.created_at DESC, fe.id DESC", direction), false
	default:
		return "fe.year DESC, fe.month DESC NULLS LAST, fe.created_at DESC, fe.id DESC", false
	}
}

// GetByID retrieves a fee expectation by ID.
func (r *PostgresFeeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeExpectation, error) {
	var fee domain.FeeExpectation
	err := r.db.GetContext(ctx, &fee, `
		SELECT id, child_id, fee_type, year, month, amount, due_date, created_at, reminder_for_id, reconciliation_year
		FROM fees.fee_expectations
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("fee not found")
		}
		return nil, err
	}
	return &fee, nil
}

// GetForChild retrieves all fee expectations for a child, optionally filtered by year.
func (r *PostgresFeeRepository) GetForChild(ctx context.Context, childID uuid.UUID, year *int) ([]domain.FeeExpectation, error) {
	var fees []domain.FeeExpectation
	var query string
	var args []interface{}

	if year != nil {
		query = `
			SELECT id, child_id, fee_type, year, month, amount, due_date, created_at, reminder_for_id, reconciliation_year
			FROM fees.fee_expectations
			WHERE child_id = $1 AND year = $2
			ORDER BY year DESC, month ASC NULLS LAST
		`
		args = []interface{}{childID, *year}
	} else {
		query = `
			SELECT id, child_id, fee_type, year, month, amount, due_date, created_at, reminder_for_id, reconciliation_year
			FROM fees.fee_expectations
			WHERE child_id = $1
			ORDER BY year DESC, month ASC NULLS LAST
		`
		args = []interface{}{childID}
	}

	err := r.db.SelectContext(ctx, &fees, query, args...)
	return fees, err
}

// ListUnpaidByMonthAndTypes returns unpaid fees for a given month/year and fee types.
func (r *PostgresFeeRepository) ListUnpaidByMonthAndTypes(ctx context.Context, year int, month int, feeTypes []domain.FeeType) ([]domain.FeeExpectation, error) {
	if len(feeTypes) == 0 {
		return []domain.FeeExpectation{}, nil
	}

	var fees []domain.FeeExpectation
	query := `
		SELECT fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year
		FROM fees.fee_expectations fe
		LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id
		WHERE fe.year = $1 AND fe.month = $2
		  AND fe.fee_type = ANY($3)
		  AND COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01
		ORDER BY fe.due_date ASC, fe.created_at ASC
	`

	err := r.db.SelectContext(ctx, &fees, query, year, month, pq.Array(feeTypes))
	return fees, err
}

// ListUnpaidWithoutReminderByMonthAndTypes returns unpaid fees without reminders for a given month/year and fee types.
func (r *PostgresFeeRepository) ListUnpaidWithoutReminderByMonthAndTypes(ctx context.Context, year int, month int, feeTypes []domain.FeeType) ([]domain.FeeExpectation, error) {
	if len(feeTypes) == 0 {
		return []domain.FeeExpectation{}, nil
	}

	var fees []domain.FeeExpectation
	query := `
		SELECT fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year
		FROM fees.fee_expectations fe
		LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id
		WHERE fe.year = $1 AND fe.month = $2
		  AND fe.fee_type = ANY($3)
		  AND COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01
		  AND NOT EXISTS (
			SELECT 1
			FROM fees.fee_expectations rem
			WHERE rem.reminder_for_id = fe.id
		  )
		ORDER BY fe.due_date ASC, fe.created_at ASC
	`

	err := r.db.SelectContext(ctx, &fees, query, year, month, pq.Array(feeTypes))
	return fees, err
}

// GetByIDs retrieves multiple fee expectations by their IDs.
func (r *PostgresFeeRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.FeeExpectation, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.FeeExpectation), nil
	}

	var fees []domain.FeeExpectation
	err := r.db.SelectContext(ctx, &fees, `
		SELECT id, child_id, fee_type, year, month, amount, due_date, created_at, reminder_for_id, reconciliation_year
		FROM fees.fee_expectations
		WHERE id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*domain.FeeExpectation, len(fees))
	for i := range fees {
		result[fees[i].ID] = &fees[i]
	}
	return result, nil
}

// Create creates a new fee expectation.
func (r *PostgresFeeRepository) Create(ctx context.Context, fee *domain.FeeExpectation) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.fee_expectations (id, child_id, fee_type, year, month, amount, due_date, created_at, reminder_for_id, reconciliation_year)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, fee.ID, fee.ChildID, fee.FeeType, fee.Year, fee.Month, fee.Amount, fee.DueDate, fee.CreatedAt, fee.ReminderForID, fee.ReconciliationYear)
	return err
}

// Update updates an existing fee expectation.
func (r *PostgresFeeRepository) Update(ctx context.Context, fee *domain.FeeExpectation) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.fee_expectations
		SET amount = $2, due_date = $3
		WHERE id = $1
	`, fee.ID, fee.Amount, fee.DueDate)
	return err
}

// Delete deletes a fee expectation.
func (r *PostgresFeeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.fee_expectations WHERE id = $1`, id)
	return err
}

// Exists checks if a fee expectation already exists.
func (r *PostgresFeeRepository) Exists(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int) (bool, error) {
	var count int
	var err error

	if month != nil {
		err = r.db.GetContext(ctx, &count, `
			SELECT COUNT(*)
			FROM fees.fee_expectations
			WHERE child_id = $1 AND fee_type = $2 AND year = $3 AND month = $4
		`, childID, feeType, year, *month)
	} else {
		err = r.db.GetContext(ctx, &count, `
			SELECT COUNT(*)
			FROM fees.fee_expectations
			WHERE child_id = $1 AND fee_type = $2 AND year = $3 AND month IS NULL
		`, childID, feeType, year)
	}

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindUnpaid finds an unpaid fee expectation for matching.
func (r *PostgresFeeRepository) FindUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int) (*domain.FeeExpectation, error) {
	var fee domain.FeeExpectation
	var err error

	query := `
		SELECT fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year
		FROM fees.fee_expectations fe
		LEFT JOIN fees.payment_matches pm ON fe.id = pm.expectation_id
		WHERE fe.child_id = $1 AND fe.fee_type = $2 AND fe.year = $3
		  AND pm.id IS NULL
	`

	if month != nil {
		query += " AND fe.month = $4"
		err = r.db.GetContext(ctx, &fee, query, childID, feeType, year, *month)
	} else {
		query += " AND fe.month IS NULL"
		err = r.db.GetContext(ctx, &fee, query, childID, feeType, year)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &fee, nil
}

// FindOldestUnpaid finds the oldest unpaid fee expectation for a child by fee type and amount.
// This is used for matching when a family has multiple unpaid fees - we want to pay the oldest first.
func (r *PostgresFeeRepository) FindOldestUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64) (*domain.FeeExpectation, error) {
	var fee domain.FeeExpectation

	query := `
		SELECT fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year
		FROM fees.fee_expectations fe
		LEFT JOIN fees.payment_matches pm ON fe.id = pm.expectation_id
		WHERE fe.child_id = $1 AND fe.fee_type = $2 AND fe.amount = $3
		  AND pm.id IS NULL
		ORDER BY fe.due_date ASC, fe.created_at ASC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &fee, query, childID, feeType, amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &fee, nil
}

// FindBestUnpaid finds the best matching unpaid fee, preferring fees for the payment month.
// Priority: 1) Same month/year as payment, 2) Oldest unpaid fee.
// This handles the common case where a payment on Jan 5 should match January's fee, not December's.
func (r *PostgresFeeRepository) FindBestUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64, paymentDate time.Time) (*domain.FeeExpectation, error) {
	var fee domain.FeeExpectation
	paymentMonth := int(paymentDate.Month())
	paymentYear := paymentDate.Year()

	// First, try to find a fee for the same month/year as the payment
	queryCurrentMonth := `
		SELECT fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year
		FROM fees.fee_expectations fe
		LEFT JOIN fees.payment_matches pm ON fe.id = pm.expectation_id
		WHERE fe.child_id = $1 AND fe.fee_type = $2 AND fe.amount = $3
		  AND fe.year = $4 AND fe.month = $5
		  AND pm.id IS NULL
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &fee, queryCurrentMonth, childID, feeType, amount, paymentYear, paymentMonth)
	if err == nil {
		return &fee, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// No fee for current month, fall back to oldest unpaid
	return r.FindOldestUnpaid(ctx, childID, feeType, amount)
}

// CountUnpaidByType counts all unpaid fees of a specific type for a child.
// This is used to determine if auto-matching should occur (only when count == 1).
func (r *PostgresFeeRepository) CountUnpaidByType(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64) (int, error) {
	var count int

	query := `
		SELECT COUNT(*)
		FROM fees.fee_expectations fe
		LEFT JOIN fees.payment_matches pm ON fe.id = pm.expectation_id
		WHERE fe.child_id = $1 AND fe.fee_type = $2 AND fe.amount = $3
		  AND pm.id IS NULL
	`

	err := r.db.GetContext(ctx, &count, query, childID, feeType, amount)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindOldestUnpaidWithReminder finds the oldest unpaid fee with its linked reminder
// where the combined amount matches the transaction amount.
// Returns both the original fee and the reminder if they exist and are both unpaid.
func (r *PostgresFeeRepository) FindOldestUnpaidWithReminder(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, combinedAmount float64) ([]domain.FeeExpectation, error) {
	// Find unpaid fees that have an unpaid reminder linked to them,
	// where fee.amount + reminder.amount = combinedAmount
	query := `
		SELECT 
			fe.id, fe.child_id, fe.fee_type, fe.year, fe.month, fe.amount, fe.due_date, fe.created_at, fe.reminder_for_id, fe.reconciliation_year,
			rem.id as rem_id, rem.child_id as rem_child_id, rem.fee_type as rem_fee_type, rem.year as rem_year, 
			rem.month as rem_month, rem.amount as rem_amount, rem.due_date as rem_due_date, rem.created_at as rem_created_at, 
			rem.reminder_for_id as rem_reminder_for_id, rem.reconciliation_year as rem_reconciliation_year
		FROM fees.fee_expectations fe
		LEFT JOIN fees.payment_matches pm_fe ON fe.id = pm_fe.expectation_id
		JOIN fees.fee_expectations rem ON rem.reminder_for_id = fe.id
		LEFT JOIN fees.payment_matches pm_rem ON rem.id = pm_rem.expectation_id
		WHERE fe.child_id = $1 
		  AND fe.fee_type = $2 
		  AND (fe.amount + rem.amount) = $3
		  AND pm_fe.id IS NULL
		  AND pm_rem.id IS NULL
		ORDER BY fe.due_date ASC, fe.created_at ASC
		LIMIT 1
	`

	rows, err := r.db.QueryContext(ctx, query, childID, feeType, combinedAmount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var fee, reminder domain.FeeExpectation
	err = rows.Scan(
		&fee.ID, &fee.ChildID, &fee.FeeType, &fee.Year, &fee.Month, &fee.Amount, &fee.DueDate, &fee.CreatedAt, &fee.ReminderForID, &fee.ReconciliationYear,
		&reminder.ID, &reminder.ChildID, &reminder.FeeType, &reminder.Year, &reminder.Month, &reminder.Amount, &reminder.DueDate, &reminder.CreatedAt, &reminder.ReminderForID, &reminder.ReconciliationYear,
	)
	if err != nil {
		return nil, err
	}

	return []domain.FeeExpectation{fee, reminder}, nil
}

// GetOverview returns fee statistics for a given year.
func (r *PostgresFeeRepository) GetOverview(ctx context.Context, year int) (*domain.FeeOverview, error) {
	overview := &domain.FeeOverview{}
	now := time.Now()

	// Get totals (consider partial payments)
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			fe.id,
			fe.amount,
			fe.due_date,
			CASE WHEN COALESCE(pm_sum.matched_amount, 0) >= fe.amount - 0.01 THEN true ELSE false END as is_paid
		FROM fees.fee_expectations fe
		LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id
		WHERE fe.year = $1
	`, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var amount float64
		var dueDate time.Time
		var isPaid bool

		if err := rows.Scan(&id, &amount, &dueDate, &isPaid); err != nil {
			return nil, err
		}

		if isPaid {
			overview.TotalPaid++
			overview.AmountPaid += amount
		} else if now.After(dueDate) {
			overview.TotalOverdue++
			overview.AmountOverdue += amount
		} else {
			overview.TotalOpen++
			overview.AmountOpen += amount
		}
	}

	// Get monthly breakdown
	monthRows, err := r.db.QueryContext(ctx, `
		SELECT 
			fe.month,
			COUNT(*) FILTER (WHERE COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01) as open_count,
			COUNT(*) FILTER (WHERE COALESCE(pm_sum.matched_amount, 0) >= fe.amount - 0.01) as paid_count,
			COALESCE(SUM(fe.amount) FILTER (WHERE COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01), 0) as open_amount,
			COALESCE(SUM(fe.amount) FILTER (WHERE COALESCE(pm_sum.matched_amount, 0) >= fe.amount - 0.01), 0) as paid_amount
		FROM fees.fee_expectations fe
		LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id
		WHERE fe.year = $1 AND fe.month IS NOT NULL
		GROUP BY fe.month
		ORDER BY fe.month
	`, year)
	if err != nil {
		return nil, err
	}
	defer monthRows.Close()

	for monthRows.Next() {
		var ms domain.MonthSummary
		ms.Year = year
		if err := monthRows.Scan(&ms.Month, &ms.OpenCount, &ms.PaidCount, &ms.OpenAmount, &ms.PaidAmount); err != nil {
			return nil, err
		}
		overview.ByMonth = append(overview.ByMonth, ms)
	}

	// Count children with open fees
	err = r.db.GetContext(ctx, &overview.ChildrenWithOpenFees, `
		SELECT COUNT(DISTINCT fe.child_id)
		FROM fees.fee_expectations fe
		LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id
		WHERE fe.year = $1 AND COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01
	`, year)
	if err != nil {
		return nil, err
	}

	// Count open fees by type (unpaid regardless of overdue)
	typeRows, err := r.db.QueryContext(ctx, `
		SELECT fe.fee_type, COUNT(*) FILTER (WHERE COALESCE(pm_sum.matched_amount, 0) < fe.amount - 0.01) as open_count
		FROM fees.fee_expectations fe
		LEFT JOIN (
			SELECT expectation_id, COALESCE(SUM(amount), 0) AS matched_amount
			FROM fees.payment_matches
			GROUP BY expectation_id
		) pm_sum ON fe.id = pm_sum.expectation_id
		WHERE fe.year = $1
		GROUP BY fe.fee_type
	`, year)
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var feeType string
		var openCount int
		if err := typeRows.Scan(&feeType, &openCount); err != nil {
			return nil, err
		}
		if openCount == 0 {
			continue
		}
		switch feeType {
		case string(domain.FeeTypeMembership):
			overview.OpenMembershipCount = openCount
		case string(domain.FeeTypeFood):
			overview.OpenFoodCount = openCount
		case string(domain.FeeTypeChildcare):
			overview.OpenChildcareCount = openCount
		}
	}

	return overview, nil
}
