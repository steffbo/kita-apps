package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// PostgresSpecialDayRepository is the PostgreSQL implementation of SpecialDayRepository.
type PostgresSpecialDayRepository struct {
	db *sqlx.DB
}

// NewPostgresSpecialDayRepository creates a new PostgreSQL special day repository.
func NewPostgresSpecialDayRepository(db *sqlx.DB) *PostgresSpecialDayRepository {
	return &PostgresSpecialDayRepository{db: db}
}

// List retrieves special days between dates.
func (r *PostgresSpecialDayRepository) List(ctx context.Context, startDate, endDate time.Time) ([]domain.SpecialDay, error) {
	var days []domain.SpecialDay
	if err := r.db.SelectContext(ctx, &days, `
		SELECT id, date, end_date, name, day_type, affects_all, notes, created_at
		FROM special_days
		WHERE date BETWEEN $1 AND $2
		ORDER BY date
	`, startDate, endDate); err != nil {
		return nil, err
	}
	return days, nil
}

// ListByType retrieves special days of a specific type between dates.
func (r *PostgresSpecialDayRepository) ListByType(ctx context.Context, startDate, endDate time.Time, dayType domain.SpecialDayType) ([]domain.SpecialDay, error) {
	var days []domain.SpecialDay
	if err := r.db.SelectContext(ctx, &days, `
		SELECT id, date, end_date, name, day_type, affects_all, notes, created_at
		FROM special_days
		WHERE date BETWEEN $1 AND $2 AND day_type = $3
		ORDER BY date
	`, startDate, endDate, dayType); err != nil {
		return nil, err
	}
	return days, nil
}

// GetByID retrieves a special day by ID.
func (r *PostgresSpecialDayRepository) GetByID(ctx context.Context, id int64) (*domain.SpecialDay, error) {
	var day domain.SpecialDay
	if err := r.db.GetContext(ctx, &day, `
		SELECT id, date, end_date, name, day_type, affects_all, notes, created_at
		FROM special_days
		WHERE id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &day, nil
}

// Create inserts a special day.
func (r *PostgresSpecialDayRepository) Create(ctx context.Context, day *domain.SpecialDay) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO special_days (date, end_date, name, day_type, affects_all, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, day.Date, day.EndDate, day.Name, day.DayType, day.AffectsAll, day.Notes).
		Scan(&day.ID, &day.CreatedAt)
}

// Update updates a special day and returns the updated record.
func (r *PostgresSpecialDayRepository) Update(ctx context.Context, day *domain.SpecialDay) (*domain.SpecialDay, error) {
	var updated domain.SpecialDay
	if err := r.db.GetContext(ctx, &updated, `
		UPDATE special_days
		SET date = $2,
		    end_date = $3,
		    name = $4,
		    day_type = $5,
		    affects_all = $6,
		    notes = $7
		WHERE id = $1
		RETURNING id, date, end_date, name, day_type, affects_all, notes, created_at
	`, day.ID, day.Date, day.EndDate, day.Name, day.DayType, day.AffectsAll, day.Notes); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Delete deletes a special day by ID.
func (r *PostgresSpecialDayRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM special_days WHERE id = $1`, id)
	return err
}
