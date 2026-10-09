package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// PostgresChildRepository is the PostgreSQL implementation of ChildRepository.
type PostgresChildRepository struct {
	db *sqlx.DB
}

// NewPostgresChildRepository creates a new PostgreSQL child repository.
func NewPostgresChildRepository(db *sqlx.DB) *PostgresChildRepository {
	return &PostgresChildRepository{db: db}
}

// Care hours (Betreuungszeit) and legal hours (Rechtsanspruch) are stored exclusively in
// fees.child_care_hours_history and fees.child_legal_hours_history. The values exposed on
// domain.Child are derived from the period that is effective today, so they can no longer
// go stale the way the previously denormalized fees.children columns did.
const childCurrentHoursJoins = `
		LEFT JOIN LATERAL (
			SELECT h.care_hours
			FROM fees.child_care_hours_history h
			WHERE h.child_id = c.id
			  AND h.effective_from <= CURRENT_DATE
			  AND (h.effective_until IS NULL OR h.effective_until >= CURRENT_DATE)
			ORDER BY h.effective_from DESC, h.created_at DESC
			LIMIT 1
		) cch ON TRUE
		LEFT JOIN LATERAL (
			SELECT h.legal_hours, h.effective_until
			FROM fees.child_legal_hours_history h
			WHERE h.child_id = c.id
			  AND h.effective_from <= CURRENT_DATE
			  AND (h.effective_until IS NULL OR h.effective_until >= CURRENT_DATE)
			ORDER BY h.effective_from DESC, h.created_at DESC
			LIMIT 1
		) clh ON TRUE`

