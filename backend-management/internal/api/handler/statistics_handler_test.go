package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

// OverviewStatisticsResponse for parsing API responses
type OverviewStatisticsResponse struct {
	Month               string                 `json:"month"`
	TotalEmployees      int                    `json:"totalEmployees"`
	EmployeeStats       []EmployeeStatsSummary `json:"employeeStats"`
	TotalScheduledHours float64                `json:"totalScheduledHours"`
	TotalWorkedHours    float64                `json:"totalWorkedHours"`
	TotalOvertimeHours  float64                `json:"totalOvertimeHours"`
	SickDays            int                    `json:"sickDays"`
	VacationDays        int                    `json:"vacationDays"`
}

type EmployeeStatsSummary struct {
	Employee              testutil.EmployeeResponse `json:"employee"`
	ScheduledHours        float64                   `json:"scheduledHours"`
	WorkedHours           float64                   `json:"workedHours"`
	OvertimeHours         float64                   `json:"overtimeHours"`
	RemainingVacationDays float64                   `json:"remainingVacationDays"`
}

// EmployeeStatisticsResponse for parsing employee stats API responses
type EmployeeStatisticsResponse struct {
	Employee              testutil.EmployeeResponse `json:"employee"`
	Month                 string                    `json:"month"`
	ContractedHours       float64                   `json:"contractedHours"`
	ScheduledHours        float64                   `json:"scheduledHours"`
	WorkedHours           float64                   `json:"workedHours"`
	OvertimeHours         float64                   `json:"overtimeHours"`
	OvertimeBalance       float64                   `json:"overtimeBalance"`
	VacationDaysUsed      int                       `json:"vacationDaysUsed"`
	VacationDaysRemaining float64                   `json:"vacationDaysRemaining"`
	SickDays              int                       `json:"sickDays"`
	DailyBreakdown        []DayStatistics           `json:"dailyBreakdown"`
}

type DayStatistics struct {
	Date           string  `json:"date"`
	ScheduledHours float64 `json:"scheduledHours"`
	WorkedHours    float64 `json:"workedHours"`
	EntryType      *string `json:"entryType,omitempty"`
}

// WeeklyStatisticsResponse for parsing weekly stats API responses
type WeeklyStatisticsResponse struct {
	WeekStart           string                `json:"weekStart"`
	WeekEnd             string                `json:"weekEnd"`
	ByEmployee          []EmployeeWeekSummary `json:"byEmployee"`
	ByGroup             []GroupWeekSummary    `json:"byGroup"`
	TotalScheduledHours float64               `json:"totalScheduledHours"`
	TotalWorkedHours    float64               `json:"totalWorkedHours"`
}

type EmployeeWeekSummary struct {
	Employee       testutil.EmployeeResponse `json:"employee"`
	ScheduledHours float64                   `json:"scheduledHours"`
	WorkedHours    float64                   `json:"workedHours"`
	DaysWorked     int                       `json:"daysWorked"`
}

type GroupWeekSummary struct {
	Group               GroupResponse `json:"group"`
	TotalScheduledHours float64       `json:"totalScheduledHours"`
	StaffedDays         int           `json:"staffedDays"`
	UnderstaffedDays    int           `json:"understaffedDays"`
}

type GroupResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       string  `json:"color"`
}

func TestStatisticsHandler_Overview(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("get overview statistics for month", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminoverview@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Create employees
		_, err = testutil.NewEmployeeBuilder().
			WithEmail("emp1overview@example.com").
			WithWeeklyHours(40).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		_, err = testutil.NewEmployeeBuilder().
			WithEmail("emp2overview@example.com").
			WithWeeklyHours(30).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Request overview for current month
		currentMonth := time.Now().Format("2006-01-02")

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/overview?month=%s", currentMonth), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var response OverviewStatisticsResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, response.TotalEmployees, 3) // admin + 2 employees
	})
}

func TestStatisticsHandler_Overview_WithScheduleAndTimeEntries(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2overview@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("empoverview2@example.com").
		WithWeeklyHours(40).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group Overview").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create schedule entry for today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	startTime := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
	endTime := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		WithDate(today).
		WithTimes(startTime, endTime).
		WithBreak(30).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create time entry
	clockIn := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
	clockOut := time.Date(today.Year(), today.Month(), today.Day(), 16, 30, 0, 0, time.UTC)

	_, err = testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		WithClockOut(clockOut).
		WithBreak(30).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	currentMonth := today.Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/overview?month=%s", currentMonth), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response OverviewStatisticsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, response.TotalScheduledHours, 0.0)
	assert.GreaterOrEqual(t, response.TotalWorkedHours, 0.0)
}

