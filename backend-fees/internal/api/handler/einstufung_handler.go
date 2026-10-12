package handler

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// EinstufungHandler handles Einstufung-related requests.
type EinstufungHandler struct {
	einstufungService *service.EinstufungService
	letterService     *service.EinstufungLetterService
}

// NewEinstufungHandler creates a new Einstufung handler.
func NewEinstufungHandler(einstufungService *service.EinstufungService,
	letterService *service.EinstufungLetterService) *EinstufungHandler {
	return &EinstufungHandler{einstufungService: einstufungService, letterService: letterService}
}

// EinstufungResponse represents an Einstufung in API responses.
// @Description Fee classification for a child
type EinstufungResponse struct {
	ID                   string                            `json:"id"`
	ChildID              string                            `json:"childId"`
	HouseholdID          string                            `json:"householdId"`
	Year                 int                               `json:"year"`
	ValidFrom            string                            `json:"validFrom"`
	ValidUntil           *string                           `json:"validUntil,omitempty" binding:"optional"`
	SourceEinstufungID   *string                           `json:"sourceEinstufungId,omitempty" binding:"optional"`
	ChangeDate           *string                           `json:"changeDate,omitempty" binding:"optional"`
	EffectiveFromMonth   string                            `json:"effectiveFromMonth"`
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	AnnualNetIncome      *float64                          `json:"annualNetIncome,omitempty" binding:"optional"` // Only set for incomeBasis INCOME.
	IncomeBasis          string                            `json:"incomeBasis" enums:"INCOME,HIGHEST_RATE,NOT_REQUIRED"`
	HighestRateVoluntary bool                              `json:"highestRateVoluntary"` // true exactly for incomeBasis HIGHEST_RATE
	Provisional          bool                              `json:"provisional"`          // Vorläufig: income proofs are still missing
	CareHoursPerWeek     int                               `json:"careHoursPerWeek"`
	CareType             string                            `json:"careType"`
	ChildrenCount        int                               `json:"childrenCount"`
	MonthlyChildcareFee  float64                           `json:"monthlyChildcareFee"`
	MonthlyFoodFee       float64                           `json:"monthlyFoodFee"`
	AnnualMembershipFee  float64                           `json:"annualMembershipFee"`
	FeeRule              string                            `json:"feeRule"`
	DiscountPercent      int                               `json:"discountPercent"`
	DiscountFactor       float64                           `json:"discountFactor"`
	BaseFee              float64                           `json:"baseFee"`
	Notes                string                            `json:"notes,omitempty" binding:"optional"`
	MonthlyTable         []domain.EinstufungMonthRow       `json:"monthlyTable,omitempty" binding:"optional"`
	CreatedAt            string                            `json:"createdAt"`
	UpdatedAt            string                            `json:"updatedAt"`
	LetterSentAt         *string                           `json:"letterSentAt,omitempty" binding:"optional"`       // Last time the letter was emailed from the app
	ExpectationChanges   *ChildcareExpectationSyncResponse `json:"expectationChanges,omitempty" binding:"optional"` // CHILDCARE expectations aligned by this save (create/update only)
	Child                interface{}                       `json:"child,omitempty" binding:"optional"`
	Household            interface{}                       `json:"household,omitempty" binding:"optional"`
} //@name Einstufung

// EinstufungListResponse represents a paginated list of Einstufungen.
// @Description Paginated list of fee classifications
type EinstufungListResponse struct {
	Data       []EinstufungResponse `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"perPage"`
	TotalPages int                  `json:"totalPages"`
} //@name EinstufungList

// CreateEinstufungRequest represents a request to create an Einstufung.
// @Description Request body for creating a fee classification
type CreateEinstufungRequest struct {
	ChildID              string                            `json:"childId"`
	Year                 int                               `json:"year"`
	ValidFrom            string                            `json:"validFrom,omitempty" binding:"optional"` // Deprecated: Ignored; validity is derived from year and child entry date. Only used as year fallback.
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	IncomeBasis          string                            `json:"incomeBasis,omitempty" enums:"INCOME,HIGHEST_RATE,NOT_REQUIRED" binding:"optional"` // Defaults to HIGHEST_RATE or INCOME from highestRateVoluntary.
	HighestRateVoluntary bool                              `json:"highestRateVoluntary,omitempty" binding:"optional"`                                 // Deprecated: Use incomeBasis HIGHEST_RATE.
	Provisional          bool                              `json:"provisional,omitempty" binding:"optional"`
	CareHoursPerWeek     int                               `json:"careHoursPerWeek,omitempty" binding:"optional"` // Deprecated: Ignored; resolved from the child's care-hours history.
	ChildrenCount        int                               `json:"childrenCount"`
	Notes                string                            `json:"notes"`
} //@name CreateEinstufungRequest

