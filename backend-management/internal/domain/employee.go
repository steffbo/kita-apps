package domain

import "time"

// EmployeeRole defines the access level of an employee.
type EmployeeRole string

const (
	EmployeeRoleAdmin    EmployeeRole = "ADMIN"
	EmployeeRoleEmployee EmployeeRole = "EMPLOYEE"
)

// Employee represents an employee record.
type Employee struct {
	ID                    int64        `db:"id"`
	Email                 string       `db:"email"`
	PasswordHash          string       `db:"password_hash"`
	FirstName             string       `db:"first_name"`
	LastName              string       `db:"last_name"`
	Role                  EmployeeRole `db:"role"`
	WeeklyHours           float64      `db:"weekly_hours"`
	VacationDaysPerYear   int          `db:"vacation_days_per_year"`
	RemainingVacationDays float64      `db:"remaining_vacation_days"`
	OvertimeBalance       float64      `db:"overtime_balance"`
	Active                bool         `db:"active"`
	CreatedAt             time.Time    `db:"created_at"`
	UpdatedAt             time.Time    `db:"updated_at"`
}

// FullName returns the employee's full name.
func (e Employee) FullName() string {
	return e.FirstName + " " + e.LastName
}