// childSelectColumns lists all fields of domain.Child, including the derived hour values.
// It requires fees.children to be aliased as c and childCurrentHoursJoins to be joined.
const childSelectColumns = `c.id, c.household_id, c.member_number, c.first_name, c.last_name, c.birth_date, c.entry_date, c.exit_date,
		       c.street, c.street_no, c.postal_code, c.city,
		       clh.legal_hours, clh.effective_until AS legal_hours_until, cch.care_hours,
		       c.is_active, c.created_at, c.updated_at`

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
		) ofe ON ofe.child_id = c.id` + childCurrentHoursJoins + `
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
		args = append(args, util.Today().AddDate(-3, 0, 0))
		argIdx++
	}

	if hasWarnings {
		// Children with warnings:
		// 1. No parents linked
		// 2. No legal hours recorded at all (in any period)
		// 3. No care hours recorded at all (in any period)
		// 4. U3 children where neither household nor any parent has income info
		//    (income is NOT required if status is MAX_ACCEPTED, NOT_REQUIRED, FOSTER_FAMILY, or HISTORIC)
		//
		// Hours are checked against the history, not against the currently effective period,
		// so a child whose care hours start in the future is not reported as missing data.
		baseQuery += ` AND (
			-- No parents linked
			NOT EXISTS (SELECT 1 FROM fees.child_parents cp WHERE cp.child_id = c.id)
			-- No legal hours
			OR NOT EXISTS (
				SELECT 1 FROM fees.child_legal_hours_history lh
				WHERE lh.child_id = c.id AND lh.legal_hours IS NOT NULL
			)
			-- No care hours
			OR NOT EXISTS (
				SELECT 1 FROM fees.child_care_hours_history ch
				WHERE ch.child_id = c.id AND ch.care_hours IS NOT NULL
			)
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
		args = append(args, util.Today().AddDate(-3, 0, 0))
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
	err := conn(ctx, r.db).GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		log.Error().Err(err).Str("query", countQuery).Msg("Child count query failed")
		return nil, 0, err
	}

	// Determine sort order
	orderClause := getChildSortOrder(sortBy, sortDir)

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT %s, ofe.open_fees_count
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, childSelectColumns, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = conn(ctx, r.db).SelectContext(ctx, &children, selectQuery, args...)
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
	err := conn(ctx, r.db).GetContext(ctx, &child, `
		SELECT `+childSelectColumns+`
		FROM fees.children c`+childCurrentHoursJoins+`
		WHERE c.id = $1
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
	err := conn(ctx, r.db).SelectContext(ctx, &children, `
		SELECT `+childSelectColumns+`
		FROM fees.children c`+childCurrentHoursJoins+`
		WHERE c.id = ANY($1)
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

// GetByHouseholdID retrieves all children linked to a household.
func (r *PostgresChildRepository) GetByHouseholdID(ctx context.Context, householdID uuid.UUID) ([]domain.Child, error) {
	var children []domain.Child
	err := conn(ctx, r.db).SelectContext(ctx, &children, `
		SELECT `+childSelectColumns+`
		FROM fees.children c`+childCurrentHoursJoins+`
		WHERE c.household_id = $1
		ORDER BY c.entry_date ASC, c.created_at ASC, c.id ASC
	`, householdID)
	if err != nil {
		return nil, err
	}
	return children, nil
}

// GetByMemberNumber retrieves a child by member number.
func (r *PostgresChildRepository) GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Child, error) {
	var child domain.Child
	err := conn(ctx, r.db).GetContext(ctx, &child, `
		SELECT `+childSelectColumns+`
		FROM fees.children c`+childCurrentHoursJoins+`
		WHERE c.member_number = $1
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
	err := conn(ctx, r.db).GetContext(ctx, &maxNum, `
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
	tx, err := beginTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO fees.children (id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
			                           street, street_no, postal_code, city,
			                           is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, child.ID, child.HouseholdID, child.MemberNumber, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
		child.Street, child.StreetNo, child.PostalCode, child.City,
		child.IsActive, child.CreatedAt, child.UpdatedAt)
	if err != nil {
		return err
	}

	// Hours are only persisted as history periods; the child row itself stores no hours.
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

	return tx.Commit()
}

// Update updates an existing child.
func (r *PostgresChildRepository) Update(ctx context.Context, child *domain.Child) error {
	child.UpdatedAt = util.Now()
	tx, err := beginTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Lock the child row and read the hours that are effective today from the history.
	// Changed hours are written as a new history period effective from today.
	var lockedID uuid.UUID
	if err := tx.GetContext(ctx, &lockedID, `SELECT id FROM fees.children WHERE id = $1 FOR UPDATE`, child.ID); err != nil {
		return err
	}
	now := util.Today()
	previousCareHours, err := currentCareHoursTx(ctx, tx, child.ID, now)
	if err != nil {
		return err
	}
	previousLegalHours, previousLegalHoursUntil, err := currentLegalHoursTx(ctx, tx, child.ID, now)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE fees.children
		SET household_id = $2, first_name = $3, last_name = $4, birth_date = $5, entry_date = $6, exit_date = $7,
		    street = $8, street_no = $9, postal_code = $10, city = $11,
		    is_active = $12, updated_at = $13
		WHERE id = $1
	`, child.ID, child.HouseholdID, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
		child.Street, child.StreetNo, child.PostalCode, child.City,
		child.IsActive, child.UpdatedAt)
	if err != nil {
		return err
	}

	if !nullableIntEqual(previousLegalHours, child.LegalHours) || !nullableTimeEqual(previousLegalHoursUntil, child.LegalHoursUntil) {
		if err := r.upsertLegalHoursHistoryTx(ctx, tx, child.ID, child.LegalHours, now, child.LegalHoursUntil); err != nil {
			return err
		}
	}
	if !nullableIntEqual(previousCareHours, child.CareHours) {
		if err := r.upsertCareHoursHistoryTx(ctx, tx, child.ID, child.CareHours, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete deletes a child (hard delete).
func (r *PostgresChildRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.children WHERE id = $1`, id)
	return err
}

// GetParents retrieves all parents linked to a child.
func (r *PostgresChildRepository) GetParents(ctx context.Context, childID uuid.UUID) ([]domain.Parent, error) {
	var parents []domain.Parent
	err := conn(ctx, r.db).SelectContext(ctx, &parents, `
		SELECT p.id, p.household_id, p.member_id, p.first_name, p.last_name, p.birth_date,
		       COALESCE(NULLIF(TRIM(p.email), ''), m.email) AS email,
		       p.phone, p.street, p.street_no, p.postal_code, p.city,
		       p.annual_household_income, p.income_status, p.created_at, p.updated_at
		FROM fees.parents p
		INNER JOIN fees.child_parents cp ON p.id = cp.parent_id
		LEFT JOIN fees.members m ON m.id = p.member_id
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
		SELECT p.id, p.household_id, p.member_id, p.first_name, p.last_name, p.birth_date,
		       COALESCE(NULLIF(TRIM(p.email), ''), m.email) AS email,
		       p.phone, p.street, p.street_no, p.postal_code, p.city,
		       p.annual_household_income, p.income_status, p.created_at, p.updated_at,
		       cp.child_id
		FROM fees.parents p
		INNER JOIN fees.child_parents cp ON p.id = cp.parent_id
		LEFT JOIN fees.members m ON m.id = p.member_id
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
	if err := conn(ctx, r.db).SelectContext(ctx, &rows, query, args...); err != nil {
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
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.child_parents (child_id, parent_id, is_primary)
		VALUES ($1, $2, $3)
		ON CONFLICT (child_id, parent_id)
		DO UPDATE SET is_primary = EXCLUDED.is_primary
	`, childID, parentID, isPrimary)
	return err
}

// UnlinkParent unlinks a parent from a child.
func (r *PostgresChildRepository) UnlinkParent(ctx context.Context, childID, parentID uuid.UUID) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		DELETE FROM fees.child_parents
		WHERE child_id = $1 AND parent_id = $2
	`, childID, parentID)
	return err
}