// UpdateEinstufungRequest represents a request to update an Einstufung.
// @Description Request body for updating a fee classification
type UpdateEinstufungRequest struct {
	IncomeCalculation    *domain.HouseholdIncomeCalculation `json:"incomeCalculation,omitempty" binding:"optional"`
	IncomeBasis          *string                            `json:"incomeBasis,omitempty" enums:"INCOME,HIGHEST_RATE,NOT_REQUIRED" binding:"optional"`
	HighestRateVoluntary *bool                              `json:"highestRateVoluntary,omitempty" binding:"optional"` // Deprecated: Use incomeBasis HIGHEST_RATE.
	Provisional          *bool                              `json:"provisional,omitempty" binding:"optional"`
	CareHoursPerWeek     *int                               `json:"careHoursPerWeek,omitempty" binding:"optional"` // Deprecated: Ignored; resolved from the child's care-hours history.
	ChildrenCount        *int                               `json:"childrenCount,omitempty" binding:"optional"`
	ValidFrom            *string                            `json:"validFrom,omitempty" binding:"optional"` // Only used for follow-up income classifications; ignored for initial classifications.
	Notes                *string                            `json:"notes,omitempty" binding:"optional"`
} //@name UpdateEinstufungRequest

// CreateFollowUpEinstufungRequest represents a request to create a follow-up Einstufung.
// @Description Request body for creating a follow-up fee classification
type CreateFollowUpEinstufungRequest struct {
	ChangeDate           string                            `json:"changeDate"` // ISO date, cut-off: 1st-14th same month, 15th+ next month
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	IncomeBasis          string                            `json:"incomeBasis,omitempty" enums:"INCOME,HIGHEST_RATE,NOT_REQUIRED" binding:"optional"` // Defaults to HIGHEST_RATE or INCOME from highestRateVoluntary.
	HighestRateVoluntary bool                              `json:"highestRateVoluntary,omitempty" binding:"optional"`                                 // Deprecated: Use incomeBasis HIGHEST_RATE.
	Provisional          bool                              `json:"provisional,omitempty" binding:"optional"`
	CareHoursPerWeek     int                               `json:"careHoursPerWeek,omitempty" binding:"optional"` // Deprecated: Ignored; resolved from the child's care-hours history at the effective month.
	ChildrenCount        int                               `json:"childrenCount"`
	Notes                string                            `json:"notes"`
} //@name CreateFollowUpEinstufungRequest

// CreditReviewPeriodResponse identifies a month needing manual credit review.
// @Description Paid month that would become overpaid after a childcare fee decrease
type CreditReviewPeriodResponse struct {
	FeeID         string  `json:"feeId"`
	Year          int     `json:"year"`
	Month         int     `json:"month"`
	OldAmount     float64 `json:"oldAmount"`
	NewAmount     float64 `json:"newAmount"`
	MatchedAmount float64 `json:"matchedAmount"`
	CreditAmount  float64 `json:"creditAmount"`
} //@name CreditReviewPeriod

// ChildcareExpectationSyncResponse describes automatic expectation updates.
// @Description Result of aligning generated CHILDCARE expectations with an Einstufung
type ChildcareExpectationSyncResponse struct {
	Created              int                          `json:"created"`
	Updated              int                          `json:"updated"`
	Removed              int                          `json:"removed"`
	Skipped              int                          `json:"skipped"`
	DeltaOpen            float64                      `json:"deltaOpen"`
	CreditReviewRequired []CreditReviewPeriodResponse `json:"creditReviewRequired"`
} //@name ChildcareExpectationSyncResult

