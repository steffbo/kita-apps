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

// SpecialDayResponse for parsing API responses
type SpecialDayResponse struct {
	ID         int64   `json:"id"`
	Date       string  `json:"date"`
	EndDate    *string `json:"endDate,omitempty"`
	Name       string  `json:"name"`
	DayType    string  `json:"dayType"`
	AffectsAll bool    `json:"affectsAll"`
	Notes      *string `json:"notes,omitempty"`
}

func TestSpecialDayHandler_List(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("list special days for year", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminlist@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Create special days for this year
		currentYear := time.Now().Year()
		date := time.Date(currentYear, 12, 25, 0, 0, 0, 0, time.UTC)

		_, err = testutil.NewSpecialDayBuilder().
			WithDate(date).
			WithName("Christmas").
			WithType(domain.SpecialDayTypeHoliday).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/special-days?year=%d", currentYear), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var response []SpecialDayResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(response), 1)

		// Find our Christmas entry
		found := false
		for _, day := range response {
			if day.Name == "Christmas" {
				found = true
				assert.Equal(t, "HOLIDAY", day.DayType)
				assert.True(t, day.AffectsAll)
			}
		}
		assert.True(t, found, "Christmas should be in the list")
	})
}

func TestSpecialDayHandler_List_WithoutHolidays(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2list@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	currentYear := time.Now().Year()

	// Create a holiday
	holidayDate := time.Date(currentYear, 12, 25, 0, 0, 0, 0, time.UTC)
	_, err = testutil.NewSpecialDayBuilder().
		WithDate(holidayDate).
		WithName("Christmas").
		WithType(domain.SpecialDayTypeHoliday).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create a closure
	closureDate := time.Date(currentYear, 8, 15, 0, 0, 0, 0, time.UTC)
	_, err = testutil.NewSpecialDayBuilder().
		WithDate(closureDate).
		WithName("Summer Closure").
		WithType(domain.SpecialDayTypeClosure).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Request without holidays
	req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/special-days?year=%d&includeHolidays=false", currentYear), nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response []SpecialDayResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	// Should not contain Christmas (holiday)
	for _, day := range response {
		assert.NotEqual(t, "HOLIDAY", day.DayType, "Should not include holidays")
	}
}

func TestSpecialDayHandler_List_RequiresYear(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin3list@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/special-days", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestSpecialDayHandler_Holidays(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("list holidays for year", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminholidays@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		currentYear := time.Now().Year()

		// Create a holiday
		holidayDate := time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err = testutil.NewSpecialDayBuilder().
			WithDate(holidayDate).
			WithName("New Year").
			WithType(domain.SpecialDayTypeHoliday).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		// Create a closure (should not appear)
		closureDate := time.Date(currentYear, 8, 15, 0, 0, 0, 0, time.UTC)
		_, err = testutil.NewSpecialDayBuilder().
			WithDate(closureDate).
			WithName("Summer Closure").
			WithType(domain.SpecialDayTypeClosure).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		req := server.AuthenticatedRequest(t, "GET", fmt.Sprintf("/api/special-days/holidays/%d", currentYear), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var response []SpecialDayResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// All returned should be holidays
		for _, day := range response {
			assert.Equal(t, "HOLIDAY", day.DayType)
		}
	})
}

func TestSpecialDayHandler_Create(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("create special day successfully", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("admincreate@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		body := map[string]interface{}{
			"date":       "2026-12-26",
			"name":       "Boxing Day",
			"dayType":    "HOLIDAY",
			"affectsAll": true,
		}

		req := server.AuthenticatedRequest(t, "POST", "/api/special-days", body, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusCreated)

		var response SpecialDayResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Boxing Day", response.Name)
		assert.Equal(t, "HOLIDAY", response.DayType)
		assert.Equal(t, "2026-12-26", response.Date)
		assert.True(t, response.AffectsAll)
	})
}

func TestSpecialDayHandler_Create_Closure_WithDateRange(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2create@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":       "2026-08-01",
		"endDate":    "2026-08-15",
		"name":       "Summer Closure",
		"dayType":    "CLOSURE",
		"affectsAll": true,
		"notes":      "Annual summer break",
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/special-days", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	var response SpecialDayResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Summer Closure", response.Name)
	assert.Equal(t, "CLOSURE", response.DayType)
	assert.Equal(t, "2026-08-01", response.Date)
	require.NotNil(t, response.EndDate)
	assert.Equal(t, "2026-08-15", *response.EndDate)
	require.NotNil(t, response.Notes)
	assert.Equal(t, "Annual summer break", *response.Notes)
}

