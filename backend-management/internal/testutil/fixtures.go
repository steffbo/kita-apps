package testutil

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// Default test password (hashed version of "Test1234!")
var DefaultPasswordHash string

func init() {
	hash, _ := auth.HashPassword("Test1234!")
	DefaultPasswordHash = hash
}

// EmployeeBuilder helps create test employees with sensible defaults.
type EmployeeBuilder struct {
	employee domain.Employee
}

// NewEmployeeBuilder creates a new builder with default values.
func NewEmployeeBuilder() *EmployeeBuilder {
	return &EmployeeBuilder{
		employee: domain.Employee{
			Email:                 "test@example.com",
			PasswordHash:          DefaultPasswordHash,
			FirstName:             "Test",
			LastName:              "User",
			Role:                  domain.EmployeeRoleEmployee,
			WeeklyHours:           40.0,
			VacationDaysPerYear:   30,
			RemainingVacationDays: 30.0,
			OvertimeBalance:       0.0,
			Active:                true,
		},
	}
}

func (b *EmployeeBuilder) WithEmail(email string) *EmployeeBuilder {
	b.employee.Email = email
	return b
}

func (b *EmployeeBuilder) WithName(first, last string) *EmployeeBuilder {
	b.employee.FirstName = first
	b.employee.LastName = last
	return b
}

func (b *EmployeeBuilder) WithNickname(nickname string) *EmployeeBuilder {
	b.employee.Nickname = &nickname
	return b
}

func (b *EmployeeBuilder) WithRole(role domain.EmployeeRole) *EmployeeBuilder {
	b.employee.Role = role
	return b
}

func (b *EmployeeBuilder) AsAdmin() *EmployeeBuilder {
	b.employee.Role = domain.EmployeeRoleAdmin
	return b
}

func (b *EmployeeBuilder) WithWeeklyHours(hours float64) *EmployeeBuilder {
	b.employee.WeeklyHours = hours
	return b
}

func (b *EmployeeBuilder) WithVacationDays(total int, remaining float64) *EmployeeBuilder {
	b.employee.VacationDaysPerYear = total
	b.employee.RemainingVacationDays = remaining
	return b
}

func (b *EmployeeBuilder) Inactive() *EmployeeBuilder {
	b.employee.Active = false
	return b
}

func (b *EmployeeBuilder) WithPassword(password string) *EmployeeBuilder {
	hash, _ := auth.HashPassword(password)
	b.employee.PasswordHash = hash
	return b
}

func (b *EmployeeBuilder) Build() *domain.Employee {
	return &b.employee
}

// Create inserts the employee into the database and returns the saved employee with ID.
func (b *EmployeeBuilder) Create(ctx context.Context, db *sqlx.DB) (*domain.Employee, error) {
	repo := repository.NewPostgresEmployeeRepository(db)
	emp := b.Build()
	if err := repo.Create(ctx, emp); err != nil {
		return nil, err
	}
	if err := repo.CreateContract(ctx, &domain.EmployeeContract{
		EmployeeID:  emp.ID,
		ValidFrom:   time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC),
		WeeklyHours: emp.WeeklyHours,
		Workdays:    defaultTestWorkdays(emp.WeeklyHours),
	}); err != nil {
		return nil, err
	}
	return emp, nil
}

func defaultTestWorkdays(weeklyHours float64) []domain.EmployeeContractWorkday {
	if weeklyHours == 33 {
		return []domain.EmployeeContractWorkday{
			{Weekday: 1, PlannedMinutes: 420},
			{Weekday: 2, PlannedMinutes: 420},
			{Weekday: 3, PlannedMinutes: 420},
			{Weekday: 4, PlannedMinutes: 360},
			{Weekday: 5, PlannedMinutes: 360},
		}
	}
	dailyMinutes := int(math.Round(weeklyHours * 60 / 5))
	workdays := make([]domain.EmployeeContractWorkday, 0, 5)
	for weekday := 1; weekday <= 5; weekday++ {
		workdays = append(workdays, domain.EmployeeContractWorkday{Weekday: weekday, PlannedMinutes: dailyMinutes})
	}
	return workdays
}

// GroupBuilder helps create test groups with sensible defaults.
type GroupBuilder struct {
	group domain.Group
}

// NewGroupBuilder creates a new builder with default values.
func NewGroupBuilder() *GroupBuilder {
	return &GroupBuilder{
		group: domain.Group{
			Name:  "Test Group",
			Color: "#3B82F6",
		},
	}
}

func (b *GroupBuilder) WithName(name string) *GroupBuilder {
	b.group.Name = name
	return b
}

func (b *GroupBuilder) WithDescription(desc string) *GroupBuilder {
	b.group.Description = &desc
	return b
}

func (b *GroupBuilder) WithColor(color string) *GroupBuilder {
	b.group.Color = color
	return b
}

func (b *GroupBuilder) Build() *domain.Group {
	return &b.group
}

