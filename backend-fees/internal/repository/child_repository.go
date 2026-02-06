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
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.children (id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
			                           street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
			                           is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, child.ID, child.HouseholdID, child.MemberNumber, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
		child.Street, child.StreetNo, child.PostalCode, child.City, child.LegalHours, child.LegalHoursUntil, child.CareHours,
		child.IsActive, child.CreatedAt, child.UpdatedAt)
	return err
}

// Update updates an existing child.
func (r *PostgresChildRepository) Update(ctx context.Context, child *domain.Child) error {
	child.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
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
	return err
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
// Foster families (income_status = 'FOSTER_FAMILY') are excluded from income breakdown counts.
func (r *PostgresChildRepository) GetStichtagsmeldungStats(ctx context.Context, stichtag time.Time) (*domain.StichtagsmeldungStats, error) {
	// U3 threshold: children born after stichtag - 3 years are U3
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	// Query for income breakdown - only U3 children, excluding foster families
	var breakdown struct {
		UpTo20k     int `db:"up_to_20k"`
		From20To35k int `db:"from_20_to_35k"`
		From35To55k int `db:"from_35_to_55k"`
		Total       int `db:"total"`
	}

	err := r.db.GetContext(ctx, &breakdown, `
		SELECT
			COUNT(*) FILTER (WHERE COALESCE(h.annual_household_income, 0) <= 20000) AS up_to_20k,
			COUNT(*) FILTER (WHERE h.annual_household_income > 20000 AND h.annual_household_income <= 35000) AS from_20_to_35k,
			COUNT(*) FILTER (WHERE h.annual_household_income > 35000 AND h.annual_household_income <= 55000) AS from_35_to_55k,
			COUNT(*) AS total
		FROM fees.children c
		LEFT JOIN fees.households h ON c.household_id = h.id
		WHERE c.is_active = true
		  AND c.entry_date <= $1
		  AND c.birth_date > $2
		  AND COALESCE(h.income_status, '') != 'FOSTER_FAMILY'
	`, stichtag, u3Threshold)
	if err != nil {
		return nil, err
	}

	// Query for total active children
	var totalChildren int
	err = r.db.GetContext(ctx, &totalChildren, `
		SELECT COUNT(*)
		FROM fees.children
		WHERE is_active = true
		  AND entry_date <= $1
	`, stichtag)
	if err != nil {
		return nil, err
	}

	return &domain.StichtagsmeldungStats{
		U3IncomeBreakdown: domain.U3IncomeBreakdown{
			UpTo20k:     breakdown.UpTo20k,
			From20To35k: breakdown.From20To35k,
			From35To55k: breakdown.From35To55k,
			Total:       breakdown.Total,
		},
		TotalChildrenInKita: totalChildren,
	}, nil
}
