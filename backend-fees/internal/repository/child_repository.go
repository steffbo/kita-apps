package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

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
func (r *PostgresChildRepository) List(ctx context.Context, activeOnly bool, u3Only bool, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Child, int64, error) {
	var children []domain.Child
	var total int64

	baseQuery := `FROM fees.children WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if activeOnly {
		baseQuery += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if u3Only {
		// Filter for children under 3 years old (born less than 3 years ago)
		baseQuery += fmt.Sprintf(" AND birth_date > $%d", argIdx)
		args = append(args, time.Now().AddDate(-3, 0, 0))
		argIdx++
	}

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR member_number ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Determine sort order
	orderClause := getChildSortOrder(sortBy, sortDir)

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
		       street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
		       is_active, created_at, updated_at
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &children, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return children, total, nil
}

// getChildSortOrder returns a safe ORDER BY clause for children.
func getChildSortOrder(sortBy, sortDir string) string {
	// Whitelist of allowed sort columns to prevent SQL injection
	allowedColumns := map[string]string{
		"memberNumber": "member_number",
		"name":         "last_name, first_name",
		"birthDate":    "birth_date",
		"age":          "birth_date", // age sorts by birth_date (reversed direction)
		"entryDate":    "entry_date",
		"createdAt":    "created_at",
	}

	// Default sort
	column := "last_name, first_name"
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
		SELECT id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
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

// GetByMemberNumber retrieves a child by member number.
func (r *PostgresChildRepository) GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Child, error) {
	var child domain.Child
	err := r.db.GetContext(ctx, &child, `
		SELECT id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
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

// Create creates a new child.
func (r *PostgresChildRepository) Create(ctx context.Context, child *domain.Child) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.children (id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
		                           street, street_no, postal_code, city, legal_hours, legal_hours_until, care_hours,
		                           is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, child.ID, child.MemberNumber, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
		child.Street, child.StreetNo, child.PostalCode, child.City, child.LegalHours, child.LegalHoursUntil, child.CareHours,
		child.IsActive, child.CreatedAt, child.UpdatedAt)
	return err
}

// Update updates an existing child.
func (r *PostgresChildRepository) Update(ctx context.Context, child *domain.Child) error {
	child.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.children
		SET first_name = $2, last_name = $3, birth_date = $4, entry_date = $5, exit_date = $6,
		    street = $7, street_no = $8, postal_code = $9, city = $10,
		    legal_hours = $11, legal_hours_until = $12, care_hours = $13,
		    is_active = $14, updated_at = $15
		WHERE id = $1
	`, child.ID, child.FirstName, child.LastName, child.BirthDate, child.EntryDate, child.ExitDate,
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
		SELECT p.id, p.first_name, p.last_name, p.birth_date, p.email, p.phone,
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