// Create inserts the group into the database.
func (b *GroupBuilder) Create(ctx context.Context, db *sqlx.DB) (*domain.Group, error) {
	repo := repository.NewPostgresGroupRepository(db)
	grp := b.Build()
	if err := repo.Create(ctx, grp); err != nil {
		return nil, err
	}
	return grp, nil
}

// ScheduleEntryBuilder helps create test schedule entries.
type ScheduleEntryBuilder struct {
	entry domain.ScheduleEntry
}

// NewScheduleEntryBuilder creates a new builder with default values.
func NewScheduleEntryBuilder() *ScheduleEntryBuilder {
	today := time.Now().Truncate(24 * time.Hour)
	startTime := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
	endTime := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

	return &ScheduleEntryBuilder{
		entry: domain.ScheduleEntry{
			Date:         today,
			StartTime:    &startTime,
			EndTime:      &endTime,
			BreakMinutes: 30,
			EntryType:    domain.ScheduleEntryTypeWork,
			ShiftKind:    domain.ShiftKindManual,
		},
	}
}

func (b *ScheduleEntryBuilder) WithEmployeeID(id int64) *ScheduleEntryBuilder {
	b.entry.EmployeeID = id
	return b
}

func (b *ScheduleEntryBuilder) WithDate(date time.Time) *ScheduleEntryBuilder {
	b.entry.Date = date
	return b
}

func (b *ScheduleEntryBuilder) WithTimes(start, end time.Time) *ScheduleEntryBuilder {
	b.entry.StartTime = &start
	b.entry.EndTime = &end
	return b
}

func (b *ScheduleEntryBuilder) WithBreak(minutes int) *ScheduleEntryBuilder {
	b.entry.BreakMinutes = minutes
	return b
}

func (b *ScheduleEntryBuilder) WithGroupID(id int64) *ScheduleEntryBuilder {
	b.entry.GroupID = &id
	return b
}

func (b *ScheduleEntryBuilder) WithType(entryType domain.ScheduleEntryType) *ScheduleEntryBuilder {
	b.entry.EntryType = entryType
	return b
}

func (b *ScheduleEntryBuilder) WithNotes(notes string) *ScheduleEntryBuilder {
	b.entry.Notes = &notes
	return b
}

func (b *ScheduleEntryBuilder) Build() *domain.ScheduleEntry {
	return &b.entry
}

