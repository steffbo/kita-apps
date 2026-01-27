package handler

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
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

// UploadResponse represents the result of a CSV upload
// @Description CSV upload and processing result
type UploadResponse struct {
	ImportID    string            `json:"importId" example:"550e8400-e29b-41d4-a716-446655440000"`
	FileName    string            `json:"fileName" example:"kontoauszug_2024_03.csv"`
	TotalRows   int               `json:"totalRows" example:"150"`
	Matched     int               `json:"matched" example:"120"`
	Unmatched   int               `json:"unmatched" example:"25"`
	Duplicates  int               `json:"duplicates" example:"5"`
	Suggestions []MatchSuggestion `json:"suggestions,omitempty"`
} //@name UploadResponse

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

// TransactionResponse represents a bank transaction
// @Description Bank transaction from CSV import
type TransactionResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BookingDate     string  `json:"bookingDate" example:"2024-03-15"`
	ValueDate       string  `json:"valueDate" example:"2024-03-15"`
	Amount          float64 `json:"amount" example:"250.00"`
	Currency        string  `json:"currency" example:"EUR"`
	IBAN            *string `json:"iban,omitempty" example:"DE89370400440532013000"`
	BIC             *string `json:"bic,omitempty" example:"COBADEFFXXX"`
	AccountHolder   *string `json:"accountHolder,omitempty" example:"Max Mustermann"`
	Purpose         *string `json:"purpose,omitempty" example:"Betreuungsgebühr März 2024"`
	TransactionType *string `json:"transactionType,omitempty" example:"SEPA-Überweisung"`
	Status          string  `json:"status" example:"unmatched" enums:"matched,unmatched,dismissed"`
	MatchedFeeID    *string `json:"matchedFeeId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
} //@name Transaction

// TransactionListResponse represents a paginated list of transactions
// @Description Paginated list of transactions
type TransactionListResponse struct {
	Data       []TransactionResponse `json:"data"`
	Total      int                   `json:"total" example:"25"`
	Page       int                   `json:"page" example:"1"`
	PerPage    int                   `json:"perPage" example:"20"`
	TotalPages int                   `json:"totalPages" example:"2"`
} //@name TransactionList

// ImportHistoryEntry represents an import history entry
// @Description Import history entry
type ImportHistoryEntry struct {
	ID         string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FileName   string `json:"fileName" example:"kontoauszug_2024_03.csv"`
	ImportedAt string `json:"importedAt" example:"2024-03-15T10:30:00Z"`
	ImportedBy string `json:"importedBy" example:"admin@knirpsenstadt.de"`
	TotalRows  int    `json:"totalRows" example:"150"`
	Matched    int    `json:"matched" example:"120"`
	Unmatched  int    `json:"unmatched" example:"25"`
	Duplicates int    `json:"duplicates" example:"5"`
} //@name ImportHistoryEntry

// ImportHistoryListResponse represents a paginated list of import history
// @Description Paginated import history
type ImportHistoryListResponse struct {
	Data       []ImportHistoryEntry `json:"data"`
	Total      int                  `json:"total" example:"10"`
	Page       int                  `json:"page" example:"1"`
	PerPage    int                  `json:"perPage" example:"20"`
	TotalPages int                  `json:"totalPages" example:"1"`
} //@name ImportHistoryList

// IBANEntry represents an IBAN in blacklist/trusted list
// @Description IBAN entry
type IBANEntry struct {
	IBAN          string  `json:"iban" example:"DE89370400440532013000"`
	AccountHolder *string `json:"accountHolder,omitempty" example:"Max Mustermann"`
	ChildID       *string `json:"childId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChildName     *string `json:"childName,omitempty" example:"Max Mustermann"`
	AddedAt       string  `json:"addedAt" example:"2024-03-15T10:30:00Z"`
	Reason        *string `json:"reason,omitempty" example:"Fremdkonto"`
} //@name IBANEntry

// IBANListResponse represents a paginated list of IBANs
// @Description Paginated IBAN list
type IBANListResponse struct {
	Data       []IBANEntry `json:"data"`
	Total      int         `json:"total" example:"5"`
	Page       int         `json:"page" example:"1"`
	PerPage    int         `json:"perPage" example:"20"`
	TotalPages int         `json:"totalPages" example:"1"`
} //@name IBANList

