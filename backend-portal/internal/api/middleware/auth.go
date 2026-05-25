package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/auth"
)

type contextKey string

const userContextKey contextKey = "portal-user"

func AuthMiddleware(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}

			parts := strings.Fields(authHeader)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			claims, err := jwtService.ValidateAccessToken(parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(userContextKey).(*auth.Claims)
	return claims
}
