package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

// Employee Handler Tests

func TestEmployeeHandler_List_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		WithName("Admin", "User").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// Create additional employees
	_, err = testutil.NewEmployeeBuilder().
		WithEmail("employee1@example.com").
		WithName("Employee", "One").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("employee2@example.com").
		WithName("Employee", "Two").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/employees", nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response []testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response, 3)
}

func TestEmployeeHandler_List_ExcludesInactive(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("active@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("inactive@example.com").
		Inactive().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// Without includeInactive
	req := server.AuthenticatedRequest(t, "GET", "/api/employees", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var response []testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response, 2) // Only active employees
}

func TestEmployeeHandler_List_IncludesInactive(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("active@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("inactive@example.com").
		Inactive().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// With includeInactive=true
	req := server.AuthenticatedRequest(t, "GET", "/api/employees?includeInactive=true", nil, admin)
	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var response []testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response, 3) // All employees including inactive
}

func TestEmployeeHandler_Get_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		WithName("Test", "Employee").
		WithWeeklyHours(35.0).
		WithVacationDays(28, 25.5).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/employees/"+itoa(employee.ID), nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "employee@example.com", response.Email)
	assert.Equal(t, "Test", response.FirstName)
	assert.Equal(t, "Employee", response.LastName)
	assert.Equal(t, 35.0, response.WeeklyHours)
	assert.Equal(t, 28, response.VacationDaysPerYear)
	assert.Equal(t, 25.5, response.RemainingVacationDays)
}

func TestEmployeeHandler_Get_NotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/employees/99999", nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNotFound)
}

func TestEmployeeHandler_Create_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/employees", map[string]interface{}{
		"email":               "new@example.com",
		"firstName":           "New",
		"lastName":            "Employee",
		"role":                "EMPLOYEE",
		"weeklyHours":         40.0,
		"vacationDaysPerYear": 30,
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	var response testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotZero(t, response.ID)
	assert.Equal(t, "new@example.com", response.Email)
	assert.Equal(t, "New", response.FirstName)
	assert.Equal(t, "Employee", response.LastName)
	assert.Equal(t, "EMPLOYEE", response.Role)
	assert.Equal(t, 40.0, response.WeeklyHours)
	assert.True(t, response.Active)
}

func TestEmployeeHandler_Create_AsAdmin(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/employees", map[string]interface{}{
		"email":       "newadmin@example.com",
		"firstName":   "New",
		"lastName":    "Admin",
		"role":        "ADMIN",
		"weeklyHours": 40.0,
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)

	var response testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ADMIN", response.Role)
}

func TestEmployeeHandler_Create_Forbidden_NonAdmin(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/employees", map[string]interface{}{
		"email":       "new@example.com",
		"firstName":   "New",
		"lastName":    "Employee",
		"weeklyHours": 40.0,
	}, employee)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusForbidden)
}

func TestEmployeeHandler_Create_ValidationError(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// Missing required fields
	req := server.AuthenticatedRequest(t, "POST", "/api/employees", map[string]interface{}{
		"email": "invalid",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestEmployeeHandler_Create_DuplicateEmail(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("existing@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/employees", map[string]interface{}{
		"email":       "existing@example.com",
		"firstName":   "New",
		"lastName":    "Employee",
		"weeklyHours": 40.0,
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusConflict)
}

func TestEmployeeHandler_Update_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		WithName("Old", "Name").
		WithWeeklyHours(40.0).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "PUT", "/api/employees/"+itoa(employee.ID), map[string]interface{}{
		"firstName":   "Updated",
		"lastName":    "Employee",
		"weeklyHours": 35.0,
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Updated", response.FirstName)
	assert.Equal(t, "Employee", response.LastName)
	assert.Equal(t, 35.0, response.WeeklyHours)
}

func TestEmployeeHandler_Delete_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("todelete@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "DELETE", "/api/employees/"+itoa(employee.ID), nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNoContent)

	// Verify employee is deactivated (soft delete)
	getReq := server.AuthenticatedRequest(t, "GET", "/api/employees/"+itoa(employee.ID), nil, admin)
	getResp, err := server.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	var response testutil.EmployeeResponse
	err = json.NewDecoder(getResp.Body).Decode(&response)
	require.NoError(t, err)

	assert.False(t, response.Active)
}

func TestEmployeeHandler_Assignments_Success(t *testing.T) {
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

	_, err = testutil.NewGroupAssignmentBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/employees/"+itoa(employee.ID)+"/assignments", nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

// Helper function
func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
