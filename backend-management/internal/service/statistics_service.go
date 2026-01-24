package service

import (
	"context"
	"fmt"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

// StatisticsService handles reporting.
type StatisticsService struct {
	employees   repository.EmployeeRepository
	schedules   repository.ScheduleRepository
	timeEntries repository.TimeEntryRepository
	groups      repository.GroupRepository
}

// NewStatisticsService creates a new StatisticsService.
func NewStatisticsService(
	employees repository.EmployeeRepository,
	schedules repository.ScheduleRepository,
	timeEntries repository.TimeEntryRepository,
	groups repository.GroupRepository,
) *StatisticsService {
	return &StatisticsService{
		employees:   employees,
		schedules:   schedules,
		timeEntries: timeEntries,
		groups:      groups,
	}
}

// OverviewStatistics represents the overview stats.
type OverviewStatistics struct {
	Month               time.Time
	TotalEmployees      int
	EmployeeStats       []EmployeeStatisticsSummary
	TotalScheduledHours float64
	TotalWorkedHours    float64
	TotalOvertimeHours  float64
	SickDays            int
	VacationDays        int
}

// EmployeeStatisticsSummary represents summary stats for an employee.
type EmployeeStatisticsSummary struct {
	Employee              domain.Employee
	ScheduledHours        float64
	WorkedHours           float64
	OvertimeHours         float64
	RemainingVacationDays float64
}

// EmployeeStatistics represents detailed employee stats.
type EmployeeStatistics struct {
	Employee              domain.Employee
	Month                 time.Time
	ContractedHours       float64
	ScheduledHours        float64
	WorkedHours           float64
	OvertimeHours         float64
	OvertimeBalance       float64
	VacationDaysUsed      int
	VacationDaysRemaining float64
	SickDays              int
	DailyBreakdown        []DayStatistics
}

// DayStatistics represents a daily breakdown.
type DayStatistics struct {
	Date           time.Time
	ScheduledHours float64
	WorkedHours    float64
	EntryType      *domain.ScheduleEntryType
}

// WeeklyStatistics represents weekly stats.
type WeeklyStatistics struct {
	WeekStart           time.Time
	WeekEnd             time.Time
	ByEmployee          []EmployeeWeekSummary
	ByGroup             []GroupWeekSummary
	TotalScheduledHours float64
	TotalWorkedHours    float64
}

// EmployeeWeekSummary represents weekly stats per employee.
type EmployeeWeekSummary struct {
	Employee       domain.Employee
	ScheduledHours float64
	WorkedHours    float64
	DaysWorked     int
}

// GroupWeekSummary represents weekly stats per group.
type GroupWeekSummary struct {
	Group               domain.Group
	TotalScheduledHours float64
	StaffedDays         int
	UnderstaffedDays    int
}

// Overview computes overview statistics for a month.
func (s *StatisticsService) Overview(ctx context.Context, month time.Time) (*OverviewStatistics, error) {
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	employees, err := s.employees.List(ctx, true)
	if err != nil {
		return nil, err
	}

	scheduleEntries, err := s.schedules.List(ctx, startOfMonth, endOfMonth, nil, nil)
	if err != nil {
		return nil, err
	}

	timeEntries, err := s.timeEntries.List(ctx, startOfMonth, endOfMonth, nil)
	if err != nil {
		return nil, err
	}

	scheduleByEmployee := make(map[int64][]domain.ScheduleEntry)
	for _, entry := range scheduleEntries {
		scheduleByEmployee[entry.EmployeeID] = append(scheduleByEmployee[entry.EmployeeID], entry)
	}

	timeByEmployee := make(map[int64][]domain.TimeEntry)
	for _, entry := range timeEntries {
		timeByEmployee[entry.EmployeeID] = append(timeByEmployee[entry.EmployeeID], entry)
	}

	stats := &OverviewStatistics{
		Month:          startOfMonth,
		TotalEmployees: len(employees),
	}

	for _, employee := range employees {
		entries := scheduleByEmployee[employee.ID]
		times := timeByEmployee[employee.ID]

		scheduledHours := calculateScheduledHours(entries)
		workedHours := calculateWorkedHours(times)

		workingDays := countWorkingDays(startOfMonth, endOfMonth)
		expectedMonthlyHours := (employee.WeeklyHours / 5) * float64(workingDays)
		overtimeHours := workedHours - expectedMonthlyHours

		stats.TotalScheduledHours += scheduledHours
		stats.TotalWorkedHours += workedHours
		stats.TotalOvertimeHours += overtimeHours

		for _, entry := range entries {
			switch entry.EntryType {
			case domain.ScheduleEntryTypeSick:
				stats.SickDays++
			case domain.ScheduleEntryTypeVacation:
				stats.VacationDays++
			}
		}

		stats.EmployeeStats = append(stats.EmployeeStats, EmployeeStatisticsSummary{
			Employee:              employee,
			ScheduledHours:        scheduledHours,
			WorkedHours:           workedHours,
			OvertimeHours:         overtimeHours,
			RemainingVacationDays: employee.RemainingVacationDays,
		})
	}

	return stats, nil
}

// Weekly computes weekly statistics.
func (s *StatisticsService) Weekly(ctx context.Context, weekStart time.Time) (*WeeklyStatistics, error) {
	monday := weekStart
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, -1)
	}
	sunday := monday.AddDate(0, 0, 6)

	employees, err := s.employees.List(ctx, true)
	if err != nil {
		return nil, err
	}

	scheduleEntries, err := s.schedules.List(ctx, monday, sunday, nil, nil)
	if err != nil {
		return nil, err
	}

	timeEntries, err := s.timeEntries.List(ctx, monday, sunday, nil)
	if err != nil {
		return nil, err
	}

	scheduleByEmployee := make(map[int64][]domain.ScheduleEntry)
	for _, entry := range scheduleEntries {
		scheduleByEmployee[entry.EmployeeID] = append(scheduleByEmployee[entry.EmployeeID], entry)
	}

	timeByEmployee := make(map[int64][]domain.TimeEntry)
	for _, entry := range timeEntries {
		timeByEmployee[entry.EmployeeID] = append(timeByEmployee[entry.EmployeeID], entry)
	}

	scheduleByGroup := make(map[int64][]domain.ScheduleEntry)
	for _, entry := range scheduleEntries {
		if entry.GroupID != nil {
			scheduleByGroup[*entry.GroupID] = append(scheduleByGroup[*entry.GroupID], entry)
		}
	}

	result := &WeeklyStatistics{
		WeekStart: monday,
		WeekEnd:   sunday,
	}

	for _, employee := range employees {
		entries := scheduleByEmployee[employee.ID]
		times := timeByEmployee[employee.ID]

		scheduledHours := calculateScheduledHours(entries)
		workedHours := calculateWorkedHours(times)

		daysWorked := countWorkedDays(times)

		result.TotalScheduledHours += scheduledHours
		result.TotalWorkedHours += workedHours

		result.ByEmployee = append(result.ByEmployee, EmployeeWeekSummary{
			Employee:       employee,
			ScheduledHours: scheduledHours,
			WorkedHours:    workedHours,
			DaysWorked:     daysWorked,
		})
	}

	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		entries := scheduleByGroup[group.ID]
		groupHours := calculateScheduledHours(entries)
		staffedDays := countScheduledDays(entries)

		result.ByGroup = append(result.ByGroup, GroupWeekSummary{
			Group:               group,
			TotalScheduledHours: groupHours,
			StaffedDays:         staffedDays,
			UnderstaffedDays:    5 - staffedDays,
		})
	}

	return result, nil
}

