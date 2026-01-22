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

// HouseholdHandler handles household-related requests.
type HouseholdHandler struct {
	householdService *service.HouseholdService
}

// NewHouseholdHandler creates a new household handler.
func NewHouseholdHandler(householdService *service.HouseholdService) *HouseholdHandler {
	return &HouseholdHandler{householdService: householdService}
}

// CreateHouseholdRequest represents a request to create a household.
type CreateHouseholdRequest struct {
	Name                  string   `json:"name"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty"`
}

// List handles GET /households
func (h *HouseholdHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")

	households, total, err := h.householdService.List(r.Context(), search, sortBy, sortDir, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list households")
		return
	}

	response.Paginated(w, households, total, pagination.Page, pagination.PerPage)
}

// Create handles POST /households
func (h *HouseholdHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateHouseholdRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}

	incomeStatus := domain.IncomeStatusUnknown
	if req.IncomeStatus != nil {
		incomeStatus = domain.IncomeStatus(*req.IncomeStatus)
	}

	household, err := h.householdService.Create(r.Context(), service.CreateHouseholdInput{
		Name:                  req.Name,
		AnnualHouseholdIncome: req.AnnualHouseholdIncome,
		IncomeStatus:          incomeStatus,
	})
	if err != nil {
		response.InternalError(w, "failed to create household")
		return
	}

	response.Created(w, household)
}

// Get handles GET /households/{id}
func (h *HouseholdHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid household ID")
		return
	}

	household, err := h.householdService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "household not found")
			return
		}
		response.InternalError(w, "failed to get household")
		return
	}

	response.Success(w, household)
}

// UpdateHouseholdRequest represents a request to update a household.
type UpdateHouseholdRequest struct {
	Name                  *string  `json:"name,omitempty"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty"`
}

// Update handles PUT /households/{id}
func (h *HouseholdHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid household ID")
		return
	}

	var req UpdateHouseholdRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	input := service.UpdateHouseholdInput{
		Name:                  req.Name,
		AnnualHouseholdIncome: req.AnnualHouseholdIncome,
	}
	if req.IncomeStatus != nil {
		status := domain.IncomeStatus(*req.IncomeStatus)
		input.IncomeStatus = &status
	}

	household, err := h.householdService.Update(r.Context(), id, input)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "household not found")
			return
		}
		response.InternalError(w, "failed to update household")
		return
	}

	response.Success(w, household)
}

// Delete handles DELETE /households/{id}
func (h *HouseholdHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid household ID")
		return
	}

	if err := h.householdService.Delete(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "household not found")
			return
		}
		response.InternalError(w, "failed to delete household")
		return
	}

	response.NoContent(w)
}

// HouseholdLinkParentRequest represents a request to link a parent to a household.
type HouseholdLinkParentRequest struct {
	ParentID string `json:"parentId"`
}

// LinkParent handles POST /households/{id}/parents
func (h *HouseholdHandler) LinkParent(w http.ResponseWriter, r *http.Request) {
	householdID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid household ID")
		return
	}

	var req HouseholdLinkParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	parentID, err := uuid.Parse(req.ParentID)
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	if err := h.householdService.LinkParent(r.Context(), householdID, parentID); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "household or parent not found")
			return
		}
		response.InternalError(w, "failed to link parent")
		return
	}

	response.NoContent(w)
}

// HouseholdLinkChildRequest represents a request to link a child to a household.
type HouseholdLinkChildRequest struct {
	ChildID string `json:"childId"`
}

// LinkChild handles POST /households/{id}/children
func (h *HouseholdHandler) LinkChild(w http.ResponseWriter, r *http.Request) {
	householdID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid household ID")
		return
	}

	var req HouseholdLinkChildRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	childID, err := uuid.Parse(req.ChildID)
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	if err := h.householdService.LinkChild(r.Context(), householdID, childID); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "household or child not found")
			return
		}
		response.InternalError(w, "failed to link child")
		return
	}

	response.NoContent(w)
}
