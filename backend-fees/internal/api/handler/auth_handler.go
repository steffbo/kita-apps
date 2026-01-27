package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	authService *service.AuthService
	jwtService  *auth.JWTService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService *service.AuthService, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtService:  jwtService,
	}
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" example:"admin@example.com"`
	Password string `json:"password" example:"password123"`
} //@name LoginRequest

// LoginResponse represents a login response.
type LoginResponse struct {
	AccessToken  string       `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string       `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresAt    string       `json:"expiresAt" example:"2024-01-27T15:04:05Z"`
	User         UserResponse `json:"user"`
} //@name LoginResponse

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string  `json:"email" example:"user@example.com"`
	FirstName *string `json:"firstName,omitempty" example:"Max"`
	LastName  *string `json:"lastName,omitempty" example:"Mustermann"`
	Role      string  `json:"role" example:"ADMIN" enums:"ADMIN,USER"`
} //@name User

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
} //@name MessageResponse

// Login handles user authentication
// @Summary User login
// @Description Authenticate a user with email and password, returns JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Successful login"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Invalid credentials"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.BadRequest(w, "email and password are required")
		return
	}

	user, err := h.authService.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Unauthorized(w, "invalid credentials")
		return
	}

	tokenPair, err := h.jwtService.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		response.InternalError(w, "failed to generate tokens")
		return
	}

	// Store refresh token
	if err := h.authService.StoreRefreshToken(r.Context(), user.ID, tokenPair.RefreshToken); err != nil {
		response.InternalError(w, "failed to store refresh token")
		return
	}

	resp := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		User: UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      string(user.Role),
		},
	}

	response.Success(w, resp)
}

// RefreshRequest represents a refresh token request.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIs..."`
} //@name RefreshTokenRequest

// Refresh refreshes an access token using a refresh token
// @Summary Refresh access token
// @Description Exchange a refresh token for a new access token and refresh token pair
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} LoginResponse "Token refreshed"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Invalid or revoked refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		response.BadRequest(w, "refresh token is required")
		return
	}

	claims, err := h.jwtService.ValidateToken(req.RefreshToken, auth.TokenTypeRefresh)
	if err != nil {
		response.Unauthorized(w, "invalid refresh token")
		return
	}

	// Verify token is still valid in database
	valid, err := h.authService.ValidateRefreshToken(r.Context(), claims.UserID, req.RefreshToken)
	if err != nil || !valid {
		response.Unauthorized(w, "refresh token has been revoked")
		return
	}

	// Revoke old refresh token
	if err := h.authService.RevokeRefreshToken(r.Context(), req.RefreshToken); err != nil {
		// Log but continue
	}

	// Generate new token pair
	tokenPair, err := h.jwtService.GenerateTokenPair(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		response.InternalError(w, "failed to generate tokens")
		return
	}

	// Store new refresh token
	if err := h.authService.StoreRefreshToken(r.Context(), claims.UserID, tokenPair.RefreshToken); err != nil {
		response.InternalError(w, "failed to store refresh token")
		return
	}

	response.Success(w, map[string]interface{}{
		"accessToken":  tokenPair.AccessToken,
		"refreshToken": tokenPair.RefreshToken,
		"expiresAt":    tokenPair.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Logout handles user logout
// @Summary Logout
// @Description Invalidate the refresh token, ending the session
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefreshRequest false "Refresh token to revoke"
// @Success 204 "Logged out successfully"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := request.DecodeJSON(r, &req); err == nil && req.RefreshToken != "" {
		h.authService.RevokeRefreshToken(r.Context(), req.RefreshToken)
	}
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
// @Description Change the password for the currently authenticated user
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Current and new password"
// @Success 200 {object} MessageResponse "Password changed successfully"
// @Failure 400 {object} response.ErrorBody "Invalid request or current password incorrect"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "User not found"
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

	if len(req.NewPassword) < 8 {
		response.BadRequest(w, "new password must be at least 8 characters")
		return
	}

	userID, err := uuid.Parse(userCtx.UserID)
	if err != nil {
		response.InternalError(w, "invalid user ID")
		return
	}

	err = h.authService.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch err {
		case service.ErrUnauthorized:
			response.BadRequest(w, "current password is incorrect")
		case service.ErrNotFound:
			response.NotFound(w, "user not found")
		default:
			response.InternalError(w, "failed to change password")
		}
		return
	}

	response.Success(w, map[string]string{"message": "password changed successfully"})
}

// PasswordResetRequest represents a password reset request.
type PasswordResetRequest struct {
	Email string `json:"email" example:"user@example.com"`
} //@name PasswordResetRequest

// PasswordResetConfirmRequest represents a password reset confirmation.
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" example:"abc123def456"`
	NewPassword string `json:"newPassword" example:"newPassword456" minLength:"8"`
} //@name PasswordResetConfirmRequest

// RequestPasswordReset initiates the password reset flow
// @Summary Request password reset email
// @Description Sends a password reset email to the specified address if a user exists
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body PasswordResetRequest true "Email address"
// @Success 200 {object} MessageResponse "Reset email sent (if email exists)"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Router /auth/password-reset/request [post]
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" {
		response.BadRequest(w, "email is required")
		return
	}

	// Always call the service - it handles user existence check internally
	h.authService.RequestPasswordReset(r.Context(), req.Email)

	// Always return success to not reveal if user exists
	response.Success(w, map[string]string{"message": "If the email exists, a password reset link has been sent"})
}

// ConfirmPasswordReset completes the password reset using a token
// @Summary Confirm password reset with token
// @Description Complete the password reset process using the token from the email
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body PasswordResetConfirmRequest true "Token and new password"
// @Success 200 {object} MessageResponse "Password reset successful"
// @Failure 400 {object} response.ErrorBody "Invalid or expired token"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /auth/password-reset/confirm [post]
func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetConfirmRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		response.BadRequest(w, "token and new password are required")
		return
	}

	if len(req.NewPassword) < 8 {
		response.BadRequest(w, "new password must be at least 8 characters")
		return
	}

	err := h.authService.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		switch err {
		case service.ErrInvalidInput:
			response.BadRequest(w, "invalid or expired token")
		default:
			response.InternalError(w, "failed to reset password")
		}
		return
	}

	response.Success(w, map[string]string{"message": "password has been reset successfully"})
}
