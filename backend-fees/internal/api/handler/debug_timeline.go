package handler

import (
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
)

// DebugFeeInfo represents diagnostic information about a fee and its matches.
type DebugFeeInfo struct {
	FeeID      string           `json:"feeId"`
	Month      *int             `json:"month,omitempty"`
	Amount     float64          `json:"amount"`
	FeeType    string           `json:"feeType"`
	MatchCount int              `json:"matchCount"`
	Matches    []DebugMatchInfo `json:"matches"`
}

// DebugMatchInfo represents diagnostic information about a payment match.
type DebugMatchInfo struct {
	MatchID           string  `json:"matchId"`
	TransactionID     string  `json:"transactionId"`
	TransactionAmount float64 `json:"transactionAmount"`
	BookingDate       string  `json:"bookingDate"`
	Description       *string `json:"description,omitempty"`
}

// DebugTimelineResponse represents the diagnostic response for timeline debugging.
type DebugTimelineResponse struct {
	ChildID  string         `json:"childId"`
	Year     int            `json:"year"`
	FeeCount int            `json:"feeCount"`
	Fees     []DebugFeeInfo `json:"fees"`
}

// DebugTimeline returns diagnostic information about fees and their payment matches.
// @Summary Debug timeline data
// @Description Returns diagnostic information about fees and payment matches for debugging timeline issues
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param year query int false "Year (defaults to 2026)"
// @Success 200 {object} DebugTimelineResponse "Debug information"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/debug-timeline [get]
func (h *ChildHandler) DebugTimeline(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	year := request.GetQueryIntOptional(r, "year")
	if year == nil {
		y2026 := 2026
		year = &y2026
	}

	// Get all fees for the child in the year
	fees, err := h.feeRepo.GetForChild(r.Context(), childID, year)
	if err != nil {
		response.InternalError(w, "failed to get fees for child")
		return
	}

	resp := DebugTimelineResponse{
		ChildID:  childID.String(),
		Year:     *year,
		FeeCount: len(fees),
		Fees:     []DebugFeeInfo{},
	}

	for _, fee := range fees {
		feeInfo := DebugFeeInfo{
			FeeID:   fee.ID.String(),
			Month:   fee.Month,
			Amount:  fee.Amount,
			FeeType: string(fee.FeeType),
			Matches: []DebugMatchInfo{},
		}

		// Get all matches for this fee
		matches, err := h.matchRepo.GetAllByExpectation(r.Context(), fee.ID)
		if err != nil {
			// Log error but continue
			feeInfo.MatchCount = -1
		} else {
			feeInfo.MatchCount = len(matches)
			for _, match := range matches {
				// Load transaction details separately
				tx, txErr := h.transactionRepo.GetByID(r.Context(), match.TransactionID)
				if txErr != nil {
					// If can't load transaction, still show match info
					matchInfo := DebugMatchInfo{
						MatchID:       match.ID.String(),
						TransactionID: match.TransactionID.String(),
					}
					feeInfo.Matches = append(feeInfo.Matches, matchInfo)
				} else {
					matchInfo := DebugMatchInfo{
						MatchID:           match.ID.String(),
						TransactionID:     match.TransactionID.String(),
						TransactionAmount: tx.Amount,
						BookingDate:       tx.BookingDate.Format("2006-01-02"),
						Description:       tx.Description,
					}
					feeInfo.Matches = append(feeInfo.Matches, matchInfo)
				}
			}
		}

		resp.Fees = append(resp.Fees, feeInfo)
	}

	response.Success(w, resp)
}
