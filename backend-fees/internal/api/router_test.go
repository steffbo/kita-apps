package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

const apiPrefix = "/api/fees/v1"

// Routes that answer without a bearer token. Everything else under the API
// prefix must reject anonymous requests before a handler runs.
var publicRoutes = map[string]bool{
	"POST " + apiPrefix + "/auth/login":               true,
	"POST " + apiPrefix + "/auth/refresh":             true,
	"POST " + apiPrefix + "/auth/logout":              true,
	"POST " + apiPrefix + "/auth/invitation-password": true,
	"GET " + apiPrefix + "/childcare-fee/calculate":   true,
}

// Routes restricted to role ADMIN (RequireRole in router.go).
var adminRoutes = []string{
	"POST /banking-sync/run",
	"GET /banking-sync/status",
	"POST /banking-sync/cancel",
	"GET /fees/reminders/settings",
	"PUT /fees/reminders/settings",
	"GET /fees/email-logs",
	"GET /fees/reminder-cases",
	"POST /fees/reminder-cases/{householdId}/preview",
	"POST /fees/reminder-cases/{householdId}/send",
	"POST /fee-schedules/",
	"PUT /fee-schedules/{id}",
	"DELETE /fee-schedules/{id}",
	"GET /users/",
	"POST /users/",
	"GET /users/invitation-candidates",
	"POST /users/invitations",
	"POST /users/{id}/invitation",
	"PUT /users/{id}",
	"POST /users/{id}/password",
	"POST /users/{id}/impersonate",
	"GET /activity",
	"GET /parents/{id}/changes",
	"GET /children/{id}/changes",
	"GET /parent-reports/",
	"POST /parent-reports/{id}/resolve",
}

// testRouter wires the real router with nil handlers: every request asserted
// here must be answered by middleware before a handler is reached.
func testRouter(t *testing.T, importToken string) (http.Handler, *auth.JWTService) {
	t.Helper()
	jwtService := auth.NewJWTService("router-test-secret", time.Minute, time.Hour, "test")
	cfg := &config.Config{}
	cfg.Import.Token = importToken
	workRepo := routerParentWorkRepo{}
	workHandler := handler.NewParentWorkHandler(service.NewParentWorkService(workRepo))
	invitation := handler.NewAccountInvitationHandler(nil,
		auth.NewLoginLimiter(auth.DefaultLoginMaxPerAccount,
			auth.DefaultLoginMaxPerIP, auth.DefaultLoginWindow))
	return NewRouter(cfg, &Handlers{
		JWTService: jwtService, ParentWork: workHandler, Invitation: invitation,
	}), jwtService
}

type routerParentWorkRepo struct {
	repository.ParentWorkRepository
}

func (routerParentWorkRepo) ListRules(context.Context) ([]domain.ParentWorkRule, error) {
	return []domain.ParentWorkRule{}, nil
}

func (routerParentWorkRepo) ListTerms(context.Context) ([]domain.BoardTerm, error) {
	return []domain.BoardTerm{}, nil
}

func concretePath(pattern string) string {
	path := strings.ReplaceAll(pattern, "/*", "/")
	for strings.Contains(path, "{") {
		start := strings.Index(path, "{")
		end := strings.Index(path[start:], "}") + start
		path = path[:start] + uuid.NewString() + path[end+1:]
	}
	return path
}

func TestRouter_ProtectedRoutesRequireToken(t *testing.T) {
	router, _ := testRouter(t, "")
	mux, ok := router.(chi.Routes)
	if !ok {
		t.Fatal("router is not a chi.Routes")
	}

	checked := 0
	err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, apiPrefix) || publicRoutes[method+" "+route] {
			return nil
		}
		checked++
		req := httptest.NewRequest(method, concretePath(route), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: status %d, want 401", method, route, rec.Code)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 100 {
		t.Fatalf("only %d protected routes walked, router wiring changed?", checked)
	}
}

func TestRouter_AdminRoutesRejectUserRole(t *testing.T) {
	router, jwtService := testRouter(t, "")
	tokens, err := jwtService.GenerateTokenPair(uuid.New(), "user@example.org", "USER")
	if err != nil {
		t.Fatal(err)
	}

	for _, route := range adminRoutes {
		method, pattern, _ := strings.Cut(route, " ")
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(method, apiPrefix+concretePath(pattern), nil)
			req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status %d, want 403", rec.Code)
			}
		})
	}
}

