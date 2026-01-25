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

// TimeEntryResponse for parsing API responses
type TimeEntryResponse struct {
	ID            int64   `json:"id"`
	EmployeeID    int64   `json:"employeeId"`
	Date          string  `json:"date"`
	ClockIn       string  `json:"clockIn"`
	ClockOut      *string `json:"clockOut,omitempty"`
	BreakMinutes  int     `json:"breakMinutes"`
	EntryType     string  `json:"entryType"`
	WorkedMinutes *int    `json:"workedMinutes,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	EditedBy      *int64  `json:"editedBy,omitempty"`
	EditReason    *string `json:"editReason,omitempty"`
}

// TimeScheduleComparisonResponse for parsing comparison API responses
type TimeScheduleComparisonResponse struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Entries   []struct {
		Date              string `json:"date"`
		ScheduledMinutes  int    `json:"scheduledMinutes"`
		ActualMinutes     int    `json:"actualMinutes"`
		DifferenceMinutes int    `json:"differenceMinutes"`
		Status            string `json:"status"`
	} `json:"entries"`
	Summary struct {
		TotalScheduledMinutes  int `json:"totalScheduledMinutes"`
		TotalActualMinutes     int `json:"totalActualMinutes"`
		TotalDifferenceMinutes int `json:"totalDifferenceMinutes"`
		DaysWorked             int `json:"daysWorked"`
		DaysScheduled          int `json:"daysScheduled"`
	} `json:"summary"`
}

func TestTimeTrackingHandler_ClockIn(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("clock in successfully", func(t *testing.T) {
		// Create an employee
		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("clockin@example.com").
			WithName("Clock", "In").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", nil, employee)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var result TimeEntryResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, employee.ID, result.EmployeeID)
		assert.NotEmpty(t, result.ClockIn)
		assert.Nil(t, result.ClockOut)
		assert.Equal(t, "WORK", result.EntryType)
	})
}

func TestTimeTrackingHandler_ClockIn_WithNotes(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("clockin2@example.com").
		WithName("Clock", "In2").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]string{"notes": "Starting early today"}
	req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", body, employee)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var result TimeEntryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.NotNil(t, result.Notes)
	assert.Equal(t, "Starting early today", *result.Notes)
}

func TestTimeTrackingHandler_ClockIn_AlreadyClockedIn(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	emp, err := testutil.NewEmployeeBuilder().
		WithEmail("doubleclick@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// First clock in
	req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", nil, emp)
	resp, err := server.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	testutil.AssertStatus(t, resp, http.StatusOK)

	// Second clock in should fail
	req = server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", nil, emp)
	resp, err = server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The service returns BadRequest for "already clocked in"
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestTimeTrackingHandler_ClockIn_Unauthorized(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("POST", "/api/time-tracking/clock-in", nil)
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestTimeTrackingHandler_ClockOut(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("clock out successfully", func(t *testing.T) {
		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("clockout@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// First clock in
		req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", nil, employee)
		resp, err := server.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		// Then clock out
		req = server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-out", nil, employee)
		resp, err = server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var result TimeEntryResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, employee.ID, result.EmployeeID)
		assert.NotNil(t, result.ClockOut)
		assert.NotNil(t, result.WorkedMinutes)
	})
}

func TestTimeTrackingHandler_ClockOut_WithBreak(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("clockoutbreak@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Clock in
	req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", nil, employee)
	resp, err := server.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Clock out with break
	breakMinutes := 30
	body := map[string]interface{}{
		"breakMinutes": breakMinutes,
		"notes":        "Lunch break",
	}
	req = server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-out", body, employee)
	resp, err = server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var result TimeEntryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, breakMinutes, result.BreakMinutes)
	require.NotNil(t, result.Notes)
	assert.Equal(t, "Lunch break", *result.Notes)
}

func TestTimeTrackingHandler_ClockOut_NotClockedIn(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("notclockedin@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-out", nil, employee)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestTimeTrackingHandler_Current(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("returns current open entry", func(t *testing.T) {
		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("current@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Clock in
		req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/clock-in", nil, employee)
		resp, err := server.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		// Get current
		req = server.AuthenticatedRequest(t, "GET", "/api/time-tracking/current", nil, employee)
		resp, err = server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var result TimeEntryResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, employee.ID, result.EmployeeID)
		assert.Nil(t, result.ClockOut)
	})
}

func TestTimeTrackingHandler_Current_NoOpenEntry(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("nocurrent@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/time-tracking/current", nil, employee)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNoContent)
}

func TestTimeTrackingHandler_List(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("list entries for date range", func(t *testing.T) {
		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("list@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Create time entries
		today := time.Now().Truncate(24 * time.Hour)
		clockIn := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
		clockOut := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

		_, err = testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(today).
			WithClockIn(clockIn).
			WithClockOut(clockOut).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		startDate := today.AddDate(0, 0, -7).Format("2006-01-02")
		endDate := today.AddDate(0, 0, 7).Format("2006-01-02")

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/time-tracking/entries?startDate=%s&endDate=%s", startDate, endDate), nil, employee)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var result []TimeEntryResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Len(t, result, 1)
		assert.Equal(t, employee.ID, result[0].EmployeeID)
	})
}

func TestTimeTrackingHandler_List_AdminViewOther(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("adminlist@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emplist@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create time entry for employee
	today := time.Now().Truncate(24 * time.Hour)
	clockIn := time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, time.UTC)
	clockOut := time.Date(today.Year(), today.Month(), today.Day(), 17, 0, 0, 0, time.UTC)

	_, err = testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		WithClockOut(clockOut).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	startDate := today.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := today.AddDate(0, 0, 1).Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/time-tracking/entries?startDate=%s&endDate=%s&employeeId=%d", startDate, endDate, employee.ID), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var result []TimeEntryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Equal(t, employee.ID, result[0].EmployeeID)
}

func TestTimeTrackingHandler_List_RequiresDateRange(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("nodates@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/time-tracking/entries", nil, employee)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestTimeTrackingHandler_Create(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("create time entry successfully", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("admincreate@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("empcreate@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		today := time.Now().Truncate(24 * time.Hour)
		clockIn := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
		clockOut := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

		body := map[string]interface{}{
			"employeeId":   employee.ID,
			"date":         today.Format("2006-01-02"),
			"clockIn":      clockIn.Format(time.RFC3339),
			"clockOut":     clockOut.Format(time.RFC3339),
			"breakMinutes": 30,
			"entryType":    "WORK",
			"editReason":   "Manual entry",
		}

		req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/entries", body, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusCreated)

		var result TimeEntryResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, employee.ID, result.EmployeeID)
		assert.Equal(t, 30, result.BreakMinutes)
		assert.Equal(t, "WORK", result.EntryType)
	})
}

func TestTimeTrackingHandler_Create_Vacation(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2create@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emp2create@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	clockIn := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	clockOut := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)

	body := map[string]interface{}{
		"employeeId": employee.ID,
		"date":       today.Format("2006-01-02"),
		"clockIn":    clockIn.Format(time.RFC3339),
		"clockOut":   clockOut.Format(time.RFC3339),
		"entryType":  "VACATION",
		"notes":      "Annual leave",
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/entries", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	var result TimeEntryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "VACATION", result.EntryType)
	require.NotNil(t, result.Notes)
	assert.Equal(t, "Annual leave", *result.Notes)
}

func TestTimeTrackingHandler_Create_ValidationError(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("adminval@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"employeeId": 999,
		// missing required fields
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/time-tracking/entries", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestTimeTrackingHandler_Update(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("update time entry successfully", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminupdate@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("empupdate@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		today := time.Now().Truncate(24 * time.Hour)
		clockIn := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
		clockOut := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

		entry, err := testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(today).
			WithClockIn(clockIn).
			WithClockOut(clockOut).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		newClockOut := time.Date(today.Year(), today.Month(), today.Day(), 17, 0, 0, 0, time.UTC)
		body := map[string]interface{}{
			"clockOut":     newClockOut.Format(time.RFC3339),
			"breakMinutes": 45,
			"editReason":   "Correction",
		}

		req := server.AuthenticatedRequest(t, "PUT", fmt.Sprintf("/api/time-tracking/entries/%d", entry.ID), body, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var result TimeEntryResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, 45, result.BreakMinutes)
		require.NotNil(t, result.EditReason)
		assert.Equal(t, "Correction", *result.EditReason)
		require.NotNil(t, result.EditedBy)
		assert.Equal(t, admin.ID, *result.EditedBy)
	})
}

func TestTimeTrackingHandler_Update_ChangeType(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2update@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("emp2update@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	clockIn := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
	clockOut := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

	entry, err := testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		WithClockOut(clockOut).
		WithType(domain.TimeEntryTypeWork).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"entryType":  "SICK",
		"editReason": "Employee was actually sick",
	}

	req := server.AuthenticatedRequest(t, "PUT", fmt.Sprintf("/api/time-tracking/entries/%d", entry.ID), body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var result TimeEntryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "SICK", result.EntryType)
}

func TestTimeTrackingHandler_Update_NotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin404@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"breakMinutes": 30,
	}

	req := server.AuthenticatedRequest(t, "PUT", "/api/time-tracking/entries/99999", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNotFound)
}

func TestTimeTrackingHandler_Delete(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("delete time entry successfully", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("admindelete@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("empdelete@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		today := time.Now().Truncate(24 * time.Hour)
		clockIn := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
		clockOut := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

		entry, err := testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(today).
			WithClockIn(clockIn).
			WithClockOut(clockOut).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		req := server.AuthenticatedRequest(t, "DELETE", fmt.Sprintf("/api/time-tracking/entries/%d", entry.ID), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusNoContent)
	})
}

func TestTimeTrackingHandler_Delete_NotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admindelete404@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "DELETE", "/api/time-tracking/entries/99999", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNotFound)
}

func TestTimeTrackingHandler_Comparison(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("get comparison for date range", func(t *testing.T) {
		employee, err := testutil.NewEmployeeBuilder().
			WithEmail("comparison@example.com").
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		today := time.Now().Truncate(24 * time.Hour)
		startDate := today.Format("2006-01-02")
		endDate := today.AddDate(0, 0, 7).Format("2006-01-02")

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/time-tracking/comparison?startDate=%s&endDate=%s", startDate, endDate), nil, employee)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var result TimeScheduleComparisonResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, startDate, result.StartDate)
		assert.Equal(t, endDate, result.EndDate)
	})
}

func TestTimeTrackingHandler_Comparison_RequiresDateRange(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("compnodates@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/time-tracking/comparison", nil, employee)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestTimeTrackingHandler_Comparison_AdminViewOther(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admincomp@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("empcomp@example.com").
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)
	startDate := today.Format("2006-01-02")
	endDate := today.AddDate(0, 0, 7).Format("2006-01-02")

	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/time-tracking/comparison?startDate=%s&endDate=%s&employeeId=%d", startDate, endDate, employee.ID), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}
