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
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresChildRepository is the PostgreSQL implementation of ChildRepository.
type PostgresChildRepository struct {
	db *sqlx.DB
}

// NewPostgresChildRepository creates a new PostgreSQL child repository.
func NewPostgresChildRepository(db *sqlx.DB) *PostgresChildRepository {
	return &PostgresChildRepository{db: db}
}

type careHoursHistoryRow struct {
	ID             uuid.UUID  `db:"id"`
	ChildID        uuid.UUID  `db:"child_id"`
	CareHours      *int       `db:"care_hours"`
	EffectiveFrom  time.Time  `db:"effective_from"`
	EffectiveUntil *time.Time `db:"effective_until"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

type legalHoursHistoryRow struct {
	ID             uuid.UUID  `db:"id"`
	ChildID        uuid.UUID  `db:"child_id"`
	LegalHours     *int       `db:"legal_hours"`
	EffectiveFrom  time.Time  `db:"effective_from"`
	EffectiveUntil *time.Time `db:"effective_until"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// List retrieves children with optional filtering and sorting.
func (r *PostgresChildRepository) List(ctx context.Context, activeOnly bool, u3Only bool, hasWarnings bool, hasOpenFees bool, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Child, int64, error) {
	var children []domain.Child
	var total int64

	baseQuery := `
		FROM fees.children c
		LEFT JOIN (
			SELECT fe.child_id,
				   COUNT(*) FILTER (WHERE pm.id IS NULL) AS open_fees_count
			FROM fees.fee_expectations fe
			LEFT JOIN fees.payment_matches pm ON fe.id = pm.expectation_id
			GROUP BY fe.child_id
		) ofe ON ofe.child_id = c.id
		WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if activeOnly {
		baseQuery += fmt.Sprintf(" AND c.is_active = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if u3Only {
		// Filter for children under 3 years old (born less than 3 years ago)
		baseQuery += fmt.Sprintf(" AND c.birth_date > $%d", argIdx)
		args = append(args, time.Now().AddDate(-3, 0, 0))
		argIdx++
	}

	if hasWarnings {
		// Children with warnings:
		// 1. No parents linked
		// 2. No legal_hours set
		// 3. No care_hours set
		// 4. U3 children where neither household nor any parent has income info
		//    (income is NOT required if status is MAX_ACCEPTED, NOT_REQUIRED, FOSTER_FAMILY, or HISTORIC)
		baseQuery += ` AND (
			-- No parents linked
			NOT EXISTS (SELECT 1 FROM fees.child_parents cp WHERE cp.child_id = c.id)
			-- No legal hours
			OR c.legal_hours IS NULL
			-- No care hours
			OR c.care_hours IS NULL
			-- U3 without income: born less than 3 years ago AND no valid income source
			OR (c.birth_date > $` + fmt.Sprintf("%d", argIdx) + ` AND NOT EXISTS (
				-- Check household first
				SELECT 1 FROM fees.households h
				WHERE h.id = c.household_id
				AND (
					h.annual_household_income IS NOT NULL
					OR h.income_status IN ('MAX_ACCEPTED', 'NOT_REQUIRED', 'FOSTER_FAMILY', 'HISTORIC')
				)
			) AND NOT EXISTS (
				-- Fallback: check parent income_status (legacy data)
				SELECT 1 FROM fees.child_parents cp2
				JOIN fees.parents p ON p.id = cp2.parent_id
				WHERE cp2.child_id = c.id
				AND (
					p.annual_household_income IS NOT NULL
					OR p.income_status IN ('MAX_ACCEPTED', 'NOT_REQUIRED', 'FOSTER_FAMILY', 'HISTORIC')
				)
			))
		)`
		args = append(args, time.Now().AddDate(-3, 0, 0))
		argIdx++
	}

	if hasOpenFees {
		baseQuery += " AND COALESCE(ofe.open_fees_count, 0) > 0"
	}

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (c.first_name ILIKE $%d OR c.last_name ILIKE $%d OR c.member_number ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		log.Error().Err(err).Str("query", countQuery).Msg("Child count query failed")
		return nil, 0, err
	}

	// Determine sort order
	orderClause := getChildSortOrder(sortBy, sortDir)

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT c.id, c.household_id, c.member_number, c.first_name, c.last_name, c.birth_date, c.entry_date, c.exit_date,
		       c.street, c.street_no, c.postal_code, c.city, c.legal_hours, c.legal_hours_until, c.care_hours,
		       c.is_active, c.created_at, c.updated_at, ofe.open_fees_count
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &children, selectQuery, args...)
	if err != nil {
		log.Error().Err(err).Str("query", selectQuery).Msg("Child list query failed")
		return nil, 0, err
	}

	return children, total, nil
}

// getChildSortOrder returns a safe ORDER BY clause for children.
func getChildSortOrder(sortBy, sortDir string) string {
	// Whitelist of allowed sort columns to prevent SQL injection
	allowedColumns := map[string]string{
		"memberNumber": "c.member_number",
		"firstName":    "c.first_name",
		"lastName":     "c.last_name",
		"name":         "c.last_name, c.first_name", // Legacy: combined name sorting
		"birthDate":    "c.birth_date",
		"age":          "c.birth_date", // age sorts by birth_date (reversed direction)
		"entryDate":    "c.entry_date",
		"createdAt":    "c.created_at",
	}

	// Default sort
	column := "c.last_name, c.first_name"
	if col, ok := allowedColumns[sortBy]; ok {
		column = col
	}

	// Validate direction
	direction := "ASC"
	if sortDir == "desc" {
		direction = "DESC"
	}

	// Special case: sorting by "age" should reverse the direction
	// (older = earlier birth_date, so ASC birth_date = DESC age)
	if sortBy == "age" {
		if direction == "ASC" {
			direction = "DESC"
		} else {
			direction = "ASC"
		}
	}

	return column + " " + direction
}

// GetByID retrieves a child by ID.
func (r *PostgresChildRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	var child domain.Child
	err := r.db.GetContext(ctx, &child, `
		SELECT id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
		       street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
		       is_active, created_at, updated_at
		FROM fees.children
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("child not found")
		}
		return nil, err
	}
	return &child, nil
}

