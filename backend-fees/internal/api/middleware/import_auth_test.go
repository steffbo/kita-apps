package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
)

func TestImportAuthMiddleware(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", time.Minute, time.Hour, "test")
	tokens, err := jwtService.GenerateTokenPair(uuid.New(), "admin@example.org", "ADMIN")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	tests := []struct {
		name          string
		configured    string
		authorization string
		importToken   string
		wantStatus    int
		wantUserID    string
	}{
		{name: "valid import token", configured: "s3cret", importToken: "s3cret", wantStatus: http.StatusOK, wantUserID: ImportUserID},
		{name: "wrong import token", configured: "s3cret", importToken: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "token auth disabled", configured: "", importToken: "anything", wantStatus: http.StatusUnauthorized},
		{name: "no credentials", configured: "s3cret", wantStatus: http.StatusUnauthorized},
		{name: "valid bearer", configured: "s3cret", authorization: "Bearer " + tokens.AccessToken, wantStatus: http.StatusOK},
		{name: "invalid bearer wins over valid import token", configured: "s3cret", authorization: "Bearer nope", importToken: "s3cret", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser *UserContext
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = GetUserFromContext(r)
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/import/upload", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			if tt.importToken != "" {
				req.Header.Set(ImportTokenHeader, tt.importToken)
			}
			rec := httptest.NewRecorder()

			ImportAuthMiddleware(jwtService, tt.configured)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantUserID != "" && (gotUser == nil || gotUser.UserID != tt.wantUserID) {
				t.Fatalf("user = %+v, want ID %s", gotUser, tt.wantUserID)
			}
		})
	}
}
