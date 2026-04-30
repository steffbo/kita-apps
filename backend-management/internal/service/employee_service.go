package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// EmployeeWithGroup represents an employee and their primary group.
type EmployeeWithGroup struct {
	Employee        domain.Employee
	PrimaryGroup    *domain.Group
	PrimaryGroupID  *int64
	CurrentContract *domain.EmployeeContract
}

// EmployeeService handles employee operations.
type EmployeeService struct {
	employees   repository.EmployeeRepository
	assignments repository.GroupAssignmentRepository
	groups      repository.GroupRepository
}

// NewEmployeeService creates a new EmployeeService.
func NewEmployeeService(
	employees repository.EmployeeRepository,
	assignments repository.GroupAssignmentRepository,
	groups repository.GroupRepository,
) *EmployeeService {
	return &EmployeeService{
		employees:   employees,
		assignments: assignments,
		groups:      groups,
	}
}

// List retrieves employees with primary group info.
func (s *EmployeeService) List(ctx context.Context, activeOnly bool) ([]EmployeeWithGroup, error) {
	employees, err := s.employees.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(employees))
	for _, emp := range employees {
		ids = append(ids, emp.ID)
	}

	assignments, err := s.assignments.ListPrimaryAssignments(ctx, ids)
	if err != nil {
		return nil, err
	}

	assignmentByEmployee := make(map[int64]domain.GroupAssignment)
	for _, assignment := range assignments {
		assignmentByEmployee[assignment.EmployeeID] = assignment
	}

	result := make([]EmployeeWithGroup, 0, len(employees))
	for _, emp := range employees {
		var primaryGroup *domain.Group
		var primaryGroupID *int64
		if assignment, ok := assignmentByEmployee[emp.ID]; ok {
			primaryGroup = assignment.Group
			groupID := assignment.GroupID
			primaryGroupID = &groupID
		}
		currentContract, err := s.employees.GetContractForDate(ctx, emp.ID, time.Now())
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		result = append(result, EmployeeWithGroup{
			Employee:        emp,
			PrimaryGroup:    primaryGroup,
			PrimaryGroupID:  primaryGroupID,
			CurrentContract: currentContract,
		})
	}

	return result, nil
}

// Get retrieves a single employee with primary group info.
func (s *EmployeeService) Get(ctx context.Context, id int64) (*EmployeeWithGroup, error) {
	emp, err := s.employees.GetByID(ctx, id)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", id))
	}

	assignment, err := s.assignments.GetPrimaryAssignment(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var primaryGroup *domain.Group
	var primaryGroupID *int64
	if assignment != nil {
		primaryGroup = assignment.Group
		groupID := assignment.GroupID
		primaryGroupID = &groupID
	}

	currentContract, err := s.employees.GetContractForDate(ctx, id, time.Now())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &EmployeeWithGroup{
		Employee:        *emp,
		PrimaryGroup:    primaryGroup,
		PrimaryGroupID:  primaryGroupID,
		CurrentContract: currentContract,
	}, nil
}

// CreateEmployeeInput represents input for creating an employee.
type CreateEmployeeInput struct {
	Email               string
	FirstName           string
	LastName            string
	Nickname            *string
	Role                domain.EmployeeRole
	WeeklyHours         float64
	VacationDaysPerYear int
	PrimaryGroupID      *int64
}

// EmployeeContractInput represents a historical weekly-hours setup.
type EmployeeContractInput struct {
	ValidFrom   time.Time
	WeeklyHours float64
	Workdays    []EmployeeContractWorkdayInput
}

// EmployeeContractWorkdayInput represents planned net minutes for a weekday.
type EmployeeContractWorkdayInput struct {
	Weekday        int
	PlannedMinutes int
}

// Create creates a new employee.
func (s *EmployeeService) Create(ctx context.Context, input CreateEmployeeInput) (*EmployeeWithGroup, error) {
	exists, err := s.employees.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, NewConflict("E-Mail-Adresse wird bereits verwendet")
	}

	tempPassword := generateTemporaryPassword()
	hash, err := auth.HashPassword(tempPassword)
	if err != nil {
		return nil, err
	}

	role := input.Role
	if role == "" {
		role = domain.EmployeeRoleEmployee
	}
	vacDays := input.VacationDaysPerYear
	if vacDays == 0 {
		vacDays = 30
	}

	employee := &domain.Employee{
		Email:                 input.Email,
		PasswordHash:          hash,
		FirstName:             input.FirstName,
		LastName:              input.LastName,
		Nickname:              normalizeOptionalString(input.Nickname),
		Role:                  role,
		WeeklyHours:           input.WeeklyHours,
		VacationDaysPerYear:   vacDays,
		RemainingVacationDays: float64(vacDays),
		OvertimeBalance:       0,
		Active:                true,
	}

	if err := s.employees.Create(ctx, employee); err != nil {
		return nil, err
	}

	if err := s.createDefaultContract(ctx, employee.ID, input.WeeklyHours); err != nil {
		return nil, err
	}

	if input.PrimaryGroupID != nil && *input.PrimaryGroupID > 0 {
		if err := s.setPrimaryGroup(ctx, employee.ID, *input.PrimaryGroupID); err != nil {
			return nil, err
		}
	}

	log.Info().Str("email", employee.Email).Str("password", tempPassword).Msg("temporary password generated")

	return s.Get(ctx, employee.ID)
}

