package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// UserContext holds the authenticated user's information.
type UserContext struct {
	UserID string
	Email  string
	Role   string
}

// AuthMiddleware creates an authentication middleware.
func AuthMiddleware(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			userCtx, ok := authenticateBearer(w, jwtService, authHeader)
			if !ok {
				return
			}

			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), userCtx)))
		})
	}
}

// authenticateBearer validates a "Bearer <token>" header. On failure it writes
// the 401 response and returns ok=false.
func authenticateBearer(w http.ResponseWriter, jwtService *auth.JWTService, authHeader string) (*UserContext, bool) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
		return nil, false
	}

	claims, err := jwtService.ValidateToken(parts[1], auth.TokenTypeAccess)
	if err != nil {
		switch err {
		case auth.ErrExpiredToken:
			response.Error(w, http.StatusUnauthorized, "token has expired")
		default:
			response.Error(w, http.StatusUnauthorized, "invalid token")
		}
		return nil, false
	}

	return &UserContext{
		UserID: claims.UserID.String(),
		Email:  claims.Email,
		Role:   claims.Role,
	}, true
}

func withUser(ctx context.Context, userCtx *UserContext) context.Context {
	return context.WithValue(ctx, UserContextKey, userCtx)
}

// RequireRole creates a middleware that requires a specific role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userCtx, ok := r.Context().Value(UserContextKey).(*UserContext)
			if !ok {
				response.Error(w, http.StatusUnauthorized, "user not authenticated")
				return
			}

			for _, role := range roles {
				if userCtx.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Error(w, http.StatusForbidden, "insufficient permissions")
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