// GetByIDs retrieves multiple children by their IDs.
func (r *PostgresChildRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Child, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.Child), nil
	}

	var children []domain.Child
	err := r.db.SelectContext(ctx, &children, `
		SELECT id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
		       street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
		       is_active, created_at, updated_at
		FROM fees.children
		WHERE id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*domain.Child, len(children))
	for i := range children {
		result[children[i].ID] = &children[i]
	}
	return result, nil
}

// GetByMemberNumber retrieves a child by member number.
func (r *PostgresChildRepository) GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Child, error) {
	var child domain.Child
	err := r.db.GetContext(ctx, &child, `
		SELECT id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
		       street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
		       is_active, created_at, updated_at
		FROM fees.children
		WHERE member_number = $1
	`, memberNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("child not found")
		}
		return nil, err
	}
	return &child, nil
}

// GetNextMemberNumber generates the next available numeric member number.
func (r *PostgresChildRepository) GetNextMemberNumber(ctx context.Context) (string, error) {
	var maxNum sql.NullInt64
	err := r.db.GetContext(ctx, &maxNum, `
		SELECT MAX(CAST(member_number AS INTEGER))
		FROM fees.children
		WHERE member_number ~ '^[0-9]+$'
	`)
	if err != nil {
		return "", err
	}

	nextNum := int64(1)
	if maxNum.Valid {
		nextNum = maxNum.Int64 + 1
	}

	return fmt.Sprintf("%d", nextNum), nil
}

// Create creates a new child.
func (r *PostgresChildRepository) Create(ctx context.Context, child *domain.Child) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO fees.children (id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
			                           street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
			                           is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, child.ID, child.HouseholdID, child.MemberNumber, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
		child.Street, child.StreetNo, child.PostalCode, child.City, child.LegalHours, child.LegalHoursUntil, child.CareHours,
		child.IsActive, child.CreatedAt, child.UpdatedAt)
	if err != nil {
		return err
	}

	if child.CareHours != nil {
		if err := r.upsertCareHoursHistoryTx(ctx, tx, child.ID, child.CareHours, child.EntryDate); err != nil {
			return err
		}
	}
	if child.LegalHours != nil || child.LegalHoursUntil != nil {
		if err := r.upsertLegalHoursHistoryTx(ctx, tx, child.ID, child.LegalHours, child.EntryDate, child.LegalHoursUntil); err != nil {
			return err
		}
	}
	if err := r.syncCurrentLegalHoursTx(ctx, tx, child.ID, time.Now()); err != nil {
		return err
	}
	if err := r.syncCurrentCareHoursTx(ctx, tx, child.ID, time.Now()); err != nil {
		return err
	}

	return tx.Commit()
}

// Update updates an existing child.
func (r *PostgresChildRepository) Update(ctx context.Context, child *domain.Child) error {
	child.UpdatedAt = time.Now()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var previousCareHours *int
	var previousLegalHours *int
	var previousLegalHoursUntil *time.Time
	type previousHoursRow struct {
		CareHours       sql.NullInt64 `db:"care_hours"`
		LegalHours      sql.NullInt64 `db:"legal_hours"`
		LegalHoursUntil *time.Time    `db:"legal_hours_until"`
	}
	var previous previousHoursRow
	if err := tx.GetContext(ctx, &previous, `SELECT care_hours, legal_hours, legal_hours_until FROM fees.children WHERE id = $1 FOR UPDATE`, child.ID); err != nil {
		return err
	}
	if previous.CareHours.Valid {
		value := int(previous.CareHours.Int64)
		previousCareHours = &value
	}
	if previous.LegalHours.Valid {
		value := int(previous.LegalHours.Int64)
		previousLegalHours = &value
	}
	previousLegalHoursUntil = previous.LegalHoursUntil

	_, err = tx.ExecContext(ctx, `
		UPDATE fees.children
		SET household_id = $2, first_name = $3, last_name = $4, birth_date = $5, entry_date = $6, exit_date = $7,
		    street = $8, street_no = $9, postal_code = $10, city = $11,
		    legal_hours = $12, legal_hours_until = $13, care_hours = $14,
		    is_active = $15, updated_at = $16
		WHERE id = $1
	`, child.ID, child.HouseholdID, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
		child.Street, child.StreetNo, child.PostalCode, child.City,
		child.LegalHours, child.LegalHoursUntil, child.CareHours,
		child.IsActive, child.UpdatedAt)
	if err != nil {
		return err
	}

	if !nullableIntEqual(previousLegalHours, child.LegalHours) || !nullableTimeEqual(previousLegalHoursUntil, child.LegalHoursUntil) {
		if err := r.upsertLegalHoursHistoryTx(ctx, tx, child.ID, child.LegalHours, time.Now(), child.LegalHoursUntil); err != nil {
			return err
		}
	}
	if !nullableIntEqual(previousCareHours, child.CareHours) {
		if err := r.upsertCareHoursHistoryTx(ctx, tx, child.ID, child.CareHours, time.Now()); err != nil {
			return err
		}
	}
	if err := r.syncCurrentLegalHoursTx(ctx, tx, child.ID, time.Now()); err != nil {
		return err
	}
	if err := r.syncCurrentCareHoursTx(ctx, tx, child.ID, time.Now()); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete deletes a child (hard delete).
func (r *PostgresChildRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.children WHERE id = $1`, id)
	return err
}

