package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// AccountInvitationHandler handles admin invitations and public password setup.
type AccountInvitationHandler struct {
	svc     *service.AccountInvitationService
	limiter *auth.LoginLimiter
}

func NewAccountInvitationHandler(svc *service.AccountInvitationService,
	limiter *auth.LoginLimiter) *AccountInvitationHandler {
	return &AccountInvitationHandler{svc: svc, limiter: limiter}
}

type InviteParentsRequest struct {
	ParentIDs []uuid.UUID `json:"parentIds"`
} //@name InviteParentsRequest

type InviteParentsResponse struct {
	Results []service.InvitationResult `json:"results"`
} //@name InviteParentsResponse

type InvitationPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password" minLength:"8"`
} //@name InvitationPasswordRequest

// Candidates handles GET /users/invitation-candidates.
// @Summary List parent invitation candidates
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {array} repository.InvitationCandidate "Candidates"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Router /users/invitation-candidates [get]
func (h *AccountInvitationHandler) Candidates(w http.ResponseWriter, r *http.Request) {
	candidates, err := h.svc.Candidates(r.Context())
	if err != nil {
		response.InternalError(w, "Eltern konnten nicht geladen werden")
		return
	}
	response.Success(w, candidates)
}

// Invite handles POST /users/invitations.
// @Summary Invite selected parents
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body InviteParentsRequest true "Parent IDs"
// @Success 200 {object} InviteParentsResponse "Result for each parent"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Router /users/invitations [post]
func (h *AccountInvitationHandler) Invite(w http.ResponseWriter, r *http.Request) {
	var req InviteParentsRequest
	if err := request.DecodeJSON(r, &req); err != nil || len(req.ParentIDs) == 0 {
		response.BadRequest(w, "Bitte mindestens einen Elternteil auswählen")
		return
	}
	actorID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	results, err := h.svc.Invite(r.Context(), req.ParentIDs, actorID)
	if err != nil {
		response.InternalError(w, "Einladungen fehlgeschlagen")
		return
	}
	response.Success(w, InviteParentsResponse{Results: results})
}

// Resend handles POST /users/{id}/invitation.
// @Summary Resend pending account invitation
// @Tags Users
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 204 "Invitation sent"
// @Failure 400 {object} response.ErrorBody "No pending invitation"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Admin role required"
// @Failure 404 {object} response.ErrorBody "User not found"
// @Router /users/{id}/invitation [post]
func (h *AccountInvitationHandler) Resend(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	actorID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Resend(r.Context(), id, actorID); err != nil {
		writeUserError(w, err)
		return
	}
	response.NoContent(w)
}

// SetPassword handles POST /auth/invitation-password.
// @Summary Set first password with invitation token
// @Description Invalid or expired tokens are throttled per IP and return the same response.
// @Tags Auth
// @Accept json
// @Param request body InvitationPasswordRequest true "Invitation token and password"
// @Success 204 "Password set"
// @Failure 400 {object} response.ErrorBody "Invalid password"
// @Failure 404 {object} response.ErrorBody "Invalid or expired token"
// @Failure 429 {object} response.ErrorBody "Too many attempts"
// @Router /auth/invitation-password [post]
func (h *AccountInvitationHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	const account = "invitation-password"
	if blocked, retry := h.limiter.Blocked(ip, account); blocked {
		tooManyAttempts(w, retry)
		return
	}
	var req InvitationPasswordRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	err := h.svc.SetPassword(r.Context(), req.Token, req.Password)
	switch {
	case err == nil:
		h.limiter.Succeed(ip, account)
		response.NoContent(w)
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, err.Error())
	case errors.Is(err, service.ErrNotFound), errors.Is(err, repository.ErrNotFound):
		h.limiter.Fail(ip, account)
		response.NotFound(w, "Einladung ist ungültig oder abgelaufen")
	default:
		response.InternalError(w, "Passwort konnte nicht gesetzt werden")
	}
}
