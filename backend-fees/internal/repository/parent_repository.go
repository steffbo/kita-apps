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

// PostgresParentRepository is the PostgreSQL implementation of ParentRepository.
type PostgresParentRepository struct {
	db *sqlx.DB
}

// NewPostgresParentRepository creates a new PostgreSQL parent repository.
func NewPostgresParentRepository(db *sqlx.DB) *PostgresParentRepository {
	return &PostgresParentRepository{db: db}
}

// List retrieves parents with optional search filtering and sorting.
func (r *PostgresParentRepository) List(ctx context.Context, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Parent, int64, error) {
	var parents []domain.Parent
	var total int64

	baseQuery := `FROM fees.parents WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get sort order
	orderClause := getParentSortOrder(sortBy, sortDir)

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, household_id, member_id, first_name, last_name, birth_date, email, phone,
		       street, street_no, postal_code, city,
		       annual_household_income, income_status, created_at, updated_at
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &parents, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return parents, total, nil
}

// getParentSortOrder returns a safe ORDER BY clause for parents.
func getParentSortOrder(sortBy, sortDir string) string {
	// Validate sort direction
	dir := "ASC"
	if sortDir == "desc" {
		dir = "DESC"
	}

	// Map allowed column names to actual database columns
	allowedColumns := map[string]string{
		"name":  "last_name",
		"email": "email",
	}

	if col, ok := allowedColumns[sortBy]; ok {
		if sortBy == "name" {
			// Sort by last_name, then first_name
			return fmt.Sprintf("last_name %s, first_name %s", dir, dir)
		}
		return fmt.Sprintf("%s %s NULLS LAST", col, dir)
	}

	// Default sort
	return fmt.Sprintf("last_name %s, first_name %s", dir, dir)
}

// GetByID retrieves a parent by ID.
func (r *PostgresParentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	var parent domain.Parent
	err := r.db.GetContext(ctx, &parent, `
		SELECT id, household_id, member_id, first_name, last_name, birth_date, email, phone,
		       street, street_no, postal_code, city,
		       annual_household_income, income_status, created_at, updated_at
		FROM fees.parents
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("parent not found")
		}
		return nil, err
	}
	return &parent, nil
}

// FindByNameAndEmail finds a parent by first name, last name, and email.
// Returns nil if no matching parent is found (not an error).
func (r *PostgresParentRepository) FindByNameAndEmail(ctx context.Context, firstName, lastName, email string) (*domain.Parent, error) {
	var parent domain.Parent
	err := r.db.GetContext(ctx, &parent, `
		SELECT id, household_id, member_id, first_name, last_name, birth_date, email, phone,
		       street, street_no, postal_code, city,
		       annual_household_income, income_status, created_at, updated_at
		FROM fees.parents
		WHERE LOWER(first_name) = LOWER($1) 
		  AND LOWER(last_name) = LOWER($2) 
		  AND LOWER(email) = LOWER($3)
	`, firstName, lastName, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &parent, nil
}

// Create creates a new parent.
func (r *PostgresParentRepository) Create(ctx context.Context, parent *domain.Parent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.parents (id, household_id, member_id, first_name, last_name, birth_date, email, phone,
		                          street, street_no, postal_code, city,
		                          annual_household_income, income_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, parent.ID, parent.HouseholdID, parent.MemberID, parent.FirstName, parent.LastName, parent.BirthDate, parent.Email, parent.Phone,
		parent.Street, parent.StreetNo, parent.PostalCode, parent.City,
		parent.AnnualHouseholdIncome, parent.IncomeStatus, parent.CreatedAt, parent.UpdatedAt)
	return err
}

// Update updates an existing parent.
func (r *PostgresParentRepository) Update(ctx context.Context, parent *domain.Parent) error {
	parent.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.parents
		SET household_id = $2, member_id = $3, first_name = $4, last_name = $5, birth_date = $6, email = $7, phone = $8,
		    street = $9, street_no = $10, postal_code = $11, city = $12,
		    annual_household_income = $13, income_status = $14, updated_at = $15
		WHERE id = $1
	`, parent.ID, parent.HouseholdID, parent.MemberID, parent.FirstName, parent.LastName, parent.BirthDate, parent.Email, parent.Phone,
		parent.Street, parent.StreetNo, parent.PostalCode, parent.City,
		parent.AnnualHouseholdIncome, parent.IncomeStatus, parent.UpdatedAt)
	return err
}

// Delete deletes a parent.
func (r *PostgresParentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.parents WHERE id = $1`, id)
	return err
}

// GetChildren retrieves all children linked to a parent.
func (r *PostgresParentRepository) GetChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Child, error) {
	var children []domain.Child
	err := r.db.SelectContext(ctx, &children, `
		SELECT c.id, c.member_number, c.first_name, c.last_name, c.birth_date, c.entry_date,
		       c.street, c.street_no, c.postal_code, c.city,
		       c.is_active, c.created_at, c.updated_at
		FROM fees.children c
		INNER JOIN fees.child_parents cp ON c.id = cp.child_id
		WHERE cp.parent_id = $1
		ORDER BY c.last_name, c.first_name
	`, parentID)
	if err != nil {
		return nil, err
	}
	return children, nil
}

// childWithParentID is a helper struct for batch loading children with their parent ID.
type childWithParentID struct {
	domain.Child
	ParentID uuid.UUID `db:"parent_id"`
}

// GetChildrenForParents retrieves children for multiple parents in a single query.
func (r *PostgresParentRepository) GetChildrenForParents(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID][]domain.Child, error) {
	if len(parentIDs) == 0 {
		return make(map[uuid.UUID][]domain.Child), nil
	}

	query, args, err := sqlx.In(`
		SELECT c.id, c.member_number, c.first_name, c.last_name, c.birth_date, c.entry_date,
		       c.street, c.street_no, c.postal_code, c.city,
		       c.is_active, c.created_at, c.updated_at,
		       cp.parent_id
		FROM fees.children c
		INNER JOIN fees.child_parents cp ON c.id = cp.child_id
		WHERE cp.parent_id IN (?)
		ORDER BY c.last_name, c.first_name
	`, parentIDs)
	if err != nil {
		return nil, err
	}

	// Rebind for PostgreSQL
	query = r.db.Rebind(query)

	var rows []childWithParentID
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	// Group by parent ID
	result := make(map[uuid.UUID][]domain.Child)
	for _, row := range rows {
		result[row.ParentID] = append(result[row.ParentID], row.Child)
	}

	return result, nil
}
