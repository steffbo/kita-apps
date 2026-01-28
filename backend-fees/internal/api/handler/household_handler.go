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

// HouseholdResponse represents a household in API responses.
// @Description Household information with relationships
type HouseholdResponse struct {
	ID                    string      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name                  string      `json:"name" example:"Familie Müller"`
	AnnualHouseholdIncome *float64    `json:"annualHouseholdIncome,omitempty" example:"65000.00"`
	IncomeStatus          *string     `json:"incomeStatus,omitempty" example:"PROVIDED" enums:"PROVIDED,MAX_ACCEPTED,PENDING,NOT_REQUIRED,HISTORIC,FOSTER_FAMILY"`
	CreatedAt             string      `json:"createdAt" example:"2023-01-15T10:00:00Z"`
	UpdatedAt             string      `json:"updatedAt" example:"2023-01-15T10:00:00Z"`
	Parents               interface{} `json:"parents,omitempty"`
	Children              interface{} `json:"children,omitempty"`
} //@name Household

// HouseholdListResponse represents a paginated list of households.
// @Description Paginated list of households
type HouseholdListResponse struct {
	Data       []HouseholdResponse `json:"data"`
	Total      int64               `json:"total" example:"25"`
	Page       int                 `json:"page" example:"1"`
	PerPage    int                 `json:"perPage" example:"20"`
	TotalPages int                 `json:"totalPages" example:"2"`
} //@name HouseholdList

// CreateHouseholdRequest represents a request to create a household.
// @Description Request body for creating a new household
type CreateHouseholdRequest struct {
	Name                  string   `json:"name" example:"Familie Müller"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty" example:"65000.00"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty" example:"PROVIDED" enums:"PROVIDED,MAX_ACCEPTED,PENDING,NOT_REQUIRED,HISTORIC,FOSTER_FAMILY"`
} //@name CreateHouseholdRequest

// List returns all households with pagination
// @Summary List all households
// @Description Get a paginated list of households with optional search and sorting
// @Tags Households
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param search query string false "Search by name"
// @Param sortBy query string false "Sort field (name)" default(name)
// @Param sortDir query string false "Sort direction (asc, desc)" default(asc)
// @Success 200 {object} HouseholdListResponse "Paginated list of households"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households [get]
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

// Create creates a new household
// @Summary Create new household
// @Description Register a new household in the system
// @Tags Households
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateHouseholdRequest true "Household data"
// @Success 201 {object} HouseholdResponse "Household created successfully"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households [post]
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

// Get returns a household by ID
// @Summary Get household by ID
// @Description Retrieve detailed information about a specific household
// @Tags Households
// @Produce json
// @Security BearerAuth
// @Param id path string true "Household ID (UUID)"
// @Success 200 {object} HouseholdResponse "Household found"
// @Failure 400 {object} response.ErrorBody "Invalid household ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Household not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households/{id} [get]
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
// @Description Request body for updating a household
type UpdateHouseholdRequest struct {
	Name                  *string  `json:"name,omitempty" example:"Familie Müller"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty" example:"65000.00"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty" example:"PROVIDED" enums:"PROVIDED,MAX_ACCEPTED,PENDING,NOT_REQUIRED,HISTORIC,FOSTER_FAMILY"`
	ChildrenCountForFees  *int     `json:"childrenCountForFees,omitempty" example:"2"`
} //@name UpdateHouseholdRequest

// Update updates a household
// @Summary Update household
// @Description Update household information
// @Tags Households
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Household ID (UUID)"
// @Param request body UpdateHouseholdRequest true "Updated household data"
// @Success 200 {object} HouseholdResponse "Household updated"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Household not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households/{id} [put]
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
		ChildrenCountForFees:  req.ChildrenCountForFees,
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

// Delete deletes a household
// @Summary Delete household
// @Description Remove a household from the system
// @Tags Households
// @Security BearerAuth
// @Param id path string true "Household ID (UUID)"
// @Success 204 "Household deleted"
// @Failure 400 {object} response.ErrorBody "Invalid household ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Household not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households/{id} [delete]
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
// @Description Request body for linking a parent to a household
type HouseholdLinkParentRequest struct {
	ParentID string `json:"parentId" example:"550e8400-e29b-41d4-a716-446655440000"`
} //@name HouseholdLinkParentRequest

// LinkParent links a parent to a household
// @Summary Link parent to household
// @Description Associate a parent with a household
// @Tags Households
// @Accept json
// @Security BearerAuth
// @Param id path string true "Household ID (UUID)"
// @Param request body HouseholdLinkParentRequest true "Parent link data"
// @Success 204 "Parent linked"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Household or parent not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households/{id}/parents [post]
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
// @Description Request body for linking a child to a household
type HouseholdLinkChildRequest struct {
	ChildID string `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
} //@name HouseholdLinkChildRequest

// LinkChild links a child to a household
// @Summary Link child to household
// @Description Associate a child with a household
// @Tags Households
// @Accept json
// @Security BearerAuth
// @Param id path string true "Household ID (UUID)"
// @Param request body HouseholdLinkChildRequest true "Child link data"
// @Success 204 "Child linked"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Household or child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /households/{id}/children [post]
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
