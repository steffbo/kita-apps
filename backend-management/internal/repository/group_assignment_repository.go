package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// PostgresGroupAssignmentRepository is the PostgreSQL implementation of GroupAssignmentRepository.
type PostgresGroupAssignmentRepository struct {
	db *sqlx.DB
}

// NewPostgresGroupAssignmentRepository creates a new PostgreSQL group assignment repository.
func NewPostgresGroupAssignmentRepository(db *sqlx.DB) *PostgresGroupAssignmentRepository {
	return &PostgresGroupAssignmentRepository{db: db}
}

type groupAssignmentRow struct {
	ID              int64                 `db:"id"`
	EmployeeID      int64                 `db:"employee_id"`
	GroupID         int64                 `db:"group_id"`
	AssignmentType  domain.AssignmentType `db:"assignment_type"`
	CreatedAt       sql.NullTime          `db:"created_at"`
	EmployeeEmail   sql.NullString        `db:"employee_email"`
	EmployeeFirst   sql.NullString        `db:"employee_first_name"`
	EmployeeLast    sql.NullString        `db:"employee_last_name"`
	EmployeeRole    sql.NullString        `db:"employee_role"`
	EmployeeHours   sql.NullFloat64       `db:"employee_weekly_hours"`
	EmployeeVacPer  sql.NullInt64         `db:"employee_vacation_days_per_year"`
	EmployeeVacRem  sql.NullFloat64       `db:"employee_remaining_vacation_days"`
	EmployeeOT      sql.NullFloat64       `db:"employee_overtime_balance"`
	EmployeeActive  sql.NullBool          `db:"employee_active"`
	EmployeeCreated sql.NullTime          `db:"employee_created_at"`
	EmployeeUpdated sql.NullTime          `db:"employee_updated_at"`
	GroupName       sql.NullString        `db:"group_name"`
	GroupDesc       sql.NullString        `db:"group_description"`
	GroupColor      sql.NullString        `db:"group_color"`
	GroupCreated    sql.NullTime          `db:"group_created_at"`
	GroupUpdated    sql.NullTime          `db:"group_updated_at"`
}

func mapGroupAssignment(row groupAssignmentRow) domain.GroupAssignment {
	assignment := domain.GroupAssignment{
		ID:             row.ID,
		EmployeeID:     row.EmployeeID,
		GroupID:        row.GroupID,
		AssignmentType: row.AssignmentType,
	}
	if row.CreatedAt.Valid {
		assignment.CreatedAt = row.CreatedAt.Time
	}

	if row.EmployeeEmail.Valid {
		assignment.Employee = &domain.Employee{
			ID:                    row.EmployeeID,
			Email:                 row.EmployeeEmail.String,
			FirstName:             row.EmployeeFirst.String,
			LastName:              row.EmployeeLast.String,
			Role:                  domain.EmployeeRole(row.EmployeeRole.String),
			WeeklyHours:           row.EmployeeHours.Float64,
			VacationDaysPerYear:   int(row.EmployeeVacPer.Int64),
			RemainingVacationDays: row.EmployeeVacRem.Float64,
			OvertimeBalance:       row.EmployeeOT.Float64,
			Active:                row.EmployeeActive.Bool,
		}
		if row.EmployeeCreated.Valid {
			assignment.Employee.CreatedAt = row.EmployeeCreated.Time
		}
		if row.EmployeeUpdated.Valid {
			assignment.Employee.UpdatedAt = row.EmployeeUpdated.Time
		}
	}

	if row.GroupName.Valid {
		assignment.Group = &domain.Group{
			ID:          row.GroupID,
			Name:        row.GroupName.String,
			Color:       row.GroupColor.String,
			Description: nil,
		}
		if row.GroupDesc.Valid {
			assignment.Group.Description = &row.GroupDesc.String
		}
		if row.GroupCreated.Valid {
			assignment.Group.CreatedAt = row.GroupCreated.Time
		}
		if row.GroupUpdated.Valid {
			assignment.Group.UpdatedAt = row.GroupUpdated.Time
		}
	}

	return assignment
}

