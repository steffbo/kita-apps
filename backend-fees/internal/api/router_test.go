package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/config"
)

const apiPrefix = "/api/fees/v1"

// Routes that answer without a bearer token. Everything else under the API
// prefix must reject anonymous requests before a handler runs.
var publicRoutes = map[string]bool{
	"POST " + apiPrefix + "/auth/login":             true,
	"POST " + apiPrefix + "/auth/refresh":           true,
	"GET " + apiPrefix + "/childcare-fee/calculate": true,
}

// Routes restricted to role ADMIN (RequireRole in router.go).
var adminRoutes = []string{
	"POST /banking-sync/run",
	"GET /banking-sync/status",
	"POST /banking-sync/cancel",
	"POST /fees/reminders/run",
	"POST /fees/membership-reminders/run",
	"GET /fees/reminders/settings",
	"PUT /fees/reminders/settings",
	"GET /fees/email-logs",
	"GET /fees/reminder-cases",
	"POST /fees/reminder-cases/{householdId}/preview",
	"POST /fees/reminder-cases/{householdId}/send",
	"POST /fee-schedules/",
	"PUT /fee-schedules/{id}",
	"DELETE /fee-schedules/{id}",
}

// testRouter wires the real router with nil handlers: every request asserted
// here must be answered by middleware before a handler is reached.
func testRouter(t *testing.T, importToken string) (http.Handler, *auth.JWTService) {
	t.Helper()
	jwtService := auth.NewJWTService("router-test-secret", time.Minute, time.Hour, "test")
	cfg := &config.Config{}
	cfg.Import.Token = importToken
	return NewRouter(cfg, &Handlers{JWTService: jwtService}), jwtService
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
		{name: "oversized upload with import token", token: "import-secret", wantStatus: http.StatusRequestEntityTooLarge},
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
