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

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

type passwordResetConfirm struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type authResponse struct {
	AccessToken  string           `json:"accessToken"`
	RefreshToken string           `json:"refreshToken"`
	ExpiresIn    int64            `json:"expiresIn"`
	User         EmployeeResponse `json:"user"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.BadRequest(w, "E-Mail und Passwort sind erforderlich")
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
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.RefreshToken == "" {
		response.BadRequest(w, "Refresh Token ist erforderlich")
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
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	var req changePasswordRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		response.BadRequest(w, "Aktuelles und neues Passwort sind erforderlich")
		return
	}

	if err := h.authService.ChangePassword(r.Context(), user.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Passwort wurde erfolgreich geändert"})
}

// RequestPasswordReset handles POST /auth/password-reset/request.
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.Email == "" {
		response.BadRequest(w, "E-Mail ist erforderlich")
		return
	}

	h.authService.RequestPasswordReset(r.Context(), req.Email)

	response.Success(w, map[string]string{"message": "Falls die E-Mail-Adresse existiert, wurde eine Anleitung zum Zurücksetzen gesendet"})
}

// ConfirmPasswordReset handles POST /auth/password-reset/confirm.
func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirm
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.Token == "" || req.NewPassword == "" {
		response.BadRequest(w, "Token und neues Passwort sind erforderlich")
		return
	}

	if err := h.authService.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword); err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Passwort wurde erfolgreich zurückgesetzt"})
}
