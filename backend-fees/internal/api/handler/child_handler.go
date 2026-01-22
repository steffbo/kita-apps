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

// CreateChildRequest represents a request to create a child.
type CreateChildRequest struct {
	MemberNumber    string  `json:"memberNumber"`
	FirstName       string  `json:"firstName"`
	LastName        string  `json:"lastName"`
	BirthDate       string  `json:"birthDate"`
	EntryDate       string  `json:"entryDate"`
	ExitDate        *string `json:"exitDate,omitempty"`
	Street          *string `json:"street,omitempty"`
	StreetNo        *string `json:"streetNo,omitempty"`
	PostalCode      *string `json:"postalCode,omitempty"`
	City            *string `json:"city,omitempty"`
	LegalHours      *int    `json:"legalHours,omitempty"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty"`
	CareHours       *int    `json:"careHours,omitempty"`
}

// List handles GET /children
func (h *ChildHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	activeOnly := request.GetQueryBool(r, "active")
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")

	filter := service.ChildFilter{
		ActiveOnly: activeOnly != nil && *activeOnly,
		Search:     search,
		SortBy:     sortBy,
		SortDir:    sortDir,
	}

	children, total, err := h.childService.List(r.Context(), filter, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list children")
		return
	}

	response.Paginated(w, children, total, pagination.Page, pagination.PerPage)
}

// Create handles POST /children
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

// Get handles GET /children/{id}
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
type UpdateChildRequest struct {
	FirstName       *string `json:"firstName,omitempty"`
	LastName        *string `json:"lastName,omitempty"`
	BirthDate       *string `json:"birthDate,omitempty"`
	EntryDate       *string `json:"entryDate,omitempty"`
	ExitDate        *string `json:"exitDate,omitempty"`
	Street          *string `json:"street,omitempty"`
	StreetNo        *string `json:"streetNo,omitempty"`
	PostalCode      *string `json:"postalCode,omitempty"`
	City            *string `json:"city,omitempty"`
	LegalHours      *int    `json:"legalHours,omitempty"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty"`
	CareHours       *int    `json:"careHours,omitempty"`
	IsActive        *bool   `json:"isActive,omitempty"`
}

// Update handles PUT /children/{id}
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

// Delete handles DELETE /children/{id}
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
type LinkParentRequest struct {
	ParentID  string `json:"parentId"`
	IsPrimary bool   `json:"isPrimary"`
}

// LinkParent handles POST /children/{id}/parents
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

// UnlinkParent handles DELETE /children/{id}/parents/{parentId}
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
