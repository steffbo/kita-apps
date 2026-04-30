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
	ShiftKind                 domain.ShiftKind         `db:"shift_kind"`
	Notes                     *string                  `db:"notes"`
	CreatedAt                 sql.NullTime             `db:"created_at"`
	UpdatedAt                 sql.NullTime             `db:"updated_at"`
	EmployeeEmail             string                   `db:"employee_email"`
	EmployeeFirstName         string                   `db:"employee_first_name"`
	EmployeeLastName          string                   `db:"employee_last_name"`
	EmployeeNickname          *string                  `db:"employee_nickname"`
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

type scheduleEntrySegmentRow struct {
	ID               int64        `db:"id"`
	ScheduleEntryID  int64        `db:"schedule_entry_id"`
	GroupID          int64        `db:"group_id"`
	StartTime        sql.NullTime `db:"start_time"`
	EndTime          sql.NullTime `db:"end_time"`
	Notes            *string      `db:"notes"`
	SortOrder        int          `db:"sort_order"`
	CreatedAt        sql.NullTime `db:"created_at"`
	UpdatedAt        sql.NullTime `db:"updated_at"`
	GroupName        string       `db:"group_name"`
	GroupDescription *string      `db:"group_description"`
	GroupColor       *string      `db:"group_color"`
	GroupCreatedAt   sql.NullTime `db:"group_created_at"`
	GroupUpdatedAt   sql.NullTime `db:"group_updated_at"`
}

type scheduleRequestRow struct {
	ID                        int64                        `db:"id"`
	EmployeeID                int64                        `db:"employee_id"`
	Date                      sql.NullTime                 `db:"date"`
	StartTime                 sql.NullTime                 `db:"start_time"`
	EndTime                   sql.NullTime                 `db:"end_time"`
	RequestType               domain.ScheduleRequestType   `db:"request_type"`
	Text                      string                       `db:"text"`
	Status                    domain.ScheduleRequestStatus `db:"status"`
	CreatedAt                 sql.NullTime                 `db:"created_at"`
	UpdatedAt                 sql.NullTime                 `db:"updated_at"`
	EmployeeEmail             string                       `db:"employee_email"`
	EmployeeFirstName         string                       `db:"employee_first_name"`
	EmployeeLastName          string                       `db:"employee_last_name"`
	EmployeeNickname          *string                      `db:"employee_nickname"`
	EmployeeRole              string                       `db:"employee_role"`
	EmployeeWeeklyHours       float64                      `db:"employee_weekly_hours"`
	EmployeeVacationDays      int                          `db:"employee_vacation_days_per_year"`
	EmployeeRemainingVacation float64                      `db:"employee_remaining_vacation_days"`
	EmployeeOvertimeBalance   float64                      `db:"employee_overtime_balance"`
	EmployeeActive            bool                         `db:"employee_active"`
	EmployeeCreatedAt         sql.NullTime                 `db:"employee_created_at"`
	EmployeeUpdatedAt         sql.NullTime                 `db:"employee_updated_at"`
}