// GetParents retrieves all parents linked to a child.
func (r *PostgresChildRepository) GetParents(ctx context.Context, childID uuid.UUID) ([]domain.Parent, error) {
	var parents []domain.Parent
	err := r.db.SelectContext(ctx, &parents, `
		SELECT p.id, p.household_id, p.member_id, p.first_name, p.last_name, p.birth_date, p.email, p.phone,
		       p.street, p.street_no, p.postal_code, p.city,
		       p.annual_household_income, p.income_status, p.created_at, p.updated_at
		FROM fees.parents p
		INNER JOIN fees.child_parents cp ON p.id = cp.parent_id
		WHERE cp.child_id = $1
		ORDER BY cp.is_primary DESC, p.last_name, p.first_name
	`, childID)
	if err != nil {
		return nil, err
	}
	return parents, nil
}

// GetParentsForChildren batch-loads parents for multiple children.
func (r *PostgresChildRepository) GetParentsForChildren(ctx context.Context, childIDs []uuid.UUID) (map[uuid.UUID][]domain.Parent, error) {
	if len(childIDs) == 0 {
		return make(map[uuid.UUID][]domain.Parent), nil
	}

	// Query all parents for all children in one query
	query, args, err := sqlx.In(`
		SELECT p.id, p.household_id, p.member_id, p.first_name, p.last_name, p.birth_date, p.email, p.phone,
		       p.street, p.street_no, p.postal_code, p.city,
		       p.annual_household_income, p.income_status, p.created_at, p.updated_at,
		       cp.child_id
		FROM fees.parents p
		INNER JOIN fees.child_parents cp ON p.id = cp.parent_id
		WHERE cp.child_id IN (?)
		ORDER BY cp.is_primary DESC, p.last_name, p.first_name
	`, childIDs)
	if err != nil {
		return nil, err
	}

	// Rebind for postgres ($1, $2, ... instead of ?)
	query = r.db.Rebind(query)

	// Temp struct to capture child_id alongside parent data
	type parentWithChildID struct {
		domain.Parent
		ChildID uuid.UUID `db:"child_id"`
	}

	var rows []parentWithChildID
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	// Group parents by child ID
	result := make(map[uuid.UUID][]domain.Parent)
	for _, row := range rows {
		result[row.ChildID] = append(result[row.ChildID], row.Parent)
	}

	return result, nil
}

