package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// TransactionListResponse represents a paginated list of bank transactions
// @Description Paginated list of bank transactions as returned by the import transaction endpoints
type TransactionListResponse struct {
	Data       []domain.BankTransaction `json:"data"`
	Total      int                      `json:"total" example:"25"`
	Page       int                      `json:"page" example:"1"`
	PerPage    int                      `json:"perPage" example:"20"`
	TotalPages int                      `json:"totalPages" example:"2"`
} //@name TransactionList

// UnmatchedTransactions handles GET /import/transactions
// @Summary Get unmatched transactions
// @Description Get a paginated list of transactions that haven't been matched to fees
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param search query string false "Search by payer name or description"
// @Param sortBy query string false "Sort field: date, payer, description, amount" default(date)
// @Param sortDir query string false "Sort direction: asc, desc" default(desc)
// @Success 200 {object} TransactionListResponse "Unmatched transactions"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions [get]
func (h *ImportHandler) UnmatchedTransactions(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sortBy")
	sortDir := r.URL.Query().Get("sortDir")

	transactions, total, err := h.importService.GetUnmatchedTransactions(r.Context(), search, sortBy, sortDir, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get unmatched transactions")
		return
	}

	response.Paginated(w, transactions, total, pagination.Page, pagination.PerPage)
}

// ManualMatchRequest represents a request to manually match a transaction.
// @Description Request body for manual matching
type ManualMatchRequest struct {
	TransactionID string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExpectationID string `json:"expectationId" example:"550e8400-e29b-41d4-a716-446655440001"`
} //@name ManualMatchRequest

// ManualMatchResponse represents the result of a manual match
// @Description Manual match result
type ManualMatchResponse struct {
	TransactionID string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExpectationID string `json:"expectationId" example:"550e8400-e29b-41d4-a716-446655440001"`
	MatchedAt     string `json:"matchedAt" example:"2024-03-15T10:30:00Z"`
	MatchedBy     string `json:"matchedBy" example:"admin@knirpsenstadt.de"`
} //@name ManualMatchResponse

// ManualMatch handles POST /import/match
// @Summary Manually match a transaction
// @Description Create a manual match between a transaction and a fee expectation
// @Tags Import
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param match body ManualMatchRequest true "Match data"
// @Success 201 {object} ManualMatchResponse "Match created"
// @Failure 400 {object} response.ErrorBody "Invalid transaction or expectation ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction or expectation not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/match [post]
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

// DismissTransactionResponse represents the result of dismissing a transaction
// @Description Dismiss transaction result
type DismissTransactionResponse struct {
	TransactionID    string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	IBAN             string `json:"iban" example:"DE89370400440532013000"`
	AddedToBlacklist bool   `json:"addedToBlacklist" example:"true"`
} //@name DismissTransactionResponse

// HideTransactionResponse represents the result of hiding a transaction
// @Description Hide transaction result
type HideTransactionResponse struct {
	TransactionID string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
} //@name HideTransactionResponse

// UnmatchTransactionRequest represents a request to unmatch (and optionally delete) a transaction
// @Description Unmatch transaction request
type UnmatchTransactionRequest struct {
	DeleteTransaction bool `json:"deleteTransaction" example:"false"`
} //@name UnmatchTransactionRequest

// UnmatchTransactionResponse represents the result of unmatching a transaction
// @Description Unmatch transaction result
type UnmatchTransactionResponse struct {
	TransactionID      string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	MatchesRemoved     int64  `json:"matchesRemoved" example:"1"`
	TransactionDeleted bool   `json:"transactionDeleted" example:"false"`
} //@name UnmatchTransactionResponse

