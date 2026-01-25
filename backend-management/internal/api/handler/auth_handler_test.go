package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

var suite *testutil.TestSuite

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	container, err := testutil.SetupPostgres(ctx)
	if err != nil {
		panic("failed to setup postgres: " + err.Error())
	}

	suite = &testutil.TestSuite{
		Container: container,
		Ctx:       ctx,
	}

	code := m.Run()

	container.Cleanup(ctx)
	if code != 0 {
		panic("tests failed")
	}
}

func setupHandlerTest(t *testing.T) *testutil.TestServer {
	t.Helper()

	err := testutil.CleanupTables(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	return testutil.NewTestServer(t, suite.Container.DB)
}

// Auth Handler Tests

func TestAuthHandler_Login_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	// Create test employee
	_, err := testutil.NewEmployeeBuilder().
		WithEmail("login@example.com").
		WithPassword("Test1234!").
		WithName("Login", "User").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// Login
	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "login@example.com",
		"password": "Test1234!",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response testutil.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Greater(t, response.ExpiresIn, int64(0))
	assert.Equal(t, "login@example.com", response.User.Email)
	assert.Equal(t, "Login", response.User.FirstName)
}

func TestAuthHandler_Login_InvalidPassword(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	// Create test employee
	_, err := testutil.NewEmployeeBuilder().
		WithEmail("wrongpw@example.com").
		WithPassword("CorrectPassword123!").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// Login with wrong password
	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "wrongpw@example.com",
		"password": "WrongPassword123!",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuthHandler_Login_UserNotFound(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "nonexistent@example.com",
		"password": "Test1234!",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuthHandler_Login_InactiveUser(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	// Create inactive employee
	_, err := testutil.NewEmployeeBuilder().
		WithEmail("inactive@example.com").
		WithPassword("Test1234!").
		Inactive().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "inactive@example.com",
		"password": "Test1234!",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuthHandler_Login_ValidationError_MissingEmail(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"password": "Test1234!",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestAuthHandler_Login_ValidationError_InvalidEmail(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "not-an-email",
		"password": "Test1234!",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestAuthHandler_Login_ValidationError_ShortPassword(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "short",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	// Create and login
	_, err := testutil.NewEmployeeBuilder().
		WithEmail("refresh@example.com").
		WithPassword("Test1234!").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	auth := server.Login(t, "refresh@example.com", "Test1234!")

	// Refresh token
	req, err := server.Request("POST", "/api/auth/refresh", map[string]string{
		"refreshToken": auth.RefreshToken,
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response testutil.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	// New tokens should be different (though not strictly required)
	assert.NotEqual(t, auth.AccessToken, response.AccessToken)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("POST", "/api/auth/refresh", map[string]string{
		"refreshToken": "invalid-token",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuthHandler_Me_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("me@example.com").
		WithPassword("Test1234!").
		WithName("Me", "User").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/auth/me", nil, employee)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	var response testutil.EmployeeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "me@example.com", response.Email)
	assert.Equal(t, "Me", response.FirstName)
	assert.Equal(t, "User", response.LastName)
	assert.Equal(t, "ADMIN", response.Role)
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	req, err := server.Request("GET", "/api/auth/me", nil)
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("changepw@example.com").
		WithPassword("OldPassword123!").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/auth/change-password", map[string]string{
		"currentPassword": "OldPassword123!",
		"newPassword":     "NewPassword456!",
	}, employee)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	// Verify can login with new password
	loginReq, err := server.Request("POST", "/api/auth/login", map[string]string{
		"email":    "changepw@example.com",
		"password": "NewPassword456!",
	})
	require.NoError(t, err)

	loginResp, err := server.Do(loginReq)
	require.NoError(t, err)
	defer loginResp.Body.Close()

	testutil.AssertStatus(t, loginResp, http.StatusOK)
}

func TestAuthHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("wrongcurrent@example.com").
		WithPassword("CorrectPassword123!").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/auth/change-password", map[string]string{
		"currentPassword": "WrongPassword123!",
		"newPassword":     "NewPassword456!",
	}, employee)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestAuthHandler_PasswordReset_Request(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	_, err := testutil.NewEmployeeBuilder().
		WithEmail("reset@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	// Request password reset - should always return success (security)
	req, err := server.Request("POST", "/api/auth/password-reset/request", map[string]string{
		"email": "reset@example.com",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestAuthHandler_PasswordReset_Request_NonexistentEmail(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	// Should still return success (to prevent email enumeration)
	req, err := server.Request("POST", "/api/auth/password-reset/request", map[string]string{
		"email": "nonexistent@example.com",
	})
	require.NoError(t, err)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}
