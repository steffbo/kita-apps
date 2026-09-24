package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
)

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Message
}

func TestAuthMiddleware(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", time.Minute, time.Hour, "test")
	userID := uuid.New()
	tokens, err := jwtService.GenerateTokenPair(userID, "admin@example.org", "ADMIN")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	expiredService := auth.NewJWTService("test-secret", -time.Minute, time.Hour, "test")
	expired, err := expiredService.GenerateTokenPair(userID, "admin@example.org", "ADMIN")
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	otherSecret := auth.NewJWTService("other-secret", time.Minute, time.Hour, "test")
	forged, err := otherSecret.GenerateTokenPair(userID, "admin@example.org", "ADMIN")
	if err != nil {
		t.Fatalf("generate forged token: %v", err)
	}

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
		wantMessage   string
	}{
		{name: "missing header", wantStatus: http.StatusUnauthorized, wantMessage: "missing authorization header"},
		{name: "wrong scheme", authorization: "Basic abc", wantStatus: http.StatusUnauthorized, wantMessage: "invalid authorization header format"},
		{name: "no token", authorization: "Bearer", wantStatus: http.StatusUnauthorized, wantMessage: "invalid authorization header format"},
		{name: "garbage token", authorization: "Bearer not-a-jwt", wantStatus: http.StatusUnauthorized, wantMessage: "invalid token"},
		{name: "wrong signing secret", authorization: "Bearer " + forged.AccessToken, wantStatus: http.StatusUnauthorized, wantMessage: "invalid token"},
		{name: "refresh token used as access token", authorization: "Bearer " + tokens.RefreshToken, wantStatus: http.StatusUnauthorized, wantMessage: "invalid token"},
		{name: "expired access token", authorization: "Bearer " + expired.AccessToken, wantStatus: http.StatusUnauthorized, wantMessage: "token has expired"},
		{name: "valid access token", authorization: "Bearer " + tokens.AccessToken, wantStatus: http.StatusOK},
		{name: "scheme is case-insensitive", authorization: "bearer " + tokens.AccessToken, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser *UserContext
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = GetUserFromContext(r)
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/children", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			AuthMiddleware(jwtService)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				if gotUser != nil {
					t.Fatal("next handler must not run on auth failure")
				}
				if msg := errorMessage(t, rec); msg != tt.wantMessage {
					t.Fatalf("message = %q, want %q", msg, tt.wantMessage)
				}
				return
			}
			if gotUser == nil || gotUser.UserID != userID.String() || gotUser.Email != "admin@example.org" || gotUser.Role != "ADMIN" {
				t.Fatalf("user context = %+v", gotUser)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		user       *UserContext
		roles      []string
		wantStatus int
	}{
		{name: "no user in context", roles: []string{"ADMIN"}, wantStatus: http.StatusUnauthorized},
		{name: "role not allowed", user: &UserContext{Role: "USER"}, roles: []string{"ADMIN"}, wantStatus: http.StatusForbidden},
		{name: "role allowed", user: &UserContext{Role: "ADMIN"}, roles: []string{"ADMIN"}, wantStatus: http.StatusOK},
		{name: "one of several roles", user: &UserContext{Role: "USER"}, roles: []string{"ADMIN", "USER"}, wantStatus: http.StatusOK},
		{name: "role match is case-sensitive", user: &UserContext{Role: "admin"}, roles: []string{"ADMIN"}, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/fees/email-logs", nil)
			if tt.user != nil {
				req = req.WithContext(withUser(req.Context(), tt.user))
			}
			rec := httptest.NewRecorder()

			RequireRole(tt.roles...)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != (tt.wantStatus == http.StatusOK) {
				t.Fatalf("next called = %v", called)
			}
		})
	}
}