// ChildUnmatchedSuggestionsResponse represents likely unmatched transactions for a child
// @Description Likely unmatched transactions for a child
type ChildUnmatchedSuggestionsResponse struct {
	ChildID     string                   `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Scanned     int                      `json:"scanned" example:"250"`
	Suggestions []domain.MatchSuggestion `json:"suggestions"`
} //@name ChildUnmatchedSuggestionsResponse

// AllocationRequest represents a single fee allocation.
// @Description Fee allocation entry
type AllocationRequest struct {
	ExpectationID string  `json:"expectationId" example:"550e8400-e29b-41d4-a716-446655440001"`
	Amount        float64 `json:"amount" example:"45.40"`
} //@name AllocationRequest

// AllocateTransactionRequest represents a request to allocate a transaction.
// @Description Allocate a transaction across multiple fees
type AllocateTransactionRequest struct {
	Allocations []AllocationRequest `json:"allocations"`
} //@name AllocateTransactionRequest

// AllocateTransactionResponse represents the result of an allocation.
// @Description Allocation result
type AllocateTransactionResponse struct {
	TransactionID      string  `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	AllocationsCreated int     `json:"allocationsCreated" example:"2"`
	TotalAllocated     float64 `json:"totalAllocated" example:"90.80"`
	Overpayment        float64 `json:"overpayment" example:"0.00"`
} //@name AllocateTransactionResponse

// DismissTransaction handles POST /import/transactions/{id}/dismiss
// @Summary Dismiss a transaction
// @Description Dismiss an unmatched transaction and optionally add its IBAN to blacklist
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Success 200 {object} DismissTransactionResponse "Dismissal result"
// @Failure 400 {object} response.ErrorBody "Invalid transaction ID or transaction has no IBAN"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/dismiss [post]
func (h *ImportHandler) DismissTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}

	result, err := h.importService.DismissTransaction(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "transaction not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "transaction has no IBAN")
			return
		}
		response.InternalError(w, "failed to dismiss transaction")
		return
	}

	response.Success(w, result)
}

// HideTransaction handles POST /import/transactions/{id}/hide
// @Summary Hide a transaction
// @Description Hide an unmatched transaction without blacklisting its IBAN
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Success 200 {object} HideTransactionResponse "Hide result"
// @Failure 400 {object} response.ErrorBody "Invalid transaction ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/hide [post]
func (h *ImportHandler) HideTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}

	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}
	userID, _ := uuid.Parse(userCtx.UserID)

	result, err := h.importService.HideTransaction(r.Context(), id, userID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "transaction not found")
			return
		}
		response.InternalError(w, "failed to hide transaction")
		return
	}

	response.Success(w, result)
}

// UnmatchTransaction handles POST /import/transactions/{id}/unmatch
// @Summary Unmatch a transaction
// @Description Remove all matches for a transaction and optionally delete the transaction itself
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Param request body UnmatchTransactionRequest true "Unmatch options"
// @Success 200 {object} UnmatchTransactionResponse "Unmatch result"
// @Failure 400 {object} response.ErrorBody "Invalid transaction ID or transaction has no matches"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/unmatch [post]
func (h *ImportHandler) UnmatchTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}

	var req UnmatchTransactionRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	result, err := h.importService.UnmatchTransaction(r.Context(), id, req.DeleteTransaction)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "transaction not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "transaction has no matches")
			return
		}
		response.InternalError(w, "failed to unmatch transaction")
		return
	}

	resp := UnmatchTransactionResponse{
		TransactionID:      result.TransactionID.String(),
		MatchesRemoved:     result.MatchesRemoved,
		TransactionDeleted: result.TransactionDeleted,
	}
	response.Success(w, resp)
}

