package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
		SELECT id, email, password_hash, first_name, last_name, nickname, role, weekly_hours,
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
		SELECT id, email, password_hash, first_name, last_name, nickname, role, weekly_hours,
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
		SELECT id, email, password_hash, first_name, last_name, nickname, role, weekly_hours,
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
			email, password_hash, first_name, last_name, nickname, role, weekly_hours,
			vacation_days_per_year, remaining_vacation_days, overtime_balance, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, employee.Email, employee.PasswordHash, employee.FirstName, employee.LastName,
		employee.Nickname, employee.Role, employee.WeeklyHours, employee.VacationDaysPerYear,
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
		    nickname = $5,
		    role = $6,
		    weekly_hours = $7,
		    vacation_days_per_year = $8,
		    remaining_vacation_days = $9,
		    overtime_balance = $10,
		    active = $11
		WHERE id = $1
		RETURNING id, email, password_hash, first_name, last_name, nickname, role, weekly_hours,
		          vacation_days_per_year, remaining_vacation_days, overtime_balance,
		          active, created_at, updated_at
	`, employee.ID, employee.Email, employee.FirstName, employee.LastName, employee.Nickname,
		employee.Role, employee.WeeklyHours, employee.VacationDaysPerYear, employee.RemainingVacationDays,
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

// ListContracts retrieves all historical contracts for an employee.
func (r *PostgresEmployeeRepository) ListContracts(ctx context.Context, employeeID int64) ([]domain.EmployeeContract, error) {
	var contracts []domain.EmployeeContract
	if err := r.db.SelectContext(ctx, &contracts, `
		SELECT id, employee_id, valid_from, weekly_hours, created_at, updated_at
		FROM employee_contracts
		WHERE employee_id = $1
		ORDER BY valid_from DESC
	`, employeeID); err != nil {
		return nil, err
	}
	return r.attachWorkdays(ctx, contracts)
}

// GetContractByID retrieves a contract with its workday pattern.
func (r *PostgresEmployeeRepository) GetContractByID(ctx context.Context, id int64) (*domain.EmployeeContract, error) {
	var contract domain.EmployeeContract
	if err := r.db.GetContext(ctx, &contract, `
		SELECT id, employee_id, valid_from, weekly_hours, created_at, updated_at
		FROM employee_contracts
		WHERE id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	workdays, err := r.listContractWorkdays(ctx, contract.ID)
	if err != nil {
		return nil, err
	}
	contract.Workdays = workdays
	return &contract, nil
}

// GetContractForDate retrieves the latest contract valid for the given date.
func (r *PostgresEmployeeRepository) GetContractForDate(ctx context.Context, employeeID int64, date time.Time) (*domain.EmployeeContract, error) {
	var contract domain.EmployeeContract
	if err := r.db.GetContext(ctx, &contract, `
		SELECT id, employee_id, valid_from, weekly_hours, created_at, updated_at
		FROM employee_contracts
		WHERE employee_id = $1 AND valid_from <= date_trunc('month', $2::date)::date
		ORDER BY valid_from DESC
		LIMIT 1
	`, employeeID, date); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	workdays, err := r.listContractWorkdays(ctx, contract.ID)
	if err != nil {
		return nil, err
	}
	contract.Workdays = workdays
	return &contract, nil
}

// CreateContract inserts a contract and its workday pattern.
func (r *PostgresEmployeeRepository) CreateContract(ctx context.Context, contract *domain.EmployeeContract) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.QueryRowxContext(ctx, `
		INSERT INTO employee_contracts (employee_id, valid_from, weekly_hours)
		VALUES ($1, date_trunc('month', $2::date)::date, $3)
		RETURNING id, valid_from, created_at, updated_at
	`, contract.EmployeeID, contract.ValidFrom, contract.WeeklyHours).
		Scan(&contract.ID, &contract.ValidFrom, &contract.CreatedAt, &contract.UpdatedAt); err != nil {
		return err
	}

	if err := replaceContractWorkdays(ctx, tx, contract.ID, contract.Workdays); err != nil {
		return err
	}
	if err := syncEmployeeWeeklyHours(ctx, tx, contract.EmployeeID); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateContract updates a contract and replaces its workday pattern.
func (r *PostgresEmployeeRepository) UpdateContract(ctx context.Context, contract *domain.EmployeeContract) (*domain.EmployeeContract, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var updated domain.EmployeeContract
	if err := tx.GetContext(ctx, &updated, `
		UPDATE employee_contracts
		SET valid_from = date_trunc('month', $2::date)::date,
		    weekly_hours = $3
		WHERE id = $1
		RETURNING id, employee_id, valid_from, weekly_hours, created_at, updated_at
	`, contract.ID, contract.ValidFrom, contract.WeeklyHours); err != nil {
		return nil, err
	}

	if err := replaceContractWorkdays(ctx, tx, updated.ID, contract.Workdays); err != nil {
		return nil, err
	}
	if err := syncEmployeeWeeklyHours(ctx, tx, updated.EmployeeID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetContractByID(ctx, updated.ID)
}

func (r *PostgresEmployeeRepository) attachWorkdays(ctx context.Context, contracts []domain.EmployeeContract) ([]domain.EmployeeContract, error) {
	for i := range contracts {
		workdays, err := r.listContractWorkdays(ctx, contracts[i].ID)
		if err != nil {
			return nil, err
		}
		contracts[i].Workdays = workdays
	}
	return contracts, nil
}

func (r *PostgresEmployeeRepository) listContractWorkdays(ctx context.Context, contractID int64) ([]domain.EmployeeContractWorkday, error) {
	var workdays []domain.EmployeeContractWorkday
	if err := r.db.SelectContext(ctx, &workdays, `
		SELECT id, contract_id, weekday, planned_minutes
		FROM employee_contract_workdays
		WHERE contract_id = $1
		ORDER BY weekday
	`, contractID); err != nil {
		return nil, err
	}
	return workdays, nil
}

type txExecutor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryRowxContext(context.Context, string, ...interface{}) *sqlx.Row
}

func replaceContractWorkdays(ctx context.Context, tx txExecutor, contractID int64, workdays []domain.EmployeeContractWorkday) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM employee_contract_workdays WHERE contract_id = $1`, contractID); err != nil {
		return err
	}
	seen := make(map[int]bool)
	for _, day := range workdays {
		if day.Weekday < 1 || day.Weekday > 5 {
			return fmt.Errorf("weekday must be between 1 and 5")
		}
		if seen[day.Weekday] {
			return fmt.Errorf("duplicate weekday %d", day.Weekday)
		}
		seen[day.Weekday] = true
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO employee_contract_workdays (contract_id, weekday, planned_minutes)
			VALUES ($1, $2, $3)
		`, contractID, day.Weekday, day.PlannedMinutes); err != nil {
			return err
		}
	}
	return nil
}

func syncEmployeeWeeklyHours(ctx context.Context, tx txExecutor, employeeID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE employees
		SET weekly_hours = (
			SELECT weekly_hours
			FROM employee_contracts
			WHERE employee_id = $1
			ORDER BY valid_from DESC
			LIMIT 1
		)
		WHERE id = $1
	`, employeeID)
	return err
}