// LinkParent links a parent to a child.
func (r *PostgresChildRepository) LinkParent(ctx context.Context, childID, parentID uuid.UUID, isPrimary bool) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.child_parents (child_id, parent_id, is_primary)
		VALUES ($1, $2, $3)
		ON CONFLICT (child_id, parent_id)
		DO UPDATE SET is_primary = EXCLUDED.is_primary
	`, childID, parentID, isPrimary)
	return err
}

// UnlinkParent unlinks a parent from a child.
func (r *PostgresChildRepository) UnlinkParent(ctx context.Context, childID, parentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM fees.child_parents
		WHERE child_id = $1 AND parent_id = $2
	`, childID, parentID)
	return err
}

// GetStichtagsmeldungStats retrieves statistics for the Stichtagsmeldung report.
// It counts active children at the given stichtag date and breaks down U3 children by income.
func (r *PostgresChildRepository) GetStichtagsmeldungStats(ctx context.Context, stichtag time.Time) (*domain.StichtagsmeldungStats, error) {
	report, err := r.GetStichtagsmeldungReport(ctx, stichtag)
	if err != nil {
		return nil, err
	}

	return &domain.StichtagsmeldungStats{
		U3IncomeBreakdown:   report.U3IncomeBreakdown,
		TotalChildrenInKita: report.TotalChildrenInKita,
	}, nil
}

// GetStichtagsmeldungReport retrieves the full Stichtagsmeldung report for a specific date.
func (r *PostgresChildRepository) GetStichtagsmeldungReport(ctx context.Context, stichtag time.Time) (*domain.StichtagsmeldungReport, error) {
	u3Breakdown, totalChildren, u3ChildrenCount, err := r.getStichtagSummary(ctx, stichtag)
	if err != nil {
		return nil, err
	}

	var breakdownRows []struct {
		CareHours *int `db:"care_hours"`
		Count     int  `db:"count"`
		U3Count   int  `db:"u3_count"`
		Ue3Count  int  `db:"ue3_count"`
	}
	var legalBreakdownRows []struct {
		LegalHours *int `db:"legal_hours"`
		Count      int  `db:"count"`
		U3Count    int  `db:"u3_count"`
		Ue3Count   int  `db:"ue3_count"`
	}

	err = r.loadHoursBreakdown(ctx, &breakdownRows, stichtag, "fees.child_care_hours_history", "care_hours", "c.care_hours")
	if err != nil {
		return nil, err
	}
	err = r.loadHoursBreakdown(ctx, &legalBreakdownRows, stichtag, "fees.child_legal_hours_history", "legal_hours", "c.legal_hours")
	if err != nil {
		return nil, err
	}

	breakdown := make([]domain.CareHoursBreakdown, len(breakdownRows))
	for i, row := range breakdownRows {
		breakdown[i] = domain.CareHoursBreakdown{
			CareHours: row.CareHours,
			Count:     row.Count,
			U3Count:   row.U3Count,
			Ue3Count:  row.Ue3Count,
		}
	}
	legalBreakdown := make([]domain.LegalHoursBreakdown, len(legalBreakdownRows))
	for i, row := range legalBreakdownRows {
		legalBreakdown[i] = domain.LegalHoursBreakdown{
			LegalHours: row.LegalHours,
			Count:      row.Count,
			U3Count:    row.U3Count,
			Ue3Count:   row.Ue3Count,
		}
	}

	return &domain.StichtagsmeldungReport{
		ReportDate:          stichtag,
		U3IncomeBreakdown:   u3Breakdown,
		TotalChildrenInKita: totalChildren,
		U3ChildrenCount:     u3ChildrenCount,
		Ue3ChildrenCount:    totalChildren - u3ChildrenCount,
		CareHoursBreakdown:  breakdown,
		LegalHoursBreakdown: legalBreakdown,
	}, nil
}

