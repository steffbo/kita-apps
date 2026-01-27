package handler

import (
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// loginRequest represents login credentials.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"mitarbeiter@knirpsenstadt.de"`
	Password string `json:"password" validate:"required,min=8" example:"sicheres-passwort"`
} //@name LoginRequest

// refreshRequest represents refresh token request.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
} //@name RefreshTokenRequest

// passwordResetRequest represents password reset request.
type passwordResetRequest struct {
	Email string `json:"email" validate:"required,email" example:"mitarbeiter@knirpsenstadt.de"`
} //@name PasswordResetRequest

// passwordResetConfirm represents password reset confirmation.
type passwordResetConfirm struct {
	Token       string `json:"token" validate:"required" example:"abc123def456"`
	NewPassword string `json:"newPassword" validate:"required,min=8" example:"neues-sicheres-passwort"`
} //@name PasswordResetConfirm

// changePasswordRequest represents change password request.
type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required" example:"altes-passwort"`
	NewPassword     string `json:"newPassword" validate:"required,min=8" example:"neues-sicheres-passwort"`
} //@name ChangePasswordRequest

// authResponse represents authentication response.
type authResponse struct {
	AccessToken  string           `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string           `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int64            `json:"expiresIn" example:"3600"`
	User         EmployeeResponse `json:"user"`
} //@name AuthResponse

// Login handles POST /auth/login.
// @Summary User login
// @Description Authenticate with email and password to receive access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body loginRequest true "Login credentials"
// @Success 200 {object} authResponse "Authentication successful"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	result, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		User:         mapEmployeeResponse(*result.Employee, nil, nil),
	})
}

// Refresh handles POST /auth/refresh.
// @Summary Refresh access token
// @Description Use a refresh token to get a new access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param token body refreshRequest true "Refresh token"
// @Success 200 {object} authResponse "New tokens"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid or expired refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	result, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		User:         mapEmployeeResponse(*result.Employee, nil, nil),
	})
}

// Me handles GET /auth/me.
// @Summary Get current user
// @Description Get the currently authenticated user's information
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} EmployeeResponse "Current user"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	employee, err := h.authService.GetCurrentUser(r.Context(), user.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapEmployeeResponse(*employee, nil, nil))
}

// ChangePassword handles POST /auth/change-password.
// @Summary Change password
// @Description Change the current user's password
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param passwords body changePasswordRequest true "Current and new password"
// @Success 200 {object} map[string]string "Password changed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated or wrong current password"
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	var req changePasswordRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	if err := h.authService.ChangePassword(r.Context(), user.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Passwort wurde erfolgreich geändert"})
}

// RequestPasswordReset handles POST /auth/password-reset/request.
// @Summary Request password reset
// @Description Request a password reset email
// @Tags Auth
// @Accept json
// @Produce json
// @Param email body passwordResetRequest true "Email address"
// @Success 200 {object} map[string]string "Reset email sent (if email exists)"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Router /auth/password-reset/request [post]
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	h.authService.RequestPasswordReset(r.Context(), req.Email)

	response.Success(w, map[string]string{"message": "Falls die E-Mail-Adresse existiert, wurde eine Anleitung zum Zurücksetzen gesendet"})
}

// ConfirmPasswordReset handles POST /auth/password-reset/confirm.
// @Summary Confirm password reset
// @Description Set a new password using a reset token
// @Tags Auth
// @Accept json
// @Produce json
// @Param reset body passwordResetConfirm true "Reset token and new password"
// @Success 200 {object} map[string]string "Password reset successful"
// @Failure 400 {object} map[string]interface{} "Invalid request or token"
// @Router /auth/password-reset/confirm [post]
func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirm
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	if err := h.authService.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword); err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Passwort wurde erfolgreich zurückgesetzt"})
}