// CreateFollowUpEinstufungResponse returns the created follow-up and contribution changes.
// @Description Follow-up Einstufung creation result
type CreateFollowUpEinstufungResponse struct {
	Einstufung         EinstufungResponse               `json:"einstufung"`
	ExpectationChanges ChildcareExpectationSyncResponse `json:"expectationChanges"`
} //@name CreateFollowUpEinstufungResponse

// CalculateIncomeRequest is a lightweight endpoint for calculating fee-relevant income
// without creating a full Einstufung record.
// @Description Calculate household income from parent details
type CalculateIncomeRequest struct {
	Parent1 domain.IncomeDetails `json:"parent1"`
	Parent2 domain.IncomeDetails `json:"parent2"`
} //@name CalculateIncomeRequest

// CalculateIncomeResponse returns the computed income breakdown.
// @Description Computed income breakdown
type CalculateIncomeResponse struct {
	Parent1NetIncome         float64 `json:"parent1NetIncome"`
	Parent2NetIncome         float64 `json:"parent2NetIncome"`
	Parent1FeeRelevantIncome float64 `json:"parent1FeeRelevantIncome"`
	Parent2FeeRelevantIncome float64 `json:"parent2FeeRelevantIncome"`
	Parent1AdvertisingCosts  float64 `json:"parent1AdvertisingCosts"` // Applied WK-Pauschale, capped at the gross wages
	Parent2AdvertisingCosts  float64 `json:"parent2AdvertisingCosts"` // Applied WK-Pauschale, capped at the gross wages
	HouseholdFeeIncome       float64 `json:"householdFeeIncome"`
	HouseholdFullIncome      float64 `json:"householdFullIncome"`
} //@name CalculateIncomeResponse

// PreviewEinstufungRequest is an unsaved Einstufung form. einstufungId previews
// an edit, sourceEinstufungId a follow-up, otherwise childId and year a new one.
// @Description Unsaved Einstufung form for the live preview
type PreviewEinstufungRequest struct {
	EinstufungID       string                            `json:"einstufungId,omitempty" binding:"optional"`
	SourceEinstufungID string                            `json:"sourceEinstufungId,omitempty" binding:"optional"`
	ChildID            string                            `json:"childId,omitempty" binding:"optional"`
	Year               int                               `json:"year,omitempty" binding:"optional"`
	ChangeDate         string                            `json:"changeDate,omitempty" binding:"optional"` // Follow-ups and edits of follow-ups, YYYY-MM-DD
	IncomeCalculation  domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	IncomeBasis        string                            `json:"incomeBasis" enums:"INCOME,HIGHEST_RATE,NOT_REQUIRED"`
	ChildrenCount      int                               `json:"childrenCount"`
	Provisional        bool                              `json:"provisional,omitempty" binding:"optional"`
} //@name PreviewEinstufungRequest

// EinstufungPreviewResponse is the computed, unsaved result of a form.
// @Description Live preview of an Einstufung; nothing is saved
type EinstufungPreviewResponse struct {
	Einstufung EinstufungResponse      `json:"einstufung"`
	Income     CalculateIncomeResponse `json:"income"`
	Warnings   []string                `json:"warnings"` // Need a confirmation before saving
	Notices    []string                `json:"notices"`
} //@name EinstufungPreview

// Create handles POST /einstufungen
// @Summary Create a fee classification
// @Description Creates a new Einstufung by calculating fees from income proofs and aligns already generated CHILDCARE expectations of its months
// @Tags Einstufungen
// @Accept json
// @Produce json
// @Param body body CreateEinstufungRequest true "Einstufung data"
// @Success 201 {object} EinstufungResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen [post]
func (h *EinstufungHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateEinstufungRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	childID, err := uuid.Parse(req.ChildID)
	if err != nil {
		response.BadRequest(w, "invalid childId")
		return
	}

	if req.Year == 0 && req.ValidFrom != "" {
		validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
		if err != nil {
			response.BadRequest(w, "invalid validFrom date (expected YYYY-MM-DD)")
			return
		}
		req.Year = validFrom.Year()
	}

	result, changes, err := h.einstufungService.Create(r.Context(), service.CreateEinstufungInput{
		ChildID:              childID,
		Year:                 req.Year,
		IncomeCalculation:    req.IncomeCalculation,
		IncomeBasis:          domain.IncomeBasis(req.IncomeBasis),
		HighestRateVoluntary: req.HighestRateVoluntary,
		ChildrenCount:        req.ChildrenCount,
		Provisional:          req.Provisional,
		Notes:                req.Notes,
	})
	if err != nil {
		handleEinstufungError(w, err)
		return
	}

	resp := toEinstufungResponse(result)
	resp.ExpectationChanges = syncResponse(changes)
	response.Created(w, resp)
}