// GetU3ChildrenDetails retrieves details of U3 children for the Stichtagsmeldung modal.
func (r *PostgresChildRepository) GetU3ChildrenDetails(ctx context.Context, stichtag time.Time) ([]domain.U3ChildDetail, error) {
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	var children []struct {
		ID              string   `db:"id"`
		MemberNumber    string   `db:"member_number"`
		FirstName       string   `db:"first_name"`
		LastName        string   `db:"last_name"`
		BirthDate       string   `db:"birth_date"`
		HouseholdIncome *float64 `db:"annual_household_income"`
		IncomeStatus    *string  `db:"income_status"`
	}

	err := r.db.SelectContext(ctx, &children, `
		SELECT
			c.id::text AS id,
			c.member_number,
			c.first_name,
			c.last_name,
			TO_CHAR(c.birth_date, 'YYYY-MM-DD') AS birth_date,
			h.annual_household_income,
			h.income_status
		FROM fees.children c
		LEFT JOIN fees.households h ON c.household_id = h.id
		WHERE c.is_active = true
		  AND c.entry_date <= $1
		  AND (c.exit_date IS NULL OR c.exit_date >= $1)
		  AND c.birth_date > $2
		ORDER BY c.last_name, c.first_name
	`, stichtag, u3Threshold)
	if err != nil {
		log.Error().Err(err).Msg("GetU3ChildrenDetails query failed")
		return nil, err
	}

	result := make([]domain.U3ChildDetail, len(children))
	for i, c := range children {
		isFoster := c.IncomeStatus != nil && *c.IncomeStatus == "FOSTER_FAMILY"
		var income *int
		if c.HouseholdIncome != nil {
			incomeInt := int(*c.HouseholdIncome)
			income = &incomeInt
		}
		result[i] = domain.U3ChildDetail{
			ID:              c.ID,
			MemberNumber:    c.MemberNumber,
			FirstName:       c.FirstName,
			LastName:        c.LastName,
			BirthDate:       c.BirthDate,
			HouseholdIncome: income,
			IncomeStatus:    c.IncomeStatus,
			IsFosterFamily:  isFoster,
		}
	}

	return result, nil
}

// ListCareHoursHistory returns the care hours history for a child.
func (r *PostgresChildRepository) ListCareHoursHistory(ctx context.Context, childID uuid.UUID) ([]domain.ChildCareHoursHistory, error) {
	var rows []domain.ChildCareHoursHistory
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, child_id, care_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_care_hours_history
		WHERE child_id = $1
		ORDER BY effective_from DESC, created_at DESC
	`, childID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertCareHoursHistory creates or updates a care hours period for a child.
func (r *PostgresChildRepository) UpsertCareHoursHistory(ctx context.Context, childID uuid.UUID, careHours *int, validFrom time.Time) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.upsertCareHoursHistoryTx(ctx, tx, childID, careHours, validFrom); err != nil {
		return err
	}
	if err := r.syncCurrentCareHoursTx(ctx, tx, childID, time.Now()); err != nil {
		return err
	}

	return tx.Commit()
}

// ListLegalHoursHistory returns the legal hours history for a child.
func (r *PostgresChildRepository) ListLegalHoursHistory(ctx context.Context, childID uuid.UUID) ([]domain.ChildLegalHoursHistory, error) {
	var rows []domain.ChildLegalHoursHistory
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, child_id, legal_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_legal_hours_history
		WHERE child_id = $1
		ORDER BY effective_from DESC, created_at DESC
	`, childID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertLegalHoursHistory creates or updates a legal hours period for a child.
func (r *PostgresChildRepository) UpsertLegalHoursHistory(ctx context.Context, childID uuid.UUID, legalHours *int, validFrom time.Time) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.upsertLegalHoursHistoryTx(ctx, tx, childID, legalHours, validFrom, nil); err != nil {
		return err
	}
	if err := r.syncCurrentLegalHoursTx(ctx, tx, childID, time.Now()); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresChildRepository) getStichtagSummary(ctx context.Context, stichtag time.Time) (domain.U3IncomeBreakdown, int, int, error) {
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	var breakdown struct {
		UpTo20k      int `db:"up_to_20k"`
		From20To35k  int `db:"from_20_to_35k"`
		From35To55k  int `db:"from_35_to_55k"`
		MaxAccepted  int `db:"max_accepted"`
		FosterFamily int `db:"foster_family"`
		Total        int `db:"total"`
	}

	err := r.db.GetContext(ctx, &breakdown, `
		SELECT
			COUNT(*) FILTER (WHERE COALESCE(h.income_status, '') NOT IN ('MAX_ACCEPTED', 'FOSTER_FAMILY') AND COALESCE(h.annual_household_income, 0) <= 20000) AS up_to_20k,
			COUNT(*) FILTER (WHERE COALESCE(h.income_status, '') NOT IN ('MAX_ACCEPTED', 'FOSTER_FAMILY') AND h.annual_household_income > 20000 AND h.annual_household_income <= 35000) AS from_20_to_35k,
			COUNT(*) FILTER (WHERE COALESCE(h.income_status, '') NOT IN ('MAX_ACCEPTED', 'FOSTER_FAMILY') AND h.annual_household_income > 35000 AND h.annual_household_income <= 55000) AS from_35_to_55k,
			COUNT(*) FILTER (WHERE h.income_status = 'MAX_ACCEPTED') AS max_accepted,
			COUNT(*) FILTER (WHERE h.income_status = 'FOSTER_FAMILY') AS foster_family,
			COUNT(*) AS total
		FROM fees.children c
		LEFT JOIN fees.households h ON c.household_id = h.id
		WHERE c.is_active = true
		  AND c.entry_date <= $1
		  AND (c.exit_date IS NULL OR c.exit_date >= $1)
		  AND c.birth_date > $2
	`, stichtag, u3Threshold)
	if err != nil {
		return domain.U3IncomeBreakdown{}, 0, 0, err
	}

	var totalChildren int
	err = r.db.GetContext(ctx, &totalChildren, `
		SELECT COUNT(*)
		FROM fees.children
		WHERE is_active = true
		  AND entry_date <= $1
		  AND (exit_date IS NULL OR exit_date >= $1)
	`, stichtag)
	if err != nil {
		return domain.U3IncomeBreakdown{}, 0, 0, err
	}

	return domain.U3IncomeBreakdown{
		UpTo20k:      breakdown.UpTo20k,
		From20To35k:  breakdown.From20To35k,
		From35To55k:  breakdown.From35To55k,
		MaxAccepted:  breakdown.MaxAccepted,
		FosterFamily: breakdown.FosterFamily,
		Total:        breakdown.Total,
	}, totalChildren, breakdown.Total, nil
}

