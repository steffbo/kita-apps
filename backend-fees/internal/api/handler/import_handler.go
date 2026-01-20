package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ImportHandler handles CSV import-related requests.
type ImportHandler struct {
	importService *service.ImportService
}

// NewImportHandler creates a new import handler.
func NewImportHandler(importService *service.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

// Upload handles POST /import/upload
func (h *ImportHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Max 10MB file
	r.ParseMultipartForm(10 << 20)

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
type ConfirmMatchRequest struct {
	Matches []MatchConfirmation `json:"matches"`
}

// MatchConfirmation represents a single match confirmation.
type MatchConfirmation struct {
	TransactionID string `json:"transactionId"`
	ExpectationID string `json:"expectationId"`
}

// Confirm handles POST /import/confirm
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
func (h *ImportHandler) History(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	history, total, err := h.importService.GetHistory(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get import history")
		return
	}

	response.Paginated(w, history, total, pagination.Page, pagination.PerPage)
}

// UnmatchedTransactions handles GET /import/transactions
func (h *ImportHandler) UnmatchedTransactions(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	transactions, total, err := h.importService.GetUnmatchedTransactions(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get unmatched transactions")
		return
	}

	response.Paginated(w, transactions, total, pagination.Page, pagination.PerPage)
}

// ManualMatchRequest represents a request to manually match a transaction.
type ManualMatchRequest struct {
	TransactionID string `json:"transactionId"`
	ExpectationID string `json:"expectationId"`
}

// ManualMatch handles POST /import/match
func (h *ImportHandler) ManualMatch(w http.ResponseWriter, r *http.Request) {
	var req ManualMatchRequest
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

	txID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}

	expID, err := uuid.Parse(req.ExpectationID)
	if err != nil {
		response.BadRequest(w, "invalid expectation ID")
		return
	}

	match, err := h.importService.CreateManualMatch(r.Context(), txID, expID, userID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "transaction or expectation not found")
			return
		}
		response.InternalError(w, "failed to create manual match")
		return
	}

	response.Created(w, match)
}
