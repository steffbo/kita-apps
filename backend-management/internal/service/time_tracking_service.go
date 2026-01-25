package service

import (
	"context"
	"fmt"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// TimeTrackingService handles time tracking operations.
type TimeTrackingService struct {
	timeEntries repository.TimeEntryRepository
	employees   repository.EmployeeRepository
	schedules   repository.ScheduleRepository
}

// NewTimeTrackingService creates a new TimeTrackingService.
func NewTimeTrackingService(
	timeEntries repository.TimeEntryRepository,
	employees repository.EmployeeRepository,
	schedules repository.ScheduleRepository,
) *TimeTrackingService {
	return &TimeTrackingService{
		timeEntries: timeEntries,
		employees:   employees,
		schedules:   schedules,
	}
}

// ClockIn clocks an employee in.
func (s *TimeTrackingService) ClockIn(ctx context.Context, employeeID int64, notes *string) (*domain.TimeEntry, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}

	openEntries, err := s.timeEntries.ListOpenByEmployeeID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if len(openEntries) > 0 {
		return nil, NewBadRequest("Sie sind bereits eingestempelt")
	}

	entry := &domain.TimeEntry{
		EmployeeID:   employeeID,
		Date:         time.Now(),
		ClockIn:      time.Now(),
		BreakMinutes: 0,
		EntryType:    domain.TimeEntryTypeWork,
		Notes:        notes,
	}

	if err := s.timeEntries.Create(ctx, entry); err != nil {
		return nil, err
	}

	return s.timeEntries.GetByID(ctx, entry.ID)
}

// ClockOut clocks an employee out.
func (s *TimeTrackingService) ClockOut(ctx context.Context, employeeID int64, breakMinutes *int, notes *string) (*domain.TimeEntry, error) {
	openEntries, err := s.timeEntries.ListOpenByEmployeeID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if len(openEntries) == 0 {
		return nil, NewBadRequest("Sie sind nicht eingestempelt")
	}

	entry := openEntries[0]
	entry.ClockOut = ptrTime(time.Now())
	if breakMinutes != nil {
		entry.BreakMinutes = *breakMinutes
	}
	if notes != nil {
		entry.Notes = notes
	}

	updated, err := s.timeEntries.Update(ctx, &entry)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Current returns the current open time entry.
func (s *TimeTrackingService) Current(ctx context.Context, employeeID int64) (*domain.TimeEntry, error) {
	openEntries, err := s.timeEntries.ListOpenByEmployeeID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if len(openEntries) == 0 {
		return nil, nil
	}
	return &openEntries[0], nil
}

// List retrieves time entries for a date range.
func (s *TimeTrackingService) List(ctx context.Context, startDate, endDate time.Time, employeeID int64) ([]domain.TimeEntry, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}
	return s.timeEntries.List(ctx, startDate, endDate, &employeeID)
}

// CreateTimeEntryInput represents input for creating a time entry.
type CreateTimeEntryInput struct {
	EmployeeID   int64
	Date         time.Time
	ClockIn      time.Time
	ClockOut     time.Time
	BreakMinutes int
	EntryType    domain.TimeEntryType
	Notes        *string
	EditReason   *string
}

// Create creates a time entry.
func (s *TimeTrackingService) Create(ctx context.Context, input CreateTimeEntryInput) (*domain.TimeEntry, error) {
	if _, err := s.employees.GetByID(ctx, input.EmployeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", input.EmployeeID))
	}

	entryType := input.EntryType
	if entryType == "" {
		entryType = domain.TimeEntryTypeWork
	}

	entry := &domain.TimeEntry{
		EmployeeID:   input.EmployeeID,
		Date:         input.Date,
		ClockIn:      input.ClockIn,
		ClockOut:     &input.ClockOut,
		BreakMinutes: input.BreakMinutes,
		EntryType:    entryType,
		Notes:        input.Notes,
		EditReason:   input.EditReason,
	}

	if err := s.timeEntries.Create(ctx, entry); err != nil {
		return nil, err
	}

	return s.timeEntries.GetByID(ctx, entry.ID)
}

