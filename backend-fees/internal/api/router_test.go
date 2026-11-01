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
	"GET /installment-plans/",
	"POST /installment-plans/",
	"GET /installment-plans/{id}",
	"PUT /installment-plans/{id}",
	"POST /installment-plans/{id}/close",
	"POST /installment-plans/{id}/payments",
	"DELETE /installment-plans/{id}/payments/{paymentId}",
	"PUT /installment-plans/{id}/final-installment",
	"DELETE /installment-plans/{id}/final-installment",
	"GET /installment-plans/reminder-cases",
	"POST /installment-plans/{id}/reminder/preview",
	"POST /installment-plans/{id}/reminder/send",
	"PUT /installment-plans/{id}/contract",
	"GET /installment-plans/{id}/contract",
	"DELETE /installment-plans/{id}/contract",

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
	for _, role := range []string{"ADMIN", "USER", "PARENT_WORK", "LEITUNG"} {
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
	staff := map[string]bool{"ADMIN": true, "USER": true}
	masterData := map[string]bool{"ADMIN": true, "USER": true, "LEITUNG": true}
	childID := uuid.NewString()
	routes := []struct {
		path    string
		allowed map[string]bool
	}{
		{"/auth/me", map[string]bool{"ADMIN": true, "USER": true, "PARENT_WORK": true, "LEITUNG": true, "PARENT": true}},
		{"/users/", map[string]bool{"ADMIN": true}},
		{"/children/", masterData},
		{"/members/", masterData},
		{"/households/", masterData},
		{"/fees/", staff},
		{"/fee-schedules/", staff},
		{"/fees/reminders/settings", map[string]bool{"ADMIN": true}},
		{"/parent-work/rules", map[string]bool{"ADMIN": true, "PARENT_WORK": true, "LEITUNG": true}},
		{"/parent-work/board-terms", map[string]bool{"ADMIN": true}},
		{"/activity", map[string]bool{"ADMIN": true}},
		{"/parent-reports/", map[string]bool{"ADMIN": true}},
		{"/children/" + uuid.NewString() + "/changes", map[string]bool{"ADMIN": true}},
		{"/parents/" + uuid.NewString(), masterData},
		{"/me", map[string]bool{"PARENT": true}},
		// Fee, Einstufung, bank and note data: never for LEITUNG.
		{"/children/" + childID + "/ledger", staff},
		{"/children/" + childID + "/timeline", staff},
		{"/children/" + childID + "/notes", staff},
		{"/notes", staff},
		{"/einstufungen/", staff},
		{"/stichtagsmeldung/stats", staff},
		{"/stichtagsmeldung/report", staff},
		{"/stichtagsmeldung/children", staff},
		{"/stichtagsmeldung/jugendamt", masterData},
		{"/stichtagsmeldung/jugendamt/lists", masterData},
		{"/import/transactions", staff},
		{"/import/history", staff},
		{"/import/donations", staff},
		{"/households/options", staff},
		{"/banking-sync/status", map[string]bool{"ADMIN": true}},
	}
	for _, role := range []string{"ADMIN", "USER", "PARENT_WORK", "LEITUNG", "PARENT"} {
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

// LEITUNG edits master data but must not delete it (deleting a child or
// household cascades into fee data) or reach the bulk child import.
func TestRouter_LeitungWriteRoutes(t *testing.T) {
	router, jwtService := testRouter(t, "")
	id := uuid.NewString()
	routes := []struct {
		method, path string
		leitung      bool
	}{
		{http.MethodPost, "/children/", true},
		{http.MethodPut, "/children/" + id, true},
		{http.MethodPost, "/children/" + id + "/parents", true},
		{http.MethodPost, "/children/" + id + "/care-hours-history", true},
		{http.MethodPost, "/parents/", true},
		{http.MethodPut, "/parents/" + id, true},
		{http.MethodPost, "/households/", true},
		{http.MethodPut, "/households/" + id, true},
		{http.MethodPost, "/households/" + id + "/children", true},
		{http.MethodPost, "/members/", true},
		{http.MethodPut, "/members/" + id, true},
		{http.MethodDelete, "/children/" + id, false},
		{http.MethodDelete, "/parents/" + id, false},
		{http.MethodDelete, "/households/" + id, false},
		{http.MethodDelete, "/members/" + id, false},
		{http.MethodPost, "/children/" + id + "/notes", false},
		{http.MethodPost, "/children/import/parse", false},
		{http.MethodPost, "/children/import/execute", false},
		{http.MethodPost, "/fees/", false},
		{http.MethodPost, "/einstufungen/", false},
		{http.MethodPost, "/import/match", false},
	}
	leitung, err := jwtService.GenerateTokenPair(uuid.New(), "leitung@example.org", "LEITUNG")
	if err != nil {
		t.Fatal(err)
	}
	user, err := jwtService.GenerateTokenPair(uuid.New(), "user@example.org", "USER")
	if err != nil {
		t.Fatal(err)
	}
	do := func(method, path, token string) int {
		req := httptest.NewRequest(method, apiPrefix+path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	for _, route := range routes {
		t.Run(route.method+route.path, func(t *testing.T) {
			code := do(route.method, route.path, leitung.AccessToken)
			if route.leitung && (code == http.StatusUnauthorized || code == http.StatusForbidden) {
				t.Fatalf("LEITUNG: status %d, want handler reached", code)
			}
			if !route.leitung && code != http.StatusForbidden {
				t.Fatalf("LEITUNG: status %d, want 403", code)
			}
			if code := do(route.method, route.path, user.AccessToken); code == http.StatusForbidden || code == http.StatusUnauthorized {
				t.Fatalf("USER: status %d, want handler reached", code)
			}
		})
	}
}

// LEITUNG sees the parent-work accounts and records hours, but corrections,
// review, overrides and the import stay with ADMIN/PARENT_WORK.
func TestRouter_LeitungParentWorkRoutes(t *testing.T) {
	router, jwtService := testRouter(t, "")
	id := uuid.NewString()
	routes := []struct {
		method, path string
		leitung      bool
	}{
		{http.MethodGet, "/parent-work/overview", true},
		{http.MethodGet, "/parent-work/households", true},
		{http.MethodGet, "/parent-work/households/" + id, true},
		{http.MethodGet, "/parent-work/rules", true},
		{http.MethodPost, "/parent-work/entries", true},
		{http.MethodPut, "/parent-work/households/" + id + "/override", false},
		{http.MethodDelete, "/parent-work/households/" + id + "/override", false},
		{http.MethodPut, "/parent-work/entries/" + id, false},
		{http.MethodPost, "/parent-work/entries/" + id + "/void", false},
		{http.MethodPost, "/parent-work/entries/" + id + "/approve", false},
		{http.MethodPost, "/parent-work/entries/" + id + "/reject", false},
		{http.MethodPost, "/parent-work/import/parse", false},
		{http.MethodPost, "/parent-work/import/preview", false},
		{http.MethodPost, "/parent-work/import/execute", false},
	}
	token := func(role string) string {
		tokens, err := jwtService.GenerateTokenPair(uuid.New(), role+"@example.org", role)
		if err != nil {
			t.Fatal(err)
		}
		return tokens.AccessToken
	}
	leitung, parentWork := token("LEITUNG"), token("PARENT_WORK")
	do := func(method, path, token string) int {
		req := httptest.NewRequest(method, apiPrefix+path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	for _, route := range routes {
		t.Run(route.method+route.path, func(t *testing.T) {
			code := do(route.method, route.path, leitung)
			if route.leitung && (code == http.StatusUnauthorized || code == http.StatusForbidden) {
				t.Fatalf("LEITUNG: status %d, want handler reached", code)
			}
			if !route.leitung && code != http.StatusForbidden {
				t.Fatalf("LEITUNG: status %d, want 403", code)
			}
			if code := do(route.method, route.path, parentWork); code == http.StatusForbidden || code == http.StatusUnauthorized {
				t.Fatalf("PARENT_WORK: status %d, want handler reached", code)
			}
		})
	}
}

func TestRouter_InstallmentsRejectAllNonAdminRoles(t *testing.T) {
	router, jwtService := testRouter(t, "")
	for _, role := range []string{"USER", "LEITUNG", "PARENT_WORK", "PARENT"} {
		tokens, err := jwtService.GenerateTokenPair(uuid.New(), "roles@example.test", role)
		if err != nil {
			t.Fatal(err)
		}
		for _, route := range adminRoutes {
			if !strings.Contains(route, "/installment-plans") {
				continue
			}
			method, path, _ := strings.Cut(route, " ")
			req := httptest.NewRequest(method, apiPrefix+concretePath(path), nil)
			req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != 403 {
				t.Fatalf("%s %s: %d", role, route, w.Code)
			}
		}
	}
}
