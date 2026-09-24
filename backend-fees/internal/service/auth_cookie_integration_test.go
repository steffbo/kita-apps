package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
)

// HTTP-level auth flow against the real DB: the refresh token only travels in
// the httpOnly cookie, rotates on refresh, dies on logout, and wrong passwords
// are throttled. Lives here because this package owns the test container.

func newAuthHandler(t *testing.T) *handler.AuthHandler {
	t.Helper()
	authSvc, _ := newUserServices(t)
	if _, err := authSvc.BootstrapAdmin(context.Background(), "admin@example.com", "env-password"); err != nil {
		t.Fatal(err)
	}
	jwtSvc := auth.NewJWTService("test-secret-test-secret-test-secret", time.Minute, time.Hour, "test")
	return handler.NewAuthHandler(authSvc, jwtSvc, auth.NewLoginLimiter(3, 10, time.Minute))
}

func doLogin(h *handler.AuthHandler, password string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/fees/v1/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"`+password+`"}`))
	req.RemoteAddr = "10.0.0.1:1234"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	return rec
}

func withCookie(method, path string, c *http.Cookie) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if c != nil {
		req.AddCookie(c)
	}
	return req
}

func refreshCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "fees_refresh" {
			return c
		}
	}
	t.Fatalf("no fees_refresh cookie in response (status %d)", rec.Code)
	return nil
}

func TestAuthHTTP_RefreshTokenOnlyInHttpOnlyCookie(t *testing.T) {
	h := newAuthHandler(t)

	rec := doLogin(h, "env-password", map[string]string{"X-Forwarded-Proto": "https"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["accessToken"] == "" || body["accessToken"] == nil {
		t.Error("login body has no accessToken")
	}
	if _, ok := body["refreshToken"]; ok {
		t.Error("login body must not contain the refresh token")
	}
	c := refreshCookie(t, rec)
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/fees/v1/auth" || c.MaxAge <= 0 {
		t.Errorf("cookie attributes = %+v", c)
	}

	// Plain HTTP (local dev) gets a non-Secure cookie.
	if c := refreshCookie(t, doLogin(h, "env-password", nil)); c.Secure {
		t.Error("cookie is Secure on plain HTTP")
	}

	// Refresh rotates the cookie; the old one is dead afterwards.
	rec = httptest.NewRecorder()
	h.Refresh(rec, withCookie(http.MethodPost, "/api/fees/v1/auth/refresh", c))
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status %d: %s", rec.Code, rec.Body)
	}
	rotated := refreshCookie(t, rec)
	if rotated.Value == c.Value {
		t.Error("refresh did not rotate the token")
	}
	rec = httptest.NewRecorder()
	h.Refresh(rec, withCookie(http.MethodPost, "/api/fees/v1/auth/refresh", c))
	if rec.Code != http.StatusUnauthorized || refreshCookie(t, rec).MaxAge >= 0 {
		t.Errorf("reused cookie: status %d, want 401 and cleared cookie", rec.Code)
	}

	// No cookie, no session.
	rec = httptest.NewRecorder()
	h.Refresh(rec, withCookie(http.MethodPost, "/api/fees/v1/auth/refresh", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh without cookie: status %d, want 401", rec.Code)
	}

	// Logout revokes the current cookie and clears it.
	rec = httptest.NewRecorder()
	h.Logout(rec, withCookie(http.MethodPost, "/api/fees/v1/auth/logout", rotated))
	if rec.Code != http.StatusNoContent || refreshCookie(t, rec).MaxAge >= 0 {
		t.Errorf("logout: status %d, want 204 and cleared cookie", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Refresh(rec, withCookie(http.MethodPost, "/api/fees/v1/auth/refresh", rotated))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout: status %d, want 401", rec.Code)
	}
}

func TestAuthHTTP_LoginThrottled(t *testing.T) {
	h := newAuthHandler(t)

	for i := 0; i < 3; i++ {
		if rec := doLogin(h, "wrong-password", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d, want 401", i+1, rec.Code)
		}
	}
	// Blocked now, even with the correct password.
	rec := doLogin(h, "env-password", nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" || !strings.Contains(rec.Body.String(), "Zu viele fehlgeschlagene Versuche") {
		t.Errorf("429 response: Retry-After=%q body=%s", rec.Header().Get("Retry-After"), rec.Body)
	}
}
