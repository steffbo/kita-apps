package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// EinstufungHandler handles Einstufung-related requests.
type EinstufungHandler struct {
	einstufungService *service.EinstufungService
}

// NewEinstufungHandler creates a new Einstufung handler.
func NewEinstufungHandler(einstufungService *service.EinstufungService) *EinstufungHandler {
	return &EinstufungHandler{einstufungService: einstufungService}
}

// EinstufungResponse represents an Einstufung in API responses.
// @Description Fee classification for a child
type EinstufungResponse struct {
	ID                   string                            `json:"id"`
	ChildID              string                            `json:"childId"`
	HouseholdID          string                            `json:"householdId"`
	Year                 int                               `json:"year"`
	ValidFrom            string                            `json:"validFrom"`
	ValidUntil           *string                           `json:"validUntil,omitempty"`
	SourceEinstufungID   *string                           `json:"sourceEinstufungId,omitempty"`
	ChangeDate           *string                           `json:"changeDate,omitempty"`
	EffectiveFromMonth   string                            `json:"effectiveFromMonth"`
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	AnnualNetIncome      float64                           `json:"annualNetIncome"`
	HighestRateVoluntary bool                              `json:"highestRateVoluntary"`
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
	Notes                string                            `json:"notes,omitempty"`
	MonthlyTable         []domain.EinstufungMonthRow       `json:"monthlyTable,omitempty"`
	CreatedAt            string                            `json:"createdAt"`
	UpdatedAt            string                            `json:"updatedAt"`
	Child                interface{}                       `json:"child,omitempty"`
	Household            interface{}                       `json:"household,omitempty"`
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
	ValidFrom            string                            `json:"validFrom"` // ISO date, e.g. "2026-01-01"
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	HighestRateVoluntary bool                              `json:"highestRateVoluntary"`
	CareHoursPerWeek     int                               `json:"careHoursPerWeek"`
	ChildrenCount        int                               `json:"childrenCount"`
	Notes                string                            `json:"notes"`
} //@name CreateEinstufungRequest

// UpdateEinstufungRequest represents a request to update an Einstufung.
// @Description Request body for updating a fee classification
type UpdateEinstufungRequest struct {
	IncomeCalculation    *domain.HouseholdIncomeCalculation `json:"incomeCalculation,omitempty"`
	HighestRateVoluntary *bool                              `json:"highestRateVoluntary,omitempty"`
	CareHoursPerWeek     *int                               `json:"careHoursPerWeek,omitempty"`
	ChildrenCount        *int                               `json:"childrenCount,omitempty"`
	ValidFrom            *string                            `json:"validFrom,omitempty"`
	Notes                *string                            `json:"notes,omitempty"`
} //@name UpdateEinstufungRequest

// CreateFollowUpEinstufungRequest represents a request to create a follow-up Einstufung.
// @Description Request body for creating a follow-up fee classification
type CreateFollowUpEinstufungRequest struct {
	ChangeDate           string                            `json:"changeDate"` // ISO date, cut-off: 1st-14th same month, 15th+ next month
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	HighestRateVoluntary bool                              `json:"highestRateVoluntary"`
	CareHoursPerWeek     int                               `json:"careHoursPerWeek"`
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
// @Description Result of synchronizing CHILDCARE expectations after a follow-up
type ChildcareExpectationSyncResponse struct {
	Created              int                          `json:"created"`
	Updated              int                          `json:"updated"`
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
	HouseholdFeeIncome       float64 `json:"householdFeeIncome"`
	HouseholdFullIncome      float64 `json:"householdFullIncome"`
} //@name CalculateIncomeResponse

// Create handles POST /einstufungen
// @Summary Create a fee classification
// @Description Creates a new Einstufung by calculating fees from income proofs
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

	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		response.BadRequest(w, "invalid validFrom date (expected YYYY-MM-DD)")
		return
	}

	if req.Year == 0 {
		req.Year = validFrom.Year()
	}

	result, err := h.einstufungService.Create(r.Context(), service.CreateEinstufungInput{
		ChildID:              childID,
		Year:                 req.Year,
		ValidFrom:            validFrom,
		IncomeCalculation:    req.IncomeCalculation,
		HighestRateVoluntary: req.HighestRateVoluntary,
		CareHoursPerWeek:     req.CareHoursPerWeek,
		ChildrenCount:        req.ChildrenCount,
		Notes:                req.Notes,
	})
	if err != nil {
		handleEinstufungError(w, err)
		return
	}

	response.Created(w, toEinstufungResponse(result))
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
// @Description Updates an existing Einstufung and recalculates fees
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
		CareHoursPerWeek:     req.CareHoursPerWeek,
		ChildrenCount:        req.ChildrenCount,
		Notes:                req.Notes,
	}

	if req.ValidFrom != nil {
		t, err := time.Parse("2006-01-02", *req.ValidFrom)
		if err != nil {
			response.BadRequest(w, "invalid validFrom date")
			return
		}
		input.ValidFrom = &t
	}

	result, err := h.einstufungService.Update(r.Context(), id, input)
	if err != nil {
		handleEinstufungError(w, err)
		return
	}

	response.Success(w, toEinstufungResponse(result))
}

// CreateFollowUp handles POST /einstufungen/{id}/follow-ups
// @Summary Create a follow-up fee classification
// @Description Creates a follow-up Einstufung, applies the monthly cut-off rule, closes the source period, and syncs open CHILDCARE expectations from the effective month.
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
		HighestRateVoluntary: req.HighestRateVoluntary,
		CareHoursPerWeek:     req.CareHoursPerWeek,
		ChildrenCount:        req.ChildrenCount,
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
		response.NotFound(w, "Einstufung not found")
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
		yearStr = strconv.Itoa(time.Now().Year())
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

	household := domain.HouseholdIncomeCalculation{
		Parent1: req.Parent1,
		Parent2: req.Parent2,
	}

	response.Success(w, CalculateIncomeResponse{
		Parent1NetIncome:         req.Parent1.CalculateNetIncome(),
		Parent2NetIncome:         req.Parent2.CalculateNetIncome(),
		Parent1FeeRelevantIncome: req.Parent1.CalculateFeeRelevantIncome(),
		Parent2FeeRelevantIncome: req.Parent2.CalculateFeeRelevantIncome(),
		HouseholdFeeIncome:       household.CalculateAnnualNetIncome(),
		HouseholdFullIncome:      household.CalculateFullNetIncome(),
	})
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
		HighestRateVoluntary: e.HighestRateVoluntary,
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

	// Generate monthly table
	var exitDate *time.Time
	if e.Child != nil {
		exitDate = e.Child.ExitDate
	}
	resp.MonthlyTable = e.GenerateMonthlyTable(exitDate)

	if e.Child != nil {
		resp.Child = e.Child
	}
	if e.Household != nil {
		resp.Household = e.Household
	}

	return resp
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
		Skipped:              result.Skipped,
		DeltaOpen:            result.DeltaOpen,
		CreditReviewRequired: creditReview,
	}
}

func handleEinstufungError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrNotFound) {
		response.NotFound(w, "Kind oder Haushalt nicht gefunden")
	} else if errors.Is(err, service.ErrInvalidInput) {
		response.BadRequest(w, "Kind muss einem Haushalt zugeordnet sein")
	} else {
		response.InternalError(w, err.Error())
	}
}
