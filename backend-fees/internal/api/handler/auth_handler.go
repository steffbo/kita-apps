package handler

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// The refresh token lives only in this httpOnly cookie, so page scripts can
// never read it. It is scoped to the auth endpoints and never sent cross-site.
const (
	refreshCookieName = "fees_refresh"
	refreshCookiePath = "/api/fees/v1/auth"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	authService  *service.AuthService
	jwtService   *auth.JWTService
	loginLimiter *auth.LoginLimiter
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService *service.AuthService, jwtService *auth.JWTService, loginLimiter *auth.LoginLimiter) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		jwtService:   jwtService,
		loginLimiter: loginLimiter,
	}
}

// isHTTPS reports whether the client connection is TLS, directly or via the
// reverse proxy. Plain-HTTP local development gets a non-Secure cookie.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func setRefreshCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func refreshCookieValue(r *http.Request) string {
	c, err := r.Cookie(refreshCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// clientIP is the remote address after chi's RealIP middleware (taken from
// the proxy's X-Forwarded-For), without a port.
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func tooManyAttempts(w http.ResponseWriter, retryAfter time.Duration) {
	minutes := int(math.Ceil(retryAfter.Minutes()))
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
	unit := "Minuten"
	if minutes == 1 {
		unit = "Minute"
	}
	response.Error(w, http.StatusTooManyRequests, fmt.Sprintf(
		"Zu viele fehlgeschlagene Versuche. Bitte in %d %s erneut versuchen.", minutes, unit))
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" example:"admin@example.com"`
	Password string `json:"password" example:"password123"`
} //@name LoginRequest

// LoginResponse represents a login response. The refresh token is not part
// of the body; it is set as the httpOnly cookie "fees_refresh".
type LoginResponse struct {
	AccessToken string       `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresAt   string       `json:"expiresAt" example:"2024-01-27T15:04:05Z"`
	User        UserResponse `json:"user"`
} //@name LoginResponse

// RefreshResponse carries a new access token; the rotated refresh token is
// set as cookie.
type RefreshResponse struct {
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresAt   string `json:"expiresAt" example:"2024-01-27T15:04:05Z"`
} //@name RefreshResponse

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string  `json:"email" example:"user@example.com"`
	FirstName *string `json:"firstName,omitempty" example:"Max" binding:"optional"`
	LastName  *string `json:"lastName,omitempty" example:"Mustermann" binding:"optional"`
	Role      string  `json:"role" example:"ADMIN" enums:"ADMIN,USER"`
} //@name User

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
} //@name MessageResponse

// Login handles user authentication
// @Summary User login
// @Description Authenticate with email and password. Returns the access token and sets the refresh token as httpOnly cookie. Repeated failures are throttled per IP and account.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Successful login"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Invalid credentials"
// @Failure 429 {object} response.ErrorBody "Too many failed attempts"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.BadRequest(w, "E-Mail und Passwort sind erforderlich")
		return
	}

	ip := clientIP(r)
	if blocked, retryAfter := h.loginLimiter.Blocked(ip, req.Email); blocked {
		tooManyAttempts(w, retryAfter)
		return
	}

	user, err := h.authService.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			h.loginLimiter.Fail(ip, req.Email)
			response.Unauthorized(w, "E-Mail oder Passwort ist falsch")
			return
		}
		response.InternalError(w, "login failed")
		return
	}
	h.loginLimiter.Succeed(ip, req.Email)

	tokenPair, err := h.issueTokens(w, r, user.ID, user.Email, string(user.Role))
	if err != nil {
		response.InternalError(w, "failed to issue tokens")
		return
	}

	response.Success(w, LoginResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresAt:   tokenPair.ExpiresAt.Format(time.RFC3339),
		User: UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      string(user.Role),
		},
	})
}

// issueTokens generates a token pair, stores the refresh token and sets it as cookie.
func (h *AuthHandler) issueTokens(w http.ResponseWriter, r *http.Request, userID uuid.UUID, email, role string) (*auth.TokenPair, error) {
	tokenPair, err := h.jwtService.GenerateTokenPair(userID, email, role)
	if err != nil {
		return nil, err
	}
	if err := h.authService.StoreRefreshToken(r.Context(), userID, tokenPair.RefreshToken); err != nil {
		return nil, err
	}
	setRefreshCookie(w, r, tokenPair.RefreshToken, tokenPair.RefreshExpiresAt)
	return tokenPair, nil
}

// Refresh refreshes an access token using the refresh token cookie
// @Summary Refresh access token
// @Description Exchange the refresh token cookie for a new access token; the refresh token is rotated
// @Tags Auth
// @Produce json
// @Success 200 {object} RefreshResponse "Token refreshed"
// @Failure 401 {object} response.ErrorBody "Missing, invalid or revoked refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := refreshCookieValue(r)
	if refreshToken == "" {
		response.Unauthorized(w, "no session")
		return
	}

	reject := func(message string) {
		clearRefreshCookie(w, r)
		response.Unauthorized(w, message)
	}

	claims, err := h.jwtService.ValidateToken(refreshToken, auth.TokenTypeRefresh)
	if err != nil {
		reject("invalid refresh token")
		return
	}

	// Verify token is still valid in database
	valid, err := h.authService.ValidateRefreshToken(r.Context(), claims.UserID, refreshToken)
	if err != nil || !valid {
		reject("refresh token has been revoked")
		return
	}

	// Email and role come from the account, so changes apply with the next refresh.
	user, err := h.authService.GetUserByID(r.Context(), claims.UserID)
	if err != nil || !user.IsActive {
		_ = h.authService.RevokeRefreshToken(r.Context(), refreshToken)
		reject("account is not active")
		return
	}

	// Revoke old refresh token (best effort; a new pair is issued anyway)
	_ = h.authService.RevokeRefreshToken(r.Context(), refreshToken)

	tokenPair, err := h.issueTokens(w, r, user.ID, user.Email, string(user.Role))
	if err != nil {
		response.InternalError(w, "failed to issue tokens")
		return
	}

	response.Success(w, RefreshResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresAt:   tokenPair.ExpiresAt.Format(time.RFC3339),
	})
}

// Logout handles user logout
// @Summary Logout
// @Description Revoke the refresh token cookie and clear it, ending the session. Works without a valid access token.
// @Tags Auth
// @Success 204 "Logged out successfully"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := refreshCookieValue(r); token != "" {
		_ = h.authService.RevokeRefreshToken(r.Context(), token)
	}
	clearRefreshCookie(w, r)
	response.NoContent(w)
}

// Me returns the current authenticated user
// @Summary Get current user info
// @Description Returns information about the currently authenticated user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse "Current user info"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "User not found"
// @Router /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	userID, err := uuid.Parse(userCtx.UserID)
	if err != nil {
		response.InternalError(w, "invalid user ID")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		response.NotFound(w, "user not found")
		return
	}

	response.Success(w, map[string]interface{}{
		"id":        user.ID.String(),
		"email":     user.Email,
		"firstName": user.FirstName,
		"lastName":  user.LastName,
		"role":      string(user.Role),
	})
}

// ChangePasswordRequest represents a change password request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" example:"oldPassword123"`
	NewPassword     string `json:"newPassword" example:"newPassword456" minLength:"8"`
} //@name ChangePasswordRequest

// ChangePassword changes the current user's password
// @Summary Change own password
// @Description Change the password for the currently authenticated user. Ends all other sessions and returns a fresh access token (new refresh cookie). Wrong current passwords are throttled.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Current and new password"
// @Success 200 {object} RefreshResponse "Password changed; new session"
// @Failure 400 {object} response.ErrorBody "Invalid request or current password incorrect"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "User not found"
// @Failure 429 {object} response.ErrorBody "Too many failed attempts"
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		response.BadRequest(w, "current and new password are required")
		return
	}

	if len(req.NewPassword) < service.MinPasswordLength {
		response.BadRequest(w, "Das neue Passwort muss mindestens 8 Zeichen haben")
		return
	}

	userID, err := uuid.Parse(userCtx.UserID)
	if err != nil {
		response.InternalError(w, "invalid user ID")
		return
	}

	// Guessing the current password with a stolen access token is throttled too.
	ip, account := clientIP(r), "password-change:"+userID.String()
	if blocked, retryAfter := h.loginLimiter.Blocked(ip, account); blocked {
		tooManyAttempts(w, retryAfter)
		return
	}

	err = h.authService.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			h.loginLimiter.Fail(ip, account)
			response.BadRequest(w, "Das aktuelle Passwort ist falsch")
		case errors.Is(err, service.ErrNotFound):
			response.NotFound(w, "user not found")
		default:
			response.InternalError(w, "failed to change password")
		}
		return
	}
	h.loginLimiter.Succeed(ip, account)

	// All sessions were revoked; start a fresh one for this browser.
	tokenPair, err := h.issueTokens(w, r, userID, userCtx.Email, userCtx.Role)
	if err != nil {
		response.InternalError(w, "failed to issue tokens")
		return
	}
	response.Success(w, RefreshResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresAt:   tokenPair.ExpiresAt.Format(time.RFC3339),
	})
}
