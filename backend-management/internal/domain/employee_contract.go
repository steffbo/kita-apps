package domain

import "time"

// EmployeeContract stores the historical weekly-hours setup for an employee.
type EmployeeContract struct {
	ID          int64                     `db:"id"`
	EmployeeID  int64                     `db:"employee_id"`
	ValidFrom   time.Time                 `db:"valid_from"`
	WeeklyHours float64                   `db:"weekly_hours"`
	CreatedAt   time.Time                 `db:"created_at"`
	UpdatedAt   time.Time                 `db:"updated_at"`
	Workdays    []EmployeeContractWorkday `db:"-"`
}

// EmployeeContractWorkday stores the planned net working minutes for a weekday.
type EmployeeContractWorkday struct {
	ID             int64 `db:"id"`
	ContractID     int64 `db:"contract_id"`
	Weekday        int   `db:"weekday"`
	PlannedMinutes int   `db:"planned_minutes"`
}