// ListContracts retrieves all contracts for an employee.
func (s *EmployeeService) ListContracts(ctx context.Context, employeeID int64) ([]domain.EmployeeContract, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}
	return s.employees.ListContracts(ctx, employeeID)
}

// CreateContract creates a monthly contract for an employee.
func (s *EmployeeService) CreateContract(ctx context.Context, employeeID int64, input EmployeeContractInput) (*domain.EmployeeContract, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}
	contract := &domain.EmployeeContract{
		EmployeeID:  employeeID,
		ValidFrom:   firstOfMonth(input.ValidFrom),
		WeeklyHours: input.WeeklyHours,
		Workdays:    mapContractWorkdays(0, input.Workdays),
	}
	if len(contract.Workdays) == 0 {
		contract.Workdays = defaultWorkdays(0, input.WeeklyHours)
	}
	if err := s.employees.CreateContract(ctx, contract); err != nil {
		return nil, err
	}
	return s.employees.GetContractByID(ctx, contract.ID)
}

// UpdateContract updates a monthly contract for an employee.
func (s *EmployeeService) UpdateContract(ctx context.Context, employeeID, contractID int64, input EmployeeContractInput) (*domain.EmployeeContract, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}
	existing, err := s.employees.GetContractByID(ctx, contractID)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Vertrag mit ID %d nicht gefunden", contractID))
	}
	if existing.EmployeeID != employeeID {
		return nil, NewNotFound(fmt.Sprintf("Vertrag mit ID %d nicht gefunden", contractID))
	}
	contract := &domain.EmployeeContract{
		ID:          contractID,
		EmployeeID:  employeeID,
		ValidFrom:   firstOfMonth(input.ValidFrom),
		WeeklyHours: input.WeeklyHours,
		Workdays:    mapContractWorkdays(contractID, input.Workdays),
	}
	if len(contract.Workdays) == 0 {
		contract.Workdays = defaultWorkdays(contractID, input.WeeklyHours)
	}
	return s.employees.UpdateContract(ctx, contract)
}

// UpdateEmployeeInput represents input for updating an employee.
type UpdateEmployeeInput struct {
	Email                 *string
	FirstName             *string
	LastName              *string
	Nickname              *string
	Role                  *domain.EmployeeRole
	WeeklyHours           *float64
	VacationDaysPerYear   *int
	RemainingVacationDays *float64
	OvertimeBalance       *float64
	Active                *bool
	PrimaryGroupID        *int64
}

// Update updates an employee.
func (s *EmployeeService) Update(ctx context.Context, id int64, input UpdateEmployeeInput) (*EmployeeWithGroup, error) {
	employee, err := s.employees.GetByID(ctx, id)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", id))
	}

	if input.Email != nil {
		employee.Email = *input.Email
	}
	if input.FirstName != nil {
		employee.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		employee.LastName = *input.LastName
	}
	if input.Nickname != nil {
		employee.Nickname = normalizeOptionalString(input.Nickname)
	}
	if input.Role != nil {
		employee.Role = *input.Role
	}
	if input.WeeklyHours != nil {
		employee.WeeklyHours = *input.WeeklyHours
	}
	if input.VacationDaysPerYear != nil {
		employee.VacationDaysPerYear = *input.VacationDaysPerYear
	}
	if input.RemainingVacationDays != nil {
		employee.RemainingVacationDays = *input.RemainingVacationDays
	}
	if input.OvertimeBalance != nil {
		employee.OvertimeBalance = *input.OvertimeBalance
	}
	if input.Active != nil {
		employee.Active = *input.Active
	}

	if _, err := s.employees.Update(ctx, employee); err != nil {
		return nil, err
	}

	if input.PrimaryGroupID != nil {
		if *input.PrimaryGroupID == 0 {
			if err := s.assignments.DeleteByEmployeeAndType(ctx, employee.ID, domain.AssignmentTypePermanent); err != nil {
				return nil, err
			}
		} else {
			if err := s.setPrimaryGroup(ctx, employee.ID, *input.PrimaryGroupID); err != nil {
				return nil, err
			}
		}
	}

	return s.Get(ctx, employee.ID)
}

