package handler

import (
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

const (
	dateLayout     = "2006-01-02"
	timeLayoutSecs = "15:04:05"
)

// EmployeeResponse represents the employee API response.
type EmployeeResponse struct {
	ID                    int64          `json:"id"`
	Email                 string         `json:"email"`
	FirstName             string         `json:"firstName"`
	LastName              string         `json:"lastName"`
	Role                  string         `json:"role"`
	WeeklyHours           float64        `json:"weeklyHours"`
	VacationDaysPerYear   int            `json:"vacationDaysPerYear"`
	RemainingVacationDays float64        `json:"remainingVacationDays"`
	OvertimeBalance       float64        `json:"overtimeBalance"`
	Active                bool           `json:"active"`
	PrimaryGroupID        *int64         `json:"primaryGroupId,omitempty"`
	PrimaryGroup          *GroupResponse `json:"primaryGroup,omitempty"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
}

// GroupResponse represents the group API response.
type GroupResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       string  `json:"color"`
}

// GroupWithMembersResponse represents a group with members.
type GroupWithMembersResponse struct {
	GroupResponse
	Members []GroupAssignmentResponse `json:"members"`
}

// GroupAssignmentResponse represents a group assignment.
type GroupAssignmentResponse struct {
	ID             int64             `json:"id"`
	EmployeeID     int64             `json:"employeeId"`
	Employee       *EmployeeResponse `json:"employee,omitempty"`
	GroupID        int64             `json:"groupId"`
	AssignmentType string            `json:"assignmentType"`
}

// ScheduleEntryResponse represents a schedule entry.
type ScheduleEntryResponse struct {
	ID           int64             `json:"id"`
	EmployeeID   int64             `json:"employeeId"`
	Employee     *EmployeeResponse `json:"employee,omitempty"`
	Date         string            `json:"date"`
	StartTime    *string           `json:"startTime,omitempty"`
	EndTime      *string           `json:"endTime,omitempty"`
	BreakMinutes int               `json:"breakMinutes"`
	GroupID      *int64            `json:"groupId,omitempty"`
	Group        *GroupResponse    `json:"group,omitempty"`
	EntryType    string            `json:"entryType"`
	Notes        *string           `json:"notes,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// WeekScheduleResponse represents the week schedule.
type WeekScheduleResponse struct {
	WeekStart   string                `json:"weekStart"`
	WeekEnd     string                `json:"weekEnd"`
	Days        []DayScheduleResponse `json:"days"`
	SpecialDays []SpecialDayResponse  `json:"specialDays,omitempty"`
}

// DayScheduleResponse represents a day's schedule.
type DayScheduleResponse struct {
	Date        string                             `json:"date"`
	DayOfWeek   string                             `json:"dayOfWeek"`
	IsHoliday   bool                               `json:"isHoliday"`
	HolidayName *string                            `json:"holidayName,omitempty"`
	Entries     []ScheduleEntryResponse            `json:"entries"`
	ByGroup     map[string][]ScheduleEntryResponse `json:"byGroup,omitempty"`
}

// TimeEntryResponse represents a time entry.
type TimeEntryResponse struct {
	ID            int64             `json:"id"`
	EmployeeID    int64             `json:"employeeId"`
	Employee      *EmployeeResponse `json:"employee,omitempty"`
	Date          string            `json:"date"`
	ClockIn       string            `json:"clockIn"`
	ClockOut      *string           `json:"clockOut,omitempty"`
	BreakMinutes  int               `json:"breakMinutes"`
	EntryType     string            `json:"entryType"`
	WorkedMinutes *int              `json:"workedMinutes,omitempty"`
	Notes         *string           `json:"notes,omitempty"`
	EditedBy      *int64            `json:"editedBy,omitempty"`
	EditedAt      *string           `json:"editedAt,omitempty"`
	EditReason    *string           `json:"editReason,omitempty"`
	CreatedAt     string            `json:"createdAt"`
}

// SpecialDayResponse represents a special day.
type SpecialDayResponse struct {
	ID         int64   `json:"id"`
	Date       string  `json:"date"`
	EndDate    *string `json:"endDate,omitempty"`
	Name       string  `json:"name"`
	DayType    string  `json:"dayType"`
	AffectsAll bool    `json:"affectsAll"`
	Notes      *string `json:"notes,omitempty"`
}

// OverviewStatisticsResponse represents overview stats.
type OverviewStatisticsResponse struct {
	Month               string                              `json:"month"`
	TotalEmployees      int                                 `json:"totalEmployees"`
	EmployeeStats       []EmployeeStatisticsSummaryResponse `json:"employeeStats"`
	TotalScheduledHours float64                             `json:"totalScheduledHours"`
	TotalWorkedHours    float64                             `json:"totalWorkedHours"`
	TotalOvertimeHours  float64                             `json:"totalOvertimeHours"`
	SickDays            int                                 `json:"sickDays"`
	VacationDays        int                                 `json:"vacationDays"`
}

// EmployeeStatisticsSummaryResponse represents employee summary stats.
type EmployeeStatisticsSummaryResponse struct {
	Employee              EmployeeResponse `json:"employee"`
	ScheduledHours        float64          `json:"scheduledHours"`
	WorkedHours           float64          `json:"workedHours"`
	OvertimeHours         float64          `json:"overtimeHours"`
	RemainingVacationDays float64          `json:"remainingVacationDays"`
}

// EmployeeStatisticsResponse represents employee stats.
type EmployeeStatisticsResponse struct {
	Employee              EmployeeResponse        `json:"employee"`
	Month                 string                  `json:"month"`
	ContractedHours       float64                 `json:"contractedHours"`
	ScheduledHours        float64                 `json:"scheduledHours"`
	WorkedHours           float64                 `json:"workedHours"`
	OvertimeHours         float64                 `json:"overtimeHours"`
	OvertimeBalance       float64                 `json:"overtimeBalance"`
	VacationDaysUsed      int                     `json:"vacationDaysUsed"`
	VacationDaysRemaining float64                 `json:"vacationDaysRemaining"`
	SickDays              int                     `json:"sickDays"`
	DailyBreakdown        []DayStatisticsResponse `json:"dailyBreakdown"`
}

// DayStatisticsResponse represents day stats.
type DayStatisticsResponse struct {
	Date           string  `json:"date"`
	ScheduledHours float64 `json:"scheduledHours"`
	WorkedHours    float64 `json:"workedHours"`
	EntryType      *string `json:"entryType,omitempty"`
}

// WeeklyStatisticsResponse represents weekly stats.
type WeeklyStatisticsResponse struct {
	WeekStart           string                        `json:"weekStart"`
	WeekEnd             string                        `json:"weekEnd"`
	ByEmployee          []EmployeeWeekSummaryResponse `json:"byEmployee"`
	ByGroup             []GroupWeekSummaryResponse    `json:"byGroup"`
	TotalScheduledHours float64                       `json:"totalScheduledHours"`
	TotalWorkedHours    float64                       `json:"totalWorkedHours"`
}

// EmployeeWeekSummaryResponse represents weekly summary per employee.
type EmployeeWeekSummaryResponse struct {
	Employee       EmployeeResponse `json:"employee"`
	ScheduledHours float64          `json:"scheduledHours"`
	WorkedHours    float64          `json:"workedHours"`
	DaysWorked     int              `json:"daysWorked"`
}

// GroupWeekSummaryResponse represents weekly summary per group.
type GroupWeekSummaryResponse struct {
	Group               GroupResponse `json:"group"`
	TotalScheduledHours float64       `json:"totalScheduledHours"`
	StaffedDays         int           `json:"staffedDays"`
	UnderstaffedDays    int           `json:"understaffedDays"`
}

// TimeScheduleComparisonResponse represents time/schedule comparison.
type TimeScheduleComparisonResponse struct {
	StartDate string                    `json:"startDate"`
	EndDate   string                    `json:"endDate"`
	Entries   []DayComparisonResponse   `json:"entries"`
	Summary   ComparisonSummaryResponse `json:"summary"`
}

// DayComparisonResponse represents a daily comparison.
type DayComparisonResponse struct {
	Date              string `json:"date"`
	ScheduledMinutes  int    `json:"scheduledMinutes"`
	ActualMinutes     int    `json:"actualMinutes"`
	DifferenceMinutes int    `json:"differenceMinutes"`
	Status            string `json:"status"`
}

// ComparisonSummaryResponse represents comparison totals.
type ComparisonSummaryResponse struct {
	TotalScheduledMinutes  int `json:"totalScheduledMinutes"`
	TotalActualMinutes     int `json:"totalActualMinutes"`
	TotalDifferenceMinutes int `json:"totalDifferenceMinutes"`
	DaysWorked             int `json:"daysWorked"`
	DaysScheduled          int `json:"daysScheduled"`
}

func mapEmployeeResponse(emp domain.Employee, primaryGroup *domain.Group, primaryGroupID *int64) EmployeeResponse {
	response := EmployeeResponse{
		ID:                    emp.ID,
		Email:                 emp.Email,
		FirstName:             emp.FirstName,
		LastName:              emp.LastName,
		Role:                  string(emp.Role),
		WeeklyHours:           emp.WeeklyHours,
		VacationDaysPerYear:   emp.VacationDaysPerYear,
		RemainingVacationDays: emp.RemainingVacationDays,
		OvertimeBalance:       emp.OvertimeBalance,
		Active:                emp.Active,
		PrimaryGroupID:        primaryGroupID,
		CreatedAt:             emp.CreatedAt,
		UpdatedAt:             emp.UpdatedAt,
	}
	if primaryGroup != nil {
		response.PrimaryGroup = mapGroupResponse(*primaryGroup)
	}
	return response
}

func mapGroupResponse(group domain.Group) *GroupResponse {
	return &GroupResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Color:       group.Color,
	}
}

func mapGroupAssignmentResponse(assignment domain.GroupAssignment, includeEmployee bool) GroupAssignmentResponse {
	response := GroupAssignmentResponse{
		ID:             assignment.ID,
		EmployeeID:     assignment.EmployeeID,
		GroupID:        assignment.GroupID,
		AssignmentType: string(assignment.AssignmentType),
	}
	if includeEmployee && assignment.Employee != nil {
		response.Employee = &EmployeeResponse{
			ID:                    assignment.Employee.ID,
			Email:                 assignment.Employee.Email,
			FirstName:             assignment.Employee.FirstName,
			LastName:              assignment.Employee.LastName,
			Role:                  string(assignment.Employee.Role),
			WeeklyHours:           assignment.Employee.WeeklyHours,
			VacationDaysPerYear:   assignment.Employee.VacationDaysPerYear,
			RemainingVacationDays: assignment.Employee.RemainingVacationDays,
			OvertimeBalance:       assignment.Employee.OvertimeBalance,
			Active:                assignment.Employee.Active,
			CreatedAt:             assignment.Employee.CreatedAt,
			UpdatedAt:             assignment.Employee.UpdatedAt,
		}
	}
	return response
}

func mapScheduleEntryResponse(entry domain.ScheduleEntry) ScheduleEntryResponse {
	response := ScheduleEntryResponse{
		ID:           entry.ID,
		EmployeeID:   entry.EmployeeID,
		Date:         entry.Date.Format(dateLayout),
		BreakMinutes: entry.BreakMinutes,
		GroupID:      entry.GroupID,
		EntryType:    string(entry.EntryType),
		Notes:        entry.Notes,
		CreatedAt:    entry.CreatedAt,
		UpdatedAt:    entry.UpdatedAt,
	}
	if entry.StartTime != nil {
		formatted := entry.StartTime.Format(timeLayoutSecs)
		response.StartTime = &formatted
	}
	if entry.EndTime != nil {
		formatted := entry.EndTime.Format(timeLayoutSecs)
		response.EndTime = &formatted
	}
	if entry.Employee != nil {
		response.Employee = &EmployeeResponse{
			ID:                    entry.Employee.ID,
			Email:                 entry.Employee.Email,
			FirstName:             entry.Employee.FirstName,
			LastName:              entry.Employee.LastName,
			Role:                  string(entry.Employee.Role),
			WeeklyHours:           entry.Employee.WeeklyHours,
			VacationDaysPerYear:   entry.Employee.VacationDaysPerYear,
			RemainingVacationDays: entry.Employee.RemainingVacationDays,
			OvertimeBalance:       entry.Employee.OvertimeBalance,
			Active:                entry.Employee.Active,
			CreatedAt:             entry.Employee.CreatedAt,
			UpdatedAt:             entry.Employee.UpdatedAt,
		}
	}
	if entry.Group != nil {
		response.Group = mapGroupResponse(*entry.Group)
	}
	return response
}

func mapSpecialDayResponse(day domain.SpecialDay) SpecialDayResponse {
	response := SpecialDayResponse{
		ID:         day.ID,
		Date:       day.Date.Format(dateLayout),
		Name:       day.Name,
		DayType:    string(day.DayType),
		AffectsAll: day.AffectsAll,
		Notes:      day.Notes,
	}
	if day.EndDate != nil {
		formatted := day.EndDate.Format(dateLayout)
		response.EndDate = &formatted
	}
	return response
}

func mapTimeEntryResponse(entry domain.TimeEntry) TimeEntryResponse {
	response := TimeEntryResponse{
		ID:           entry.ID,
		EmployeeID:   entry.EmployeeID,
		Date:         entry.Date.Format(dateLayout),
		ClockIn:      entry.ClockIn.Format(time.RFC3339),
		BreakMinutes: entry.BreakMinutes,
		EntryType:    string(entry.EntryType),
		Notes:        entry.Notes,
		EditedBy:     entry.EditedBy,
		EditReason:   entry.EditReason,
		CreatedAt:    entry.CreatedAt.Format(time.RFC3339),
	}
	if entry.ClockOut != nil {
		formatted := entry.ClockOut.Format(time.RFC3339)
		response.ClockOut = &formatted
		worked := int(entry.ClockOut.Sub(entry.ClockIn).Minutes()) - entry.BreakMinutes
		if worked >= 0 {
			response.WorkedMinutes = &worked
		}
	}
	if entry.EditedAt != nil {
		formatted := entry.EditedAt.Format(time.RFC3339)
		response.EditedAt = &formatted
	}
	if entry.Employee != nil {
		response.Employee = &EmployeeResponse{
			ID:                    entry.Employee.ID,
			Email:                 entry.Employee.Email,
			FirstName:             entry.Employee.FirstName,
			LastName:              entry.Employee.LastName,
			Role:                  string(entry.Employee.Role),
			WeeklyHours:           entry.Employee.WeeklyHours,
			VacationDaysPerYear:   entry.Employee.VacationDaysPerYear,
			RemainingVacationDays: entry.Employee.RemainingVacationDays,
			OvertimeBalance:       entry.Employee.OvertimeBalance,
			Active:                entry.Employee.Active,
			CreatedAt:             entry.Employee.CreatedAt,
			UpdatedAt:             entry.Employee.UpdatedAt,
		}
	}
	return response
}

func mapOverviewResponse(stats *service.OverviewStatistics) OverviewStatisticsResponse {
	response := OverviewStatisticsResponse{
		Month:               stats.Month.Format(dateLayout),
		TotalEmployees:      stats.TotalEmployees,
		TotalScheduledHours: stats.TotalScheduledHours,
		TotalWorkedHours:    stats.TotalWorkedHours,
		TotalOvertimeHours:  stats.TotalOvertimeHours,
		SickDays:            stats.SickDays,
		VacationDays:        stats.VacationDays,
	}
	for _, summary := range stats.EmployeeStats {
		response.EmployeeStats = append(response.EmployeeStats, EmployeeStatisticsSummaryResponse{
			Employee:              mapEmployeeResponse(summary.Employee, nil, nil),
			ScheduledHours:        summary.ScheduledHours,
			WorkedHours:           summary.WorkedHours,
			OvertimeHours:         summary.OvertimeHours,
			RemainingVacationDays: summary.RemainingVacationDays,
		})
	}
	return response
}

func mapEmployeeStatsResponse(stats *service.EmployeeStatistics) EmployeeStatisticsResponse {
	response := EmployeeStatisticsResponse{
		Employee:              mapEmployeeResponse(stats.Employee, nil, nil),
		Month:                 stats.Month.Format(dateLayout),
		ContractedHours:       stats.ContractedHours,
		ScheduledHours:        stats.ScheduledHours,
		WorkedHours:           stats.WorkedHours,
		OvertimeHours:         stats.OvertimeHours,
		OvertimeBalance:       stats.OvertimeBalance,
		VacationDaysUsed:      stats.VacationDaysUsed,
		VacationDaysRemaining: stats.VacationDaysRemaining,
		SickDays:              stats.SickDays,
	}
	for _, day := range stats.DailyBreakdown {
		var entryType *string
		if day.EntryType != nil {
			value := string(*day.EntryType)
			entryType = &value
		}
		response.DailyBreakdown = append(response.DailyBreakdown, DayStatisticsResponse{
			Date:           day.Date.Format(dateLayout),
			ScheduledHours: day.ScheduledHours,
			WorkedHours:    day.WorkedHours,
			EntryType:      entryType,
		})
	}
	return response
}

func mapWeeklyStatsResponse(stats *service.WeeklyStatistics) WeeklyStatisticsResponse {
	response := WeeklyStatisticsResponse{
		WeekStart:           stats.WeekStart.Format(dateLayout),
		WeekEnd:             stats.WeekEnd.Format(dateLayout),
		TotalScheduledHours: stats.TotalScheduledHours,
		TotalWorkedHours:    stats.TotalWorkedHours,
	}
	for _, summary := range stats.ByEmployee {
		response.ByEmployee = append(response.ByEmployee, EmployeeWeekSummaryResponse{
			Employee:       mapEmployeeResponse(summary.Employee, nil, nil),
			ScheduledHours: summary.ScheduledHours,
			WorkedHours:    summary.WorkedHours,
			DaysWorked:     summary.DaysWorked,
		})
	}
	for _, summary := range stats.ByGroup {
		response.ByGroup = append(response.ByGroup, GroupWeekSummaryResponse{
			Group:               *mapGroupResponse(summary.Group),
			TotalScheduledHours: summary.TotalScheduledHours,
			StaffedDays:         summary.StaffedDays,
			UnderstaffedDays:    summary.UnderstaffedDays,
		})
	}
	return response
}

func mapTimeScheduleComparisonResponse(comp *service.TimeScheduleComparison) TimeScheduleComparisonResponse {
	response := TimeScheduleComparisonResponse{
		StartDate: comp.StartDate.Format(dateLayout),
		EndDate:   comp.EndDate.Format(dateLayout),
		Summary: ComparisonSummaryResponse{
			TotalScheduledMinutes:  comp.Summary.TotalScheduledMinutes,
			TotalActualMinutes:     comp.Summary.TotalActualMinutes,
			TotalDifferenceMinutes: comp.Summary.TotalDifferenceMinutes,
			DaysWorked:             comp.Summary.DaysWorked,
			DaysScheduled:          comp.Summary.DaysScheduled,
		},
	}
	for _, entry := range comp.Entries {
		response.Entries = append(response.Entries, DayComparisonResponse{
			Date:              entry.Date.Format(dateLayout),
			ScheduledMinutes:  entry.ScheduledMinutes,
			ActualMinutes:     entry.ActualMinutes,
			DifferenceMinutes: entry.DifferenceMinutes,
			Status:            entry.Status,
		})
	}
	return response
}
