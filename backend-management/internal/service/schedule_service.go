package service

import (
	"context"
	"fmt"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// ScheduleService handles schedule operations.
type ScheduleService struct {
	schedules   repository.ScheduleRepository
	employees   repository.EmployeeRepository
	groups      repository.GroupRepository
	specialDays repository.SpecialDayRepository
}

// NewScheduleService creates a new ScheduleService.
func NewScheduleService(
	schedules repository.ScheduleRepository,
	employees repository.EmployeeRepository,
	groups repository.GroupRepository,
	specialDays repository.SpecialDayRepository,
) *ScheduleService {
	return &ScheduleService{
		schedules:   schedules,
		employees:   employees,
		groups:      groups,
		specialDays: specialDays,
	}
}

// CreateScheduleEntryInput represents input for creating a schedule entry.
type CreateScheduleEntryInput struct {
	EmployeeID   int64
	Date         time.Time
	StartTime    *time.Time
	EndTime      *time.Time
	BreakMinutes int
	GroupID      *int64
	EntryType    domain.ScheduleEntryType
	Notes        *string
}

// UpdateScheduleEntryInput represents input for updating a schedule entry.
type UpdateScheduleEntryInput struct {
	Date         *time.Time
	StartTime    *time.Time
	EndTime      *time.Time
	BreakMinutes *int
	GroupID      *int64
	EntryType    *domain.ScheduleEntryType
	Notes        *string
}

// List retrieves schedule entries for a date range.
func (s *ScheduleService) List(ctx context.Context, startDate, endDate time.Time, employeeID, groupID *int64) ([]domain.ScheduleEntry, error) {
	return s.schedules.List(ctx, startDate, endDate, employeeID, groupID)
}

// WeekSchedule represents a weekly schedule response.
type WeekSchedule struct {
	WeekStart   time.Time
	WeekEnd     time.Time
	Days        []DaySchedule
	SpecialDays []domain.SpecialDay
}

// DaySchedule represents entries for a day.
type DaySchedule struct {
	Date        time.Time
	DayOfWeek   time.Weekday
	IsHoliday   bool
	HolidayName *string
	Entries     []domain.ScheduleEntry
	ByGroup     map[string][]domain.ScheduleEntry
}

// Week retrieves a weekly schedule view.
func (s *ScheduleService) Week(ctx context.Context, weekStart time.Time) (*WeekSchedule, error) {
	monday := weekStart
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, -1)
	}
	sunday := monday.AddDate(0, 0, 6)

	entries, err := s.schedules.List(ctx, monday, sunday, nil, nil)
	if err != nil {
		return nil, err
	}

	specialDays, err := s.specialDays.List(ctx, monday, sunday)
	if err != nil {
		return nil, err
	}

	dayMap := make(map[string][]domain.ScheduleEntry)
	for _, entry := range entries {
		key := entry.Date.Format(dateLayout)
		dayMap[key] = append(dayMap[key], entry)
	}

	specialByDate := make(map[string]domain.SpecialDay)
	for _, day := range specialDays {
		key := day.Date.Format(dateLayout)
		specialByDate[key] = day
		if day.EndDate != nil {
			current := day.Date
			for current.Before(*day.EndDate) {
				current = current.AddDate(0, 0, 1)
				specialByDate[current.Format(dateLayout)] = day
			}
		}
	}

	days := make([]DaySchedule, 0, 7)
	for i := 0; i < 7; i++ {
		current := monday.AddDate(0, 0, i)
		key := current.Format(dateLayout)

		entriesForDay := dayMap[key]
		byGroup := make(map[string][]domain.ScheduleEntry)
		for _, entry := range entriesForDay {
			if entry.GroupID != nil {
				gid := fmt.Sprintf("%d", *entry.GroupID)
				byGroup[gid] = append(byGroup[gid], entry)
			}
		}

		daily := DaySchedule{
			Date:      current,
			DayOfWeek: current.Weekday(),
			Entries:   entriesForDay,
			ByGroup:   byGroup,
		}

		if special, ok := specialByDate[key]; ok {
			if special.DayType == domain.SpecialDayTypeHoliday || special.DayType == domain.SpecialDayTypeClosure {
				daily.IsHoliday = true
				name := special.Name
				daily.HolidayName = &name
			}
		}

		days = append(days, daily)
	}

	return &WeekSchedule{
		WeekStart:   monday,
		WeekEnd:     sunday,
		Days:        days,
		SpecialDays: specialDays,
	}, nil
}