func TestSpecialDayHandler_Create_TeamDay(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin3create@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":       "2026-03-15",
		"name":       "Team Building Day",
		"dayType":    "TEAM_DAY",
		"affectsAll": true,
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/special-days", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	var response SpecialDayResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "TEAM_DAY", response.DayType)
}

func TestSpecialDayHandler_Create_Event(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin4create@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":       "2026-06-20",
		"name":       "Summer Festival",
		"dayType":    "EVENT",
		"affectsAll": false,
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/special-days", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	var response SpecialDayResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "EVENT", response.DayType)
	assert.False(t, response.AffectsAll)
}

func TestSpecialDayHandler_Create_ValidationError(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("adminval@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	// Missing required fields
	body := map[string]interface{}{
		"date": "2026-12-26",
		// missing name and dayType
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/special-days", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestSpecialDayHandler_Create_InvalidDayType(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admininvalid@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":    "2026-12-26",
		"name":    "Test Day",
		"dayType": "INVALID_TYPE",
	}

	req := server.AuthenticatedRequest(t, "POST", "/api/special-days", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestSpecialDayHandler_Update(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("update special day successfully", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("adminupdate@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		date := time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)
		day, err := testutil.NewSpecialDayBuilder().
			WithDate(date).
			WithName("Christmas").
			WithType(domain.SpecialDayTypeHoliday).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		body := map[string]interface{}{
			"date":       "2026-12-25",
			"name":       "Christmas Day",
			"dayType":    "HOLIDAY",
			"affectsAll": true,
			"notes":      "Updated note",
		}

		req := server.AuthenticatedRequest(t, "PUT", fmt.Sprintf("/api/special-days/%d", day.ID), body, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusOK)

		var response SpecialDayResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Christmas Day", response.Name)
		require.NotNil(t, response.Notes)
		assert.Equal(t, "Updated note", *response.Notes)
	})
}

func TestSpecialDayHandler_Update_ChangeType(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin2update@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	day, err := testutil.NewSpecialDayBuilder().
		WithDate(date).
		WithName("Event Day").
		WithType(domain.SpecialDayTypeEvent).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":       "2026-06-15",
		"name":       "Team Day",
		"dayType":    "TEAM_DAY",
		"affectsAll": true,
	}

	req := server.AuthenticatedRequest(t, "PUT", fmt.Sprintf("/api/special-days/%d", day.ID), body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response SpecialDayResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Team Day", response.Name)
	assert.Equal(t, "TEAM_DAY", response.DayType)
}

func TestSpecialDayHandler_Update_AddEndDate(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin3update@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	day, err := testutil.NewSpecialDayBuilder().
		WithDate(date).
		WithName("Summer Closure").
		WithType(domain.SpecialDayTypeClosure).
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":       "2026-08-01",
		"endDate":    "2026-08-14",
		"name":       "Summer Closure",
		"dayType":    "CLOSURE",
		"affectsAll": true,
	}

	req := server.AuthenticatedRequest(t, "PUT", fmt.Sprintf("/api/special-days/%d", day.ID), body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response SpecialDayResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	require.NotNil(t, response.EndDate)
	assert.Equal(t, "2026-08-14", *response.EndDate)
}

func TestSpecialDayHandler_Update_NotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin404@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	body := map[string]interface{}{
		"date":    "2026-12-26",
		"name":    "Test",
		"dayType": "HOLIDAY",
	}

	req := server.AuthenticatedRequest(t, "PUT", "/api/special-days/99999", body, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNotFound)
}

func TestSpecialDayHandler_Delete(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("delete special day successfully", func(t *testing.T) {
		admin, err := testutil.NewEmployeeBuilder().
			WithEmail("admindelete@example.com").
			AsAdmin().
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		date := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
		day, err := testutil.NewSpecialDayBuilder().
			WithDate(date).
			WithName("New Year's Eve").
			WithType(domain.SpecialDayTypeHoliday).
			Create(ctx, suite.Container.DB)
		require.NoError(t, err)

		req := server.AuthenticatedRequest(t, "DELETE", fmt.Sprintf("/api/special-days/%d", day.ID), nil, admin)
		resp, err := server.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		testutil.AssertStatus(t, resp, http.StatusNoContent)
	})
}

func TestSpecialDayHandler_Delete_NotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	ctx := context.Background()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admindelete404@example.com").
		AsAdmin().
		Create(ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "DELETE", "/api/special-days/99999", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNotFound)
}
