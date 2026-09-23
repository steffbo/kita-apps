package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
)

const (
	ImportTokenHeader = "X-Import-Token"
	ImportUserID      = "00000000-0000-0000-0000-000000000001"
)

// ImportAuthMiddleware allows either a JWT access token or an import token.
// The import token (CRON_API_TOKEN) is intended only for automated CSV uploads
// by the banking-sync container. An empty importToken disables token auth.
func ImportAuthMiddleware(jwtService *auth.JWTService, importToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				userCtx, ok := authenticateBearer(w, jwtService, authHeader)
				if !ok {
					return
				}
				next.ServeHTTP(w, r.WithContext(withUser(r.Context(), userCtx)))
				return
			}

			provided := r.Header.Get(ImportTokenHeader)
			if provided == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header or import token")
				return
			}

			if importToken == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(importToken)) != 1 {
				response.Error(w, http.StatusUnauthorized, "invalid import token")
				return
			}

			userCtx := &UserContext{
				UserID: ImportUserID,
				Email:  "importer@system.local",
				Role:   "USER",
			}
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), userCtx)))
		})
	}
}