func TestRouter_InvitationPasswordIsPublic(t *testing.T) {
	router, _ := testRouter(t, "")
	req := httptest.NewRequest(http.MethodPost, apiPrefix+"/auth/invitation-password",
		strings.NewReader("{"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("öffentliche Route: %d, erwartet 400", rec.Code)
	}
}

func TestRouter_Health(t *testing.T) {
	router, _ := testRouter(t, "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("health: %d %s", rec.Code, rec.Body.String())
	}
}

func TestRouter_UploadBodyLimit(t *testing.T) {
	router, _ := testRouter(t, "import-secret")

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "export.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(bytes.Repeat([]byte("a"), middleware.MaxUploadBytes+1)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "without import token", wantStatus: http.StatusUnauthorized},
		{name: "oversized upload with import token", token: "import-secret",
			wantStatus: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, apiPrefix+"/import/upload", bytes.NewReader(body.Bytes()))
			req.Header.Set("Content-Type", mw.FormDataContentType())
			if tt.token != "" {
				req.Header.Set(middleware.ImportTokenHeader, tt.token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRouter_ParentWorkRuleWriteRoles(t *testing.T) {
	router, jwtService := testRouter(t, "")
	for _, role := range []string{"ADMIN", "USER", "PARENT_WORK"} {
		t.Run(role, func(t *testing.T) {
			tokens, err := jwtService.GenerateTokenPair(uuid.New(), role+"@example.org", role)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, apiPrefix+"/parent-work/rules",
				strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if role == "ADMIN" && rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want handler validation", rec.Code)
			}
			if role != "ADMIN" && rec.Code != http.StatusForbidden {
				t.Fatalf("status %d, want 403", rec.Code)
			}
		})
	}
}

func TestRouter_RoleAreas(t *testing.T) {
	router, jwtService := testRouter(t, "")
	routes := []struct {
		path    string
		allowed map[string]bool
	}{
		{"/auth/me", map[string]bool{"ADMIN": true, "USER": true, "PARENT_WORK": true, "PARENT": true}},
		{"/users/", map[string]bool{"ADMIN": true}},
		{"/children/", map[string]bool{"ADMIN": true, "USER": true}},
		{"/fees/", map[string]bool{"ADMIN": true, "USER": true}},
		{"/fee-schedules/", map[string]bool{"ADMIN": true, "USER": true}},
		{"/fees/reminders/settings", map[string]bool{"ADMIN": true}},
		{"/parent-work/rules", map[string]bool{"ADMIN": true, "PARENT_WORK": true}},
		{"/parent-work/board-terms", map[string]bool{"ADMIN": true}},
		{"/activity", map[string]bool{"ADMIN": true}},
		{"/parent-reports/", map[string]bool{"ADMIN": true}},
		{"/children/" + uuid.NewString() + "/changes", map[string]bool{"ADMIN": true}},
		{"/parents/" + uuid.NewString(), map[string]bool{"ADMIN": true, "USER": true}},
		{"/me", map[string]bool{"PARENT": true}},
	}
	for _, role := range []string{"ADMIN", "USER", "PARENT_WORK", "PARENT"} {
		tokens, err := jwtService.GenerateTokenPair(uuid.New(), role+"@example.org", role)
		if err != nil {
			t.Fatal(err)
		}
		for _, route := range routes {
			if route.path == "/me" && role == "PARENT" {
				continue
			}
			if route.path == "/activity" && role == "ADMIN" {
				continue
			}
			if strings.HasPrefix(route.path, "/parents/") && role != "PARENT" {
				continue
			}
			t.Run(role+route.path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, apiPrefix+route.path, nil)
				req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if !route.allowed[role] && rec.Code != http.StatusForbidden {
					t.Fatalf("status %d, want 403", rec.Code)
				}
				if route.allowed[role] && (rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden) {
					t.Fatalf("status %d, role should reach handler", rec.Code)
				}
				if route.path == "/parent-work/rules" && route.allowed[role] && rec.Code != http.StatusOK {
					t.Fatalf("rules: status %d, body %s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}
