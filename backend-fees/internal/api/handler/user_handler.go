package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// UserHandler handles user account management (admin only).
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// UserAccountResponse is a user account as shown in the user management.
type UserAccountResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email" example:"user@example.com"`
	FirstName *string `json:"firstName,omitempty" example:"Max" binding:"optional"`
	LastName  *string `json:"lastName,omitempty" example:"Mustermann" binding:"optional"`
	Role      string  `json:"role" example:"USER" enums:"ADMIN,USER"`
	IsActive  bool    `json:"isActive"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
} //@name UserAccount

// UserAccountRequest updates a user account.
type UserAccountRequest struct {
	Email     string  `json:"email" example:"user@example.com"`
	FirstName *string `json:"firstName,omitempty" example:"Max" binding:"optional"`
	LastName  *string `json:"lastName,omitempty" example:"Mustermann" binding:"optional"`
	Role      string  `json:"role" example:"USER" enums:"ADMIN,USER"`
	IsActive  bool    `json:"isActive"`
} //@name UserAccountRequest

// CreateUserAccountRequest creates a user account with an initial password.
type CreateUserAccountRequest struct {
	UserAccountRequest
	Password string `json:"password" minLength:"8"`
} //@name CreateUserAccountRequest

// SetUserPasswordRequest sets a new password for a user account.
type SetUserPasswordRequest struct {
	Password string `json:"password" minLength:"8"`
} //@name SetUserPasswordRequest

// List handles GET /users
// @Summary List user accounts
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {array} UserAccountResponse "Accounts"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.List(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load users")
		return
	}
	result := make([]UserAccountResponse, 0, len(users))
	for i := range users {
		result = append(result, toUserAccountResponse(&users[i]))
	}
	response.Success(w, result)
}

// Create handles POST /users
// @Summary Create user account
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserAccountRequest true "Account"
// @Success 201 {object} UserAccountResponse "Created account"
// @Failure 400 {object} response.ErrorBody "Invalid account"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Failure 409 {object} response.ErrorBody "Email already in use"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserAccountRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.userService.Create(r.Context(), req.UserAccountRequest.toInput(), req.Password)
	if err != nil {
		writeUserError(w, err)
		return
	}
	response.Created(w, toUserAccountResponse(user))
}

// Update handles PUT /users/{id}
// @Summary Update user account
// @Description Admins cannot deactivate or demote themselves. Deactivating ends the account's sessions.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Param request body UserAccountRequest true "Account"
// @Success 200 {object} UserAccountResponse "Updated account"
// @Failure 400 {object} response.ErrorBody "Invalid account"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Failure 404 {object} response.ErrorBody "User not found"
// @Failure 409 {object} response.ErrorBody "Email already in use"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /users/{id} [put]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	actorID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req UserAccountRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.userService.Update(r.Context(), actorID, id, req.toInput())
	if err != nil {
		writeUserError(w, err)
		return
	}
	response.Success(w, toUserAccountResponse(user))
}

// SetPassword handles POST /users/{id}/password
// @Summary Set user password
// @Description Sets a new password (admin reset) and ends the account's sessions.
// @Tags Users
// @Accept json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Param request body SetUserPasswordRequest true "New password"
// @Success 204 "Password set"
// @Failure 400 {object} response.ErrorBody "Password too short"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Failure 404 {object} response.ErrorBody "User not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /users/{id}/password [post]
func (h *UserHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req SetUserPasswordRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := h.userService.SetPassword(r.Context(), id, req.Password); err != nil {
		writeUserError(w, err)
		return
	}
	response.NoContent(w)
}

func (req UserAccountRequest) toInput() service.UserInput {
	return service.UserInput{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      domain.UserRole(req.Role),
		IsActive:  req.IsActive,
	}
}

func currentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(userCtx.UserID)
	if err != nil {
		response.Unauthorized(w, "invalid user ID")
		return uuid.Nil, false
	}
	return id, true
}

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, strings.TrimPrefix(err.Error(), service.ErrInvalidInput.Error()+": "))
	case errors.Is(err, service.ErrConflict):
		response.Conflict(w, strings.TrimPrefix(err.Error(), service.ErrConflict.Error()+": "))
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "Benutzer nicht gefunden")
	default:
		response.InternalError(w, "failed to save user")
	}
}

func toUserAccountResponse(u *domain.User) UserAccountResponse {
	return UserAccountResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