// Create inserts the schedule entry into the database.
func (b *ScheduleEntryBuilder) Create(ctx context.Context, db *sqlx.DB) (*domain.ScheduleEntry, error) {
	repo := repository.NewPostgresScheduleRepository(db)
	entry := b.Build()
	if err := repo.Create(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// TimeEntryBuilder helps create test time entries.
type TimeEntryBuilder struct {
	entry domain.TimeEntry
}

// NewTimeEntryBuilder creates a new builder with default values.
func NewTimeEntryBuilder() *TimeEntryBuilder {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)

	return &TimeEntryBuilder{
		entry: domain.TimeEntry{
			Date:         today,
			ClockIn:      clockIn,
			BreakMinutes: 0,
			EntryType:    domain.TimeEntryTypeWork,
		},
	}
}

func (b *TimeEntryBuilder) WithEmployeeID(id int64) *TimeEntryBuilder {
	b.entry.EmployeeID = id
	return b
}

func (b *TimeEntryBuilder) WithDate(date time.Time) *TimeEntryBuilder {
	b.entry.Date = date
	return b
}

func (b *TimeEntryBuilder) WithClockIn(clockIn time.Time) *TimeEntryBuilder {
	b.entry.ClockIn = clockIn
	return b
}

func (b *TimeEntryBuilder) WithClockOut(clockOut time.Time) *TimeEntryBuilder {
	b.entry.ClockOut = &clockOut
	return b
}

func (b *TimeEntryBuilder) WithBreak(minutes int) *TimeEntryBuilder {
	b.entry.BreakMinutes = minutes
	return b
}

func (b *TimeEntryBuilder) WithType(entryType domain.TimeEntryType) *TimeEntryBuilder {
	b.entry.EntryType = entryType
	return b
}

func (b *TimeEntryBuilder) WithNotes(notes string) *TimeEntryBuilder {
	b.entry.Notes = &notes
	return b
}

func (b *TimeEntryBuilder) Build() *domain.TimeEntry {
	return &b.entry
}

// Create inserts the time entry into the database.
func (b *TimeEntryBuilder) Create(ctx context.Context, db *sqlx.DB) (*domain.TimeEntry, error) {
	repo := repository.NewPostgresTimeEntryRepository(db)
	entry := b.Build()
	if err := repo.Create(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// SpecialDayBuilder helps create test special days.
type SpecialDayBuilder struct {
	day domain.SpecialDay
}

// NewSpecialDayBuilder creates a new builder with default values.
func NewSpecialDayBuilder() *SpecialDayBuilder {
	return &SpecialDayBuilder{
		day: domain.SpecialDay{
			Date:       time.Now().Truncate(24 * time.Hour),
			Name:       "Test Holiday",
			DayType:    domain.SpecialDayTypeHoliday,
			AffectsAll: true,
		},
	}
}

func (b *SpecialDayBuilder) WithDate(date time.Time) *SpecialDayBuilder {
	b.day.Date = date
	return b
}

func (b *SpecialDayBuilder) WithEndDate(endDate time.Time) *SpecialDayBuilder {
	b.day.EndDate = &endDate
	return b
}

func (b *SpecialDayBuilder) WithName(name string) *SpecialDayBuilder {
	b.day.Name = name
	return b
}

func (b *SpecialDayBuilder) WithType(dayType domain.SpecialDayType) *SpecialDayBuilder {
	b.day.DayType = dayType
	return b
}

func (b *SpecialDayBuilder) AffectsAll(affects bool) *SpecialDayBuilder {
	b.day.AffectsAll = affects
	return b
}

func (b *SpecialDayBuilder) WithNotes(notes string) *SpecialDayBuilder {
	b.day.Notes = &notes
	return b
}

func (b *SpecialDayBuilder) Build() *domain.SpecialDay {
	return &b.day
}

// Create inserts the special day into the database.
func (b *SpecialDayBuilder) Create(ctx context.Context, db *sqlx.DB) (*domain.SpecialDay, error) {
	repo := repository.NewPostgresSpecialDayRepository(db)
	day := b.Build()
	if err := repo.Create(ctx, day); err != nil {
		return nil, err
	}
	return day, nil
}

// GroupAssignmentBuilder helps create test group assignments.
type GroupAssignmentBuilder struct {
	assignment domain.GroupAssignment
}

// NewGroupAssignmentBuilder creates a new builder with default values.
func NewGroupAssignmentBuilder() *GroupAssignmentBuilder {
	return &GroupAssignmentBuilder{
		assignment: domain.GroupAssignment{
			AssignmentType: domain.AssignmentTypePermanent,
		},
	}
}

func (b *GroupAssignmentBuilder) WithEmployeeID(id int64) *GroupAssignmentBuilder {
	b.assignment.EmployeeID = id
	return b
}

func (b *GroupAssignmentBuilder) WithGroupID(id int64) *GroupAssignmentBuilder {
	b.assignment.GroupID = id
	return b
}

func (b *GroupAssignmentBuilder) WithType(assignmentType domain.AssignmentType) *GroupAssignmentBuilder {
	b.assignment.AssignmentType = assignmentType
	return b
}

func (b *GroupAssignmentBuilder) AsSpringer() *GroupAssignmentBuilder {
	b.assignment.AssignmentType = domain.AssignmentTypeSpringer
	return b
}

func (b *GroupAssignmentBuilder) Build() *domain.GroupAssignment {
	return &b.assignment
}

// Create inserts the group assignment into the database.
func (b *GroupAssignmentBuilder) Create(ctx context.Context, db *sqlx.DB) (*domain.GroupAssignment, error) {
	repo := repository.NewPostgresGroupAssignmentRepository(db)
	assignment := b.Build()
	if err := repo.Create(ctx, assignment); err != nil {
		return nil, err
	}
	return assignment, nil
}

// TestFixtures provides commonly used test data combinations.
type TestFixtures struct {
	DB *sqlx.DB
}

// NewTestFixtures creates a new TestFixtures instance.
func NewTestFixtures(db *sqlx.DB) *TestFixtures {
	return &TestFixtures{DB: db}
}

// CreateAdminEmployee creates an admin employee with a unique email.
func (f *TestFixtures) CreateAdminEmployee(ctx context.Context, suffix string) (*domain.Employee, error) {
	return NewEmployeeBuilder().
		WithEmail(fmt.Sprintf("admin%s@example.com", suffix)).
		WithName("Admin", "User").
		AsAdmin().
		Create(ctx, f.DB)
}

// CreateEmployee creates a regular employee with a unique email.
func (f *TestFixtures) CreateEmployee(ctx context.Context, suffix string) (*domain.Employee, error) {
	return NewEmployeeBuilder().
		WithEmail(fmt.Sprintf("employee%s@example.com", suffix)).
		WithName("Employee", "User").
		Create(ctx, f.DB)
}

// CreateGroup creates a group with a unique name.
func (f *TestFixtures) CreateGroup(ctx context.Context, suffix string) (*domain.Group, error) {
	return NewGroupBuilder().
		WithName(fmt.Sprintf("Group %s", suffix)).
		Create(ctx, f.DB)
}

// CreateGroupWithEmployee creates a group and assigns an employee to it.
func (f *TestFixtures) CreateGroupWithEmployee(ctx context.Context, suffix string, employee *domain.Employee) (*domain.Group, *domain.GroupAssignment, error) {
	group, err := f.CreateGroup(ctx, suffix)
	if err != nil {
		return nil, nil, err
	}

	assignment, err := NewGroupAssignmentBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		Create(ctx, f.DB)
	if err != nil {
		return nil, nil, err
	}

	return group, assignment, nil
}