// WarningEntry represents an import warning
// @Description Import warning for review
type WarningEntry struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type          string  `json:"type" example:"amount_mismatch" enums:"amount_mismatch,duplicate_payment,unknown_iban"`
	Message       string  `json:"message" example:"Betrag weicht um 5.00€ ab"`
	TransactionID *string `json:"transactionId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	FeeID         *string `json:"feeId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	CreatedAt     string  `json:"createdAt" example:"2024-03-15T10:30:00Z"`
	DismissedAt   *string `json:"dismissedAt,omitempty" example:"2024-03-16T14:00:00Z"`
	DismissedBy   *string `json:"dismissedBy,omitempty" example:"admin@knirpsenstadt.de"`
	DismissNote   *string `json:"dismissNote,omitempty" example:"Eltern haben Restbetrag bar bezahlt"`
} //@name WarningEntry

// WarningListResponse represents a paginated list of warnings
// @Description Paginated warnings list
type WarningListResponse struct {
	Data       []WarningEntry `json:"data"`
	Total      int            `json:"total" example:"3"`
	Page       int            `json:"page" example:"1"`
	PerPage    int            `json:"perPage" example:"20"`
	TotalPages int            `json:"totalPages" example:"1"`
} //@name WarningList

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
// @Param file formData file true "CSV file (max 10MB)"
// @Success 200 {object} UploadResponse "Upload and processing result"
// @Failure 400 {object} response.ErrorBody "No file provided or invalid format"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/upload [post]
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

// UnmatchedTransactions handles GET /import/transactions
// @Summary Get unmatched transactions
// @Description Get a paginated list of transactions that haven't been matched to fees
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} TransactionListResponse "Unmatched transactions"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions [get]
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

// RescanResponse represents the result of a rescan operation
// @Description Rescan result with new match suggestions
type RescanResponse struct {
	NewMatches  int               `json:"newMatches" example:"5"`
	Suggestions []MatchSuggestion `json:"suggestions,omitempty"`
} //@name RescanResponse

// Rescan handles POST /import/rescan
// @Summary Rescan unmatched transactions
// @Description Re-run matching algorithm on all unmatched transactions to find new matches
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
		response.InternalError(w, "failed to rescan transactions")
		return
	}

	response.Success(w, result)
}

// DismissTransactionResponse represents the result of dismissing a transaction
// @Description Dismiss transaction result
type DismissTransactionResponse struct {
	TransactionID    string `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	IBAN             string `json:"iban" example:"DE89370400440532013000"`
	AddedToBlacklist bool   `json:"addedToBlacklist" example:"true"`
} //@name DismissTransactionResponse

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

// GetBlacklist handles GET /import/blacklist
// @Summary Get blacklisted IBANs
// @Description Get a paginated list of IBANs that are ignored during import
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} IBANListResponse "Blacklisted IBANs"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/blacklist [get]
func (h *ImportHandler) GetBlacklist(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	ibans, total, err := h.importService.GetBlacklist(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get blacklist")
		return
	}

	response.Paginated(w, ibans, total, pagination.Page, pagination.PerPage)
}

// RemoveFromBlacklist handles DELETE /import/blacklist/{iban}
// @Summary Remove IBAN from blacklist
// @Description Remove an IBAN from the blacklist so transactions from it will be processed again
// @Tags Import
// @Security BearerAuth
// @Param iban path string true "IBAN (URL-encoded)"
// @Success 204 "IBAN removed from blacklist"
// @Failure 400 {object} response.ErrorBody "Invalid IBAN or IBAN is not blacklisted"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "IBAN not found in blacklist"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/blacklist/{iban} [delete]
func (h *ImportHandler) RemoveFromBlacklist(w http.ResponseWriter, r *http.Request) {
	iban, err := url.PathUnescape(chi.URLParam(r, "iban"))
	if err != nil || iban == "" {
		response.BadRequest(w, "invalid IBAN")
		return
	}

	err = h.importService.RemoveFromBlacklist(r.Context(), iban)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "IBAN not found in blacklist")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "IBAN is not blacklisted")
			return
		}
		response.InternalError(w, "failed to remove from blacklist")
		return
	}

	response.NoContent(w)
}

// GetTrustedIBANs handles GET /import/trusted
// @Summary Get trusted IBANs
// @Description Get a paginated list of trusted IBANs that are auto-matched to specific children
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} IBANListResponse "Trusted IBANs"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted [get]
func (h *ImportHandler) GetTrustedIBANs(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	ibans, total, err := h.importService.GetTrustedIBANs(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get trusted IBANs")
		return
	}

	response.Paginated(w, ibans, total, pagination.Page, pagination.PerPage)
}

// LinkIBANRequest represents a request to link an IBAN to a child.
// @Description Request body for linking an IBAN to a child
type LinkIBANRequest struct {
	ChildID string `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
} //@name LinkIBANRequest

