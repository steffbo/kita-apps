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

// MemberResponse represents a member in API responses
// @Description Member information
type MemberResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MemberNumber    *string `json:"memberNumber,omitempty" example:"M-2024-001"`
	FirstName       string  `json:"firstName" example:"Hans"`
	LastName        string  `json:"lastName" example:"Müller"`
	Email           *string `json:"email,omitempty" example:"hans.mueller@example.com"`
	Phone           *string `json:"phone,omitempty" example:"+49 123 456789"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"12345"`
	City            *string `json:"city,omitempty" example:"Berlin"`
	HouseholdID     *string `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MembershipStart string  `json:"membershipStart" example:"2024-01-01"`
	MembershipEnd   *string `json:"membershipEnd,omitempty" example:"2024-12-31"`
	IsActive        bool    `json:"isActive" example:"true"`
} //@name Member

// MemberListResponse represents a paginated list of members
// @Description Paginated list of members
type MemberListResponse struct {
	Data       []MemberResponse `json:"data"`
	Total      int              `json:"total" example:"42"`
	Page       int              `json:"page" example:"1"`
	PerPage    int              `json:"perPage" example:"20"`
	TotalPages int              `json:"totalPages" example:"3"`
} //@name MemberList

// NewMemberHandler creates a new member handler.
func NewMemberHandler(memberService *service.MemberService) *MemberHandler {
	return &MemberHandler{memberService: memberService}
}

// CreateMemberRequest represents a request to create a member.
// @Description Request body for creating a new member
type CreateMemberRequest struct {
	MemberNumber    *string `json:"memberNumber,omitempty" example:"M-2024-001"`
	FirstName       string  `json:"firstName" example:"Hans"`
	LastName        string  `json:"lastName" example:"Müller"`
	Email           *string `json:"email,omitempty" example:"hans.mueller@example.com"`
	Phone           *string `json:"phone,omitempty" example:"+49 123 456789"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"12345"`
	City            *string `json:"city,omitempty" example:"Berlin"`
	HouseholdID     *string `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MembershipStart string  `json:"membershipStart" example:"2024-01-01"`
	MembershipEnd   *string `json:"membershipEnd,omitempty" example:"2024-12-31"`
} //@name CreateMemberRequest

// List handles GET /members
// @Summary List all members
// @Description Get a paginated list of members with optional filtering and sorting
// @Tags Members
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param search query string false "Search by name or member number"
// @Param sortBy query string false "Sort by field" default(name) Enums(name, memberNumber, membershipStart)
// @Param sortDir query string false "Sort direction" default(asc) Enums(asc, desc)
// @Param active query bool false "Filter by active status"
// @Success 200 {object} MemberListResponse "Paginated list of members"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /members [get]
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
// @Summary Create a new member
// @Description Create a new membership record
// @Tags Members
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param member body CreateMemberRequest true "Member data"
// @Success 201 {object} MemberResponse "Created member"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /members [post]
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
// @Summary Get a member by ID
// @Description Get detailed information about a specific member
// @Tags Members
// @Produce json
// @Security BearerAuth
// @Param id path string true "Member ID (UUID)"
// @Success 200 {object} MemberResponse "Member details"
// @Failure 400 {object} response.ErrorBody "Invalid member ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Member not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /members/{id} [get]
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
// @Description Request body for updating a member
type UpdateMemberRequest struct {
	FirstName       *string `json:"firstName,omitempty" example:"Hans"`
	LastName        *string `json:"lastName,omitempty" example:"Müller"`
	Email           *string `json:"email,omitempty" example:"hans.mueller@example.com"`
	Phone           *string `json:"phone,omitempty" example:"+49 123 456789"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"12345"`
	City            *string `json:"city,omitempty" example:"Berlin"`
	HouseholdID     *string `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MembershipStart *string `json:"membershipStart,omitempty" example:"2024-01-01"`
	MembershipEnd   *string `json:"membershipEnd,omitempty" example:"2024-12-31"`
	IsActive        *bool   `json:"isActive,omitempty" example:"true"`
} //@name UpdateMemberRequest

// Update handles PUT /members/{id}
// @Summary Update a member
// @Description Update an existing member's information
// @Tags Members
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Member ID (UUID)"
// @Param member body UpdateMemberRequest true "Updated member data"
// @Success 200 {object} MemberResponse "Updated member"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Member not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /members/{id} [put]
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
// @Summary Delete a member
// @Description Delete a member by ID
// @Tags Members
// @Security BearerAuth
// @Param id path string true "Member ID (UUID)"
// @Success 204 "Member deleted successfully"
// @Failure 400 {object} response.ErrorBody "Invalid member ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Member not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /members/{id} [delete]
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