// Create creates a schedule entry.
func (s *ScheduleService) Create(ctx context.Context, input CreateScheduleEntryInput) (*domain.ScheduleEntry, error) {
	if _, err := s.employees.GetByID(ctx, input.EmployeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", input.EmployeeID))
	}

	if input.GroupID != nil {
		if _, err := s.groups.GetByID(ctx, *input.GroupID); err != nil {
			return nil, NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", *input.GroupID))
		}
	}

	entryType := input.EntryType
	if entryType == "" {
		entryType = domain.ScheduleEntryTypeWork
	}

	entry := &domain.ScheduleEntry{
		EmployeeID:   input.EmployeeID,
		Date:         input.Date,
		StartTime:    input.StartTime,
		EndTime:      input.EndTime,
		BreakMinutes: input.BreakMinutes,
		GroupID:      input.GroupID,
		EntryType:    entryType,
		Notes:        input.Notes,
	}

	if err := s.schedules.Create(ctx, entry); err != nil {
		return nil, err
	}

	if entryType == domain.ScheduleEntryTypeVacation {
		if err := s.employees.AdjustRemainingVacationDays(ctx, input.EmployeeID, -1); err != nil {
			return nil, err
		}
	}

	return s.schedules.GetByID(ctx, entry.ID)
}

// BulkCreate creates multiple schedule entries.
func (s *ScheduleService) BulkCreate(ctx context.Context, inputs []CreateScheduleEntryInput) ([]domain.ScheduleEntry, error) {
	entries := make([]domain.ScheduleEntry, 0, len(inputs))
	for _, input := range inputs {
		entry, err := s.Create(ctx, input)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, nil
}

// Update updates a schedule entry.
func (s *ScheduleService) Update(ctx context.Context, id int64, input UpdateScheduleEntryInput) (*domain.ScheduleEntry, error) {
	entry, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Dienstplan-Eintrag mit ID %d nicht gefunden", id))
	}

	oldType := entry.EntryType

	if input.GroupID != nil {
		if _, err := s.groups.GetByID(ctx, *input.GroupID); err != nil {
			return nil, NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", *input.GroupID))
		}
		entry.GroupID = input.GroupID
	}
	if input.Date != nil {
		entry.Date = *input.Date
	}
	if input.StartTime != nil {
		entry.StartTime = input.StartTime
	}
	if input.EndTime != nil {
		entry.EndTime = input.EndTime
	}
	if input.BreakMinutes != nil {
		entry.BreakMinutes = *input.BreakMinutes
	}
	if input.EntryType != nil {
		entry.EntryType = *input.EntryType
	}
	if input.Notes != nil {
		entry.Notes = input.Notes
	}

	updated, err := s.schedules.Update(ctx, entry)
	if err != nil {
		return nil, err
	}

	if input.EntryType != nil && oldType != *input.EntryType {
		if oldType == domain.ScheduleEntryTypeVacation && *input.EntryType != domain.ScheduleEntryTypeVacation {
			if err := s.employees.AdjustRemainingVacationDays(ctx, entry.EmployeeID, 1); err != nil {
				return nil, err
			}
		} else if oldType != domain.ScheduleEntryTypeVacation && *input.EntryType == domain.ScheduleEntryTypeVacation {
			if err := s.employees.AdjustRemainingVacationDays(ctx, entry.EmployeeID, -1); err != nil {
				return nil, err
			}
		}
	}

	return updated, nil
}

// Delete deletes a schedule entry.
func (s *ScheduleService) Delete(ctx context.Context, id int64) error {
	entry, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return NewNotFound(fmt.Sprintf("Dienstplan-Eintrag mit ID %d nicht gefunden", id))
	}

	if entry.EntryType == domain.ScheduleEntryTypeVacation {
		if err := s.employees.AdjustRemainingVacationDays(ctx, entry.EmployeeID, 1); err != nil {
			return err
		}
	}

	return s.schedules.Delete(ctx, id)
}
