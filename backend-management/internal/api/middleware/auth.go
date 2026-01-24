package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// UserContext holds the authenticated user's information.
type UserContext struct {
	UserID int64
	Email  string
	Role   string
}

// AuthMiddleware creates an authentication middleware.
func AuthMiddleware(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Authentifizierung erforderlich")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Unauthorized(w, "Authentifizierung erforderlich")
				return
			}

			tokenString := parts[1]
			claims, err := jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				response.Unauthorized(w, "Authentifizierung erforderlich")
				return
			}

			userCtx := &UserContext{
				UserID: claims.UserID,
				Email:  claims.Subject,
				Role:   claims.Role,
			}

			ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole creates a middleware that requires a specific role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userCtx, ok := r.Context().Value(UserContextKey).(*UserContext)
			if !ok {
				response.Unauthorized(w, "Authentifizierung erforderlich")
				return
			}

			for _, role := range roles {
				if userCtx.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "Unzureichende Berechtigung")
		})
	}
}

// GetUserFromContext retrieves the user context from the request.
func GetUserFromContext(r *http.Request) *UserContext {
	userCtx, ok := r.Context().Value(UserContextKey).(*UserContext)
	if !ok {
		return nil
	}
	return userCtx
}