func (r *PostgresChildRepository) upsertCareHoursHistoryTx(ctx context.Context, tx *sqlx.Tx, childID uuid.UUID, careHours *int, validFrom time.Time) error {
	var rows []careHoursHistoryRow
	err := tx.SelectContext(ctx, &rows, `
		SELECT id, child_id, care_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_care_hours_history
		WHERE child_id = $1
		ORDER BY effective_from ASC, created_at ASC
		FOR UPDATE
	`, childID)
	if err != nil {
		return err
	}

	newPeriod := careHoursHistoryRow{
		ID:            uuid.New(),
		ChildID:       childID,
		CareHours:     cloneNullableInt(careHours),
		EffectiveFrom: truncateDate(validFrom),
	}

	insertAt := len(rows)
	updatedExisting := false

	for i := range rows {
		rows[i].EffectiveFrom = truncateDate(rows[i].EffectiveFrom)
		if rows[i].EffectiveUntil != nil {
			until := truncateDate(*rows[i].EffectiveUntil)
			rows[i].EffectiveUntil = &until
		}

		if rows[i].EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
			rows[i].CareHours = cloneNullableInt(careHours)
			updatedExisting = true
			insertAt = i
			break
		}

		if rows[i].EffectiveFrom.After(newPeriod.EffectiveFrom) {
			insertAt = i
			break
		}
	}

	if !updatedExisting {
		containing := -1
		for i, row := range rows {
			if !row.EffectiveFrom.After(newPeriod.EffectiveFrom) && (row.EffectiveUntil == nil || !row.EffectiveUntil.Before(newPeriod.EffectiveFrom)) {
				containing = i
				break
			}
		}

		if containing >= 0 {
			row := rows[containing]
			if row.EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
				rows[containing].CareHours = cloneNullableInt(careHours)
			} else {
				until := newPeriod.EffectiveFrom.AddDate(0, 0, -1)
				rows[containing].EffectiveUntil = &until
				newPeriod.EffectiveUntil = row.EffectiveUntil
				insertAt = containing + 1
				rows = append(rows[:insertAt], append([]careHoursHistoryRow{newPeriod}, rows[insertAt:]...)...)
			}
		} else {
			if insertAt < len(rows) {
				until := rows[insertAt].EffectiveFrom.AddDate(0, 0, -1)
				newPeriod.EffectiveUntil = &until
			}
			rows = append(rows, careHoursHistoryRow{})
			copy(rows[insertAt+1:], rows[insertAt:])
			rows[insertAt] = newPeriod
		}
	}

	rows = normalizeCareHoursHistoryRows(rows)
	if _, err := tx.ExecContext(ctx, `DELETE FROM fees.child_care_hours_history WHERE child_id = $1`, childID); err != nil {
		return err
	}

	now := time.Now()
	for _, row := range rows {
		id := row.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO fees.child_care_hours_history (
				id, child_id, care_hours, effective_from, effective_until, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, childID, row.CareHours, row.EffectiveFrom, row.EffectiveUntil, now, now)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresChildRepository) upsertLegalHoursHistoryTx(ctx context.Context, tx *sqlx.Tx, childID uuid.UUID, legalHours *int, validFrom time.Time, validUntil *time.Time) error {
	var rows []legalHoursHistoryRow
	err := tx.SelectContext(ctx, &rows, `
		SELECT id, child_id, legal_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_legal_hours_history
		WHERE child_id = $1
		ORDER BY effective_from ASC, created_at ASC
		FOR UPDATE
	`, childID)
	if err != nil {
		return err
	}

	newPeriod := legalHoursHistoryRow{
		ID:            uuid.New(),
		ChildID:       childID,
		LegalHours:    cloneNullableInt(legalHours),
		EffectiveFrom: truncateDate(validFrom),
	}
	if validUntil != nil {
		until := truncateDate(*validUntil)
		newPeriod.EffectiveUntil = &until
	}

	insertAt := len(rows)
	updatedExisting := false

	for i := range rows {
		rows[i].EffectiveFrom = truncateDate(rows[i].EffectiveFrom)
		if rows[i].EffectiveUntil != nil {
			until := truncateDate(*rows[i].EffectiveUntil)
			rows[i].EffectiveUntil = &until
		}

		if rows[i].EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
			rows[i].LegalHours = cloneNullableInt(legalHours)
			rows[i].EffectiveUntil = cloneNullableTime(newPeriod.EffectiveUntil)
			updatedExisting = true
			insertAt = i
			break
		}

		if rows[i].EffectiveFrom.After(newPeriod.EffectiveFrom) {
			insertAt = i
			break
		}
	}

	if !updatedExisting {
		containing := -1
		for i, row := range rows {
			if !row.EffectiveFrom.After(newPeriod.EffectiveFrom) && (row.EffectiveUntil == nil || !row.EffectiveUntil.Before(newPeriod.EffectiveFrom)) {
				containing = i
				break
			}
		}

		if containing >= 0 {
			row := rows[containing]
			if row.EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
				rows[containing].LegalHours = cloneNullableInt(legalHours)
				rows[containing].EffectiveUntil = cloneNullableTime(newPeriod.EffectiveUntil)
			} else {
				originalUntil := cloneNullableTime(row.EffectiveUntil)
				until := newPeriod.EffectiveFrom.AddDate(0, 0, -1)
				rows[containing].EffectiveUntil = &until
				newPeriod.EffectiveUntil = minNullableDate(newPeriod.EffectiveUntil, originalUntil)
				insertAt = containing + 1
				rows = append(rows[:insertAt], append([]legalHoursHistoryRow{newPeriod}, rows[insertAt:]...)...)
			}
		} else {
			if insertAt < len(rows) {
				nextUntil := rows[insertAt].EffectiveFrom.AddDate(0, 0, -1)
				newPeriod.EffectiveUntil = minNullableDate(newPeriod.EffectiveUntil, &nextUntil)
			}
			rows = append(rows, legalHoursHistoryRow{})
			copy(rows[insertAt+1:], rows[insertAt:])
			rows[insertAt] = newPeriod
		}
	}

	rows = normalizeLegalHoursHistoryRows(rows)
	if _, err := tx.ExecContext(ctx, `DELETE FROM fees.child_legal_hours_history WHERE child_id = $1`, childID); err != nil {
		return err
	}

	now := time.Now()
	for _, row := range rows {
		id := row.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO fees.child_legal_hours_history (
				id, child_id, legal_hours, effective_from, effective_until, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, childID, row.LegalHours, row.EffectiveFrom, row.EffectiveUntil, now, now)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresChildRepository) syncCurrentCareHoursTx(ctx context.Context, tx *sqlx.Tx, childID uuid.UUID, at time.Time) error {
	var careHours *int
	var currentRaw sql.NullInt64
	err := tx.GetContext(ctx, &currentRaw, `
		SELECT care_hours
		FROM fees.child_care_hours_history
		WHERE child_id = $1
		  AND effective_from <= $2
		  AND (effective_until IS NULL OR effective_until >= $2)
		ORDER BY effective_from DESC, created_at DESC
		LIMIT 1
	`, childID, truncateDate(at))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		careHours = nil
	} else if currentRaw.Valid {
		value := int(currentRaw.Int64)
		careHours = &value
	}

	_, err = tx.ExecContext(ctx, `UPDATE fees.children SET care_hours = $2 WHERE id = $1`, childID, careHours)
	return err
}

