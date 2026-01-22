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

// List retrieves parents with optional search filtering.
func (r *PostgresParentRepository) List(ctx context.Context, search string, offset, limit int) ([]domain.Parent, int64, error) {
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

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, first_name, last_name, birth_date, email, phone,
		       street, street_no, postal_code, city,
		       annual_household_income, income_status, created_at, updated_at
		%s
		ORDER BY last_name, first_name
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &parents, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return parents, total, nil
}

// GetByID retrieves a parent by ID.
func (r *PostgresParentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	var parent domain.Parent
	err := r.db.GetContext(ctx, &parent, `
		SELECT id, first_name, last_name, birth_date, email, phone,
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

// Create creates a new parent.
func (r *PostgresParentRepository) Create(ctx context.Context, parent *domain.Parent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.parents (id, first_name, last_name, birth_date, email, phone,
		                          street, street_no, postal_code, city,
		                          annual_household_income, income_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, parent.ID, parent.FirstName, parent.LastName, parent.BirthDate, parent.Email, parent.Phone,
		parent.Street, parent.StreetNo, parent.PostalCode, parent.City,
		parent.AnnualHouseholdIncome, parent.IncomeStatus, parent.CreatedAt, parent.UpdatedAt)
	return err
}

// Update updates an existing parent.
func (r *PostgresParentRepository) Update(ctx context.Context, parent *domain.Parent) error {
	parent.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.parents
		SET first_name = $2, last_name = $3, birth_date = $4, email = $5, phone = $6,
		    street = $7, street_no = $8, postal_code = $9, city = $10,
		    annual_household_income = $11, income_status = $12, updated_at = $13
		WHERE id = $1
	`, parent.ID, parent.FirstName, parent.LastName, parent.BirthDate, parent.Email, parent.Phone,
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
