package repository

import (
	"context"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// EmployeeRepository handles employee persistence.
type EmployeeRepository interface {
	List(ctx context.Context, activeOnly bool) ([]domain.Employee, error)
	GetByID(ctx context.Context, id int64) (*domain.Employee, error)
	GetByEmail(ctx context.Context, email string) (*domain.Employee, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Create(ctx context.Context, employee *domain.Employee) error
	Update(ctx context.Context, employee *domain.Employee) (*domain.Employee, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Deactivate(ctx context.Context, id int64) error
	AdjustRemainingVacationDays(ctx context.Context, id int64, delta float64) error
}

// GroupRepository handles group persistence.
type GroupRepository interface {
	List(ctx context.Context) ([]domain.Group, error)
	GetByID(ctx context.Context, id int64) (*domain.Group, error)
	Create(ctx context.Context, group *domain.Group) error
	Update(ctx context.Context, group *domain.Group) (*domain.Group, error)
	Delete(ctx context.Context, id int64) error
}

// GroupAssignmentRepository handles group assignment persistence.
type GroupAssignmentRepository interface {
	ListByGroupID(ctx context.Context, groupID int64) ([]domain.GroupAssignment, error)
	ListByEmployeeID(ctx context.Context, employeeID int64) ([]domain.GroupAssignment, error)
	ListPrimaryAssignments(ctx context.Context, employeeIDs []int64) ([]domain.GroupAssignment, error)
	GetPrimaryAssignment(ctx context.Context, employeeID int64) (*domain.GroupAssignment, error)
	Create(ctx context.Context, assignment *domain.GroupAssignment) error
	UpdateGroup(ctx context.Context, assignmentID, groupID int64) error
	DeleteByGroupID(ctx context.Context, groupID int64) error
	DeleteByGroupAndEmployee(ctx context.Context, groupID, employeeID int64) error
	DeleteByEmployeeAndType(ctx context.Context, employeeID int64, assignmentType domain.AssignmentType) error
}

// ScheduleRepository handles schedule entries persistence.
type ScheduleRepository interface {
	List(ctx context.Context, startDate, endDate time.Time, employeeID, groupID *int64) ([]domain.ScheduleEntry, error)
	GetByID(ctx context.Context, id int64) (*domain.ScheduleEntry, error)
	Create(ctx context.Context, entry *domain.ScheduleEntry) error
	Update(ctx context.Context, entry *domain.ScheduleEntry) (*domain.ScheduleEntry, error)
	Delete(ctx context.Context, id int64) error
}

// TimeEntryRepository handles time entry persistence.
type TimeEntryRepository interface {
	List(ctx context.Context, startDate, endDate time.Time, employeeID *int64) ([]domain.TimeEntry, error)
	GetByID(ctx context.Context, id int64) (*domain.TimeEntry, error)
	ListOpenByEmployeeID(ctx context.Context, employeeID int64) ([]domain.TimeEntry, error)
	Create(ctx context.Context, entry *domain.TimeEntry) error
	Update(ctx context.Context, entry *domain.TimeEntry) (*domain.TimeEntry, error)
	Delete(ctx context.Context, id int64) error
}

// SpecialDayRepository handles special day persistence.
type SpecialDayRepository interface {
	List(ctx context.Context, startDate, endDate time.Time) ([]domain.SpecialDay, error)
	ListByType(ctx context.Context, startDate, endDate time.Time, dayType domain.SpecialDayType) ([]domain.SpecialDay, error)
	GetByID(ctx context.Context, id int64) (*domain.SpecialDay, error)
	Create(ctx context.Context, day *domain.SpecialDay) error
	Update(ctx context.Context, day *domain.SpecialDay) (*domain.SpecialDay, error)
	Delete(ctx context.Context, id int64) error
}