func (r *PostgresChildRepository) syncCurrentLegalHoursTx(ctx context.Context, tx *sqlx.Tx, childID uuid.UUID, at time.Time) error {
	type currentLegalRow struct {
		LegalHours     sql.NullInt64 `db:"legal_hours"`
		EffectiveUntil *time.Time    `db:"effective_until"`
	}
	var current currentLegalRow
	err := tx.GetContext(ctx, &current, `
		SELECT legal_hours, effective_until
		FROM fees.child_legal_hours_history
		WHERE child_id = $1
		  AND effective_from <= $2
		  AND (effective_until IS NULL OR effective_until >= $2)
		ORDER BY effective_from DESC, created_at DESC
		LIMIT 1
	`, childID, truncateDate(at))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var legalHours *int
	var legalHoursUntil *time.Time
	if !errors.Is(err, sql.ErrNoRows) {
		if current.LegalHours.Valid {
			value := int(current.LegalHours.Int64)
			legalHours = &value
		}
		legalHoursUntil = current.EffectiveUntil
	}

	_, err = tx.ExecContext(ctx, `UPDATE fees.children SET legal_hours = $2, legal_hours_until = $3 WHERE id = $1`, childID, legalHours, legalHoursUntil)
	return err
}

func normalizeCareHoursHistoryRows(rows []careHoursHistoryRow) []careHoursHistoryRow {
	if len(rows) == 0 {
		return rows
	}

	normalized := make([]careHoursHistoryRow, 0, len(rows))
	for _, row := range rows {
		if row.EffectiveUntil != nil && row.EffectiveUntil.Before(row.EffectiveFrom) {
			continue
		}

		if len(normalized) == 0 {
			normalized = append(normalized, row)
			continue
		}

		prev := &normalized[len(normalized)-1]
		if nullableIntEqual(prev.CareHours, row.CareHours) && prev.EffectiveUntil != nil && prev.EffectiveUntil.AddDate(0, 0, 1).Equal(row.EffectiveFrom) {
			prev.EffectiveUntil = row.EffectiveUntil
			continue
		}

		normalized = append(normalized, row)
	}

	return normalized
}

