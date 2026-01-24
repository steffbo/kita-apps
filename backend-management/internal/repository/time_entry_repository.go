package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// PostgresTimeEntryRepository is the PostgreSQL implementation of TimeEntryRepository.
type PostgresTimeEntryRepository struct {
	db *sqlx.DB
}

// NewPostgresTimeEntryRepository creates a new PostgreSQL time entry repository.
func NewPostgresTimeEntryRepository(db *sqlx.DB) *PostgresTimeEntryRepository {
	return &PostgresTimeEntryRepository{db: db}
}

type timeEntryRow struct {
	ID                        int64                `db:"id"`
	EmployeeID                int64                `db:"employee_id"`
	Date                      sql.NullTime         `db:"date"`
	ClockIn                   sql.NullTime         `db:"clock_in"`
	ClockOut                  sql.NullTime         `db:"clock_out"`
	BreakMinutes              int                  `db:"break_minutes"`
	EntryType                 domain.TimeEntryType `db:"entry_type"`
	Notes                     *string              `db:"notes"`
	EditedBy                  *int64               `db:"edited_by"`
	EditedAt                  sql.NullTime         `db:"edited_at"`
	EditReason                *string              `db:"edit_reason"`
	CreatedAt                 sql.NullTime         `db:"created_at"`
	EmployeeEmail             string               `db:"employee_email"`
	EmployeeFirstName         string               `db:"employee_first_name"`
	EmployeeLastName          string               `db:"employee_last_name"`
	EmployeeRole              string               `db:"employee_role"`
	EmployeeWeeklyHours       float64              `db:"employee_weekly_hours"`
	EmployeeVacationDays      int                  `db:"employee_vacation_days_per_year"`
	EmployeeRemainingVacation float64              `db:"employee_remaining_vacation_days"`
	EmployeeOvertimeBalance   float64              `db:"employee_overtime_balance"`
	EmployeeActive            bool                 `db:"employee_active"`
	EmployeeCreatedAt         sql.NullTime         `db:"employee_created_at"`
	EmployeeUpdatedAt         sql.NullTime         `db:"employee_updated_at"`
}

func mapTimeEntry(row timeEntryRow) domain.TimeEntry {
	entry := domain.TimeEntry{
		ID:           row.ID,
		EmployeeID:   row.EmployeeID,
		BreakMinutes: row.BreakMinutes,
		EntryType:    row.EntryType,
		Notes:        row.Notes,
		EditedBy:     row.EditedBy,
		EditReason:   row.EditReason,
	}

	if row.Date.Valid {
		entry.Date = row.Date.Time
	}
	if row.ClockIn.Valid {
		entry.ClockIn = row.ClockIn.Time
	}
	if row.ClockOut.Valid {
		entry.ClockOut = &row.ClockOut.Time
	}
	if row.EditedAt.Valid {
		entry.EditedAt = &row.EditedAt.Time
	}
	if row.CreatedAt.Valid {
		entry.CreatedAt = row.CreatedAt.Time
	}

	entry.Employee = &domain.Employee{
		ID:                    row.EmployeeID,
		Email:                 row.EmployeeEmail,
		FirstName:             row.EmployeeFirstName,
		LastName:              row.EmployeeLastName,
		Role:                  domain.EmployeeRole(row.EmployeeRole),
		WeeklyHours:           row.EmployeeWeeklyHours,
		VacationDaysPerYear:   row.EmployeeVacationDays,
		RemainingVacationDays: row.EmployeeRemainingVacation,
		OvertimeBalance:       row.EmployeeOvertimeBalance,
		Active:                row.EmployeeActive,
	}
	if row.EmployeeCreatedAt.Valid {
		entry.Employee.CreatedAt = row.EmployeeCreatedAt.Time
	}
	if row.EmployeeUpdatedAt.Valid {
		entry.Employee.UpdatedAt = row.EmployeeUpdatedAt.Time
	}

	return entry
}

