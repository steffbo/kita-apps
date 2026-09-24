package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// FeeHandler handles fee-related requests.
type FeeHandler struct {
	feeService      *service.FeeService
	importService   *service.ImportService
	reminderService *service.ReminderService
}

// FeeListResponse represents a paginated list of fees
// @Description Paginated list of fee expectations as returned by the fee endpoints
type FeeListResponse struct {
	Data       []domain.FeeExpectation `json:"data"`
	Total      int                     `json:"total" example:"100"`
	Page       int                     `json:"page" example:"1"`
	PerPage    int                     `json:"perPage" example:"20"`
	TotalPages int                     `json:"totalPages" example:"5"`
} //@name FeeList

// CreateFeeRequest represents a request to create a single fee.
// @Description Request body for creating a single fee
type CreateFeeRequest struct {
	ChildID            string   `json:"childId" example:"550e8400-e29b-41d4-a716-446655440001"`
	FeeType            string   `json:"feeType" example:"FOOD" enums:"FOOD,MEMBERSHIP,CHILDCARE,REMINDER"`
	Year               int      `json:"year" example:"2025"`
	Month              *int     `json:"month,omitempty" example:"1"`
	Amount             *float64 `json:"amount,omitempty" example:"45.40"`
	DueDate            *string  `json:"dueDate,omitempty" example:"2025-01-05"`
	ReconciliationYear *int     `json:"reconciliationYear,omitempty" example:"2024"`
} //@name CreateFeeRequest

// NewFeeHandler creates a new fee handler.
func NewFeeHandler(
	feeService *service.FeeService,
	importService *service.ImportService,
	reminderService *service.ReminderService,
) *FeeHandler {
	return &FeeHandler{
		feeService:      feeService,
		importService:   importService,
		reminderService: reminderService,
	}
}

// List handles GET /fees
// @Summary List all fees
// @Description Get a paginated list of fees with optional filtering
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param year query int false "Filter by year"
// @Param month query int false "Filter by month (1-12)"
// @Param type query string false "Filter by fee type" Enums(childcare, food, membership, reminder)
// @Param status query string false "Filter by status" Enums(open, paid, overdue)
// @Param childId query string false "Filter by child ID (UUID)"
// @Param search query string false "Search by member number or child name"
// @Param sortBy query string false "Sort by" Enums(memberNumber, childName, feeType, period, amount)
// @Param sortDir query string false "Sort direction" Enums(asc, desc)
// @Success 200 {object} FeeListResponse "Paginated list of fees"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees [get]
func (h *FeeHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	filter := service.FeeFilter{
		Year:    request.GetQueryIntOptional(r, "year"),
		Month:   request.GetQueryIntOptional(r, "month"),
		FeeType: request.GetQueryString(r, "type", ""),
		Status:  request.GetQueryString(r, "status", ""),
		Search:  request.GetQueryString(r, "search", ""),
		SortBy:  request.GetQueryString(r, "sortBy", ""),
		SortDir: request.GetQueryString(r, "sortDir", ""),
	}

	if childIDStr := request.GetQueryString(r, "childId", ""); childIDStr != "" {
		if childID, err := uuid.Parse(childIDStr); err == nil {
			filter.ChildID = &childID
		}
	}

	fees, total, err := h.feeService.List(r.Context(), filter, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list fees")
		return
	}

	response.Paginated(w, fees, total, pagination.Page, pagination.PerPage)
}

// Create handles POST /fees
// @Summary Create a single fee
// @Description Create a fee for a specific child
// @Tags Fees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateFeeRequest true "Fee creation parameters"
// @Success 201 {object} domain.FeeExpectation "Created fee expectation"
// @Failure 400 {object} response.ErrorBody "Invalid request (missing fields, invalid child ID, etc.)"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 409 {object} response.ErrorBody "Fee already exists for this period"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees [post]
func (h *FeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateFeeRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Validate required fields
	if req.ChildID == "" {
		response.BadRequest(w, "childId is required")
		return
	}
	if req.FeeType == "" {
		response.BadRequest(w, "feeType is required")
		return
	}
	if req.Year < 2000 || req.Year > 2100 {
		response.BadRequest(w, "invalid year")
		return
	}
	if req.Month != nil && (*req.Month < 1 || *req.Month > 12) {
		response.BadRequest(w, "invalid month")
		return
	}

	childID, err := uuid.Parse(req.ChildID)
	if err != nil {
		response.BadRequest(w, "invalid childId format")
		return
	}

	// Parse fee type
	feeType := domain.FeeType(req.FeeType)
	validTypes := map[domain.FeeType]bool{
		domain.FeeTypeFood:       true,
		domain.FeeTypeMembership: true,
		domain.FeeTypeChildcare:  true,
		domain.FeeTypeReminder:   true,
	}
	if !validTypes[feeType] {
		response.BadRequest(w, "invalid feeType")
		return
	}

	// Parse due date if provided
	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			response.BadRequest(w, "invalid dueDate format, use YYYY-MM-DD")
			return
		}
		dueDate = &parsed
	}

	input := service.CreateFeeInput{
		ChildID:            childID,
		FeeType:            feeType,
		Year:               req.Year,
		Month:              req.Month,
		Amount:             req.Amount,
		DueDate:            dueDate,
		ReconciliationYear: req.ReconciliationYear,
	}

	fee, err := h.feeService.Create(r.Context(), input)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		if err == service.ErrAlreadyExists {
			response.Conflict(w, "fee already exists for this child and period")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid fee type")
			return
		}
		response.InternalError(w, "failed to create fee")
		return
	}

	response.Created(w, fee)
}