// EmployeeStats computes statistics for a single employee.
func (s *StatisticsService) EmployeeStats(ctx context.Context, employeeID int64, month time.Time) (*EmployeeStatistics, error) {
	employee, err := s.employees.GetByID(ctx, employeeID)
	if err != nil {
		return nil, NewNotFound(fmt.Sprintf("Mitarbeiter mit ID %d nicht gefunden", employeeID))
	}

	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	scheduleEntries, err := s.schedules.List(ctx, startOfMonth, endOfMonth, &employeeID, nil)
	if err != nil {
		return nil, err
	}

	timeEntries, err := s.timeEntries.List(ctx, startOfMonth, endOfMonth, &employeeID)
	if err != nil {
		return nil, err
	}

	workingDays := countWorkingDays(startOfMonth, endOfMonth)
	expectedMonthlyHours := (employee.WeeklyHours / 5) * float64(workingDays)

	scheduledHours := calculateScheduledHours(scheduleEntries)
	workedHours := calculateWorkedHours(timeEntries)
	overtimeHours := workedHours - expectedMonthlyHours

	vacationDays := 0
	sickDays := 0
	for _, entry := range scheduleEntries {
		switch entry.EntryType {
		case domain.ScheduleEntryTypeVacation:
			vacationDays++
		case domain.ScheduleEntryTypeSick:
			sickDays++
		}
	}

	scheduleByDate := make(map[string][]domain.ScheduleEntry)
	for _, entry := range scheduleEntries {
		key := entry.Date.Format(dateLayout)
		scheduleByDate[key] = append(scheduleByDate[key], entry)
	}
	// time entries by date
	timeByDate := make(map[string][]domain.TimeEntry)
	for _, entry := range timeEntries {
		key := entry.Date.Format(dateLayout)
		timeByDate[key] = append(timeByDate[key], entry)
	}

	breakdown := make([]DayStatistics, 0)
	current := startOfMonth
	for !current.After(endOfMonth) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			key := current.Format(dateLayout)
			scheduleForDay := scheduleByDate[key]
			timeForDay := timeByDate[key]

			dayStats := DayStatistics{
				Date:           current,
				ScheduledHours: calculateScheduledHours(scheduleForDay),
				WorkedHours:    calculateWorkedHours(timeForDay),
			}
			if len(scheduleForDay) > 0 {
				entryType := scheduleForDay[0].EntryType
				dayStats.EntryType = &entryType
			}
			breakdown = append(breakdown, dayStats)
		}
		current = current.AddDate(0, 0, 1)
	}

	return &EmployeeStatistics{
		Employee:              *employee,
		Month:                 startOfMonth,
		ContractedHours:       expectedMonthlyHours,
		ScheduledHours:        scheduledHours,
		WorkedHours:           workedHours,
		OvertimeHours:         overtimeHours,
		OvertimeBalance:       employee.OvertimeBalance,
		VacationDaysUsed:      vacationDays,
		VacationDaysRemaining: employee.RemainingVacationDays,
		SickDays:              sickDays,
		DailyBreakdown:        breakdown,
	}, nil
}