func mapScheduleEntry(row scheduleEntryRow) domain.ScheduleEntry {
	entry := domain.ScheduleEntry{
		ID:           row.ID,
		EmployeeID:   row.EmployeeID,
		BreakMinutes: row.BreakMinutes,
		GroupID:      row.GroupID,
		EntryType:    row.EntryType,
		ShiftKind:    row.ShiftKind,
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
	if entry.ShiftKind == "" {
		entry.ShiftKind = domain.ShiftKindManual
	}

	entry.Employee = &domain.Employee{
		ID:                    row.EmployeeID,
		Email:                 row.EmployeeEmail,
		FirstName:             row.EmployeeFirstName,
		LastName:              row.EmployeeLastName,
		Nickname:              row.EmployeeNickname,
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

func mapScheduleEntrySegment(row scheduleEntrySegmentRow) domain.ScheduleEntrySegment {
	segment := domain.ScheduleEntrySegment{
		ID:              row.ID,
		ScheduleEntryID: row.ScheduleEntryID,
		GroupID:         row.GroupID,
		Notes:           row.Notes,
		SortOrder:       row.SortOrder,
		Group: &domain.Group{
			ID:    row.GroupID,
			Name:  row.GroupName,
			Color: "",
		},
	}
	if row.StartTime.Valid {
		segment.StartTime = row.StartTime.Time
	}
	if row.EndTime.Valid {
		segment.EndTime = row.EndTime.Time
	}
	if row.CreatedAt.Valid {
		segment.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		segment.UpdatedAt = row.UpdatedAt.Time
	}
	if row.GroupDescription != nil {
		segment.Group.Description = row.GroupDescription
	}
	if row.GroupColor != nil {
		segment.Group.Color = *row.GroupColor
	}
	if row.GroupCreatedAt.Valid {
		segment.Group.CreatedAt = row.GroupCreatedAt.Time
	}
	if row.GroupUpdatedAt.Valid {
		segment.Group.UpdatedAt = row.GroupUpdatedAt.Time
	}
	return segment
}

func mapScheduleRequest(row scheduleRequestRow) domain.ScheduleRequest {
	request := domain.ScheduleRequest{
		ID:          row.ID,
		EmployeeID:  row.EmployeeID,
		RequestType: row.RequestType,
		Text:        row.Text,
		Status:      row.Status,
		Employee: &domain.Employee{
			ID:                    row.EmployeeID,
			Email:                 row.EmployeeEmail,
			FirstName:             row.EmployeeFirstName,
			LastName:              row.EmployeeLastName,
			Nickname:              row.EmployeeNickname,
			Role:                  domain.EmployeeRole(row.EmployeeRole),
			WeeklyHours:           row.EmployeeWeeklyHours,
			VacationDaysPerYear:   row.EmployeeVacationDays,
			RemainingVacationDays: row.EmployeeRemainingVacation,
			OvertimeBalance:       row.EmployeeOvertimeBalance,
			Active:                row.EmployeeActive,
		},
	}
	if row.Date.Valid {
		request.Date = row.Date.Time
	}
	if row.StartTime.Valid {
		request.StartTime = &row.StartTime.Time
	}
	if row.EndTime.Valid {
		request.EndTime = &row.EndTime.Time
	}
	if row.CreatedAt.Valid {
		request.CreatedAt = row.CreatedAt.Time
		request.Employee.CreatedAt = row.EmployeeCreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		request.UpdatedAt = row.UpdatedAt.Time
	}
	if row.EmployeeUpdatedAt.Valid {
		request.Employee.UpdatedAt = row.EmployeeUpdatedAt.Time
	}
	return request
}

// List retrieves schedule entries for the given filters.
func (r *PostgresScheduleRepository) List(ctx context.Context, startDate, endDate time.Time, employeeID, groupID *int64) ([]domain.ScheduleEntry, error) {
	baseQuery := `
		SELECT se.id, se.employee_id, se.date, se.start_time, se.end_time,
		       COALESCE(se.break_minutes, 0) AS break_minutes,
		       se.group_id, se.entry_type, se.shift_kind, se.notes, se.created_at, se.updated_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.nickname AS employee_nickname,
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
	if err := r.attachSegments(ctx, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetByID retrieves a schedule entry by ID including relations.
func (r *PostgresScheduleRepository) GetByID(ctx context.Context, id int64) (*domain.ScheduleEntry, error) {
	var row scheduleEntryRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT se.id, se.employee_id, se.date, se.start_time, se.end_time,
		       COALESCE(se.break_minutes, 0) AS break_minutes,
		       se.group_id, se.entry_type, se.shift_kind, se.notes, se.created_at, se.updated_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.nickname AS employee_nickname,
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

	entries := []domain.ScheduleEntry{mapScheduleEntry(row)}
	if err := r.attachSegments(ctx, entries); err != nil {
		return nil, err
	}
	return &entries[0], nil
}

// Create inserts a schedule entry.
func (r *PostgresScheduleRepository) Create(ctx context.Context, entry *domain.ScheduleEntry) error {
	if entry.ShiftKind == "" {
		entry.ShiftKind = domain.ShiftKindManual
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.QueryRowxContext(ctx, `
		INSERT INTO schedule_entries (
			employee_id, date, start_time, end_time, break_minutes, group_id,
			entry_type, shift_kind, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, entry.EmployeeID, entry.Date, entry.StartTime, entry.EndTime,
		entry.BreakMinutes, entry.GroupID, entry.EntryType, entry.ShiftKind, entry.Notes,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
		return err
	}

	if len(entry.Segments) > 0 {
		if err := insertScheduleEntrySegments(ctx, tx, entry.ID, entry.Segments); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Update updates a schedule entry and returns the updated record.
func (r *PostgresScheduleRepository) Update(ctx context.Context, entry *domain.ScheduleEntry) (*domain.ScheduleEntry, error) {
	if entry.ShiftKind == "" {
		entry.ShiftKind = domain.ShiftKindManual
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE schedule_entries
		SET date = $2,
		    start_time = $3,
		    end_time = $4,
		    break_minutes = $5,
		    group_id = $6,
		    entry_type = $7,
		    shift_kind = $8,
		    notes = $9
		WHERE id = $1
	`, entry.ID, entry.Date, entry.StartTime, entry.EndTime, entry.BreakMinutes,
		entry.GroupID, entry.EntryType, entry.ShiftKind, entry.Notes); err != nil {
		return nil, err
	}

	if entry.SegmentsChanged {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schedule_entry_segments WHERE schedule_entry_id = $1`, entry.ID); err != nil {
			return nil, err
		}
		if len(entry.Segments) > 0 {
			if err := insertScheduleEntrySegments(ctx, tx, entry.ID, entry.Segments); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, entry.ID)
}

// Delete deletes a schedule entry by ID.
func (r *PostgresScheduleRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM schedule_entries WHERE id = $1`, id)
	return err
}

func (r *PostgresScheduleRepository) attachSegments(ctx context.Context, entries []domain.ScheduleEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ids := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	query := `
		SELECT ses.id, ses.schedule_entry_id, ses.group_id, ses.start_time, ses.end_time,
		       ses.notes, ses.sort_order, ses.created_at, ses.updated_at,
		       g.name AS group_name, g.description AS group_description, g.color AS group_color,
		       g.created_at AS group_created_at, g.updated_at AS group_updated_at
		FROM schedule_entry_segments ses
		JOIN groups g ON g.id = ses.group_id
		WHERE ses.schedule_entry_id IN (?`
	for i := 1; i < len(ids); i++ {
		query += ", ?"
	}
	query += ") ORDER BY ses.schedule_entry_id, ses.sort_order, ses.start_time"

	query = r.db.Rebind(query)
	var rows []scheduleEntrySegmentRow
	if err := r.db.SelectContext(ctx, &rows, query, ids...); err != nil {
		return err
	}

	segmentsByEntry := make(map[int64][]domain.ScheduleEntrySegment)
	for _, row := range rows {
		segmentsByEntry[row.ScheduleEntryID] = append(segmentsByEntry[row.ScheduleEntryID], mapScheduleEntrySegment(row))
	}
	for i := range entries {
		entries[i].Segments = segmentsByEntry[entries[i].ID]
	}
	return nil
}

func insertScheduleEntrySegments(ctx context.Context, tx *sqlx.Tx, entryID int64, segments []domain.ScheduleEntrySegment) error {
	for index, segment := range segments {
		sortOrder := segment.SortOrder
		if sortOrder == 0 {
			sortOrder = index + 1
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schedule_entry_segments (
				schedule_entry_id, group_id, start_time, end_time, notes, sort_order
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, entryID, segment.GroupID, segment.StartTime, segment.EndTime, segment.Notes, sortOrder); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresScheduleRepository) ListRequests(ctx context.Context, startDate, endDate time.Time, employeeID *int64) ([]domain.ScheduleRequest, error) {
	query := `
		SELECT sr.id, sr.employee_id, sr.date, sr.start_time, sr.end_time,
		       sr.request_type, sr.text, sr.status, sr.created_at, sr.updated_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.nickname AS employee_nickname,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at
		FROM schedule_requests sr
		JOIN employees e ON e.id = sr.employee_id
		WHERE sr.date BETWEEN $1 AND $2`
	args := []interface{}{startDate, endDate}
	if employeeID != nil {
		query += fmt.Sprintf(" AND sr.employee_id = $%d", len(args)+1)
		args = append(args, *employeeID)
	}
	query += " ORDER BY sr.date, sr.start_time NULLS LAST, sr.created_at"

	var rows []scheduleRequestRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	requests := make([]domain.ScheduleRequest, 0, len(rows))
	for _, row := range rows {
		requests = append(requests, mapScheduleRequest(row))
	}
	return requests, nil
}

func (r *PostgresScheduleRepository) GetRequestByID(ctx context.Context, id int64) (*domain.ScheduleRequest, error) {
	var row scheduleRequestRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT sr.id, sr.employee_id, sr.date, sr.start_time, sr.end_time,
		       sr.request_type, sr.text, sr.status, sr.created_at, sr.updated_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.nickname AS employee_nickname,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at
		FROM schedule_requests sr
		JOIN employees e ON e.id = sr.employee_id
		WHERE sr.id = $1
	`, id); err != nil {
		return nil, err
	}
	request := mapScheduleRequest(row)
	return &request, nil
}

func (r *PostgresScheduleRepository) CreateRequest(ctx context.Context, request *domain.ScheduleRequest) error {
	if request.Status == "" {
		request.Status = domain.ScheduleRequestStatusOpen
	}
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO schedule_requests (
			employee_id, date, start_time, end_time, request_type, text, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, request.EmployeeID, request.Date, request.StartTime, request.EndTime,
		request.RequestType, request.Text, request.Status,
	).Scan(&request.ID, &request.CreatedAt, &request.UpdatedAt)
}

func (r *PostgresScheduleRepository) UpdateRequest(ctx context.Context, request *domain.ScheduleRequest) (*domain.ScheduleRequest, error) {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE schedule_requests
		SET date = $2,
		    start_time = $3,
		    end_time = $4,
		    request_type = $5,
		    text = $6,
		    status = $7
		WHERE id = $1
	`, request.ID, request.Date, request.StartTime, request.EndTime,
		request.RequestType, request.Text, request.Status); err != nil {
		return nil, err
	}
	return r.GetRequestByID(ctx, request.ID)
}

func (r *PostgresScheduleRepository) DeleteRequest(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM schedule_requests WHERE id = $1`, id)
	return err
}
