package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// MemberHandler handles member-related requests.
type MemberHandler struct {
	memberService *service.MemberService
}

// NewMemberHandler creates a new member handler.
func NewMemberHandler(memberService *service.MemberService) *MemberHandler {
	return &MemberHandler{memberService: memberService}
}

// CreateMemberRequest represents a request to create a member.
type CreateMemberRequest struct {
	MemberNumber    *string `json:"memberNumber,omitempty"`
	FirstName       string  `json:"firstName"`
	LastName        string  `json:"lastName"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	Street          *string `json:"street,omitempty"`
	StreetNo        *string `json:"streetNo,omitempty"`
	PostalCode      *string `json:"postalCode,omitempty"`
	City            *string `json:"city,omitempty"`
	HouseholdID     *string `json:"householdId,omitempty"`
	MembershipStart string  `json:"membershipStart"`
	MembershipEnd   *string `json:"membershipEnd,omitempty"`
}

// List handles GET /members
func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")
	activeOnlyPtr := request.GetQueryBool(r, "active")
	activeOnly := activeOnlyPtr != nil && *activeOnlyPtr

	members, total, err := h.memberService.List(r.Context(), activeOnly, search, sortBy, sortDir, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list members")
		return
	}

	response.Paginated(w, members, total, pagination.Page, pagination.PerPage)
}

// Create handles POST /members
func (h *MemberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateMemberRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		response.BadRequest(w, "firstName and lastName are required")
		return
	}

	if req.MembershipStart == "" {
		response.BadRequest(w, "membershipStart is required")
		return
	}

	membershipStart, err := time.Parse("2006-01-02", req.MembershipStart)
	if err != nil {
		response.BadRequest(w, "invalid membershipStart format (use YYYY-MM-DD)")
		return
	}

	var membershipEnd *time.Time
	if req.MembershipEnd != nil {
		t, err := time.Parse("2006-01-02", *req.MembershipEnd)
		if err != nil {
			response.BadRequest(w, "invalid membershipEnd format (use YYYY-MM-DD)")
			return
		}
		membershipEnd = &t
	}

	var householdID *uuid.UUID
	if req.HouseholdID != nil {
		id, err := uuid.Parse(*req.HouseholdID)
		if err != nil {
			response.BadRequest(w, "invalid householdId")
			return
		}
		householdID = &id
	}

	member, err := h.memberService.Create(r.Context(), service.CreateMemberInput{
		MemberNumber:    req.MemberNumber,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Email:           req.Email,
		Phone:           req.Phone,
		Street:          req.Street,
		StreetNo:        req.StreetNo,
		PostalCode:      req.PostalCode,
		City:            req.City,
		HouseholdID:     householdID,
		MembershipStart: membershipStart,
		MembershipEnd:   membershipEnd,
	})
	if err != nil {
		response.InternalError(w, "failed to create member")
		return
	}

	response.Created(w, member)
}

// Get handles GET /members/{id}
func (h *MemberHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid member ID")
		return
	}

	member, err := h.memberService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "member not found")
			return
		}
		response.InternalError(w, "failed to get member")
		return
	}

	response.Success(w, member)
}

// UpdateMemberRequest represents a request to update a member.
type UpdateMemberRequest struct {
	FirstName       *string `json:"firstName,omitempty"`
	LastName        *string `json:"lastName,omitempty"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	Street          *string `json:"street,omitempty"`
	StreetNo        *string `json:"streetNo,omitempty"`
	PostalCode      *string `json:"postalCode,omitempty"`
	City            *string `json:"city,omitempty"`
	HouseholdID     *string `json:"householdId,omitempty"`
	MembershipStart *string `json:"membershipStart,omitempty"`
	MembershipEnd   *string `json:"membershipEnd,omitempty"`
	IsActive        *bool   `json:"isActive,omitempty"`
}

// Update handles PUT /members/{id}
func (h *MemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid member ID")
		return
	}

	var req UpdateMemberRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	input := service.UpdateMemberInput{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		Phone:      req.Phone,
		Street:     req.Street,
		StreetNo:   req.StreetNo,
		PostalCode: req.PostalCode,
		City:       req.City,
		IsActive:   req.IsActive,
	}

	if req.HouseholdID != nil {
		hid, err := uuid.Parse(*req.HouseholdID)
		if err != nil {
			response.BadRequest(w, "invalid householdId")
			return
		}
		input.HouseholdID = &hid
	}

	if req.MembershipStart != nil {
		t, err := time.Parse("2006-01-02", *req.MembershipStart)
		if err != nil {
			response.BadRequest(w, "invalid membershipStart format (use YYYY-MM-DD)")
			return
		}
		input.MembershipStart = &t
	}

	if req.MembershipEnd != nil {
		t, err := time.Parse("2006-01-02", *req.MembershipEnd)
		if err != nil {
			response.BadRequest(w, "invalid membershipEnd format (use YYYY-MM-DD)")
			return
		}
		input.MembershipEnd = &t
	}

	member, err := h.memberService.Update(r.Context(), id, input)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "member not found")
			return
		}
		response.InternalError(w, "failed to update member")
		return
	}

	response.Success(w, member)
}

// Delete handles DELETE /members/{id}
func (h *MemberHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid member ID")
		return
	}

	if err := h.memberService.Delete(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "member not found")
			return
		}
		response.InternalError(w, "failed to delete member")
		return
	}

	response.NoContent(w)
}
