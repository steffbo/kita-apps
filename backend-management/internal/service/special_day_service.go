package service

import (
	"context"
	"fmt"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// SpecialDayService handles special day operations.
type SpecialDayService struct {
	repo repository.SpecialDayRepository
}

// NewSpecialDayService creates a new SpecialDayService.
func NewSpecialDayService(repo repository.SpecialDayRepository) *SpecialDayService {
	return &SpecialDayService{repo: repo}
}

// List retrieves special days for a year.
func (s *SpecialDayService) List(ctx context.Context, year int, includeHolidays bool) ([]domain.SpecialDay, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)

	if includeHolidays {
		return s.repo.List(ctx, start, end)
	}

	all, err := s.repo.List(ctx, start, end)
	if err != nil {
		return nil, err
	}

	filtered := make([]domain.SpecialDay, 0, len(all))
	for _, day := range all {
		if day.DayType != domain.SpecialDayTypeHoliday {
			filtered = append(filtered, day)
		}
	}

	return filtered, nil
}

// Holidays retrieves holidays for a year.
func (s *SpecialDayService) Holidays(ctx context.Context, year int) ([]domain.SpecialDay, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	return s.repo.ListByType(ctx, start, end, domain.SpecialDayTypeHoliday)
}

// CreateSpecialDayInput represents input for creating a special day.
type CreateSpecialDayInput struct {
	Date       time.Time
	EndDate    *time.Time
	Name       string
	DayType    domain.SpecialDayType
	AffectsAll bool
	Notes      *string
}

// Create creates a special day.
func (s *SpecialDayService) Create(ctx context.Context, input CreateSpecialDayInput) (*domain.SpecialDay, error) {
	day := &domain.SpecialDay{
		Date:       input.Date,
		EndDate:    input.EndDate,
		Name:       input.Name,
		DayType:    input.DayType,
		AffectsAll: input.AffectsAll,
		Notes:      input.Notes,
	}

	if err := s.repo.Create(ctx, day); err != nil {
		return nil, err
	}
	return day, nil
}

// Update updates a special day.
func (s *SpecialDayService) Update(ctx context.Context, id int64, input CreateSpecialDayInput) (*domain.SpecialDay, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Besonderer Tag mit ID %d nicht gefunden", id))
	}

	day := &domain.SpecialDay{
		ID:         id,
		Date:       input.Date,
		EndDate:    input.EndDate,
		Name:       input.Name,
		DayType:    input.DayType,
		AffectsAll: input.AffectsAll,
		Notes:      input.Notes,
	}

	updated, err := s.repo.Update(ctx, day)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete deletes a special day.
func (s *SpecialDayService) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return NewNotFound(fmt.Sprintf("Besonderer Tag mit ID %d nicht gefunden", id))
	}
	return s.repo.Delete(ctx, id)
}