// LinkIBANToChild handles POST /import/trusted/{iban}/link
// @Summary Link IBAN to child
// @Description Link a trusted IBAN to a specific child for automatic matching
// @Tags Import
// @Accept json
// @Security BearerAuth
// @Param iban path string true "IBAN (URL-encoded)"
// @Param link body LinkIBANRequest true "Link data"
// @Success 204 "IBAN linked to child"
// @Failure 400 {object} response.ErrorBody "Invalid IBAN, child ID, or IBAN is not trusted"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "IBAN or child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted/{iban}/link [post]
func (h *ImportHandler) LinkIBANToChild(w http.ResponseWriter, r *http.Request) {
	iban, err := url.PathUnescape(chi.URLParam(r, "iban"))
	if err != nil || iban == "" {
		response.BadRequest(w, "invalid IBAN")
		return
	}

	var req LinkIBANRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	childID, err := uuid.Parse(req.ChildID)
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	err = h.importService.LinkIBANToChild(r.Context(), iban, childID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "IBAN or child not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "IBAN is not trusted")
			return
		}
		response.InternalError(w, "failed to link IBAN to child")
		return
	}

	response.NoContent(w)
}

// UnlinkIBANFromChild handles DELETE /import/trusted/{iban}/link
// @Summary Unlink IBAN from child
// @Description Remove the link between a trusted IBAN and a child
// @Tags Import
// @Security BearerAuth
// @Param iban path string true "IBAN (URL-encoded)"
// @Success 204 "IBAN unlinked from child"
// @Failure 400 {object} response.ErrorBody "Invalid IBAN"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "IBAN not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted/{iban}/link [delete]
func (h *ImportHandler) UnlinkIBANFromChild(w http.ResponseWriter, r *http.Request) {
	iban, err := url.PathUnescape(chi.URLParam(r, "iban"))
	if err != nil || iban == "" {
		response.BadRequest(w, "invalid IBAN")
		return
	}

	err = h.importService.UnlinkIBANFromChild(r.Context(), iban)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "IBAN not found")
			return
		}
		response.InternalError(w, "failed to unlink IBAN from child")
		return
	}

	response.NoContent(w)
}

// GetWarnings handles GET /import/warnings
// @Summary Get import warnings
// @Description Get a paginated list of warnings generated during import processing
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} WarningListResponse "Import warnings"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/warnings [get]
func (h *ImportHandler) GetWarnings(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	warnings, total, err := h.importService.GetWarnings(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get warnings")
		return
	}

	response.Paginated(w, warnings, total, pagination.Page, pagination.PerPage)
}

// DismissWarningRequest represents a request to dismiss a warning.
// @Description Request body for dismissing a warning
type DismissWarningRequest struct {
	Note string `json:"note" example:"Differenz wurde bar ausgeglichen"`
} //@name DismissWarningRequest

// DismissWarning handles POST /import/warnings/{id}/dismiss
// @Summary Dismiss a warning
// @Description Dismiss an import warning with an optional note
// @Tags Import
// @Accept json
// @Security BearerAuth
// @Param id path string true "Warning ID (UUID)"
// @Param dismiss body DismissWarningRequest true "Dismissal note"
// @Success 204 "Warning dismissed"
// @Failure 400 {object} response.ErrorBody "Invalid warning ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Warning not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/warnings/{id}/dismiss [post]
func (h *ImportHandler) DismissWarning(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid warning ID")
		return
	}

	var req DismissWarningRequest
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

	err = h.importService.DismissWarning(r.Context(), id, userID, req.Note)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "warning not found")
			return
		}
		response.InternalError(w, "failed to dismiss warning")
		return
	}

	response.NoContent(w)
}