// UpdateTimeEntryInput represents input for updating a time entry.
type UpdateTimeEntryInput struct {
	ClockIn      *time.Time
	ClockOut     *time.Time
	BreakMinutes *int
	EntryType    *domain.TimeEntryType
	Notes        *string
	EditReason   *string
}

// Update updates a time entry.
func (s *TimeTrackingService) Update(ctx context.Context, id int64, input UpdateTimeEntryInput, editorID *int64) (*domain.TimeEntry, error) {
	entry, err := s.timeEntries.GetByID(ctx, id)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Zeiteintrag mit ID %d nicht gefunden", id))
	}

	if input.ClockIn != nil {
		entry.ClockIn = *input.ClockIn
	}
	if input.ClockOut != nil {
		entry.ClockOut = input.ClockOut
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
	if input.EditReason != nil {
		entry.EditReason = input.EditReason
	}

	if editorID != nil {
		entry.EditedBy = editorID
		entry.EditedAt = ptrTime(time.Now())
	}

	updated, err := s.timeEntries.Update(ctx, entry)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// Delete deletes a time entry.
func (s *TimeTrackingService) Delete(ctx context.Context, id int64) error {
	if _, err := s.timeEntries.GetByID(ctx, id); err != nil {
		return NewNotFound(fmt.Sprintf("Zeiteintrag mit ID %d nicht gefunden", id))
	}
	return s.timeEntries.Delete(ctx, id)
}

// TimeScheduleComparison represents a comparison between schedule and time entries.
type TimeScheduleComparison struct {
	StartDate time.Time
	EndDate   time.Time
	Entries   []DayComparison
	Summary   ComparisonSummary
}

// DayComparison represents a per-day comparison.
type DayComparison struct {
	Date              time.Time
	ScheduledMinutes  int
	ActualMinutes     int
	DifferenceMinutes int
	Status            string
}

// ComparisonSummary represents comparison totals.
type ComparisonSummary struct {
	TotalScheduledMinutes  int
	TotalActualMinutes     int
	TotalDifferenceMinutes int
	DaysWorked             int
	DaysScheduled          int
}

// Comparison calculates comparison between schedule and time entries.
func (s *TimeTrackingService) Comparison(ctx context.Context, startDate, endDate time.Time, employeeID int64) (*TimeScheduleComparison, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}

	scheduleEntries, err := s.schedules.List(ctx, startDate, endDate, &employeeID, nil)
	if err != nil {
		return nil, err
	}

	timeEntries, err := s.timeEntries.List(ctx, startDate, endDate, &employeeID)
	if err != nil {
		return nil, err
	}

	scheduledMinutes := 0
	scheduledDays := make(map[string]bool)
	for _, entry := range scheduleEntries {
		if entry.EntryType != domain.ScheduleEntryTypeWork {
			continue
		}
		if entry.StartTime != nil && entry.EndTime != nil {
			minutes := int(entry.EndTime.Sub(*entry.StartTime).Minutes()) - entry.BreakMinutes
			if minutes > 0 {
				scheduledMinutes += minutes
			}
		}
		key := entry.Date.Format(dateLayout)
		scheduledDays[key] = true
	}

	actualMinutes := 0
	workedDays := make(map[string]bool)
	for _, entry := range timeEntries {
		if entry.ClockOut == nil {
			continue
		}
		minutes := int(entry.ClockOut.Sub(entry.ClockIn).Minutes()) - entry.BreakMinutes
		if minutes > 0 {
			actualMinutes += minutes
		}
		key := entry.Date.Format(dateLayout)
		workedDays[key] = true
	}

	summary := ComparisonSummary{
		TotalScheduledMinutes:  scheduledMinutes,
		TotalActualMinutes:     actualMinutes,
		TotalDifferenceMinutes: actualMinutes - scheduledMinutes,
		DaysWorked:             len(workedDays),
		DaysScheduled:          len(scheduledDays),
	}

	return &TimeScheduleComparison{
		StartDate: startDate,
		EndDate:   endDate,
		Entries:   []DayComparison{},
		Summary:   summary,
	}, nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
