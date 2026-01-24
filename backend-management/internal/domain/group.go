package domain

import "time"

// AssignmentType defines the type of a group assignment.
type AssignmentType string

const (
	AssignmentTypePermanent AssignmentType = "PERMANENT"
	AssignmentTypeSpringer  AssignmentType = "SPRINGER"
)

// Group represents a group within the Kita.
type Group struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	Color       string    `db:"color"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// GroupAssignment represents an employee's assignment to a group.
type GroupAssignment struct {
	ID             int64          `db:"id"`
	EmployeeID     int64          `db:"employee_id"`
	GroupID        int64          `db:"group_id"`
	AssignmentType AssignmentType `db:"assignment_type"`
	CreatedAt      time.Time      `db:"created_at"`
	Employee       *Employee      `db:"-"`
	Group          *Group         `db:"-"`
}
