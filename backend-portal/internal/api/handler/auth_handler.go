package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

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

type authResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresAt    string       `json:"expiresAt"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName *string           `json:"firstName,omitempty"`
	LastName  *string           `json:"lastName,omitempty"`
	Roles     []domain.UserRole `json:"roles"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	response.Success(w, mapAuthResponse(result))
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	response.Success(w, mapAuthResponse(result))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r)
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.authService.GetCurrentUser(r.Context(), claims.UserID)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	response.Success(w, mapUserResponse(user))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r)
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if err := h.authService.LogoutAll(r.Context(), claims.UserID); err != nil {
		writeAuthError(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "logged out"})
}

func mapAuthResponse(result *service.AuthResult) authResponse {
	return authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		User:         mapUserResponse(result.User),
	}
}

func mapUserResponse(user *domain.User) userResponse {
	return userResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Roles:     user.Roles,
	}
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBadRequest):
		response.Error(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrUnauthorized):
		response.Error(w, http.StatusUnauthorized, "authentication required")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