func calculateScheduledHours(entries []domain.ScheduleEntry) float64 {
	minutes := 0.0
	for _, entry := range entries {
		if entry.EntryType != domain.ScheduleEntryTypeWork {
			continue
		}
		if entry.StartTime != nil && entry.EndTime != nil {
			diff := entry.EndTime.Sub(*entry.StartTime).Minutes()
			minutes += diff - float64(entry.BreakMinutes)
		}
	}
	return minutes / 60
}

func calculateWorkedHours(entries []domain.TimeEntry) float64 {
	minutes := 0.0
	for _, entry := range entries {
		if entry.ClockOut == nil {
			continue
		}
		diff := entry.ClockOut.Sub(entry.ClockIn).Minutes()
		minutes += diff - float64(entry.BreakMinutes)
	}
	return minutes / 60
}

func countWorkingDays(start, end time.Time) int {
	count := 0
	current := start
	for !current.After(end) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			count++
		}
		current = current.AddDate(0, 0, 1)
	}
	return count
}

func countWorkedDays(entries []domain.TimeEntry) int {
	unique := make(map[string]bool)
	for _, entry := range entries {
		if entry.ClockOut == nil {
			continue
		}
		unique[entry.Date.Format(dateLayout)] = true
	}
	return len(unique)
}

func countScheduledDays(entries []domain.ScheduleEntry) int {
	unique := make(map[string]bool)
	for _, entry := range entries {
		unique[entry.Date.Format(dateLayout)] = true
	}
	return len(unique)
}