// Get handles GET /einstufungen/{id}
// @Summary Get a fee classification
// @Description Returns a single Einstufung by ID
// @Tags Einstufungen
// @Produce json
// @Param id path string true "Einstufung ID"
// @Success 200 {object} EinstufungResponse
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id} [get]
func (h *EinstufungHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}

	result, err := h.einstufungService.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Einstufung not found")
		return
	}

	response.Success(w, toEinstufungResponse(result))
}

// Update handles PUT /einstufungen/{id}
// @Summary Update a fee classification
// @Description Updates an existing Einstufung, recalculates fees and aligns already generated CHILDCARE expectations of its months
// @Tags Einstufungen
// @Accept json
// @Produce json
// @Param id path string true "Einstufung ID"
// @Param body body UpdateEinstufungRequest true "Update data"
// @Success 200 {object} EinstufungResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id} [put]
func (h *EinstufungHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}

	var req UpdateEinstufungRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	input := service.UpdateEinstufungInput{
		IncomeCalculation:    req.IncomeCalculation,
		HighestRateVoluntary: req.HighestRateVoluntary,
		ChildrenCount:        req.ChildrenCount,
		Provisional:          req.Provisional,
		Notes:                req.Notes,
	}
	if req.IncomeBasis != nil {
		basis := domain.IncomeBasis(*req.IncomeBasis)
		input.IncomeBasis = &basis
	}

	if req.ValidFrom != nil {
		t, err := time.Parse("2006-01-02", *req.ValidFrom)
		if err != nil {
			response.BadRequest(w, "invalid validFrom date")
			return
		}
		input.ValidFrom = &t
	}

	result, changes, err := h.einstufungService.Update(r.Context(), id, input)
	if err != nil {
		handleEinstufungError(w, err)
		return
	}

	resp := toEinstufungResponse(result)
	resp.ExpectationChanges = syncResponse(changes)
	response.Success(w, resp)
}

// CreateFollowUp handles POST /einstufungen/{id}/follow-ups
// @Summary Create a follow-up fee classification
// @Description Creates a follow-up Einstufung, applies the monthly cut-off rule, closes the source period, and aligns already generated CHILDCARE expectations from the effective month.
// @Tags Einstufungen
// @Accept json
// @Produce json
// @Param id path string true "Source Einstufung ID"
// @Param body body CreateFollowUpEinstufungRequest true "Follow-up data"
// @Success 201 {object} CreateFollowUpEinstufungResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id}/follow-ups [post]
func (h *EinstufungHandler) CreateFollowUp(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}

	var req CreateFollowUpEinstufungRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	changeDate, err := time.Parse("2006-01-02", req.ChangeDate)
	if err != nil {
		response.BadRequest(w, "invalid changeDate date (expected YYYY-MM-DD)")
		return
	}

	result, err := h.einstufungService.CreateFollowUp(r.Context(), sourceID, service.CreateFollowUpEinstufungInput{
		ChangeDate:           changeDate,
		IncomeCalculation:    req.IncomeCalculation,
		IncomeBasis:          domain.IncomeBasis(req.IncomeBasis),
		HighestRateVoluntary: req.HighestRateVoluntary,
		ChildrenCount:        req.ChildrenCount,
		Provisional:          req.Provisional,
		Notes:                req.Notes,
	})
	if err != nil {
		handleEinstufungError(w, err)
		return
	}

	response.Created(w, CreateFollowUpEinstufungResponse{
		Einstufung:         toEinstufungResponse(result.Einstufung),
		ExpectationChanges: toChildcareExpectationSyncResponse(result.ExpectationChanges),
	})
}