// ListByGroupID retrieves assignments for a group including employee info.
func (r *PostgresGroupAssignmentRepository) ListByGroupID(ctx context.Context, groupID int64) ([]domain.GroupAssignment, error) {
	var rows []groupAssignmentRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT ga.id, ga.employee_id, ga.group_id, ga.assignment_type, ga.created_at,
		       e.email AS employee_email, e.first_name AS employee_first_name, e.last_name AS employee_last_name,
		       e.role AS employee_role, e.weekly_hours AS employee_weekly_hours,
		       e.vacation_days_per_year AS employee_vacation_days_per_year,
		       e.remaining_vacation_days AS employee_remaining_vacation_days,
		       e.overtime_balance AS employee_overtime_balance,
		       e.active AS employee_active, e.created_at AS employee_created_at, e.updated_at AS employee_updated_at
		FROM group_assignments ga
		JOIN employees e ON e.id = ga.employee_id
		WHERE ga.group_id = $1
		ORDER BY e.last_name, e.first_name
	`, groupID); err != nil {
		return nil, err
	}

	assignments := make([]domain.GroupAssignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, mapGroupAssignment(row))
	}
	return assignments, nil
}

// ListByEmployeeID retrieves assignments for an employee including group info.
func (r *PostgresGroupAssignmentRepository) ListByEmployeeID(ctx context.Context, employeeID int64) ([]domain.GroupAssignment, error) {
	var rows []groupAssignmentRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT ga.id, ga.employee_id, ga.group_id, ga.assignment_type, ga.created_at,
		       g.name AS group_name, g.description AS group_description, g.color AS group_color,
		       g.created_at AS group_created_at, g.updated_at AS group_updated_at
		FROM group_assignments ga
		JOIN groups g ON g.id = ga.group_id
		WHERE ga.employee_id = $1
		ORDER BY g.name
	`, employeeID); err != nil {
		return nil, err
	}

	assignments := make([]domain.GroupAssignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, mapGroupAssignment(row))
	}
	return assignments, nil
}

// ListPrimaryAssignments retrieves primary assignments for a set of employees.
func (r *PostgresGroupAssignmentRepository) ListPrimaryAssignments(ctx context.Context, employeeIDs []int64) ([]domain.GroupAssignment, error) {
	if len(employeeIDs) == 0 {
		return nil, nil
	}

	var rows []groupAssignmentRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT ga.id, ga.employee_id, ga.group_id, ga.assignment_type, ga.created_at,
		       g.name AS group_name, g.description AS group_description, g.color AS group_color,
		       g.created_at AS group_created_at, g.updated_at AS group_updated_at
		FROM group_assignments ga
		JOIN groups g ON g.id = ga.group_id
		WHERE ga.assignment_type = 'PERMANENT'
		  AND ga.employee_id = ANY($1)
	`, pq.Int64Array(employeeIDs)); err != nil {
		return nil, err
	}

	assignments := make([]domain.GroupAssignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, mapGroupAssignment(row))
	}
	return assignments, nil
}

// GetPrimaryAssignment retrieves the primary assignment for an employee.
func (r *PostgresGroupAssignmentRepository) GetPrimaryAssignment(ctx context.Context, employeeID int64) (*domain.GroupAssignment, error) {
	var row groupAssignmentRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT ga.id, ga.employee_id, ga.group_id, ga.assignment_type, ga.created_at,
		       g.name AS group_name, g.description AS group_description, g.color AS group_color,
		       g.created_at AS group_created_at, g.updated_at AS group_updated_at
		FROM group_assignments ga
		JOIN groups g ON g.id = ga.group_id
		WHERE ga.employee_id = $1 AND ga.assignment_type = 'PERMANENT'
		LIMIT 1
	`, employeeID); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	assignment := mapGroupAssignment(row)
	return &assignment, nil
}

// Create inserts a group assignment.
func (r *PostgresGroupAssignmentRepository) Create(ctx context.Context, assignment *domain.GroupAssignment) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO group_assignments (employee_id, group_id, assignment_type)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, assignment.EmployeeID, assignment.GroupID, assignment.AssignmentType).
		Scan(&assignment.ID, &assignment.CreatedAt)
}

// UpdateGroup updates the group of an assignment.
func (r *PostgresGroupAssignmentRepository) UpdateGroup(ctx context.Context, assignmentID, groupID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE group_assignments
		SET group_id = $2
		WHERE id = $1
	`, assignmentID, groupID)
	return err
}

// DeleteByGroupID deletes assignments by group.
func (r *PostgresGroupAssignmentRepository) DeleteByGroupID(ctx context.Context, groupID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM group_assignments WHERE group_id = $1`, groupID)
	return err
}

// DeleteByGroupAndEmployee deletes a specific assignment.
func (r *PostgresGroupAssignmentRepository) DeleteByGroupAndEmployee(ctx context.Context, groupID, employeeID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM group_assignments
		WHERE group_id = $1 AND employee_id = $2
	`, groupID, employeeID)
	return err
}

// DeleteByEmployeeAndType deletes assignments for an employee by type.
func (r *PostgresGroupAssignmentRepository) DeleteByEmployeeAndType(ctx context.Context, employeeID int64, assignmentType domain.AssignmentType) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM group_assignments
		WHERE employee_id = $1 AND assignment_type = $2
	`, employeeID, assignmentType)
	return err
}
