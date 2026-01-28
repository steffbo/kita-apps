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

// PostgresHouseholdRepository is the PostgreSQL implementation of HouseholdRepository.
type PostgresHouseholdRepository struct {
	db *sqlx.DB
}

// NewPostgresHouseholdRepository creates a new PostgreSQL household repository.
func NewPostgresHouseholdRepository(db *sqlx.DB) *PostgresHouseholdRepository {
	return &PostgresHouseholdRepository{db: db}
}

// List retrieves households with optional search filtering and sorting.
func (r *PostgresHouseholdRepository) List(ctx context.Context, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Household, int64, error) {
	var households []domain.Household
	var total int64

	baseQuery := `FROM fees.households WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
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
	orderClause := getHouseholdSortOrder(sortBy, sortDir)

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, name, annual_household_income, income_status, children_count_for_fees, created_at, updated_at
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &households, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return households, total, nil
}

// getHouseholdSortOrder returns a safe ORDER BY clause for households.
func getHouseholdSortOrder(sortBy, sortDir string) string {
	dir := "ASC"
	if sortDir == "desc" {
		dir = "DESC"
	}

	allowedColumns := map[string]string{
		"name":   "name",
		"income": "annual_household_income",
	}

	if col, ok := allowedColumns[sortBy]; ok {
		return fmt.Sprintf("%s %s NULLS LAST", col, dir)
	}

	// Default sort by name
	return fmt.Sprintf("name %s", dir)
}

// GetByID retrieves a household by ID.
func (r *PostgresHouseholdRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Household, error) {
	var household domain.Household
	err := r.db.GetContext(ctx, &household, `
		SELECT id, name, annual_household_income, income_status, children_count_for_fees, created_at, updated_at
		FROM fees.households
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("household not found")
		}
		return nil, err
	}
	return &household, nil
}

// GetByIDs retrieves households by their IDs.
func (r *PostgresHouseholdRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Household, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.Household), nil
	}

	query, args, err := sqlx.In(`
		SELECT id, name, annual_household_income, income_status, children_count_for_fees, created_at, updated_at
		FROM fees.households
		WHERE id IN (?)
	`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var households []domain.Household
	if err := r.db.SelectContext(ctx, &households, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*domain.Household, len(households))
	for i := range households {
		result[households[i].ID] = &households[i]
	}
	return result, nil
}

// Create creates a new household.
func (r *PostgresHouseholdRepository) Create(ctx context.Context, household *domain.Household) error {
	if household.ID == uuid.Nil {
		household.ID = uuid.New()
	}
	now := time.Now()
	household.CreatedAt = now
	household.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.households (id, name, annual_household_income, income_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, household.ID, household.Name, household.AnnualHouseholdIncome, household.IncomeStatus,
		household.CreatedAt, household.UpdatedAt)
	return err
}

// Update updates an existing household.
func (r *PostgresHouseholdRepository) Update(ctx context.Context, household *domain.Household) error {
	household.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.households
		SET name = $2, annual_household_income = $3, income_status = $4, children_count_for_fees = $5, updated_at = $6
		WHERE id = $1
	`, household.ID, household.Name, household.AnnualHouseholdIncome, household.IncomeStatus,
		household.ChildrenCountForFees, household.UpdatedAt)
	return err
}

// Delete deletes a household.
func (r *PostgresHouseholdRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.households WHERE id = $1`, id)
	return err
}

// GetParents retrieves all parents linked to a household.
func (r *PostgresHouseholdRepository) GetParents(ctx context.Context, householdID uuid.UUID) ([]domain.Parent, error) {
	var parents []domain.Parent
	err := r.db.SelectContext(ctx, &parents, `
		SELECT id, household_id, member_id, first_name, last_name, birth_date, email, phone,
		       street, street_no, postal_code, city,
		       annual_household_income, income_status, created_at, updated_at
		FROM fees.parents
		WHERE household_id = $1
		ORDER BY last_name, first_name
	`, householdID)
	if err != nil {
		return nil, err
	}
	return parents, nil
}

// GetChildren retrieves all children linked to a household.
func (r *PostgresHouseholdRepository) GetChildren(ctx context.Context, householdID uuid.UUID) ([]domain.Child, error) {
	var children []domain.Child
	err := r.db.SelectContext(ctx, &children, `
		SELECT id, household_id, member_number, first_name, last_name, birth_date, entry_date, exit_date,
		       street, street_no, postal_code, city,
		       legal_hours, legal_hours_until, care_hours,
		       is_active, created_at, updated_at
		FROM fees.children
		WHERE household_id = $1
		ORDER BY last_name, first_name
	`, householdID)
	if err != nil {
		return nil, err
	}
	return children, nil
}

// GetWithMembers retrieves a household with all its parents and children.
func (r *PostgresHouseholdRepository) GetWithMembers(ctx context.Context, id uuid.UUID) (*domain.Household, error) {
	household, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	parents, err := r.GetParents(ctx, id)
	if err != nil {
		return nil, err
	}
	household.Parents = parents

	children, err := r.GetChildren(ctx, id)
	if err != nil {
		return nil, err
	}
	household.Children = children

	return household, nil
}