// Delete handles DELETE /einstufungen/{id}
// @Summary Delete a fee classification
// @Description Deletes an Einstufung
// @Tags Einstufungen
// @Param id path string true "Einstufung ID"
// @Success 204
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id} [delete]
func (h *EinstufungHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}

	if err := h.einstufungService.Delete(r.Context(), id); err != nil {
		handleEinstufungError(w, err)
		return
	}

	response.NoContent(w)
}

// ListByYear handles GET /einstufungen?year=2026
// @Summary List fee classifications
// @Description Returns Einstufungen for a year with pagination
// @Tags Einstufungen
// @Produce json
// @Param year query int true "Year"
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} EinstufungListResponse
// @Security BearerAuth
// @Router /einstufungen [get]
func (h *EinstufungHandler) List(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = strconv.Itoa(util.Now().Year())
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		response.BadRequest(w, "invalid year")
		return
	}

	pag := request.GetPagination(r)
	results, total, err := h.einstufungService.ListByYear(r.Context(), year, pag.Offset, pag.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list Einstufungen")
		return
	}

	data := make([]EinstufungResponse, len(results))
	for i, e := range results {
		data[i] = toEinstufungResponse(&e)
	}

	response.Paginated(w, data, total, pag.Page, pag.PerPage)
}

// GetForChild handles GET /einstufungen/child/{childId}?year=2026
// @Summary Get Einstufung for a child
// @Description Returns the Einstufung for a child in a given year (or latest)
// @Tags Einstufungen
// @Produce json
// @Param childId path string true "Child ID"
// @Param year query int false "Year (defaults to latest)"
// @Success 200 {object} EinstufungResponse
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/child/{childId} [get]
func (h *EinstufungHandler) GetForChild(w http.ResponseWriter, r *http.Request) {
	childID, err := uuid.Parse(chi.URLParam(r, "childId"))
	if err != nil {
		response.BadRequest(w, "invalid childId")
		return
	}

	yearStr := r.URL.Query().Get("year")
	var result *domain.Einstufung

	if yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			response.BadRequest(w, "invalid year")
			return
		}
		result, err = h.einstufungService.GetByChildAndYear(r.Context(), childID, year)
		if err != nil {
			response.NotFound(w, "keine Einstufung für dieses Kind und Jahr gefunden")
			return
		}
	} else {
		result, err = h.einstufungService.GetLatestForChild(r.Context(), childID)
		if err != nil {
			response.NotFound(w, "keine Einstufung für dieses Kind gefunden")
			return
		}
	}

	response.Success(w, toEinstufungResponse(result))
}

// ListForHousehold handles GET /einstufungen/household/{householdId}
// @Summary List Einstufungen for a household
// @Description Returns all Einstufungen for a household
// @Tags Einstufungen
// @Produce json
// @Param householdId path string true "Household ID"
// @Success 200 {array} EinstufungResponse
// @Security BearerAuth
// @Router /einstufungen/household/{householdId} [get]
func (h *EinstufungHandler) ListForHousehold(w http.ResponseWriter, r *http.Request) {
	householdID, err := uuid.Parse(chi.URLParam(r, "householdId"))
	if err != nil {
		response.BadRequest(w, "invalid householdId")
		return
	}

	results, err := h.einstufungService.ListByHousehold(r.Context(), householdID)
	if err != nil {
		response.InternalError(w, "failed to list Einstufungen")
		return
	}

	data := make([]EinstufungResponse, len(results))
	for i, e := range results {
		data[i] = toEinstufungResponse(&e)
	}

	response.Success(w, data)
}

// CalculateIncome handles POST /einstufungen/calculate-income
// @Summary Calculate household income
// @Description Calculates fee-relevant household income from parent income details without creating a record.
// @Tags Einstufungen
// @Accept json
// @Produce json
// @Param body body CalculateIncomeRequest true "Parent income details"
// @Success 200 {object} CalculateIncomeResponse
// @Router /einstufungen/calculate-income [post]
func (h *EinstufungHandler) CalculateIncome(w http.ResponseWriter, r *http.Request) {
	var req CalculateIncomeRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	response.Success(w, toCalculateIncomeResponse(service.CalculateIncomeBreakdown(domain.HouseholdIncomeCalculation{
		Parent1: req.Parent1,
		Parent2: req.Parent2,
	})))
}