func TestStatisticsHandler_Overview_RequiresMonth(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin3overview@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/statistics/overview", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestStatisticsHandler_Overview_InvalidMonth(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin4overview@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/statistics/overview?month=invalid", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestStatisticsHandler_Employee(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("get employee statistics", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminempstats@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("empstats@example.com").
			WithWeeklyHours(40).
			WithVacationDays(30, 25.5).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		currentMonth := time.Now().Format("2006-01-02")

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/employee/%d?month=%s", employee.ID, currentMonth), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var response EmployeeStatisticsResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, employee.ID, response.Employee.ID)
		assert.Equal(t, 25.5, response.VacationDaysRemaining)
	})
}

func TestStatisticsHandler_Employee_WithEntries(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2empstats@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emp2stats@example.com").
		WithWeeklyHours(40).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group Entries").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create entries for multiple days this month
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i := 0; i < 5; i++ {
		day := today.AddDate(0, 0, -i)
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}

		startTime := time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, time.UTC)
		endTime := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, time.UTC)

		_, err = testutil.NewScheduleEntryBuilder().
			WithEmployeeID(employee.ID).
			WithGroupID(group.ID).
			WithDate(day).
			WithTimes(startTime, endTime).
			WithBreak(30).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		clockIn := time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, time.UTC)
		clockOut := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, time.UTC)

		_, err = testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(day).
			WithClockIn(clockIn).
			WithClockOut(clockOut).
			WithBreak(30).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)
	}

	currentMonth := today.Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/employee/%d?month=%s", employee.ID, currentMonth), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response EmployeeStatisticsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, employee.ID, response.Employee.ID)
	assert.GreaterOrEqual(t, response.ScheduledHours, 0.0)
	assert.GreaterOrEqual(t, response.WorkedHours, 0.0)
}

func TestStatisticsHandler_Employee_WithSickDays(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin3empstats@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emp3stats@example.com").
		WithWeeklyHours(40).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group SickDays").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create a sick day schedule entry (sick days are tracked via ScheduleEntry, not TimeEntry)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		WithDate(today).
		WithType(domain.ScheduleEntryTypeSick).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	currentMonth := today.Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/employee/%d?month=%s", employee.ID, currentMonth), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response EmployeeStatisticsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, response.SickDays, 1)
}

func TestStatisticsHandler_Employee_RequiresMonth(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin4empstats@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emp4stats@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/employee/%d", employee.ID), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestStatisticsHandler_Employee_NotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin5empstats@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	currentMonth := time.Now().Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/employee/99999?month=%s", currentMonth), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNotFound)
}

func TestStatisticsHandler_Weekly(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("get weekly statistics", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminweekly@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("empweekly@example.com").
			WithWeeklyHours(40).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Create group
		group, err := testutil.NewGroupBuilder().
			WithName("Test Group").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Assign employee to group
		_, err = testutil.NewGroupAssignmentBuilder().
			WithEmployeeID(employee.ID).
			WithGroupID(group.ID).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Get Monday of current week
		today := time.Now()
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := today.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
		weekStart := monday.Format("2006-01-02")

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/weekly?weekStart=%s", weekStart), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var response WeeklyStatisticsResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, weekStart, response.WeekStart)
	})
}

func TestStatisticsHandler_Weekly_WithData(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2weekly@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emp2weekly@example.com").
		WithWeeklyHours(40).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create group
	group, err := testutil.NewGroupBuilder().
		WithName("Test Group 2").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Assign employee to group
	_, err = testutil.NewGroupAssignmentBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Get Monday of current week
	today := time.Now()
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := today.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)

	// Create schedule and time entries for this week
	for i := 0; i < 5; i++ {
		day := monday.AddDate(0, 0, i)
		startTime := time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, time.UTC)
		endTime := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, time.UTC)

		_, err = testutil.NewScheduleEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(day).
			WithTimes(startTime, endTime).
			WithBreak(30).
			WithGroupID(group.ID).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		clockIn := time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, time.UTC)
		clockOut := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, time.UTC)

		_, err = testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(day).
			WithClockIn(clockIn).
			WithClockOut(clockOut).
			WithBreak(30).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)
	}

	weekStart := monday.Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/statistics/weekly?weekStart=%s", weekStart), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response WeeklyStatisticsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Greater(t, response.TotalScheduledHours, 0.0)
	assert.Greater(t, response.TotalWorkedHours, 0.0)
}

func TestStatisticsHandler_Weekly_RequiresWeekStart(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin3weekly@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/statistics/weekly", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestStatisticsHandler_Weekly_InvalidWeekStart(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin4weekly@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/statistics/weekly?weekStart=invalid", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestStatisticsHandler_ExportTimesheet(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("adminexport@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/export/timesheet", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The handler returns NoContent for now (not implemented)
	testutil.AssertStatus(t, resp, http.StatusNoContent)
}

func TestStatisticsHandler_ExportSchedule(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2export@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/export/schedule", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The handler returns NoContent for now (not implemented)
	testutil.AssertStatus(t, resp, http.StatusNoContent)
}
