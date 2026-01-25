package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
