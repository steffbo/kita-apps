package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ParentHandler handles parent-related requests.
type ParentHandler struct {
	parentService *service.ParentService
}

// NewParentHandler creates a new parent handler.
func NewParentHandler(parentService *service.ParentService) *ParentHandler {
	return &ParentHandler{parentService: parentService}
}

// CreateParentRequest represents a request to create a parent.
type CreateParentRequest struct {
	FirstName             string   `json:"firstName"`
	LastName              string   `json:"lastName"`
	BirthDate             *string  `json:"birthDate,omitempty"`
	Email                 *string  `json:"email,omitempty"`
	Phone                 *string  `json:"phone,omitempty"`
	Street                *string  `json:"street,omitempty"`
	StreetNo              *string  `json:"streetNo,omitempty"`
	PostalCode            *string  `json:"postalCode,omitempty"`
	City                  *string  `json:"city,omitempty"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty"`
}

// List handles GET /parents
func (h *ParentHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")

	parents, total, err := h.parentService.List(r.Context(), search, sortBy, sortDir, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list parents")
		return
	}

	response.Paginated(w, parents, total, pagination.Page, pagination.PerPage)
}

// Create handles POST /parents
func (h *ParentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		response.BadRequest(w, "firstName and lastName are required")
		return
	}

	parent, err := h.parentService.Create(r.Context(), service.CreateParentInput{
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		BirthDate:             req.BirthDate,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Street:                req.Street,
		StreetNo:              req.StreetNo,
		PostalCode:            req.PostalCode,
		City:                  req.City,
		AnnualHouseholdIncome: req.AnnualHouseholdIncome,
		IncomeStatus:          req.IncomeStatus,
	})
	if err != nil {
		response.InternalError(w, "failed to create parent")
		return
	}

	response.Created(w, parent)
}

// Get handles GET /parents/{id}
func (h *ParentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	parent, err := h.parentService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		response.InternalError(w, "failed to get parent")
		return
	}

	response.Success(w, parent)
}

// UpdateParentRequest represents a request to update a parent.
type UpdateParentRequest struct {
	FirstName             *string  `json:"firstName,omitempty"`
	LastName              *string  `json:"lastName,omitempty"`
	BirthDate             *string  `json:"birthDate,omitempty"`
	Email                 *string  `json:"email,omitempty"`
	Phone                 *string  `json:"phone,omitempty"`
	Street                *string  `json:"street,omitempty"`
	StreetNo              *string  `json:"streetNo,omitempty"`
	PostalCode            *string  `json:"postalCode,omitempty"`
	City                  *string  `json:"city,omitempty"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty"`
}

// Update handles PUT /parents/{id}
func (h *ParentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	var req UpdateParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	parent, err := h.parentService.Update(r.Context(), id, service.UpdateParentInput{
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		BirthDate:             req.BirthDate,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Street:                req.Street,
		StreetNo:              req.StreetNo,
		PostalCode:            req.PostalCode,
		City:                  req.City,
		AnnualHouseholdIncome: req.AnnualHouseholdIncome,
		IncomeStatus:          req.IncomeStatus,
	})
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		response.InternalError(w, "failed to update parent")
		return
	}

	response.Success(w, parent)
}

// Delete handles DELETE /parents/{id}
func (h *ParentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	if err := h.parentService.Delete(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		response.InternalError(w, "failed to delete parent")
		return
	}

	response.NoContent(w)
}