// Preview handles POST /einstufungen/preview
// @Summary Preview a fee classification
// @Description Computes an unsaved Einstufung form exactly like create, follow-up or update would, including the monthly table. Nothing is saved.
// @Tags Einstufungen
// @Accept json
// @Produce json
// @Param body body PreviewEinstufungRequest true "Form data"
// @Success 200 {object} EinstufungPreviewResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/preview [post]
func (h *EinstufungHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var req PreviewEinstufungRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	input := service.PreviewEinstufungInput{
		Year:              req.Year,
		IncomeCalculation: req.IncomeCalculation,
		IncomeBasis:       domain.IncomeBasis(req.IncomeBasis),
		ChildrenCount:     req.ChildrenCount,
		Provisional:       req.Provisional,
	}
	parseID := func(value, field string, target **uuid.UUID) bool {
		if value == "" {
			return true
		}
		id, err := uuid.Parse(value)
		if err != nil {
			response.BadRequest(w, "invalid "+field)
			return false
		}
		*target = &id
		return true
	}
	var childID *uuid.UUID
	if !parseID(req.EinstufungID, "einstufungId", &input.EinstufungID) ||
		!parseID(req.SourceEinstufungID, "sourceEinstufungId", &input.SourceEinstufungID) ||
		!parseID(req.ChildID, "childId", &childID) {
		return
	}
	if childID != nil {
		input.ChildID = *childID
	} else if input.EinstufungID == nil && input.SourceEinstufungID == nil {
		response.BadRequest(w, "childId, sourceEinstufungId or einstufungId is required")
		return
	}
	if req.ChangeDate != "" {
		changeDate, err := time.Parse("2006-01-02", req.ChangeDate)
		if err != nil {
			response.BadRequest(w, "invalid changeDate date (expected YYYY-MM-DD)")
			return
		}
		input.ChangeDate = &changeDate
	}

	preview, err := h.einstufungService.Preview(r.Context(), input)
	if err != nil {
		handleEinstufungError(w, err)
		return
	}

	response.Success(w, EinstufungPreviewResponse{
		Einstufung: toEinstufungResponse(preview.Einstufung),
		Income:     toCalculateIncomeResponse(preview.Income),
		Warnings:   preview.Warnings,
		Notices:    preview.Notices,
	})
}

func toCalculateIncomeResponse(b service.IncomeBreakdown) CalculateIncomeResponse {
	return CalculateIncomeResponse{
		Parent1NetIncome:         b.Parent1NetIncome,
		Parent2NetIncome:         b.Parent2NetIncome,
		Parent1FeeRelevantIncome: b.Parent1FeeRelevantIncome,
		Parent2FeeRelevantIncome: b.Parent2FeeRelevantIncome,
		Parent1AdvertisingCosts:  b.Parent1AdvertisingCosts,
		Parent2AdvertisingCosts:  b.Parent2AdvertisingCosts,
		HouseholdFeeIncome:       b.HouseholdFeeIncome,
		HouseholdFullIncome:      b.HouseholdFullIncome,
	}
}