func normalizeLegalHoursHistoryRows(rows []legalHoursHistoryRow) []legalHoursHistoryRow {
	if len(rows) == 0 {
		return rows
	}

	normalized := make([]legalHoursHistoryRow, 0, len(rows))
	for _, row := range rows {
		if row.EffectiveUntil != nil && row.EffectiveUntil.Before(row.EffectiveFrom) {
			continue
		}
		if len(normalized) == 0 {
			normalized = append(normalized, row)
			continue
		}

		prev := &normalized[len(normalized)-1]
		if nullableIntEqual(prev.LegalHours, row.LegalHours) && prev.EffectiveUntil != nil && prev.EffectiveUntil.AddDate(0, 0, 1).Equal(row.EffectiveFrom) {
			prev.EffectiveUntil = row.EffectiveUntil
			continue
		}

		if prev.EffectiveUntil == nil || prev.EffectiveUntil.After(row.EffectiveFrom.AddDate(0, 0, -1)) {
			until := row.EffectiveFrom.AddDate(0, 0, -1)
			prev.EffectiveUntil = &until
		}

		normalized = append(normalized, row)
	}

	return normalized
}

func (r *PostgresChildRepository) loadHoursBreakdown(ctx context.Context, dest interface{}, stichtag time.Time, historyTable, historyColumn, fallbackColumn string) error {
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN history_match.found IS TRUE THEN history_match.value
				ELSE %s
			END AS %s,
			COUNT(*) AS count,
			COUNT(*) FILTER (WHERE c.birth_date > $2) AS u3_count,
			COUNT(*) FILTER (WHERE c.birth_date <= $2) AS ue3_count
		FROM fees.children c
		LEFT JOIN LATERAL (
			SELECT %s AS value, TRUE AS found
			FROM %s h
			WHERE h.child_id = c.id
			  AND h.effective_from <= $1
			  AND (h.effective_until IS NULL OR h.effective_until >= $1)
			ORDER BY h.effective_from DESC, h.created_at DESC
			LIMIT 1
		) history_match ON TRUE
		WHERE c.is_active = true
		  AND c.entry_date <= $1
		  AND (c.exit_date IS NULL OR c.exit_date >= $1)
		GROUP BY 1
		ORDER BY (CASE
			WHEN history_match.found IS TRUE THEN history_match.value
			ELSE %s
		END) IS NULL,
		(CASE
			WHEN history_match.found IS TRUE THEN history_match.value
			ELSE %s
		END) ASC
	`, fallbackColumn, historyColumn, historyColumn, historyTable, fallbackColumn, fallbackColumn)

	return r.db.SelectContext(ctx, dest, query, stichtag, u3Threshold)
}

func truncateDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func cloneNullableInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func nullableIntEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func nullableTimeEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func cloneNullableTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func minNullableDate(a, b *time.Time) *time.Time {
	if a == nil {
		return cloneNullableTime(b)
	}
	if b == nil {
		return cloneNullableTime(a)
	}
	if a.Before(*b) {
		return cloneNullableTime(a)
	}
	return cloneNullableTime(b)
}
