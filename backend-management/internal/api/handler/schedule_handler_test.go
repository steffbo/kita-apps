package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

// Schedule Handler Tests

func TestScheduleHandler_List_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(admin.ID).
		WithDate(today).
		WithGroupID(group.ID).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	startDate := today.Format("2006-01-02")
	endDate := today.AddDate(0, 0, 7).Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", "/api/schedule?startDate="+startDate+"&endDate="+endDate, nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestScheduleHandler_Create_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)

	req := server.AuthenticatedRequest(t, "POST", "/api/schedule", map[string]interface{}{
		"employeeId":   employee.ID,
		"date":         today.Format("2006-01-02"),
		"startTime":    "08:00",
		"endTime":      "16:00",
		"breakMinutes": 30,
		"groupId":      group.ID,
		"entryType":    "WORK",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)
}

func TestScheduleHandler_Create_Forbidden_NonAdmin(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)

	req := server.AuthenticatedRequest(t, "POST", "/api/schedule", map[string]interface{}{
		"employeeId": employee.ID,
		"date":       today.Format("2006-01-02"),
		"entryType":  "WORK",
	}, employee)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusForbidden)
}

func TestScheduleHandler_TimeSuggestion_EarlyUsesContractHours(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		WithWeeklyHours(35).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/schedule/time-suggestion", map[string]interface{}{
		"employeeId": employee.ID,
		"date":       currentMonthWeekday(time.Monday).Format("2006-01-02"),
		"shiftKind":  "EARLY",
		"startTime":  "07:00",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var result struct {
		StartTime      string `json:"startTime"`
		EndTime        string `json:"endTime"`
		BreakMinutes   int    `json:"breakMinutes"`
		PlannedMinutes int    `json:"plannedMinutes"`
		IsBlocked      bool   `json:"isBlocked"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	assert.False(t, result.IsBlocked)
	assert.Equal(t, "07:00:00", result.StartTime)
	assert.Equal(t, "14:30:00", result.EndTime)
	assert.Equal(t, 30, result.BreakMinutes)
	assert.Equal(t, 420, result.PlannedMinutes)
}

func TestScheduleHandler_TimeSuggestion_LateUsesClosingHours(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		WithWeeklyHours(30).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/schedule/time-suggestion", map[string]interface{}{
		"employeeId": employee.ID,
		"date":       currentMonthWeekday(time.Friday).Format("2006-01-02"),
		"shiftKind":  "LATE",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var result struct {
		StartTime    string `json:"startTime"`
		EndTime      string `json:"endTime"`
		BreakMinutes int    `json:"breakMinutes"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	assert.Equal(t, "10:00:00", result.StartTime)
	assert.Equal(t, "16:00:00", result.EndTime)
	assert.Equal(t, 0, result.BreakMinutes)
}

func TestScheduleHandler_Create_BlockedDayRequiresOverride(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		WithWeeklyHours(21).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)
	setContractWorkdays(t, employee.ID, []domain.EmployeeContractWorkday{
		{Weekday: 1, PlannedMinutes: 420},
		{Weekday: 2, PlannedMinutes: 420},
		{Weekday: 3, PlannedMinutes: 420},
	})

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"employeeId":   employee.ID,
		"date":         currentMonthWeekday(time.Friday).Format("2006-01-02"),
		"startTime":    "08:00",
		"endTime":      "15:30",
		"breakMinutes": 30,
		"groupId":      group.ID,
		"entryType":    "WORK",
		"shiftKind":    "EARLY",
	}

	resp, err := server.Do(server.AuthenticatedRequest(t, "POST", "/api/schedule", body, admin))
	require.NoError(t, err)
	resp.Body.Close()
	testutil.AssertStatus(t, resp, http.StatusBadRequest)

	body["overrideBlockedDay"] = true
	resp, err = server.Do(server.AuthenticatedRequest(t, "POST", "/api/schedule", body, admin))
	require.NoError(t, err)
	defer resp.Body.Close()
	testutil.AssertStatus(t, resp, http.StatusCreated)
}

func TestScheduleHandler_BulkCreate_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.AddDate(0, 0, 1)

	req := server.AuthenticatedRequest(t, "POST", "/api/schedule/bulk", []map[string]interface{}{
		{
			"employeeId":   employee.ID,
			"date":         today.Format("2006-01-02"),
			"startTime":    "08:00",
			"endTime":      "16:00",
			"breakMinutes": 30,
			"groupId":      group.ID,
			"entryType":    "WORK",
		},
		{
			"employeeId":   employee.ID,
			"date":         tomorrow.Format("2006-01-02"),
			"startTime":    "08:00",
			"endTime":      "16:00",
			"breakMinutes": 30,
			"groupId":      group.ID,
			"entryType":    "WORK",
		},
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	type ScheduleResponse struct {
		ID int64 `json:"id"`
	}
	var response []ScheduleResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
}

func TestScheduleHandler_Update_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	entry, err := testutil.NewScheduleEntryBuilder().
		WithEmployeeID(admin.ID).
		WithDate(today).
		WithGroupID(group.ID).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "PUT", "/api/schedule/"+strconv.FormatInt(entry.ID, 10), map[string]interface{}{
		"employeeId":   admin.ID,
		"date":         today.Format("2006-01-02"),
		"startTime":    "09:00",
		"endTime":      "17:00",
		"breakMinutes": 45,
		"groupId":      group.ID,
		"entryType":    "WORK",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestScheduleHandler_Delete_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	entry, err := testutil.NewScheduleEntryBuilder().
		WithEmployeeID(admin.ID).
		WithDate(today).
		WithGroupID(group.ID).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "DELETE", "/api/schedule/"+strconv.FormatInt(entry.ID, 10), nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNoContent)
}

func TestScheduleHandler_Week_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	weekStart := today.Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", "/api/schedule/week?weekStart="+weekStart, nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

func currentMonthWeekday(target time.Weekday) time.Time {
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for date.Weekday() != target {
		date = date.AddDate(0, 0, 1)
	}
	return date
}

func setContractWorkdays(t *testing.T, employeeID int64, workdays []domain.EmployeeContractWorkday) {
	t.Helper()
	repo := repository.NewPostgresEmployeeRepository(suite.Container.DB)
	contracts, err := repo.ListContracts(suite.Ctx, employeeID)
	require.NoError(t, err)
	require.NotEmpty(t, contracts)
	contract := contracts[0]
	contract.Workdays = workdays
	_, err = repo.UpdateContract(suite.Ctx, &contract)
	require.NoError(t, err)
}
