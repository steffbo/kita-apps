package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ChildHandler handles child-related requests.
type ChildHandler struct {
	childService *service.ChildService
}

// NewChildHandler creates a new child handler.
func NewChildHandler(childService *service.ChildService) *ChildHandler {
	return &ChildHandler{childService: childService}
}

// ChildResponse represents a child in API responses.
// @Description Child information
type ChildResponse struct {
	ID              string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	HouseholdID     *string          `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MemberNumber    string           `json:"memberNumber" example:"K-2024-001"`
	FirstName       string           `json:"firstName" example:"Emma"`
	LastName        string           `json:"lastName" example:"Müller"`
	BirthDate       string           `json:"birthDate" example:"2020-06-15"`
	EntryDate       string           `json:"entryDate" example:"2023-08-01"`
	ExitDate        *string          `json:"exitDate,omitempty" example:"2026-07-31"`
	Street          *string          `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string          `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string          `json:"postalCode,omitempty" example:"14467"`
	City            *string          `json:"city,omitempty" example:"Potsdam"`
	LegalHours      *int             `json:"legalHours,omitempty" example:"35"`
	LegalHoursUntil *string          `json:"legalHoursUntil,omitempty" example:"2024-12-31"`
	CareHours       *int             `json:"careHours,omitempty" example:"40"`
	IsActive        bool             `json:"isActive" example:"true"`
	CreatedAt       string           `json:"createdAt" example:"2023-08-01T10:00:00Z"`
	UpdatedAt       string           `json:"updatedAt" example:"2023-08-01T10:00:00Z"`
	Parents         []ParentResponse `json:"parents,omitempty"`
	Household       interface{}      `json:"household,omitempty"`
}

// ChildListResponse represents a paginated list of children.
// @Description Paginated list of children
type ChildListResponse struct {
	Data       []ChildResponse `json:"data"`
	Total      int64           `json:"total" example:"100"`
	Page       int             `json:"page" example:"1"`
	PerPage    int             `json:"perPage" example:"20"`
	TotalPages int             `json:"totalPages" example:"5"`
}

// ParentResponse represents a parent in API responses (summary).
// @Description Parent information (summary)
type ParentResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	FirstName string  `json:"firstName" example:"Thomas"`
	LastName  string  `json:"lastName" example:"Müller"`
	Email     *string `json:"email,omitempty" example:"thomas.mueller@example.com"`
	Phone     *string `json:"phone,omitempty" example:"+49 331 12345"`
}

// CreateChildRequest represents a request to create a child.
// @Description Request body for creating a new child
type CreateChildRequest struct {
	MemberNumber    string  `json:"memberNumber" example:"K-2024-001"`
	FirstName       string  `json:"firstName" example:"Emma"`
	LastName        string  `json:"lastName" example:"Müller"`
	BirthDate       string  `json:"birthDate" example:"2020-06-15"`
	EntryDate       string  `json:"entryDate" example:"2023-08-01"`
	ExitDate        *string `json:"exitDate,omitempty" example:"2026-07-31"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"14467"`
	City            *string `json:"city,omitempty" example:"Potsdam"`
	LegalHours      *int    `json:"legalHours,omitempty" example:"35"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty" example:"2024-12-31"`
	CareHours       *int    `json:"careHours,omitempty" example:"40"`
}

// List returns all children with pagination and filtering
// @Summary List all children
// @Description Get a paginated list of children with optional filters
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param active query bool false "Filter by active status"
// @Param u3Only query bool false "Filter for children under 3"
// @Param hasWarnings query bool false "Filter for children with warnings"
// @Param search query string false "Search by name or member number"
// @Param sortBy query string false "Sort field (name, birthDate, entryDate)" default(name)
// @Param sortDir query string false "Sort direction (asc, desc)" default(asc)
// @Success 200 {object} ChildListResponse "Paginated list of children"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children [get]
func (h *ChildHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	activeOnly := request.GetQueryBool(r, "active")
	u3Only := request.GetQueryBool(r, "u3Only")
	hasWarnings := request.GetQueryBool(r, "hasWarnings")
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")

	filter := service.ChildFilter{
		ActiveOnly:  activeOnly != nil && *activeOnly,
		U3Only:      u3Only != nil && *u3Only,
		HasWarnings: hasWarnings != nil && *hasWarnings,
		Search:      search,
		SortBy:      sortBy,
		SortDir:     sortDir,
	}

	children, total, err := h.childService.List(r.Context(), filter, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list children")
		return
	}

	response.Paginated(w, children, total, pagination.Page, pagination.PerPage)
}