// Overview handles GET /fees/overview
// @Summary Get fee overview
// @Description Get an overview of fees with totals and monthly breakdown
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param year query int false "Filter by year"
// @Success 200 {object} domain.FeeOverview "Fee overview"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/overview [get]
func (h *FeeHandler) Overview(w http.ResponseWriter, r *http.Request) {
	year := request.GetQueryIntOptional(r, "year")

	overview, err := h.feeService.GetOverview(r.Context(), year)
	if err != nil {
		response.InternalError(w, "failed to get fee overview")
		return
	}

	response.Success(w, overview)
}

// GenerateFeeRequest represents a request to generate fees.
// @Description Request body for generating fees
type GenerateFeeRequest struct {
	Year  int  `json:"year" example:"2024"`
	Month *int `json:"month,omitempty" example:"3"` // nil for yearly fees (membership)
} //@name GenerateFeeRequest

// GenerateFeeResponse represents a response from generating fees.
// @Description Result of fee generation
type GenerateFeeResponse struct {
	Created     int                      `json:"created" example:"50"`
	Skipped     int                      `json:"skipped" example:"5"`
	Suggestions []domain.MatchSuggestion `json:"suggestions,omitempty"`
} //@name GenerateFeeResponse

// Generate handles POST /fees/generate
// @Summary Generate fees for a period
// @Description Generate fee expectations for children and members for a given year/month
// @Tags Fees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body GenerateFeeRequest true "Generation parameters"
// @Success 201 {object} GenerateFeeResponse "Generation result with optional match suggestions"
// @Failure 400 {object} response.ErrorBody "Invalid request (year/month out of range)"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/generate [post]
func (h *FeeHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req GenerateFeeRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Year < 2000 || req.Year > 2100 {
		response.BadRequest(w, "invalid year")
		return
	}

	if req.Month != nil && (*req.Month < 1 || *req.Month > 12) {
		response.BadRequest(w, "invalid month")
		return
	}

	result, err := h.feeService.Generate(r.Context(), req.Year, req.Month)
	if err != nil {
		if writeNoFeeScheduleError(w, err) {
			return
		}
		response.InternalError(w, "failed to generate fees")
		return
	}

	// Auto-trigger rescan after generating fees
	resp := GenerateFeeResponse{
		Created: result.Created,
		Skipped: result.Skipped,
	}

	if result.Created > 0 && h.importService != nil {
		rescanResult, _ := h.importService.Rescan(r.Context())
		if rescanResult != nil {
			resp.Suggestions = rescanResult.Suggestions
		}
	}

	response.Created(w, resp)
}

// Get handles GET /fees/{id}
// @Summary Get a fee by ID
// @Description Get detailed information about a specific fee
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param id path string true "Fee ID (UUID)"
// @Success 200 {object} domain.FeeExpectation "Fee details"
// @Failure 400 {object} response.ErrorBody "Invalid fee ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Fee not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/{id} [get]
func (h *FeeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	fee, err := h.feeService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "fee not found")
			return
		}
		response.InternalError(w, "failed to get fee")
		return
	}

	response.Success(w, fee)
}

// UpdateFeeRequest represents a request to update a fee.
// @Description Request body for updating a fee
type UpdateFeeRequest struct {
	Amount *float64 `json:"amount,omitempty" example:"275.50"`
} //@name UpdateFeeRequest

// Update handles PUT /fees/{id}
// @Summary Update a fee
// @Description Update the amount of an existing fee
// @Tags Fees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Fee ID (UUID)"
// @Param fee body UpdateFeeRequest true "Updated fee data"
// @Success 200 {object} domain.FeeExpectation "Updated fee expectation"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Fee not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/{id} [put]
func (h *FeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req UpdateFeeRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	fee, err := h.feeService.Update(r.Context(), id, req.Amount)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "fee not found")
			return
		}
		response.InternalError(w, "failed to update fee")
		return
	}

	response.Success(w, fee)
}

// Delete handles DELETE /fees/{id}
// @Summary Delete a fee
// @Description Delete a fee by ID
// @Tags Fees
// @Security BearerAuth
// @Param id path string true "Fee ID (UUID)"
// @Success 204 "Fee deleted successfully"
// @Failure 400 {object} response.ErrorBody "Invalid fee ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Fee not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/{id} [delete]
func (h *FeeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if err := h.feeService.Delete(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "fee not found")
			return
		}
		response.InternalError(w, "failed to delete fee")
		return
	}

	response.NoContent(w)
}

// CreateReminder handles POST /fees/{id}/reminder
// @Summary Create a reminder fee
// @Description Creates a reminder fee (Mahngebühr) for an unpaid fee
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param id path string true "Parent Fee ID (UUID)"
// @Success 201 {object} domain.FeeExpectation "Created reminder fee expectation"
// @Failure 400 {object} response.ErrorBody "Invalid fee ID or fee is already paid"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Fee not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/{id}/reminder [post]
func (h *FeeHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	reminder, err := h.feeService.CreateReminder(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "fee not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "cannot create reminder for paid fee")
			return
		}
		response.InternalError(w, "failed to create reminder")
		return
	}

	response.Created(w, reminder)
}
