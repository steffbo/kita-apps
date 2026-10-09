package handler

import (
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"net/http"
)

// LinkParentRequest represents a request to link a parent to a child.
// @Description Request body for linking a parent to a child
type LinkParentRequest struct {
	ParentID  string `json:"parentId" example:"550e8400-e29b-41d4-a716-446655440000"`
	IsPrimary bool   `json:"isPrimary" example:"true"`
} //@name LinkParentRequest

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
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
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
		if err == service.ErrHouseholdMismatch {
			response.Conflict(w, "child and parent belong to different households; merge households before linking")
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
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	parentID, ok := parseUUIDParam(w, r, "parentId")
	if !ok {
		return
	}

	if err := h.childService.UnlinkParent(r.Context(), childID, parentID); err != nil {
		response.InternalError(w, "failed to unlink parent")
		return
	}

	response.NoContent(w)
}

// LedgerEntryResponse represents a single entry in the payment ledger.
// @Description Ledger entry for a child
type LedgerEntryResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Date        string  `json:"date" example:"2024-01-05"`
	Type        string  `json:"type" example:"fee" enums:"fee,payment"`
	Description string  `json:"description" example:"Essensgeld Januar 2024"`
	FeeType     string  `json:"feeType,omitempty" example:"FOOD" binding:"optional"`
	Year        int     `json:"year,omitempty" example:"2024" binding:"optional"`
	Month       *int    `json:"month,omitempty" example:"1" binding:"optional"`
	Debit       float64 `json:"debit" example:"45.40"`
	Credit      float64 `json:"credit" example:"0"`
	Balance     float64 `json:"balance" example:"45.40"`
	IsPaid      bool    `json:"isPaid,omitempty" example:"false" binding:"optional"`
	PaidAt      *string `json:"paidAt,omitempty" example:"2024-01-10" binding:"optional"`
} //@name LedgerEntry

// LedgerSummaryResponse provides totals for the ledger.
// @Description Summary totals for the ledger
type LedgerSummaryResponse struct {
	TotalFees      float64 `json:"totalFees" example:"500.00"`
	TotalPaid      float64 `json:"totalPaid" example:"400.00"`
	TotalOpen      float64 `json:"totalOpen" example:"100.00"`
	OpenFeesCount  int     `json:"openFeesCount" example:"2"`
	PaidFeesCount  int     `json:"paidFeesCount" example:"8"`
	TotalFeesCount int     `json:"totalFeesCount" example:"10"`
} //@name LedgerSummary

// ChildLedgerResponse represents the complete payment ledger for a child.
// @Description Payment ledger for a child
type ChildLedgerResponse struct {
	ChildID string                `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Child   interface{}           `json:"child,omitempty" binding:"optional"`
	Entries []LedgerEntryResponse `json:"entries"`
	Summary LedgerSummaryResponse `json:"summary"`
} //@name ChildLedger

// GetLedger returns the payment ledger for a child
// @Summary Get child payment ledger
// @Description Retrieve the payment ledger showing all fees and payments for a child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param year query int false "Filter by year"
// @Success 200 {object} ChildLedgerResponse "Payment ledger"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/ledger [get]
func (h *ChildHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	year := request.GetQueryIntOptional(r, "year")

	ledger, err := h.feeService.GetChildLedger(r.Context(), childID, year)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get ledger")
		return
	}

	response.Success(w, ledger)
}

// FeeCoverageResponse represents monthly fee coverage.
// @Description Monthly fee coverage with transaction details
type FeeCoverageResponse struct {
	Year          int                          `json:"year" example:"2024"`
	Month         int                          `json:"month" example:"3"`
	ExpectedTotal float64                      `json:"expectedTotal" example:"110.00"`
	ReceivedTotal float64                      `json:"receivedTotal" example:"110.00"`
	Balance       float64                      `json:"balance" example:"0.00"`
	Status        string                       `json:"status" example:"COVERED" enums:"UNPAID,PARTIAL,COVERED,OVERPAID"`
	Transactions  []CoveredTransactionResponse `json:"transactions"`
}

// CoveredTransactionResponse represents a transaction covering a fee period.
type CoveredTransactionResponse struct {
	TransactionID  string  `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Amount         float64 `json:"amount" example:"66.00"`
	BookingDate    string  `json:"bookingDate" example:"2024-03-05"`
	Description    *string `json:"description,omitempty" example:"Platzgeld März" binding:"optional"`
	IsForThisMonth bool    `json:"isForThisMonth" example:"true"`
}

// GetTimeline returns a month-by-month fee coverage timeline for a child.
// @Summary Get child fee timeline
// @Description Returns monthly fee coverage showing which months are paid/unpaid based on transaction dates
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param year query int false "Year (defaults to current year)"
// @Success 200 {array} FeeCoverageResponse "Monthly coverage timeline"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/timeline [get]
func (h *ChildHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	year := request.GetQueryIntOptional(r, "year")
	if year == nil {
		response.BadRequest(w, "year is required")
		return
	}

	timeline, err := h.coverageService.GetChildTimeline(r.Context(), childID, *year)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get timeline")
		return
	}

	// Convert to response format
	var resp []FeeCoverageResponse
	for _, c := range timeline {
		coverage := FeeCoverageResponse{
			Year:          c.Year,
			Month:         c.Month,
			ExpectedTotal: c.ExpectedTotal,
			ReceivedTotal: c.ReceivedTotal,
			Balance:       c.Balance,
			Status:        string(c.Status),
		}

		for _, tx := range c.Transactions {
			coverage.Transactions = append(coverage.Transactions, CoveredTransactionResponse{
				TransactionID:  tx.TransactionID.String(),
				Amount:         tx.Amount,
				BookingDate:    tx.BookingDate.Format("2006-01-02"),
				Description:    tx.Description,
				IsForThisMonth: tx.IsForThisMonth,
			})
		}

		resp = append(resp, coverage)
	}

	response.Success(w, resp)
}