// toEinstufungResponse converts a domain Einstufung to API response.
func toEinstufungResponse(e *domain.Einstufung) EinstufungResponse {
	resp := EinstufungResponse{
		ID:                   e.ID.String(),
		ChildID:              e.ChildID.String(),
		HouseholdID:          e.HouseholdID.String(),
		Year:                 e.Year,
		ValidFrom:            e.ValidFrom.Format("2006-01-02"),
		EffectiveFromMonth:   e.EffectiveFromMonth.Format("2006-01-02"),
		IncomeCalculation:    e.IncomeCalculation,
		AnnualNetIncome:      e.AnnualNetIncome,
		IncomeBasis:          string(e.IncomeBasis),
		HighestRateVoluntary: e.HighestRateVoluntary,
		Provisional:          e.Provisional,
		CareHoursPerWeek:     e.CareHoursPerWeek,
		CareType:             string(e.CareType),
		ChildrenCount:        e.ChildrenCount,
		MonthlyChildcareFee:  e.MonthlyChildcareFee,
		MonthlyFoodFee:       e.MonthlyFoodFee,
		AnnualMembershipFee:  e.AnnualMembershipFee,
		FeeRule:              e.FeeRule,
		DiscountPercent:      e.DiscountPercent,
		DiscountFactor:       e.DiscountFactor,
		BaseFee:              e.BaseFee,
		Notes:                e.Notes,
		CreatedAt:            e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            e.UpdatedAt.Format(time.RFC3339),
	}
	if e.LetterSentAt != nil {
		sentAt := e.LetterSentAt.Format(time.RFC3339)
		resp.LetterSentAt = &sentAt
	}
	if e.ValidUntil != nil {
		validUntil := e.ValidUntil.Format("2006-01-02")
		resp.ValidUntil = &validUntil
	}
	if e.SourceEinstufungID != nil {
		sourceID := e.SourceEinstufungID.String()
		resp.SourceEinstufungID = &sourceID
	}
	if e.ChangeDate != nil {
		changeDate := e.ChangeDate.Format("2006-01-02")
		resp.ChangeDate = &changeDate
	}
	if e.EffectiveFromMonth.IsZero() {
		resp.EffectiveFromMonth = e.ValidFrom.Format("2006-01-02")
	}

	// The service builds this from the child's care-hours history. Keep the
	// domain fallback for callers that do not load relations through the service.
	resp.MonthlyTable = e.MonthlyTable
	if resp.MonthlyTable == nil {
		resp.MonthlyTable = e.GenerateMonthlyTable(e.Child)
	}

	if e.Child != nil {
		resp.Child = e.Child
	}
	if e.Household != nil {
		resp.Household = e.Household
	}

	return resp
}

func syncResponse(result *service.ChildcareExpectationSyncResult) *ChildcareExpectationSyncResponse {
	if result == nil {
		return nil
	}
	resp := toChildcareExpectationSyncResponse(result)
	return &resp
}

func toChildcareExpectationSyncResponse(result *service.ChildcareExpectationSyncResult) ChildcareExpectationSyncResponse {
	if result == nil {
		return ChildcareExpectationSyncResponse{}
	}
	creditReview := make([]CreditReviewPeriodResponse, len(result.CreditReviewRequired))
	for i, item := range result.CreditReviewRequired {
		creditReview[i] = CreditReviewPeriodResponse{
			FeeID:         item.FeeID.String(),
			Year:          item.Year,
			Month:         item.Month,
			OldAmount:     item.OldAmount,
			NewAmount:     item.NewAmount,
			MatchedAmount: item.MatchedAmount,
			CreditAmount:  item.CreditAmount,
		}
	}
	return ChildcareExpectationSyncResponse{
		Created:              result.Created,
		Updated:              result.Updated,
		Removed:              result.Removed,
		Skipped:              result.Skipped,
		DeltaOpen:            result.DeltaOpen,
		CreditReviewRequired: creditReview,
	}
}

// SendEinstufungLetterRequest is the email written in the UI; the PDF is attached by the server.
// @Description Email subject and text for sending the Einstufung letter
type SendEinstufungLetterRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
} //@name SendEinstufungLetterRequest

// SendEinstufungLetterResponse reports a sent letter.
// @Description Result of sending the Einstufung letter
type SendEinstufungLetterResponse struct {
	Recipients []string `json:"recipients"`
	SentAt     string   `json:"sentAt"`
	Filename   string   `json:"filename"`
} //@name SendEinstufungLetterResponse

// EinstufungLetterResponse is a letter that was emailed to the parents.
type EinstufungLetterResponse struct {
	ID           string  `json:"id"`
	EinstufungID *string `json:"einstufungId,omitempty" binding:"optional"`
	SentAt       string  `json:"sentAt"`
	SentByName   *string `json:"sentByName,omitempty" binding:"optional"`
	Recipients   string  `json:"recipients"`
	Subject      string  `json:"subject"`
	Filename     string  `json:"filename"`
} //@name EinstufungLetter

func toEinstufungLetterResponse(l domain.EinstufungLetter) EinstufungLetterResponse {
	resp := EinstufungLetterResponse{
		ID: l.ID.String(), SentAt: l.SentAt.Format(time.RFC3339), SentByName: l.SentByName,
		Recipients: l.Recipients, Subject: l.Subject, Filename: l.Filename,
	}
	if l.EinstufungID != nil {
		id := l.EinstufungID.String()
		resp.EinstufungID = &id
	}
	return resp
}

