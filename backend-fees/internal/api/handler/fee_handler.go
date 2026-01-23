package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// FeeHandler handles fee-related requests.
type FeeHandler struct {
	feeService    *service.FeeService
	importService *service.ImportService
}

// NewFeeHandler creates a new fee handler.
func NewFeeHandler(feeService *service.FeeService, importService *service.ImportService) *FeeHandler {
	return &FeeHandler{
		feeService:    feeService,
		importService: importService,
	}
}

// List handles GET /fees
func (h *FeeHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	filter := service.FeeFilter{
		Year:    request.GetQueryIntOptional(r, "year"),
		Month:   request.GetQueryIntOptional(r, "month"),
		FeeType: request.GetQueryString(r, "type", ""),
		Status:  request.GetQueryString(r, "status", ""),
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

// OverviewResponse represents the fee overview response.
type OverviewResponse struct {
	TotalOpen     int             `json:"totalOpen"`
	TotalPaid     int             `json:"totalPaid"`
	TotalOverdue  int             `json:"totalOverdue"`
	AmountOpen    float64         `json:"amountOpen"`
	AmountPaid    float64         `json:"amountPaid"`
	AmountOverdue float64         `json:"amountOverdue"`
	ByMonth       []MonthOverview `json:"byMonth"`
}

// MonthOverview represents fee overview for a single month.
type MonthOverview struct {
	Year       int     `json:"year"`
	Month      int     `json:"month"`
	OpenCount  int     `json:"openCount"`
	PaidCount  int     `json:"paidCount"`
	OpenAmount float64 `json:"openAmount"`
	PaidAmount float64 `json:"paidAmount"`
}

// Overview handles GET /fees/overview
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
type GenerateFeeRequest struct {
	Year  int  `json:"year"`
	Month *int `json:"month,omitempty"` // nil for yearly fees (membership)
}

// GenerateFeeResponse represents a response from generating fees.
type GenerateFeeResponse struct {
	Created     int                      `json:"created"`
	Skipped     int                      `json:"skipped"`
	Suggestions []domain.MatchSuggestion `json:"suggestions,omitempty"`
}

// Generate handles POST /fees/generate
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
func (h *FeeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid fee ID")
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
type UpdateFeeRequest struct {
	Amount *float64 `json:"amount,omitempty"`
}

// Update handles PUT /fees/{id}
func (h *FeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid fee ID")
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
func (h *FeeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid fee ID")
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
// Creates a reminder fee (Mahngebühr) for an unpaid fee.
func (h *FeeHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid fee ID")
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

// CalculateChildcareFee handles GET /childcare-fee/calculate
// Query params: childAgeType, income, siblingsCount, careHours, highestRate
func (h *FeeHandler) CalculateChildcareFee(w http.ResponseWriter, r *http.Request) {
	// Parse child age type (default: krippe)
	childAgeType := domain.ChildAgeType(request.GetQueryString(r, "childAgeType", "krippe"))
	if childAgeType != domain.ChildAgeTypeKrippe && childAgeType != domain.ChildAgeTypeKindergarten {
		childAgeType = domain.ChildAgeTypeKrippe
	}

	// Parse income
	incomeStr := request.GetQueryString(r, "income", "0")
	income, err := strconv.ParseFloat(incomeStr, 64)
	if err != nil {
		response.BadRequest(w, "invalid income value")
		return
	}

	// Parse siblings count (default: 1)
	siblingsCountStr := request.GetQueryString(r, "siblingsCount", "1")
	siblingsCount, err := strconv.Atoi(siblingsCountStr)
	if err != nil || siblingsCount < 1 {
		siblingsCount = 1
	}

	// Parse care hours (default: 30, valid: 30, 35, 40, 45, 50, 55)
	careHoursStr := request.GetQueryString(r, "careHours", "30")
	careHours, err := strconv.Atoi(careHoursStr)
	if err != nil {
		careHours = 30
	}
	// Validate care hours - must be one of 30, 35, 40, 45, 50, 55
	validHours := map[int]bool{30: true, 35: true, 40: true, 45: true, 50: true, 55: true}
	if !validHours[careHours] {
		// Round to nearest valid hour
		if careHours < 30 {
			careHours = 30
		} else if careHours > 55 {
			careHours = 55
		} else {
			careHours = ((careHours + 2) / 5) * 5
		}
	}

	// Parse highest rate flag (default: false)
	highestRateStr := request.GetQueryString(r, "highestRate", "false")
	highestRate := highestRateStr == "true" || highestRateStr == "1"

	input := domain.ChildcareFeeInput{
		ChildAgeType:  childAgeType,
		NetIncome:     income,
		SiblingsCount: siblingsCount,
		CareHours:     careHours,
		HighestRate:   highestRate,
	}

	result := h.feeService.CalculateChildcareFee(input)

	response.Success(w, result)
}

// Ensure FeeHandler references domain types
var _ domain.FeeExpectation // Just for reference