// AllocateTransaction handles POST /import/transactions/{id}/allocate
// @Summary Allocate a transaction across multiple fees
// @Description Allocate a transaction across multiple fee expectations
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Param request body AllocateTransactionRequest true "Allocation data"
// @Success 200 {object} AllocateTransactionResponse "Allocation result"
// @Failure 400 {object} response.ErrorBody "Invalid request or allocation"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction or fee not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/allocate [post]
func (h *ImportHandler) AllocateTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}

	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}
	userID, _ := uuid.Parse(userCtx.UserID)

	var req AllocateTransactionRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if len(req.Allocations) == 0 {
		response.BadRequest(w, "no allocations provided")
		return
	}

	inputs := make([]service.AllocationInput, 0, len(req.Allocations))
	for _, alloc := range req.Allocations {
		expectationID, err := uuid.Parse(alloc.ExpectationID)
		if err != nil {
			response.BadRequest(w, "invalid expectation ID")
			return
		}
		inputs = append(inputs, service.AllocationInput{
			ExpectationID: expectationID,
			Amount:        alloc.Amount,
		})
	}

	result, err := h.importService.AllocateTransaction(r.Context(), id, userID, inputs)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "transaction or fee not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid allocation")
			return
		}
		response.InternalError(w, "failed to allocate transaction")
		return
	}

	resp := AllocateTransactionResponse{
		TransactionID:      result.TransactionID.String(),
		AllocationsCreated: result.AllocationsCreated,
		TotalAllocated:     result.TotalAllocated,
		Overpayment:        result.Overpayment,
	}
	response.Success(w, resp)
}

// ChildUnmatchedSuggestions handles GET /import/transactions/unmatched/child/{id}
// @Summary Get likely unmatched transactions for a child
// @Description Returns unmatched transactions that likely belong to the given child
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param minConfidence query number false "Minimum confidence (0-1)" default(0.6)
// @Param limit query int false "Max suggestions to return" default(10)
// @Success 200 {object} ChildUnmatchedSuggestionsResponse "Suggestions"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/unmatched/child/{id} [get]
func (h *ImportHandler) ChildUnmatchedSuggestions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	minConfidence := 0.6
	if minConfidenceStr := r.URL.Query().Get("minConfidence"); minConfidenceStr != "" {
		if parsed, err := strconv.ParseFloat(minConfidenceStr, 64); err == nil {
			minConfidence = parsed
		}
	}
	if minConfidence < 0 {
		minConfidence = 0
	}
	if minConfidence > 1 {
		minConfidence = 1
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	result, err := h.importService.GetUnmatchedSuggestionsForChild(r.Context(), id, minConfidence, limit)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get suggestions")
		return
	}

	resp := ChildUnmatchedSuggestionsResponse{
		ChildID:     result.ChildID.String(),
		Scanned:     result.Scanned,
		Suggestions: result.Suggestions,
	}
	response.Success(w, resp)
}

// MatchedTransactions handles GET /import/transactions/matched
// @Summary Get matched transactions
// @Description Get a paginated list of transactions that have been matched to fees
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param search query string false "Search by payer name or description"
// @Param sortBy query string false "Sort field: date, payer, description, amount" default(date)
// @Param sortDir query string false "Sort direction: asc, desc" default(desc)
// @Success 200 {object} TransactionListResponse "Matched transactions"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/matched [get]
func (h *ImportHandler) MatchedTransactions(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sortBy")
	sortDir := r.URL.Query().Get("sortDir")

	transactions, total, err := h.importService.GetMatchedTransactions(r.Context(), search, sortBy, sortDir, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get matched transactions")
		return
	}

	response.Paginated(w, transactions, total, pagination.Page, pagination.PerPage)
}

// TransactionSuggestions handles GET /import/transactions/{id}/suggestions
// @Summary Get match suggestions for a transaction
// @Description Get potential fee matches for a single unmatched transaction
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Success 200 {object} domain.MatchSuggestion "Match suggestion"
// @Failure 400 {object} response.ErrorBody "Invalid transaction ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/suggestions [get]
func (h *ImportHandler) TransactionSuggestions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}

	suggestion, err := h.importService.GetSuggestionsForTransaction(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "transaction not found")
			return
		}
		response.InternalError(w, "failed to get suggestions")
		return
	}

	response.Success(w, suggestion)
}
