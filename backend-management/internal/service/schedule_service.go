package service

import (
	"context"
	"fmt"
	"sort"
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
	EmployeeID         int64
	Date               time.Time
	StartTime          *time.Time
	EndTime            *time.Time
	BreakMinutes       int
	GroupID            *int64
	EntryType          domain.ScheduleEntryType
	ShiftKind          domain.ShiftKind
	Notes              *string
	Segments           []domain.ScheduleEntrySegment
	OverrideBlockedDay bool
}

// UpdateScheduleEntryInput represents input for updating a schedule entry.
type UpdateScheduleEntryInput struct {
	Date               *time.Time
	StartTime          *time.Time
	EndTime            *time.Time
	BreakMinutes       *int
	GroupID            *int64
	EntryType          *domain.ScheduleEntryType
	ShiftKind          *domain.ShiftKind
	Notes              *string
	Segments           *[]domain.ScheduleEntrySegment
	OverrideBlockedDay bool
}

// CreateScheduleRequestInput represents input for creating a schedule request.
type CreateScheduleRequestInput struct {
	EmployeeID  int64
	Date        time.Time
	StartTime   *time.Time
	EndTime     *time.Time
	RequestType domain.ScheduleRequestType
	Text        string
}

// UpdateScheduleRequestInput represents input for updating a schedule request.
type UpdateScheduleRequestInput struct {
	Date        *time.Time
	StartTime   *time.Time
	EndTime     *time.Time
	RequestType *domain.ScheduleRequestType
	Text        *string
	Status      *domain.ScheduleRequestStatus
}

// TimeSuggestionInput represents input for suggested schedule times.
type TimeSuggestionInput struct {
	EmployeeID int64
	Date       time.Time
	ShiftKind  domain.ShiftKind
	StartTime  *time.Time
}

