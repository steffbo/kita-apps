package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// FeeScheduleService manages versions of the fee regulation.
//
// Versions that are already in effect (valid_from <= today) are read-only:
// fees, Einstufungen and their monthly tables are computed from them, so
// changing them would silently rewrite history. Changes are made by adding a
// new version that starts on the first of a future month.
type FeeScheduleService struct {
	repo repository.FeeScheduleRepository
}

// NewFeeScheduleService creates a new fee schedule service.
func NewFeeScheduleService(repo repository.FeeScheduleRepository) *FeeScheduleService {
	return &FeeScheduleService{repo: repo}
}

// FeeScheduleInput is the editable part of a fee schedule version.
type FeeScheduleInput struct {
	ValidFrom time.Time
	Name      string
	Config    domain.FeeScheduleConfig
}

// List returns all versions ordered by valid_from.
func (s *FeeScheduleService) List(ctx context.Context) (domain.FeeSchedules, error) {
	return s.repo.List(ctx)
}

// IsEditable reports whether a version has not started yet.
func (s *FeeScheduleService) IsEditable(schedule domain.FeeSchedule) bool {
	return schedule.ValidFrom.After(util.Today())
}

// Create adds a new future version.
func (s *FeeScheduleService) Create(ctx context.Context, input FeeScheduleInput) (*domain.FeeSchedule, error) {
	if err := s.validate(&input); err != nil {
		return nil, err
	}
	schedule := &domain.FeeSchedule{ValidFrom: input.ValidFrom, Name: input.Name, Config: input.Config}
	if err := s.repo.Create(ctx, schedule); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, fmt.Errorf("%w: ab diesem Datum gibt es bereits eine Version", ErrConflict)
		}
		return nil, err
	}
	return schedule, nil
}

// Update changes a version that has not started yet.
func (s *FeeScheduleService) Update(ctx context.Context, id uuid.UUID, input FeeScheduleInput) (*domain.FeeSchedule, error) {
	existing, err := s.editable(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validate(&input); err != nil {
		return nil, err
	}
	existing.ValidFrom = input.ValidFrom
	existing.Name = input.Name
	existing.Config = input.Config
	if err := s.repo.Update(ctx, existing); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, fmt.Errorf("%w: ab diesem Datum gibt es bereits eine Version", ErrConflict)
		}
		return nil, err
	}
	return existing, nil
}

// Delete removes a version that has not started yet.
func (s *FeeScheduleService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.editable(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *FeeScheduleService) editable(ctx context.Context, id uuid.UUID) (*domain.FeeSchedule, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !s.IsEditable(*existing) {
		return nil, fmt.Errorf("%w: Versionen, die bereits gelten, können nicht mehr geändert werden", ErrConflict)
	}
	return existing, nil
}

func (s *FeeScheduleService) validate(input *FeeScheduleInput) error {
	var problems []string
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		problems = append(problems, "Name fehlt")
	}
	if input.ValidFrom.Day() != 1 {
		problems = append(problems, "Gültig ab muss ein Monatserster sein")
	}
	if !input.ValidFrom.After(util.Today()) {
		problems = append(problems, "Gültig ab muss in der Zukunft liegen")
	}
	input.Config.Normalize()
	if err := input.Config.Validate(); err != nil {
		problems = append(problems, err.Error())
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidInput, strings.Join(problems, "; "))
	}
	return nil
}