// List retrieves time entries with optional employee filter.
func (r *PostgresTimeEntryRepository) List(ctx context.Context, startDate, endDate time.Time, employeeID *int64) ([]domain.TimeEntry, error) {
	baseQuery := `
		SELECT te.id, te.employee_id, te.date, te.clock_in, te.clock_out,
		       COALESCE(te.break_minutes, 0) AS break_minutes,
		       te.entry_type, te.notes, te.edited_by, te.edited_at, te.edit_reason,
		       te.created_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at
		FROM time_entries te
		JOIN employees e ON e.id = te.employee_id
		WHERE te.date BETWEEN $1 AND $2`

	args := []interface{}{startDate, endDate}
	if employeeID != nil {
		baseQuery += fmt.Sprintf(" AND te.employee_id = $%d", len(args)+1)
		args = append(args, *employeeID)
	}

	baseQuery += " ORDER BY te.date, te.clock_in"

	var rows []timeEntryRow
	if err := r.db.SelectContext(ctx, &rows, baseQuery, args...); err != nil {
		return nil, err
	}

	entries := make([]domain.TimeEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, mapTimeEntry(row))
	}
	return entries, nil
}

// GetByID retrieves a time entry by ID.
func (r *PostgresTimeEntryRepository) GetByID(ctx context.Context, id int64) (*domain.TimeEntry, error) {
	var row timeEntryRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT te.id, te.employee_id, te.date, te.clock_in, te.clock_out,
		       COALESCE(te.break_minutes, 0) AS break_minutes,
		       te.entry_type, te.notes, te.edited_by, te.edited_at, te.edit_reason,
		       te.created_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at
		FROM time_entries te
		JOIN employees e ON e.id = te.employee_id
		WHERE te.id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	entry := mapTimeEntry(row)
	return &entry, nil
}

// ListOpenByEmployeeID retrieves open time entries for an employee.
func (r *PostgresTimeEntryRepository) ListOpenByEmployeeID(ctx context.Context, employeeID int64) ([]domain.TimeEntry, error) {
	var rows []timeEntryRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT te.id, te.employee_id, te.date, te.clock_in, te.clock_out,
		       COALESCE(te.break_minutes, 0) AS break_minutes,
		       te.entry_type, te.notes, te.edited_by, te.edited_at, te.edit_reason,
		       te.created_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at
		FROM time_entries te
		JOIN employees e ON e.id = te.employee_id
		WHERE te.employee_id = $1 AND te.clock_out IS NULL
		ORDER BY te.clock_in DESC
	`, employeeID); err != nil {
		return nil, err
	}

	entries := make([]domain.TimeEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, mapTimeEntry(row))
	}
	return entries, nil
}

// Create inserts a time entry.
func (r *PostgresTimeEntryRepository) Create(ctx context.Context, entry *domain.TimeEntry) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO time_entries (
			employee_id, date, clock_in, clock_out, break_minutes, entry_type,
			notes, edited_by, edited_at, edit_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`, entry.EmployeeID, entry.Date, entry.ClockIn, entry.ClockOut,
		entry.BreakMinutes, entry.EntryType, entry.Notes, entry.EditedBy,
		entry.EditedAt, entry.EditReason,
	).Scan(&entry.ID, &entry.CreatedAt)
}

// Update updates a time entry and returns the updated record.
func (r *PostgresTimeEntryRepository) Update(ctx context.Context, entry *domain.TimeEntry) (*domain.TimeEntry, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE time_entries
		SET clock_in = $2,
		    clock_out = $3,
		    break_minutes = $4,
		    entry_type = $5,
		    notes = $6,
		    edited_by = $7,
		    edited_at = $8,
		    edit_reason = $9
		WHERE id = $1
	`, entry.ID, entry.ClockIn, entry.ClockOut, entry.BreakMinutes,
		entry.EntryType, entry.Notes, entry.EditedBy, entry.EditedAt, entry.EditReason)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, entry.ID)
}

// Delete deletes a time entry by ID.
func (r *PostgresTimeEntryRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM time_entries WHERE id = $1`, id)
	return err
}