// TimeSuggestion contains calculated schedule times for a contract day.
type TimeSuggestion struct {
	StartTime      *time.Time
	EndTime        *time.Time
	BreakMinutes   int
	PlannedMinutes int
	IsBlocked      bool
	ContractID     *int64
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
	Requests    []domain.ScheduleRequest
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
	requests, err := s.schedules.ListRequests(ctx, monday, sunday, nil)
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
		Requests:    requests,
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
	shiftKind := input.ShiftKind
	if shiftKind == "" {
		shiftKind = domain.ShiftKindManual
	}

	if entryType == domain.ScheduleEntryTypeWork {
		if err := s.validateWorkday(ctx, input.EmployeeID, input.Date, input.OverrideBlockedDay); err != nil {
			return nil, err
		}
		if err := s.validateSegments(ctx, input.StartTime, input.EndTime, input.Segments); err != nil {
			return nil, err
		}
	} else if len(input.Segments) > 0 {
		return nil, NewBadRequest("Segmente sind nur für Arbeitseinträge erlaubt")
	}

	entry := &domain.ScheduleEntry{
		EmployeeID:   input.EmployeeID,
		Date:         input.Date,
		StartTime:    input.StartTime,
		EndTime:      input.EndTime,
		BreakMinutes: input.BreakMinutes,
		GroupID:      input.GroupID,
		EntryType:    entryType,
		ShiftKind:    shiftKind,
		Notes:        input.Notes,
		Segments:     input.Segments,
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
	if input.ShiftKind != nil {
		entry.ShiftKind = *input.ShiftKind
	}
	if input.Notes != nil {
		entry.Notes = input.Notes
	}
	if entry.ShiftKind == "" {
		entry.ShiftKind = domain.ShiftKindManual
	}
	if entry.EntryType == domain.ScheduleEntryTypeWork {
		if err := s.validateWorkday(ctx, entry.EmployeeID, entry.Date, input.OverrideBlockedDay); err != nil {
			return nil, err
		}
	}
	if input.Segments != nil {
		if entry.EntryType != domain.ScheduleEntryTypeWork && len(*input.Segments) > 0 {
			return nil, NewBadRequest("Segmente sind nur für Arbeitseinträge erlaubt")
		}
		if err := s.validateSegments(ctx, entry.StartTime, entry.EndTime, *input.Segments); err != nil {
			return nil, err
		}
		entry.Segments = *input.Segments
		entry.SegmentsChanged = true
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

// SuggestTimes calculates schedule times from the active contract for the date.
func (s *ScheduleService) SuggestTimes(ctx context.Context, input TimeSuggestionInput) (*TimeSuggestion, error) {
	if _, err := s.employees.GetByID(ctx, input.EmployeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", input.EmployeeID))
	}

	contract, err := s.employees.GetContractForDate(ctx, input.EmployeeID, input.Date)
	if err != nil {
		return &TimeSuggestion{IsBlocked: true}, nil
	}

	weekday := int(input.Date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	workday, ok := findContractWorkday(contract, weekday)
	if !ok {
		id := contract.ID
		return &TimeSuggestion{IsBlocked: true, ContractID: &id}, nil
	}

	breakMinutes := plannedBreakMinutes(workday.PlannedMinutes)
	shiftKind := input.ShiftKind
	if shiftKind == "" {
		shiftKind = domain.ShiftKindEarly
	}

	var start, end time.Time
	switch shiftKind {
	case domain.ShiftKindLate:
		end = closingTime(input.Date)
		start = end.Add(-time.Duration(workday.PlannedMinutes+breakMinutes) * time.Minute)
	default:
		if input.StartTime != nil {
			start = normalizeClockTime(*input.StartTime)
		} else {
			start = time.Date(2000, 1, 1, 7, 0, 0, 0, time.UTC)
		}
		end = start.Add(time.Duration(workday.PlannedMinutes+breakMinutes) * time.Minute)
	}

	id := contract.ID
	return &TimeSuggestion{
		StartTime:      &start,
		EndTime:        &end,
		BreakMinutes:   breakMinutes,
		PlannedMinutes: workday.PlannedMinutes,
		IsBlocked:      false,
		ContractID:     &id,
	}, nil
}

func (s *ScheduleService) validateWorkday(ctx context.Context, employeeID int64, date time.Time, override bool) error {
	if override {
		return nil
	}
	contract, err := s.employees.GetContractForDate(ctx, employeeID, date)
	if err != nil {
		return NewBadRequest("Für diesen Tag ist kein gültiger Vertrag hinterlegt")
	}
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	if _, ok := findContractWorkday(contract, weekday); !ok {
		return NewBadRequest("Dieser Wochentag ist für den Mitarbeiter blockiert")
	}
	return nil
}

func findContractWorkday(contract *domain.EmployeeContract, weekday int) (domain.EmployeeContractWorkday, bool) {
	for _, workday := range contract.Workdays {
		if workday.Weekday == weekday {
			return workday, true
		}
	}
	return domain.EmployeeContractWorkday{}, false
}

func plannedBreakMinutes(plannedMinutes int) int {
	if plannedMinutes > 6*60 {
		return 30
	}
	return 0
}

func closingTime(date time.Time) time.Time {
	if date.Weekday() == time.Friday {
		return time.Date(2000, 1, 1, 16, 0, 0, 0, time.UTC)
	}
	return time.Date(2000, 1, 1, 16, 30, 0, 0, time.UTC)
}

func normalizeClockTime(value time.Time) time.Time {
	return time.Date(2000, 1, 1, value.Hour(), value.Minute(), value.Second(), 0, time.UTC)
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

// ListRequests retrieves schedule requests for a date range.
func (s *ScheduleService) ListRequests(ctx context.Context, startDate, endDate time.Time, employeeID *int64) ([]domain.ScheduleRequest, error) {
	return s.schedules.ListRequests(ctx, startDate, endDate, employeeID)
}

// CreateRequest creates a non-working schedule request.
func (s *ScheduleService) CreateRequest(ctx context.Context, input CreateScheduleRequestInput, actorID int64, isAdmin bool) (*domain.ScheduleRequest, error) {
	if !isAdmin && input.EmployeeID != actorID {
		return nil, NewForbidden("Wünsche können nur für den eigenen Account angelegt werden")
	}
	if _, err := s.employees.GetByID(ctx, input.EmployeeID); err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", input.EmployeeID))
	}
	requestType := input.RequestType
	if requestType == "" {
		requestType = domain.ScheduleRequestTypeWish
	}
	if err := validateRequestTimeRange(input.StartTime, input.EndTime); err != nil {
		return nil, err
	}
	if input.Text == "" {
		return nil, NewBadRequest("Text ist erforderlich")
	}

	request := &domain.ScheduleRequest{
		EmployeeID:  input.EmployeeID,
		Date:        input.Date,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		RequestType: requestType,
		Text:        input.Text,
		Status:      domain.ScheduleRequestStatusOpen,
	}
	if err := s.schedules.CreateRequest(ctx, request); err != nil {
		return nil, err
	}
	return s.schedules.GetRequestByID(ctx, request.ID)
}

// UpdateRequest updates a non-working schedule request.
func (s *ScheduleService) UpdateRequest(ctx context.Context, id int64, input UpdateScheduleRequestInput, actorID int64, isAdmin bool) (*domain.ScheduleRequest, error) {
	request, err := s.schedules.GetRequestByID(ctx, id)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Wunsch mit ID %d nicht gefunden", id))
	}
	if !isAdmin && request.EmployeeID != actorID {
		return nil, NewForbidden("Wünsche können nur für den eigenen Account geändert werden")
	}
	if !isAdmin && request.Status != domain.ScheduleRequestStatusOpen {
		return nil, NewForbidden("Erledigte Wünsche können nur von der Leitung geändert werden")
	}
	if !isAdmin && input.Status != nil && *input.Status != request.Status {
		return nil, NewForbidden("Nur die Leitung kann den Status ändern")
	}

	if input.Date != nil {
		request.Date = *input.Date
	}
	if input.StartTime != nil {
		request.StartTime = input.StartTime
	}
	if input.EndTime != nil {
		request.EndTime = input.EndTime
	}
	if input.RequestType != nil {
		request.RequestType = *input.RequestType
	}
	if input.Text != nil {
		request.Text = *input.Text
	}
	if input.Status != nil {
		request.Status = *input.Status
	}
	if err := validateRequestTimeRange(request.StartTime, request.EndTime); err != nil {
		return nil, err
	}
	if request.Text == "" {
		return nil, NewBadRequest("Text ist erforderlich")
	}

	return s.schedules.UpdateRequest(ctx, request)
}

// DeleteRequest deletes a non-working schedule request.
func (s *ScheduleService) DeleteRequest(ctx context.Context, id int64, actorID int64, isAdmin bool) error {
	request, err := s.schedules.GetRequestByID(ctx, id)
	if err != nil {
		return NewNotFound(fmt.Sprintf("Wunsch mit ID %d nicht gefunden", id))
	}
	if !isAdmin && request.EmployeeID != actorID {
		return NewForbidden("Wünsche können nur für den eigenen Account gelöscht werden")
	}
	if !isAdmin && request.Status != domain.ScheduleRequestStatusOpen {
		return NewForbidden("Erledigte Wünsche können nur von der Leitung gelöscht werden")
	}
	return s.schedules.DeleteRequest(ctx, id)
}

func (s *ScheduleService) validateSegments(ctx context.Context, startTime, endTime *time.Time, segments []domain.ScheduleEntrySegment) error {
	if len(segments) == 0 {
		return nil
	}
	if startTime == nil || endTime == nil {
		return NewBadRequest("Segmente benötigen Beginn und Ende des Arbeitseintrags")
	}

	shiftStart := clockMinutes(*startTime)
	shiftEnd := clockMinutes(*endTime)
	sortedSegments := append([]domain.ScheduleEntrySegment(nil), segments...)
	sort.SliceStable(sortedSegments, func(i, j int) bool {
		return clockMinutes(sortedSegments[i].StartTime) < clockMinutes(sortedSegments[j].StartTime)
	})

	lastEnd := -1
	for index, segment := range sortedSegments {
		if _, err := s.groups.GetByID(ctx, segment.GroupID); err != nil {
			return NewNotFound(fmt.Sprintf("Gruppe mit ID %d nicht gefunden", segment.GroupID))
		}
		segmentStart := clockMinutes(segment.StartTime)
		segmentEnd := clockMinutes(segment.EndTime)
		if segmentEnd <= segmentStart {
			return NewBadRequest("Segment-Ende muss nach Segment-Beginn liegen")
		}
		if segmentStart < shiftStart || segmentEnd > shiftEnd {
			return NewBadRequest("Segmentzeiten müssen innerhalb der Arbeitszeit liegen")
		}
		if index > 0 && segmentStart < lastEnd {
			return NewBadRequest("Segmente dürfen sich nicht überlappen")
		}
		lastEnd = segmentEnd
	}
	return nil
}

func validateRequestTimeRange(startTime, endTime *time.Time) error {
	if startTime != nil && endTime != nil && clockMinutes(*endTime) <= clockMinutes(*startTime) {
		return NewBadRequest("Ende muss nach Beginn liegen")
	}
	return nil
}

func clockMinutes(value time.Time) int {
	return value.Hour()*60 + value.Minute()
}
