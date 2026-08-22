package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
)

// BankingSyncHandler proxies requests to the banking-sync runner.
type BankingSyncHandler struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewBankingSyncHandler creates a new banking sync handler.
func NewBankingSyncHandler(baseURL, token string, timeout time.Duration) *BankingSyncHandler {
	return &BankingSyncHandler{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client: &http.Client{Timeout: timeout},
	}
}

// BankingSyncStatusResponse mirrors the state object served by the banking-sync service.
// @Description Status of the banking-sync runner (proxied 1:1 from the banking-sync service)
type BankingSyncStatusResponse struct {
	Status       string      `json:"status" example:"idle" enums:"idle,running,waiting_for_2fa,success,error,cancelled"`
	RunID        *string     `json:"runId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartedAt    *string     `json:"startedAt,omitempty" example:"2024-03-15T10:30:00Z"`
	FinishedAt   *string     `json:"finishedAt,omitempty" example:"2024-03-15T10:35:00Z"`
	LastError    *string     `json:"lastError,omitempty"`
	LastMessage  *string     `json:"lastMessage,omitempty" example:"Sync abgeschlossen"`
	DownloadPath *string     `json:"downloadPath,omitempty"`
	UploadResult interface{} `json:"uploadResult,omitempty"`
	Logs         []string    `json:"logs,omitempty"`
	UpdatedAt    string      `json:"updatedAt" example:"2024-03-15T10:35:00Z"`
} //@name BankingSyncStatus

// Run handles POST /banking-sync/run
// @Summary Start a banking sync run
// @Description Triggers a sync run in the banking-sync service and returns its current state
// @Tags Banking-Sync
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BankingSyncStatusResponse "Sync started, current state returned"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 409 {object} response.ErrorBody "A sync run is already in progress"
// @Failure 503 {object} response.ErrorBody "Banking sync not configured or unreachable"
// @Router /banking-sync/run [post]
func (h *BankingSyncHandler) Run(w http.ResponseWriter, r *http.Request) {
	if !h.isConfigured() {
		response.Error(w, http.StatusServiceUnavailable, "banking sync not configured")
		return
	}

	payload, status, err := h.call(r.Context(), http.MethodPost, "/run")
	if err != nil {
		response.InternalError(w, "failed to start banking sync")
		return
	}

	if status >= 300 {
		response.Error(w, status, parseErrorMessage(payload))
		return
	}

	response.JSON(w, status, json.RawMessage(payload))
}

// Status handles GET /banking-sync/status
// @Summary Get banking sync status
// @Description Returns the current state of the banking-sync service
// @Tags Banking-Sync
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BankingSyncStatusResponse "Current sync state"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 503 {object} response.ErrorBody "Banking sync not configured or unreachable"
// @Router /banking-sync/status [get]
func (h *BankingSyncHandler) Status(w http.ResponseWriter, r *http.Request) {
	if !h.isConfigured() {
		response.Error(w, http.StatusServiceUnavailable, "banking sync not configured")
		return
	}

	payload, status, err := h.call(r.Context(), http.MethodGet, "/status")
	if err != nil {
		response.InternalError(w, "failed to fetch banking sync status")
		return
	}

	if status >= 300 {
		response.Error(w, status, parseErrorMessage(payload))
		return
	}

	response.JSON(w, status, json.RawMessage(payload))
}

// Cancel handles POST /banking-sync/cancel
// @Summary Cancel a banking sync run
// @Description Cancels the running sync in the banking-sync service
// @Tags Banking-Sync
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BankingSyncStatusResponse "Sync cancelled, resulting state returned"
// @Failure 400 {object} response.ErrorBody "No sync in progress to cancel"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 503 {object} response.ErrorBody "Banking sync not configured or unreachable"
// @Router /banking-sync/cancel [post]
func (h *BankingSyncHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if !h.isConfigured() {
		response.Error(w, http.StatusServiceUnavailable, "banking sync not configured")
		return
	}

	payload, status, err := h.call(r.Context(), http.MethodPost, "/cancel")
	if err != nil {
		response.InternalError(w, "failed to cancel banking sync")
		return
	}

	if status >= 300 {
		response.Error(w, status, parseErrorMessage(payload))
		return
	}

	response.JSON(w, status, json.RawMessage(payload))
}

func (h *BankingSyncHandler) isConfigured() bool {
	return h.baseURL != "" && h.token != ""
}

func (h *BankingSyncHandler) call(ctx context.Context, method, path string) ([]byte, int, error) {
	url := h.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Sync-Token", h.token)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(body) == 0 {
		body = []byte("null")
	}

	return body, resp.StatusCode, nil
}

func parseErrorMessage(payload []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(payload, &parsed); err == nil {
		if value, ok := parsed["message"].(string); ok && value != "" {
			return value
		}
		if value, ok := parsed["error"].(string); ok && value != "" {
			return value
		}
	}

	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || trimmed == "null" {
		return "request failed"
	}
	return trimmed
}
