package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresFeeScheduleRepository stores fee regulation versions.
type PostgresFeeScheduleRepository struct {
	db *sqlx.DB
}

// NewPostgresFeeScheduleRepository creates a new fee schedule repository.
func NewPostgresFeeScheduleRepository(db *sqlx.DB) *PostgresFeeScheduleRepository {
	return &PostgresFeeScheduleRepository{db: db}
}

const feeScheduleColumns = `id, valid_from, name, config, created_at, updated_at`

// List returns all versions ordered by valid_from.
func (r *PostgresFeeScheduleRepository) List(ctx context.Context) (domain.FeeSchedules, error) {
	var schedules domain.FeeSchedules
	err := conn(ctx, r.db).SelectContext(ctx, &schedules, `
		SELECT `+feeScheduleColumns+` FROM fees.fee_schedules ORDER BY valid_from
	`)
	return schedules, err
}

// GetByID returns one version.
func (r *PostgresFeeScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeSchedule, error) {
	var schedule domain.FeeSchedule
	err := conn(ctx, r.db).GetContext(ctx, &schedule, `
		SELECT `+feeScheduleColumns+` FROM fees.fee_schedules WHERE id = $1
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &schedule, err
}

// GetAt returns the version valid at date.
func (r *PostgresFeeScheduleRepository) GetAt(ctx context.Context, date time.Time) (*domain.FeeSchedule, error) {
	var schedule domain.FeeSchedule
	err := conn(ctx, r.db).GetContext(ctx, &schedule, `
		SELECT `+feeScheduleColumns+` FROM fees.fee_schedules
		WHERE valid_from <= $1 ORDER BY valid_from DESC LIMIT 1
	`, date.Format("2006-01-02"))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNoFeeSchedule
	}
	return &schedule, err
}

// Create inserts a version.
func (r *PostgresFeeScheduleRepository) Create(ctx context.Context, schedule *domain.FeeSchedule) error {
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.fee_schedules (id, valid_from, name, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, schedule.ID, schedule.ValidFrom, schedule.Name, schedule.Config, schedule.CreatedAt, schedule.UpdatedAt)
	return mapUniqueViolation(err)
}

// Update changes a version.
func (r *PostgresFeeScheduleRepository) Update(ctx context.Context, schedule *domain.FeeSchedule) error {
	schedule.UpdatedAt = time.Now()
	result, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.fee_schedules SET valid_from = $2, name = $3, config = $4, updated_at = $5 WHERE id = $1
	`, schedule.ID, schedule.ValidFrom, schedule.Name, schedule.Config, schedule.UpdatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return requireRow(result)
}

// Delete removes a version.
func (r *PostgresFeeScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.fee_schedules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return requireRow(result)
}

// mapUniqueViolation turns a unique constraint violation into ErrDuplicate.
func mapUniqueViolation(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func requireRow(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
