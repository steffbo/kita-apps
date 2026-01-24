package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// PostgresScheduleRepository is the PostgreSQL implementation of ScheduleRepository.
type PostgresScheduleRepository struct {
	db *sqlx.DB
}

// NewPostgresScheduleRepository creates a new PostgreSQL schedule repository.
func NewPostgresScheduleRepository(db *sqlx.DB) *PostgresScheduleRepository {
	return &PostgresScheduleRepository{db: db}
}

type scheduleEntryRow struct {
	ID                        int64                    `db:"id"`
	EmployeeID                int64                    `db:"employee_id"`
	Date                      sql.NullTime             `db:"date"`
	StartTime                 sql.NullTime             `db:"start_time"`
	EndTime                   sql.NullTime             `db:"end_time"`
	BreakMinutes              int                      `db:"break_minutes"`
	GroupID                   *int64                   `db:"group_id"`
	EntryType                 domain.ScheduleEntryType `db:"entry_type"`
	Notes                     *string                  `db:"notes"`
	CreatedAt                 sql.NullTime             `db:"created_at"`
	UpdatedAt                 sql.NullTime             `db:"updated_at"`
	EmployeeEmail             string                   `db:"employee_email"`
	EmployeeFirstName         string                   `db:"employee_first_name"`
	EmployeeLastName          string                   `db:"employee_last_name"`
	EmployeeRole              string                   `db:"employee_role"`
	EmployeeWeeklyHours       float64                  `db:"employee_weekly_hours"`
	EmployeeVacationDays      int                      `db:"employee_vacation_days_per_year"`
	EmployeeRemainingVacation float64                  `db:"employee_remaining_vacation_days"`
	EmployeeOvertimeBalance   float64                  `db:"employee_overtime_balance"`
	EmployeeActive            bool                     `db:"employee_active"`
	EmployeeCreatedAt         sql.NullTime             `db:"employee_created_at"`
	EmployeeUpdatedAt         sql.NullTime             `db:"employee_updated_at"`
	GroupName                 *string                  `db:"group_name"`
	GroupDescription          *string                  `db:"group_description"`
	GroupColor                *string                  `db:"group_color"`
	GroupCreatedAt            sql.NullTime             `db:"group_created_at"`
	GroupUpdatedAt            sql.NullTime             `db:"group_updated_at"`
}

func mapScheduleEntry(row scheduleEntryRow) domain.ScheduleEntry {
	entry := domain.ScheduleEntry{
		ID:           row.ID,
		EmployeeID:   row.EmployeeID,
		BreakMinutes: row.BreakMinutes,
		GroupID:      row.GroupID,
		EntryType:    row.EntryType,
		Notes:        row.Notes,
	}

	if row.Date.Valid {
		entry.Date = row.Date.Time
	}
	if row.StartTime.Valid {
		entry.StartTime = &row.StartTime.Time
	}
	if row.EndTime.Valid {
		entry.EndTime = &row.EndTime.Time
	}
	if row.CreatedAt.Valid {
		entry.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		entry.UpdatedAt = row.UpdatedAt.Time
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

	if row.GroupName != nil && row.GroupID != nil {
		group := &domain.Group{
			ID:    *row.GroupID,
			Name:  *row.GroupName,
			Color: "",
		}
		if row.GroupDescription != nil {
			group.Description = row.GroupDescription
		}
		if row.GroupColor != nil {
			group.Color = *row.GroupColor
		}
		if row.GroupCreatedAt.Valid {
			group.CreatedAt = row.GroupCreatedAt.Time
		}
		if row.GroupUpdatedAt.Valid {
			group.UpdatedAt = row.GroupUpdatedAt.Time
		}
		entry.Group = group
	}

	return entry
}

// List retrieves schedule entries for the given filters.
func (r *PostgresScheduleRepository) List(ctx context.Context, startDate, endDate time.Time, employeeID, groupID *int64) ([]domain.ScheduleEntry, error) {
	baseQuery := `
		SELECT se.id, se.employee_id, se.date, se.start_time, se.end_time,
		       COALESCE(se.break_minutes, 0) AS break_minutes,
		       se.group_id, se.entry_type, se.notes, se.created_at, se.updated_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at,
		       g.name AS group_name, g.description AS group_description, g.color AS group_color,
		       g.created_at AS group_created_at, g.updated_at AS group_updated_at
		FROM schedule_entries se
		JOIN employees e ON e.id = se.employee_id
		LEFT JOIN groups g ON g.id = se.group_id
		WHERE se.date BETWEEN $1 AND $2`

	args := []interface{}{startDate, endDate}

	if employeeID != nil {
		baseQuery += fmt.Sprintf(" AND se.employee_id = $%d", len(args)+1)
		args = append(args, *employeeID)
	}
	if groupID != nil {
		baseQuery += fmt.Sprintf(" AND se.group_id = $%d", len(args)+1)
		args = append(args, *groupID)
	}

	baseQuery += " ORDER BY se.date, se.start_time"

	var rows []scheduleEntryRow
	if err := r.db.SelectContext(ctx, &rows, baseQuery, args...); err != nil {
		return nil, err
	}

	entries := make([]domain.ScheduleEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, mapScheduleEntry(row))
	}
	return entries, nil
}

// GetByID retrieves a schedule entry by ID including relations.
func (r *PostgresScheduleRepository) GetByID(ctx context.Context, id int64) (*domain.ScheduleEntry, error) {
	var row scheduleEntryRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT se.id, se.employee_id, se.date, se.start_time, se.end_time,
		       COALESCE(se.break_minutes, 0) AS break_minutes,
		       se.group_id, se.entry_type, se.notes, se.created_at, se.updated_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at,
		       g.name AS group_name, g.description AS group_description, g.color AS group_color,
		       g.created_at AS group_created_at, g.updated_at AS group_updated_at
		FROM schedule_entries se
		JOIN employees e ON e.id = se.employee_id
		LEFT JOIN groups g ON g.id = se.group_id
		WHERE se.id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	entry := mapScheduleEntry(row)
	return &entry, nil
}

// Create inserts a schedule entry.
func (r *PostgresScheduleRepository) Create(ctx context.Context, entry *domain.ScheduleEntry) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO schedule_entries (
			employee_id, date, start_time, end_time, break_minutes, group_id,
			entry_type, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, entry.EmployeeID, entry.Date, entry.StartTime, entry.EndTime,
		entry.BreakMinutes, entry.GroupID, entry.EntryType, entry.Notes,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)
}

// Update updates a schedule entry and returns the updated record.
func (r *PostgresScheduleRepository) Update(ctx context.Context, entry *domain.ScheduleEntry) (*domain.ScheduleEntry, error) {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE schedule_entries
		SET date = $2,
		    start_time = $3,
		    end_time = $4,
		    break_minutes = $5,
		    group_id = $6,
		    entry_type = $7,
		    notes = $8
		WHERE id = $1
	`, entry.ID, entry.Date, entry.StartTime, entry.EndTime, entry.BreakMinutes,
		entry.GroupID, entry.EntryType, entry.Notes); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, entry.ID)
}

// Delete deletes a schedule entry by ID.
func (r *PostgresScheduleRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM schedule_entries WHERE id = $1`, id)
	return err
}
