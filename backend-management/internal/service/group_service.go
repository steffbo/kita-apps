package service

import (
	"context"
	"fmt"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// GroupService handles group operations.
type GroupService struct {
	groups      repository.GroupRepository
	assignments repository.GroupAssignmentRepository
	employees   repository.EmployeeRepository
}

// NewGroupService creates a new GroupService.
func NewGroupService(
	groups repository.GroupRepository,
	assignments repository.GroupAssignmentRepository,
	employees repository.EmployeeRepository,
) *GroupService {
	return &GroupService{
		groups:      groups,
		assignments: assignments,
		employees:   employees,
	}
}

// List retrieves all groups.
func (s *GroupService) List(ctx context.Context) ([]domain.Group, error) {
	return s.groups.List(ctx)
}

// Get retrieves a group with members.
func (s *GroupService) Get(ctx context.Context, id int64) (*domain.Group, []domain.GroupAssignment, error) {
	group, err := s.groups.GetByID(ctx, id)
	if err != nil {
		return nil, nil, NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", id))
	}

	assignments, err := s.assignments.ListByGroupID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return group, assignments, nil
}

// CreateGroupInput represents input for creating a group.
type CreateGroupInput struct {
	Name        string
	Description *string
	Color       *string
}

// Create creates a new group.
func (s *GroupService) Create(ctx context.Context, input CreateGroupInput) (*domain.Group, error) {
	color := "#3B82F6"
	if input.Color != nil {
		color = *input.Color
	}

	group := &domain.Group{
		Name:        input.Name,
		Description: input.Description,
		Color:       color,
	}

	if err := s.groups.Create(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}

// Update updates a group.
func (s *GroupService) Update(ctx context.Context, id int64, input CreateGroupInput) (*domain.Group, error) {
	group, err := s.groups.GetByID(ctx, id)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", id))
	}

	group.Name = input.Name
	if input.Description != nil {
		group.Description = input.Description
	}
	if input.Color != nil {
		group.Color = *input.Color
	}

	updated, err := s.groups.Update(ctx, group)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete deletes a group.
func (s *GroupService) Delete(ctx context.Context, id int64) error {
	if _, err := s.groups.GetByID(ctx, id); err != nil {
		return NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", id))
	}

	if err := s.assignments.DeleteByGroupID(ctx, id); err != nil {
		return err
	}

	return s.groups.Delete(ctx, id)
}

// GroupAssignmentInput represents input for assignments.
type GroupAssignmentInput struct {
	EmployeeID     int64
	AssignmentType domain.AssignmentType
}

// Assignments retrieves assignments for a group.
func (s *GroupService) Assignments(ctx context.Context, groupID int64) ([]domain.GroupAssignment, error) {
	if _, err := s.groups.GetByID(ctx, groupID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", groupID))
	}
	return s.assignments.ListByGroupID(ctx, groupID)
}

// UpdateAssignments replaces assignments for a group.
func (s *GroupService) UpdateAssignments(ctx context.Context, groupID int64, inputs []GroupAssignmentInput) ([]domain.GroupAssignment, error) {
	group, err := s.groups.GetByID(ctx, groupID)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", groupID))
	}

	if err := s.assignments.DeleteByGroupID(ctx, groupID); err != nil {
		return nil, err
	}

	assignments := make([]domain.GroupAssignment, 0, len(inputs))
	for _, input := range inputs {
		if _, err := s.employees.GetByID(ctx, input.EmployeeID); err != nil {
			return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", input.EmployeeID))
		}

		assignment := &domain.GroupAssignment{
			EmployeeID:     input.EmployeeID,
			GroupID:        group.ID,
			AssignmentType: input.AssignmentType,
		}
		if err := s.assignments.Create(ctx, assignment); err != nil {
			return nil, err
		}
		assignments = append(assignments, *assignment)
	}

	return assignments, nil
}
