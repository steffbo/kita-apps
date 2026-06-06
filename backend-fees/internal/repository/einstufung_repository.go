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

const einstufungColumns = `id, child_id, household_id, year, valid_from, valid_until,
	source_einstufung_id, change_date, effective_from_month,
	income_calculation, annual_net_income,
	highest_rate_voluntary, care_hours_per_week, care_type, children_count,
	monthly_childcare_fee, monthly_food_fee, annual_membership_fee,
	fee_rule, discount_percent, discount_factor, base_fee,
	notes, created_at, updated_at`

// PostgresEinstufungRepository is the PostgreSQL implementation of EinstufungRepository.
type PostgresEinstufungRepository struct {
	db *sqlx.DB
}

// NewPostgresEinstufungRepository creates a new PostgreSQL einstufung repository.
func NewPostgresEinstufungRepository(db *sqlx.DB) *PostgresEinstufungRepository {
	return &PostgresEinstufungRepository{db: db}
}

// Create creates a new Einstufung record.
func (r *PostgresEinstufungRepository) Create(ctx context.Context, e *domain.Einstufung) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.EffectiveFromMonth.IsZero() {
		e.EffectiveFromMonth = e.ValidFrom
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.einstufungen (
			id, child_id, household_id, year, valid_from, valid_until,
			source_einstufung_id, change_date, effective_from_month,
			income_calculation, annual_net_income,
			highest_rate_voluntary, care_hours_per_week, care_type, children_count,
			monthly_childcare_fee, monthly_food_fee, annual_membership_fee,
			fee_rule, discount_percent, discount_factor, base_fee,
			notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`,
		e.ID, e.ChildID, e.HouseholdID, e.Year, e.ValidFrom, e.ValidUntil,
		e.SourceEinstufungID, e.ChangeDate, e.EffectiveFromMonth,
		e.IncomeCalculation, e.AnnualNetIncome,
		e.HighestRateVoluntary, e.CareHoursPerWeek, e.CareType, e.ChildrenCount,
		e.MonthlyChildcareFee, e.MonthlyFoodFee, e.AnnualMembershipFee,
		e.FeeRule, e.DiscountPercent, e.DiscountFactor, e.BaseFee,
		e.Notes, e.CreatedAt, e.UpdatedAt,
	)
	return err
}

// CreateFollowUp creates a follow-up Einstufung and closes the source period atomically.
func (r *PostgresEinstufungRepository) CreateFollowUp(ctx context.Context, sourceID uuid.UUID, sourceValidUntil time.Time, e *domain.Einstufung) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.EffectiveFromMonth.IsZero() {
		e.EffectiveFromMonth = e.ValidFrom
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	if _, err := tx.ExecContext(ctx, `
		UPDATE fees.einstufungen
		SET valid_until = $2, updated_at = $3
		WHERE id = $1
	`, sourceID, sourceValidUntil, now); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO fees.einstufungen (
			id, child_id, household_id, year, valid_from, valid_until,
			source_einstufung_id, change_date, effective_from_month,
			income_calculation, annual_net_income,
			highest_rate_voluntary, care_hours_per_week, care_type, children_count,
			monthly_childcare_fee, monthly_food_fee, annual_membership_fee,
			fee_rule, discount_percent, discount_factor, base_fee,
			notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`,
		e.ID, e.ChildID, e.HouseholdID, e.Year, e.ValidFrom, e.ValidUntil,
		e.SourceEinstufungID, e.ChangeDate, e.EffectiveFromMonth,
		e.IncomeCalculation, e.AnnualNetIncome,
		e.HighestRateVoluntary, e.CareHoursPerWeek, e.CareType, e.ChildrenCount,
		e.MonthlyChildcareFee, e.MonthlyFoodFee, e.AnnualMembershipFee,
		e.FeeRule, e.DiscountPercent, e.DiscountFactor, e.BaseFee,
		e.Notes, e.CreatedAt, e.UpdatedAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID retrieves an Einstufung by ID.
func (r *PostgresEinstufungRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Einstufung, error) {
	var e domain.Einstufung
	err := r.db.GetContext(ctx, &e, fmt.Sprintf(`
		SELECT %s FROM fees.einstufungen WHERE id = $1
	`, einstufungColumns), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

// GetByChildAndYear retrieves the Einstufung for a specific child and year.
func (r *PostgresEinstufungRepository) GetByChildAndYear(ctx context.Context, childID uuid.UUID, year int) (*domain.Einstufung, error) {
	var e domain.Einstufung
	err := r.db.GetContext(ctx, &e, fmt.Sprintf(`
		SELECT %s FROM fees.einstufungen
		WHERE child_id = $1 AND year = $2
		ORDER BY effective_from_month DESC, created_at DESC
		LIMIT 1
	`, einstufungColumns), childID, year)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

// Update updates an existing Einstufung.
func (r *PostgresEinstufungRepository) Update(ctx context.Context, e *domain.Einstufung) error {
	e.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.einstufungen SET
			child_id = $2, household_id = $3, year = $4, valid_from = $5, valid_until = $6,
			source_einstufung_id = $7, change_date = $8, effective_from_month = $9,
			income_calculation = $10, annual_net_income = $11,
			highest_rate_voluntary = $12, care_hours_per_week = $13, care_type = $14, children_count = $15,
			monthly_childcare_fee = $16, monthly_food_fee = $17, annual_membership_fee = $18,
			fee_rule = $19, discount_percent = $20, discount_factor = $21, base_fee = $22,
			notes = $23, updated_at = $24
		WHERE id = $1
	`,
		e.ID, e.ChildID, e.HouseholdID, e.Year, e.ValidFrom, e.ValidUntil,
		e.SourceEinstufungID, e.ChangeDate, e.EffectiveFromMonth,
		e.IncomeCalculation, e.AnnualNetIncome,
		e.HighestRateVoluntary, e.CareHoursPerWeek, e.CareType, e.ChildrenCount,
		e.MonthlyChildcareFee, e.MonthlyFoodFee, e.AnnualMembershipFee,
		e.FeeRule, e.DiscountPercent, e.DiscountFactor, e.BaseFee,
		e.Notes, e.UpdatedAt,
	)
	return err
}

// Delete deletes an Einstufung.
func (r *PostgresEinstufungRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.einstufungen WHERE id = $1`, id)
	return err
}

// ListByHousehold retrieves all Einstufungen for a household, ordered by year desc.
func (r *PostgresEinstufungRepository) ListByHousehold(ctx context.Context, householdID uuid.UUID) ([]domain.Einstufung, error) {
	var results []domain.Einstufung
	err := r.db.SelectContext(ctx, &results, fmt.Sprintf(`
		SELECT %s FROM fees.einstufungen
		WHERE household_id = $1
		ORDER BY year DESC, effective_from_month DESC, created_at DESC
	`, einstufungColumns), householdID)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ListByYear retrieves all Einstufungen for a given year with pagination.
func (r *PostgresEinstufungRepository) ListByYear(ctx context.Context, year int, offset, limit int) ([]domain.Einstufung, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM fees.einstufungen WHERE year = $1
	`, year)
	if err != nil {
		return nil, 0, err
	}

	var results []domain.Einstufung
	err = r.db.SelectContext(ctx, &results, fmt.Sprintf(`
		SELECT %s FROM fees.einstufungen
		WHERE year = $1
		ORDER BY effective_from_month DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`, einstufungColumns), year, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// GetLatestForChild retrieves the most recent Einstufung for a child.
func (r *PostgresEinstufungRepository) GetLatestForChild(ctx context.Context, childID uuid.UUID) (*domain.Einstufung, error) {
	var e domain.Einstufung
	err := r.db.GetContext(ctx, &e, fmt.Sprintf(`
		SELECT %s FROM fees.einstufungen
		WHERE child_id = $1
		ORDER BY effective_from_month DESC, year DESC, created_at DESC
		LIMIT 1
	`, einstufungColumns), childID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}
