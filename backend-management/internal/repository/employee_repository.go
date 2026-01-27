package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// PostgresEmployeeRepository is the PostgreSQL implementation of EmployeeRepository.
type PostgresEmployeeRepository struct {
	db *sqlx.DB
}

// NewPostgresEmployeeRepository creates a new PostgreSQL employee repository.
func NewPostgresEmployeeRepository(db *sqlx.DB) *PostgresEmployeeRepository {
	return &PostgresEmployeeRepository{db: db}
}

// List retrieves employees with optional active filter.
func (r *PostgresEmployeeRepository) List(ctx context.Context, activeOnly bool) ([]domain.Employee, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, weekly_hours,
		       vacation_days_per_year, remaining_vacation_days, overtime_balance,
		       active, created_at, updated_at
		FROM employees`

	if activeOnly {
		query += " WHERE active = true"
	}
	query += " ORDER BY last_name, first_name"

	var employees []domain.Employee
	if err := r.db.SelectContext(ctx, &employees, query); err != nil {
		return nil, err
	}
	return employees, nil
}

// GetByID retrieves an employee by ID.
func (r *PostgresEmployeeRepository) GetByID(ctx context.Context, id int64) (*domain.Employee, error) {
	var employee domain.Employee
	if err := r.db.GetContext(ctx, &employee, `
		SELECT id, email, password_hash, first_name, last_name, role, weekly_hours,
		       vacation_days_per_year, remaining_vacation_days, overtime_balance,
		       active, created_at, updated_at
		FROM employees
		WHERE id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &employee, nil
}

// GetByEmail retrieves an employee by email.
func (r *PostgresEmployeeRepository) GetByEmail(ctx context.Context, email string) (*domain.Employee, error) {
	var employee domain.Employee
	if err := r.db.GetContext(ctx, &employee, `
		SELECT id, email, password_hash, first_name, last_name, role, weekly_hours,
		       vacation_days_per_year, remaining_vacation_days, overtime_balance,
		       active, created_at, updated_at
		FROM employees
		WHERE email = $1
	`, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &employee, nil
}

// ExistsByEmail checks if an employee exists by email.
func (r *PostgresEmployeeRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `SELECT EXISTS (SELECT 1 FROM employees WHERE email = $1)`, email); err != nil {
		return false, err
	}
	return exists, nil
}

// Create inserts a new employee.
func (r *PostgresEmployeeRepository) Create(ctx context.Context, employee *domain.Employee) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO employees (
			email, password_hash, first_name, last_name, role, weekly_hours,
			vacation_days_per_year, remaining_vacation_days, overtime_balance, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, employee.Email, employee.PasswordHash, employee.FirstName, employee.LastName,
		employee.Role, employee.WeeklyHours, employee.VacationDaysPerYear,
		employee.RemainingVacationDays, employee.OvertimeBalance, employee.Active,
	).Scan(&employee.ID, &employee.CreatedAt, &employee.UpdatedAt)
}

// Update updates an employee and returns the updated record.
func (r *PostgresEmployeeRepository) Update(ctx context.Context, employee *domain.Employee) (*domain.Employee, error) {
	var updated domain.Employee
	if err := r.db.GetContext(ctx, &updated, `
		UPDATE employees
		SET email = $2,
		    first_name = $3,
		    last_name = $4,
		    role = $5,
		    weekly_hours = $6,
		    vacation_days_per_year = $7,
		    remaining_vacation_days = $8,
		    overtime_balance = $9,
		    active = $10
		WHERE id = $1
		RETURNING id, email, password_hash, first_name, last_name, role, weekly_hours,
		          vacation_days_per_year, remaining_vacation_days, overtime_balance,
		          active, created_at, updated_at
	`, employee.ID, employee.Email, employee.FirstName, employee.LastName, employee.Role,
		employee.WeeklyHours, employee.VacationDaysPerYear, employee.RemainingVacationDays,
		employee.OvertimeBalance, employee.Active); err != nil {
		return nil, err
	}
	return &updated, nil
}

// UpdatePassword updates an employee's password hash.
func (r *PostgresEmployeeRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE employees SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	return err
}

// Deactivate deactivates an employee (soft delete).
func (r *PostgresEmployeeRepository) Deactivate(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE employees SET active = false WHERE id = $1`, id)
	return err
}

// Delete permanently removes an employee from the database.
// This cascades to related records (tokens, assignments, schedule/time entries).
func (r *PostgresEmployeeRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM employees WHERE id = $1`, id)
	return err
}

// AdjustRemainingVacationDays adjusts remaining vacation days by a delta.
func (r *PostgresEmployeeRepository) AdjustRemainingVacationDays(ctx context.Context, id int64, delta float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE employees
		SET remaining_vacation_days = remaining_vacation_days + $2
		WHERE id = $1
	`, id, delta)
	return err
}
