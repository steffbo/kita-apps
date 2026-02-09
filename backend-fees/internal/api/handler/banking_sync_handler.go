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

// Run handles POST /banking-sync/run
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
