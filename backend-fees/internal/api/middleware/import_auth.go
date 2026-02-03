package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
)

const (
	ImportTokenHeader = "X-Import-Token"
	ImportUserID      = "00000000-0000-0000-0000-000000000001"
)

// ImportAuthMiddleware allows either a JWT access token or an import token.
// The import token is intended only for automated CSV uploads.
func ImportAuthMiddleware(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
					response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
					return
				}

				tokenString := parts[1]
				claims, err := jwtService.ValidateToken(tokenString, auth.TokenTypeAccess)
				if err != nil {
					switch err {
					case auth.ErrExpiredToken:
						response.Error(w, http.StatusUnauthorized, "token has expired")
					default:
						response.Error(w, http.StatusUnauthorized, "invalid token")
					}
					return
				}

				userCtx := &UserContext{
					UserID: claims.UserID.String(),
					Email:  claims.Email,
					Role:   claims.Role,
				}

				ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			importToken := r.Header.Get(ImportTokenHeader)
			if importToken == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header or import token")
				return
			}

			expectedToken := os.Getenv("CRON_API_TOKEN")
			if expectedToken == "" || importToken != expectedToken {
				response.Error(w, http.StatusUnauthorized, "invalid import token")
				return
			}

	userCtx := &UserContext{
		UserID: ImportUserID,
		Email:  "importer@system.local",
		Role:   "USER",
	}

			ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