// Delete deactivates an employee (soft delete).
func (s *EmployeeService) Delete(ctx context.Context, id int64) error {
	if _, err := s.employees.GetByID(ctx, id); err != nil {
		return NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", id))
	}
	return s.employees.Deactivate(ctx, id)
}

// PermanentDelete permanently removes an employee and all related data.
func (s *EmployeeService) PermanentDelete(ctx context.Context, id int64) error {
	if _, err := s.employees.GetByID(ctx, id); err != nil {
		return NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", id))
	}
	return s.employees.Delete(ctx, id)
}

// ResetPassword resets an employee's password.
func (s *EmployeeService) ResetPassword(ctx context.Context, id int64) (string, error) {
	emp, err := s.employees.GetByID(ctx, id)
	if err != nil {
		return "", NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", id))
	}

	tempPassword := generateTemporaryPassword()
	hash, err := auth.HashPassword(tempPassword)
	if err != nil {
		return "", err
	}

	if err := s.employees.UpdatePassword(ctx, emp.ID, hash); err != nil {
		return "", err
	}

	log.Info().Str("email", emp.Email).Str("password", tempPassword).Msg("password reset generated")

	return tempPassword, nil
}

// Assignments retrieves assignments for an employee.
func (s *EmployeeService) Assignments(ctx context.Context, employeeID int64) ([]domain.GroupAssignment, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}
	return s.assignments.ListByEmployeeID(ctx, employeeID)
}

func (s *EmployeeService) setPrimaryGroup(ctx context.Context, employeeID, groupID int64) error {
	if _, err := s.groups.GetByID(ctx, groupID); err != nil {
		return NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", groupID))
	}

	assignment, err := s.assignments.GetPrimaryAssignment(ctx, employeeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if assignment != nil {
		if assignment.GroupID == groupID {
			return nil
		}
		return s.assignments.UpdateGroup(ctx, assignment.ID, groupID)
	}

	newAssignment := &domain.GroupAssignment{
		EmployeeID:     employeeID,
		GroupID:        groupID,
		AssignmentType: domain.AssignmentTypePermanent,
	}
	return s.assignments.Create(ctx, newAssignment)
}

func generateTemporaryPassword() string {
	return uuid.NewString()[:8]
}

func (s *EmployeeService) createDefaultContract(ctx context.Context, employeeID int64, weeklyHours float64) error {
	contract := &domain.EmployeeContract{
		EmployeeID:  employeeID,
		ValidFrom:   firstOfMonth(time.Now()),
		WeeklyHours: weeklyHours,
		Workdays:    defaultWorkdays(0, weeklyHours),
	}
	return s.employees.CreateContract(ctx, contract)
}

func firstOfMonth(date time.Time) time.Time {
	if date.IsZero() {
		date = time.Now()
	}
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func defaultWorkdays(contractID int64, weeklyHours float64) []domain.EmployeeContractWorkday {
	wholeHours := int(math.Floor(weeklyHours))
	baseMinutes := (wholeHours / 5) * 60
	remainderDays := wholeHours % 5
	fractionalMinutes := int(math.Round((weeklyHours - float64(wholeHours)) * 60))

	workdays := make([]domain.EmployeeContractWorkday, 0, 5)
	for weekday := 1; weekday <= 5; weekday++ {
		plannedMinutes := baseMinutes
		if weekday > 5-remainderDays {
			plannedMinutes += 60
		}
		if weekday == 5 {
			plannedMinutes += fractionalMinutes
		}
		workdays = append(workdays, domain.EmployeeContractWorkday{
			ContractID:     contractID,
			Weekday:        weekday,
			PlannedMinutes: plannedMinutes,
		})
	}
	return workdays
}

func mapContractWorkdays(contractID int64, input []EmployeeContractWorkdayInput) []domain.EmployeeContractWorkday {
	workdays := make([]domain.EmployeeContractWorkday, 0, len(input))
	for _, day := range input {
		workdays = append(workdays, domain.EmployeeContractWorkday{
			ContractID:     contractID,
			Weekday:        day.Weekday,
			PlannedMinutes: day.PlannedMinutes,
		})
	}
	return workdays
}

func normalizeOptionalString(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}
