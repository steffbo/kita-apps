package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/email"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// TestServer wraps an HTTP test server with helpers.
type TestServer struct {
	Server     *httptest.Server
	DB         *sqlx.DB
	JWTService *auth.JWTService
	Config     *config.Config
	Handlers   *api.Handlers
}

// NewTestServer creates a fully wired test server with all handlers.
func NewTestServer(t *testing.T, db *sqlx.DB) *TestServer {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         "8080",
			CORSOrigins:  []string{"*"},
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-for-jwt-tokens-minimum-256-bits",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "kita-test",
		},
	}

	// Create JWT service
	jwtService := auth.NewJWTService(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.JWT.Issuer,
	)

	// Create repositories
	employeeRepo := repository.NewPostgresEmployeeRepository(db)
	groupRepo := repository.NewPostgresGroupRepository(db)
	groupAssignmentRepo := repository.NewPostgresGroupAssignmentRepository(db)
	scheduleRepo := repository.NewPostgresScheduleRepository(db)
	timeEntryRepo := repository.NewPostgresTimeEntryRepository(db)
	specialDayRepo := repository.NewPostgresSpecialDayRepository(db)

	// Create services
	emailService := email.NewService(email.Config{}) // Disabled for tests
	authService := service.NewAuthService(employeeRepo, jwtService, emailService, "http://localhost:5173", 1*time.Hour)
	employeeService := service.NewEmployeeService(employeeRepo, groupAssignmentRepo, groupRepo)
	groupService := service.NewGroupService(groupRepo, groupAssignmentRepo, employeeRepo)
	scheduleService := service.NewScheduleService(scheduleRepo, employeeRepo, groupRepo, specialDayRepo)
	timeTrackingService := service.NewTimeTrackingService(timeEntryRepo, employeeRepo, scheduleRepo)
	specialDayService := service.NewSpecialDayService(specialDayRepo)
	statisticsService := service.NewStatisticsService(
		employeeRepo,
		scheduleRepo,
		timeEntryRepo,
		groupRepo,
	)

	// Create handlers
	handlers := &api.Handlers{
		Auth:         handler.NewAuthHandler(authService),
		Employee:     handler.NewEmployeeHandler(employeeService),
		Group:        handler.NewGroupHandler(groupService),
		Schedule:     handler.NewScheduleHandler(scheduleService),
		TimeTracking: handler.NewTimeTrackingHandler(timeTrackingService),
		SpecialDay:   handler.NewSpecialDayHandler(specialDayService),
		Statistics:   handler.NewStatisticsHandler(statisticsService),
		JWTService:   jwtService,
	}

	// Create router
	router := api.NewRouter(cfg, handlers)

	// Create test server
	server := httptest.NewServer(router)

	return &TestServer{
		Server:     server,
		DB:         db,
		JWTService: jwtService,
		Config:     cfg,
		Handlers:   handlers,
	}
}

// Close shuts down the test server.
func (ts *TestServer) Close() {
	ts.Server.Close()
}

// URL returns the base URL of the test server.
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// Request creates a new HTTP request to the test server.
func (ts *TestServer) Request(method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, ts.URL()+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// Do executes an HTTP request and returns the response.
func (ts *TestServer) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// AuthenticatedRequest creates an authenticated request with a valid JWT token.
func (ts *TestServer) AuthenticatedRequest(t *testing.T, method, path string, body interface{}, employee *domain.Employee) *http.Request {
	t.Helper()

	req, err := ts.Request(method, path, body)
	require.NoError(t, err)

	token, err := ts.JWTService.GenerateAccessToken(
		employee.ID,
		employee.Email,
		string(employee.Role),
		employee.FullName(),
	)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// Login performs a login and returns the auth response.
func (ts *TestServer) Login(t *testing.T, email, password string) *AuthResponse {
	t.Helper()

	req, err := ts.Request("POST", "/api/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)

	resp, err := ts.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var response AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	return &response
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error   string            `json:"error,omitempty"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// AuthResponse represents the auth endpoint response.
type AuthResponse struct {
	AccessToken  string           `json:"accessToken"`
	RefreshToken string           `json:"refreshToken"`
	ExpiresIn    int64            `json:"expiresIn"`
	User         EmployeeResponse `json:"user"`
}

// EmployeeResponse represents the employee in API responses.
type EmployeeResponse struct {
	ID                    int64   `json:"id"`
	Email                 string  `json:"email"`
	FirstName             string  `json:"firstName"`
	LastName              string  `json:"lastName"`
	Role                  string  `json:"role"`
	WeeklyHours           float64 `json:"weeklyHours"`
	VacationDaysPerYear   int     `json:"vacationDaysPerYear"`
	RemainingVacationDays float64 `json:"remainingVacationDays"`
	OvertimeBalance       float64 `json:"overtimeBalance"`
	Active                bool    `json:"active"`
}

// AssertStatus checks the response status code.
func AssertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// ParseJSON parses the response body into the given type.
func ParseJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result T
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "failed to parse response: %s", string(body))

	return result
}

// TestSuite provides a base for integration test suites.
type TestSuite struct {
	Container *TestContainer
	Server    *TestServer
	Fixtures  *TestFixtures
	Ctx       context.Context
}

// SetupSuite initializes the test container and server.
func SetupSuite(t *testing.T) *TestSuite {
	t.Helper()

	ctx := context.Background()

	container, err := SetupPostgres(ctx)
	require.NoError(t, err, "failed to setup postgres container")

	server := NewTestServer(t, container.DB)
	fixtures := NewTestFixtures(container.DB)

	return &TestSuite{
		Container: container,
		Server:    server,
		Fixtures:  fixtures,
		Ctx:       ctx,
	}
}

// TeardownSuite cleans up resources.
func (s *TestSuite) TeardownSuite(t *testing.T) {
	t.Helper()

	s.Server.Close()
	if err := s.Container.Cleanup(s.Ctx); err != nil {
		t.Logf("warning: failed to cleanup container: %v", err)
	}
}

// Cleanup truncates all tables between tests.
func (s *TestSuite) Cleanup(t *testing.T) {
	t.Helper()
	err := CleanupTables(s.Ctx, s.Container.DB)
	require.NoError(t, err, "failed to cleanup tables")
}
