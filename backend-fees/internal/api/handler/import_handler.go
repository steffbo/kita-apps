package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// ImportHandler handles CSV import-related requests.
type ImportHandler struct {
	importService *service.ImportService
}

// MatchSuggestion represents a suggested match between transaction and expectation
// @Description Suggested match for manual review
type MatchSuggestion struct {
	TransactionID   string  `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExpectationID   string  `json:"expectationId" example:"550e8400-e29b-41d4-a716-446655440001"`
	TransactionInfo string  `json:"transactionInfo" example:"SEPA-Überweisung Max Mustermann"`
	ExpectationInfo string  `json:"expectationInfo" example:"Betreuungsgebühr März 2024 - Max Mustermann"`
	Confidence      float64 `json:"confidence" example:"0.85"`
	Reason          string  `json:"reason" example:"Name match"`
} //@name MatchSuggestion

// ImportHistoryListResponse represents a paginated list of import history
// @Description Paginated import history
type ImportHistoryListResponse struct {
	Data       []domain.ImportBatch `json:"data"`
	Total      int                  `json:"total" example:"10"`
	Page       int                  `json:"page" example:"1"`
	PerPage    int                  `json:"perPage" example:"20"`
	TotalPages int                  `json:"totalPages" example:"1"`
} //@name ImportHistoryList

// NewImportHandler creates a new import handler.
func NewImportHandler(importService *service.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

// Upload handles POST /import/upload
// @Summary Upload a CSV file for import
// @Description Upload and process a bank statement CSV file to match transactions with fees
// @Tags Import
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file (max 5MB)"
// @Success 200 {object} service.ImportResult "Upload and processing result"
// @Failure 400 {object} response.ErrorBody "No file provided or invalid format"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 413 {object} response.ErrorBody "File larger than 5MB"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/upload [post]
func (h *ImportHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if !parseUpload(w, r) {
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "no file provided")
		return
	}
	defer file.Close()

	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	result, err := h.importService.ProcessCSV(r.Context(), file, header.Filename, userID)
	if err != nil {
		response.InternalError(w, "failed to process CSV: "+err.Error())
		return
	}

	response.Success(w, result)
}

// ConfirmMatchRequest represents a request to confirm matches.
// @Description Request body for confirming transaction-fee matches
type ConfirmMatchRequest struct {
	Matches []MatchConfirmation `json:"matches"`
} //@name ConfirmMatchRequest

// MatchConfirmation represents a single match confirmation.
// @Description Single match confirmation
type MatchConfirmation struct {
	TransactionID string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExpectationID string `json:"expectationId" example:"550e8400-e29b-41d4-a716-446655440001"`
} //@name MatchConfirmation

// ConfirmMatchResponse represents the result of confirming matches
// @Description Result of confirming matches
type ConfirmMatchResponse struct {
	Confirmed int `json:"confirmed" example:"5"`
	Failed    int `json:"failed" example:"0"`
} //@name ConfirmMatchResponse

// Confirm handles POST /import/confirm
// @Summary Confirm suggested matches
// @Description Confirm multiple transaction-fee matches from suggestions
// @Tags Import
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param matches body ConfirmMatchRequest true "Matches to confirm"
// @Success 200 {object} ConfirmMatchResponse "Confirmation result"
// @Failure 400 {object} response.ErrorBody "Invalid request body or IDs"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/confirm [post]
func (h *ImportHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req ConfirmMatchRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	var matches []service.MatchConfirmation
	for _, m := range req.Matches {
		txID, err := uuid.Parse(m.TransactionID)
		if err != nil {
			response.BadRequest(w, "invalid transaction ID: "+m.TransactionID)
			return
		}
		expID, err := uuid.Parse(m.ExpectationID)
		if err != nil {
			response.BadRequest(w, "invalid expectation ID: "+m.ExpectationID)
			return
		}
		matches = append(matches, service.MatchConfirmation{
			TransactionID: txID,
			ExpectationID: expID,
		})
	}

	result, err := h.importService.ConfirmMatches(r.Context(), matches, userID)
	if err != nil {
		response.InternalError(w, "failed to confirm matches")
		return
	}

	response.Success(w, result)
}

// History handles GET /import/history
// @Summary Get import history
// @Description Get a paginated list of past CSV imports
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} ImportHistoryListResponse "Import history"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/history [get]
func (h *ImportHandler) History(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	history, total, err := h.importService.GetHistory(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get import history")
		return
	}

	response.Paginated(w, history, total, pagination.Page, pagination.PerPage)
}

// RescanResponse represents the result of a rescan operation
// @Description Rescan result with new match suggestions
type RescanResponse struct {
	Scanned     int               `json:"scanned" example:"226"`
	AutoMatched int               `json:"autoMatched" example:"150"`
	NewMatches  int               `json:"newMatches" example:"5"`
	Suggestions []MatchSuggestion `json:"suggestions,omitempty"`
	// Errors lists transactions whose warning or automatic match could not be saved.
	Errors []domain.ImportError `json:"errors"`
} //@name RescanResponse

// Rescan handles POST /import/rescan
// @Summary Rescan unmatched transactions
// @Description Re-run matching algorithm on all unmatched transactions. High-confidence matches (95%+) are automatically confirmed.
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Success 200 {object} RescanResponse "Rescan result"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/rescan [post]
func (h *ImportHandler) Rescan(w http.ResponseWriter, r *http.Request) {
	result, err := h.importService.Rescan(r.Context())
	if err != nil {
		response.InternalError(w, "failed to rescan transactions: "+err.Error())
		return
	}

	resp := RescanResponse{
		Scanned:     result.Scanned,
		AutoMatched: result.AutoMatched,
		NewMatches:  len(result.Suggestions),
		Suggestions: make([]MatchSuggestion, 0, len(result.Suggestions)),
		Errors:      result.Errors,
	}

	for _, s := range result.Suggestions {
		ms := MatchSuggestion{
			TransactionID: s.Transaction.ID.String(),
			Confidence:    s.Confidence,
			Reason:        s.MatchedBy,
		}

		// Build transaction info
		txInfo := fmt.Sprintf("%.2f EUR", s.Transaction.Amount)
		if s.Transaction.PayerName != nil {
			txInfo = *s.Transaction.PayerName + " - " + txInfo
		}
		if s.Transaction.Description != nil {
			txInfo += " (" + *s.Transaction.Description + ")"
		}
		ms.TransactionInfo = txInfo

		// Build expectation info
		if s.Expectation != nil {
			ms.ExpectationID = s.Expectation.ID.String()
			expInfo := fmt.Sprintf("%s %d", s.Expectation.FeeType, s.Expectation.Year)
			if s.Expectation.Month != nil {
				expInfo = fmt.Sprintf("%s %s %d", s.Expectation.FeeType, util.MonthToGerman(*s.Expectation.Month), s.Expectation.Year)
			}
			if s.Child != nil {
				expInfo += " - " + s.Child.FirstName + " " + s.Child.LastName
			}
			ms.ExpectationInfo = expInfo
		} else if len(s.Expectations) > 0 {
			// Combined match - use first expectation
			ms.ExpectationID = s.Expectations[0].ID.String()
			var expParts []string
			for _, e := range s.Expectations {
				expInfo := string(e.FeeType)
				if e.Month != nil {
					expInfo = fmt.Sprintf("%s %s %d", e.FeeType, util.MonthToGerman(*e.Month), e.Year)
				}
				expParts = append(expParts, expInfo)
			}
			ms.ExpectationInfo = strings.Join(expParts, " + ")
			if s.Child != nil {
				ms.ExpectationInfo += " - " + s.Child.FirstName + " " + s.Child.LastName
			}
		}

		resp.Suggestions = append(resp.Suggestions, ms)
	}

	response.Success(w, resp)
}
