package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// EmployeeWithGroup represents an employee and their primary group.
type EmployeeWithGroup struct {
	Employee       domain.Employee
	PrimaryGroup   *domain.Group
	PrimaryGroupID *int64
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
		result = append(result, EmployeeWithGroup{
			Employee:       emp,
			PrimaryGroup:   primaryGroup,
			PrimaryGroupID: primaryGroupID,
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

	return &EmployeeWithGroup{
		Employee:       *emp,
		PrimaryGroup:   primaryGroup,
		PrimaryGroupID: primaryGroupID,
	}, nil
}

// CreateEmployeeInput represents input for creating an employee.
type CreateEmployeeInput struct {
	Email               string
	FirstName           string
	LastName            string
	Role                domain.EmployeeRole
	WeeklyHours         float64
	VacationDaysPerYear int
	PrimaryGroupID      *int64
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

	if input.PrimaryGroupID != nil && *input.PrimaryGroupID > 0 {
		if err := s.setPrimaryGroup(ctx, employee.ID, *input.PrimaryGroupID); err != nil {
			return nil, err
		}
	}

	log.Info().Str("email", employee.Email).Str("password", tempPassword).Msg("temporary password generated")

	return s.Get(ctx, employee.ID)
}

// UpdateEmployeeInput represents input for updating an employee.
type UpdateEmployeeInput struct {
	Email                 *string
	FirstName             *string
	LastName              *string
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

// Delete deactivates an employee.
func (s *EmployeeService) Delete(ctx context.Context, id int64) error {
	if _, err := s.employees.GetByID(ctx, id); err != nil {
		return NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", id))
	}
	return s.employees.Deactivate(ctx, id)
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