// Create creates a new child
// @Summary Create new child
// @Description Register a new child in the system
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateChildRequest true "Child data"
// @Success 201 {object} ChildResponse "Child created successfully"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 409 {object} response.ErrorBody "Member number already exists"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children [post]
func (h *ChildHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateChildRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.MemberNumber == "" || req.FirstName == "" || req.LastName == "" || req.BirthDate == "" || req.EntryDate == "" {
		response.BadRequest(w, "memberNumber, firstName, lastName, birthDate and entryDate are required")
		return
	}

	child, err := h.childService.Create(r.Context(), service.CreateChildInput{
		MemberNumber:    req.MemberNumber,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		BirthDate:       req.BirthDate,
		EntryDate:       req.EntryDate,
		ExitDate:        req.ExitDate,
		Street:          req.Street,
		StreetNo:        req.StreetNo,
		PostalCode:      req.PostalCode,
		City:            req.City,
		LegalHours:      req.LegalHours,
		LegalHoursUntil: req.LegalHoursUntil,
		CareHours:       req.CareHours,
	})
	if err != nil {
		if err == service.ErrDuplicateMemberNumber {
			response.Conflict(w, "member number already exists")
			return
		}
		response.InternalError(w, "failed to create child")
		return
	}

	response.Created(w, child)
}

// Get returns a child by ID
// @Summary Get child by ID
// @Description Retrieve detailed information about a specific child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 200 {object} ChildResponse "Child found"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id} [get]
func (h *ChildHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	child, err := h.childService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get child")
		return
	}

	response.Success(w, child)
}

// UpdateChildRequest represents a request to update a child.
// @Description Request body for updating a child
type UpdateChildRequest struct {
	FirstName       *string `json:"firstName,omitempty" example:"Emma"`
	LastName        *string `json:"lastName,omitempty" example:"Müller"`
	BirthDate       *string `json:"birthDate,omitempty" example:"2020-06-15"`
	EntryDate       *string `json:"entryDate,omitempty" example:"2023-08-01"`
	ExitDate        *string `json:"exitDate,omitempty" example:"2026-07-31"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"14467"`
	City            *string `json:"city,omitempty" example:"Potsdam"`
	LegalHours      *int    `json:"legalHours,omitempty" example:"35"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty" example:"2024-12-31"`
	CareHours       *int    `json:"careHours,omitempty" example:"40"`
	IsActive        *bool   `json:"isActive,omitempty" example:"true"`
}

// Update updates a child
// @Summary Update child
// @Description Update child information
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body UpdateChildRequest true "Updated child data"
// @Success 200 {object} ChildResponse "Child updated"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id} [put]
func (h *ChildHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	var req UpdateChildRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	child, err := h.childService.Update(r.Context(), id, service.UpdateChildInput{
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		BirthDate:       req.BirthDate,
		EntryDate:       req.EntryDate,
		ExitDate:        req.ExitDate,
		Street:          req.Street,
		StreetNo:        req.StreetNo,
		PostalCode:      req.PostalCode,
		City:            req.City,
		LegalHours:      req.LegalHours,
		LegalHoursUntil: req.LegalHoursUntil,
		CareHours:       req.CareHours,
		IsActive:        req.IsActive,
	})
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to update child")
		return
	}

	response.Success(w, child)
}

// Delete deactivates a child
// @Summary Delete child
// @Description Soft-delete (deactivate) a child record
// @Tags Children
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 204 "Child deleted"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id} [delete]
func (h *ChildHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	if err := h.childService.Deactivate(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to deactivate child")
		return
	}

	response.NoContent(w)
}

// LinkParentRequest represents a request to link a parent to a child.
// @Description Request body for linking a parent to a child
type LinkParentRequest struct {
	ParentID  string `json:"parentId" example:"550e8400-e29b-41d4-a716-446655440000"`
	IsPrimary bool   `json:"isPrimary" example:"true"`
}

// LinkParent links a parent to a child
// @Summary Link parent to child
// @Description Create a relationship between a parent and a child
// @Tags Children
// @Accept json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body LinkParentRequest true "Parent link data"
// @Success 204 "Parent linked"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child or parent not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/parents [post]
func (h *ChildHandler) LinkParent(w http.ResponseWriter, r *http.Request) {
	childID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	var req LinkParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	parentID, err := uuid.Parse(req.ParentID)
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	if err := h.childService.LinkParent(r.Context(), childID, parentID, req.IsPrimary); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child or parent not found")
			return
		}
		response.InternalError(w, "failed to link parent")
		return
	}

	response.NoContent(w)
}

// UnlinkParent removes the link between a parent and a child
// @Summary Unlink parent from child
// @Description Remove the relationship between a parent and a child
// @Tags Children
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param parentId path string true "Parent ID (UUID)"
// @Success 204 "Parent unlinked"
// @Failure 400 {object} response.ErrorBody "Invalid ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/parents/{parentId} [delete]
func (h *ChildHandler) UnlinkParent(w http.ResponseWriter, r *http.Request) {
	childID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	parentID, err := uuid.Parse(chi.URLParam(r, "parentId"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	if err := h.childService.UnlinkParent(r.Context(), childID, parentID); err != nil {
		response.InternalError(w, "failed to unlink parent")
		return
	}

	response.NoContent(w)
}

// Ensure ChildHandler implements the interface
var _ domain.Child // Just for reference