// LetterPDF handles GET /einstufungen/{id}/pdf
// @Summary Download the Einstufung letter
// @Description Renders the letter as PDF. The letter is dated with the send date while the Einstufung is unchanged since sending, else today.
// @Tags Einstufungen
// @Produce application/pdf
// @Param id path string true "Einstufung ID"
// @Param inline query bool false "Show in the browser instead of downloading"
// @Success 200 {file} binary
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id}/pdf [get]
func (h *EinstufungHandler) LetterPDF(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}
	file, err := h.letterService.PDF(r.Context(), id)
	if err != nil {
		handleEinstufungError(w, err)
		return
	}
	writeLetterPDF(w, r, file)
}

func writeLetterPDF(w http.ResponseWriter, r *http.Request, file *service.LetterFile) {
	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" || r.URL.Query().Get("inline") == "true" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.Filename}))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Data)
}

// ListLetters handles GET /einstufungen/{id}/letters
// @Summary List sent Einstufung letters
// @Description Lists all letters emailed for the Einstufung's child, newest first, without the PDF data.
// @Tags Einstufungen
// @Produce json
// @Param id path string true "Einstufung ID"
// @Success 200 {array} EinstufungLetterResponse
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id}/letters [get]
func (h *EinstufungHandler) ListLetters(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}
	letters, err := h.letterService.ListLetters(r.Context(), id)
	if err != nil {
		handleEinstufungError(w, err)
		return
	}
	result := make([]EinstufungLetterResponse, 0, len(letters))
	for _, l := range letters {
		result = append(result, toEinstufungLetterResponse(l))
	}
	response.Success(w, result)
}

// SentLetterPDF handles GET /einstufungen/letters/{letterId}/pdf
// @Summary Download a sent Einstufung letter
// @Description Returns the PDF exactly as it was attached to the email.
// @Tags Einstufungen
// @Produce application/pdf
// @Param letterId path string true "Letter ID"
// @Param inline query bool false "Show in the browser instead of downloading"
// @Success 200 {file} binary
// @Failure 404 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/letters/{letterId}/pdf [get]
func (h *EinstufungHandler) SentLetterPDF(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "letterId"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}
	file, err := h.letterService.SentLetter(r.Context(), id)
	if err != nil {
		handleEinstufungError(w, err)
		return
	}
	writeLetterPDF(w, r, file)
}

// SendLetter handles POST /einstufungen/{id}/send
// @Summary Email the Einstufung letter
// @Description Sends the email to all parent addresses of the household with the letter PDF attached, logs it and records the send date.
// @Tags Einstufungen
// @Accept json
// @Produce json
// @Param id path string true "Einstufung ID"
// @Param body body SendEinstufungLetterRequest true "Email subject and text"
// @Success 200 {object} SendEinstufungLetterResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Failure 503 {object} response.ErrorBody
// @Security BearerAuth
// @Router /einstufungen/{id}/send [post]
func (h *EinstufungHandler) SendLetter(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid ID")
		return
	}
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}
	actorID, _ := uuid.Parse(userCtx.UserID)
	var req SendEinstufungLetterRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	result, err := h.letterService.Send(r.Context(), id, actorID, service.SendEinstufungLetterInput{
		Subject: req.Subject, Body: req.Body,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailDisabled) {
			response.Error(w, http.StatusServiceUnavailable, "E-Mail-Versand ist auf diesem Server nicht eingerichtet.")
			return
		}
		handleEinstufungError(w, err)
		return
	}
	response.Success(w, SendEinstufungLetterResponse{
		Recipients: result.Recipients, SentAt: result.SentAt.Format(time.RFC3339), Filename: result.Filename,
	})
}

func handleEinstufungError(w http.ResponseWriter, err error) {
	if writeNoFeeScheduleError(w, err) {
		return
	}
	var validation *service.ValidationError
	switch {
	case errors.As(err, &validation):
		response.BadRequest(w, validation.Message)
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "Einstufung, Kind oder Haushalt nicht gefunden")
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, "Ungültige Eingabe")
	default:
		response.InternalError(w, err.Error())
	}
}
