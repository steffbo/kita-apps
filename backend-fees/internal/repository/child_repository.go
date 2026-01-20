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

// List retrieves children with optional filtering.
func (r *PostgresChildRepository) List(ctx context.Context, activeOnly bool, search string, offset, limit int) ([]domain.Child, int64, error) {
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

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, member_number, first_name, last_name, birth_date, entry_date,
		       street, house_number, postal_code, city, is_active, created_at, updated_at
		%s
		ORDER BY last_name, first_name
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &children, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return children, total, nil
}

// GetByID retrieves a child by ID.
func (r *PostgresChildRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	var child domain.Child
	err := r.db.GetContext(ctx, &child, `
		SELECT id, member_number, first_name, last_name, birth_date, entry_date,
		       street, house_number, postal_code, city, is_active, created_at, updated_at
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
		SELECT id, member_number, first_name, last_name, birth_date, entry_date,
		       street, house_number, postal_code, city, is_active, created_at, updated_at
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
		INSERT INTO fees.children (id, member_number, first_name, last_name, birth_date, entry_date,
		                           street, house_number, postal_code, city, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, child.ID, child.MemberNumber, child.FirstName, child.LastName, child.BirthDate, child.EntryDate,
		child.Street, child.HouseNumber, child.PostalCode, child.City, child.IsActive, child.CreatedAt, child.UpdatedAt)
	return err
}

// Update updates an existing child.
func (r *PostgresChildRepository) Update(ctx context.Context, child *domain.Child) error {
	child.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.children
		SET first_name = $2, last_name = $3, birth_date = $4, entry_date = $5,
		    street = $6, house_number = $7, postal_code = $8, city = $9,
		    is_active = $10, updated_at = $11
		WHERE id = $1
	`, child.ID, child.FirstName, child.LastName, child.BirthDate, child.EntryDate,
		child.Street, child.HouseNumber, child.PostalCode, child.City, child.IsActive, child.UpdatedAt)
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
		       p.street, p.house_number, p.postal_code, p.city,
		       p.annual_household_income, p.created_at, p.updated_at
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
