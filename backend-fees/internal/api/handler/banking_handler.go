package handler

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/banking"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/banking/encrypt"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// BankingHandler handles banking synchronization-related requests.
type BankingHandler struct {
	configRepo repository.BankingConfigRepository
	bankingSvc *banking.Service
	encryptor  *encrypt.Encryptor
}

// NewBankingHandler creates a new banking handler.
func NewBankingHandler(
	configRepo repository.BankingConfigRepository,
	bankingSvc *banking.Service,
	encryptor *encrypt.Encryptor,
) *BankingHandler {
	return &BankingHandler{
		configRepo: configRepo,
		bankingSvc: bankingSvc,
		encryptor:  encryptor,
	}
}

// BankingConfigRequest represents a request to configure banking settings.
type BankingConfigRequest struct {
	BankName      string `json:"bankName" example:"SozialBank"`
	BankBLZ       string `json:"bankBlz" example:"37020500"`
	UserID        string `json:"userId" example:"DE123456789"`
	AccountNumber string `json:"accountNumber" example:"1234567890"`
	PIN           string `json:"pin" example:"12345"`
	FinTSURL      string `json:"fintsUrl" example:"https://fints.sozialbank.com/fints"`
	SyncEnabled   bool   `json:"syncEnabled" example:"true"`
}

// BankingConfigResponse represents the banking configuration response.
type BankingConfigResponse struct {
	ID            string  `json:"id,omitempty"`
	BankName      string  `json:"bankName"`
	BankBLZ       string  `json:"bankBlz"`
	UserID        string  `json:"userId"`
	AccountNumber string  `json:"accountNumber"`
	FinTSURL      string  `json:"fintsUrl"`
	LastSyncAt    *string `json:"lastSyncAt,omitempty"`
	SyncEnabled   bool    `json:"syncEnabled"`
	IsConfigured  bool    `json:"isConfigured"`
}

// SyncResponse represents the result of a sync operation.
type SyncResponse struct {
	Success              bool     `json:"success"`
	TransactionsFetched  int      `json:"transactionsFetched"`
	TransactionsImported int      `json:"transactionsImported"`
	TransactionsSkipped  int      `json:"transactionsSkipped"`
	Errors               []string `json:"errors,omitempty"`
	LastSyncAt           string   `json:"lastSyncAt"`
}

// RegisterRoutes registers all banking routes on the given router.
// The authMiddleware should be created using middleware.AuthMiddleware(jwtService).
func (h *BankingHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/banking", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/config", h.GetConfig)
		r.Post("/config", h.SaveConfig)
		r.Delete("/config", h.DeleteConfig)

		r.Get("/sync/status", h.GetSyncStatus)
		r.Post("/sync", h.TriggerSync)
		r.Post("/test-connection", h.TestConnection)
	})
}

// GetConfig godoc
// @Summary Get banking configuration
// @Description Returns the current banking configuration (PIN is not included)
// @Tags banking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BankingConfigResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /banking/config [get]
func (h *BankingHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := h.configRepo.Get(ctx)
	if err != nil {
		response.NotFound(w, "Banking configuration not found")
		return
	}

	resp := BankingConfigResponse{
		ID:            config.ID.String(),
		BankName:      config.BankName,
		BankBLZ:       config.BankBLZ,
		UserID:        config.UserID,
		AccountNumber: config.AccountNumber,
		FinTSURL:      config.FinTSURL,
		SyncEnabled:   config.SyncEnabled,
		IsConfigured:  config.IsConfigured(),
	}

	if config.LastSyncAt != nil {
		formatted := config.LastSyncAt.Format("2006-01-02 15:04:05")
		resp.LastSyncAt = &formatted
	}

	response.JSON(w, http.StatusOK, resp)
}

// SaveConfig godoc
// @Summary Save or update banking configuration
// @Description Creates or updates the banking configuration. PIN is encrypted before storage.
// @Tags banking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param config body BankingConfigRequest true "Banking configuration"
// @Success 200 {object} BankingConfigResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /banking/config [post]
func (h *BankingHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BankingConfigRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Validate required fields
	if req.BankBLZ == "" || req.UserID == "" || req.PIN == "" || req.FinTSURL == "" {
		response.BadRequest(w, "Missing required fields: bankBlz, userId, pin, fintsUrl")
		return
	}

	// Encrypt PIN
	encryptedPIN, err := h.encryptor.Encrypt(req.PIN)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to encrypt PIN")
		return
	}

	// Check if config already exists
	existingConfig, err := h.configRepo.Get(ctx)
	if err == nil && existingConfig != nil {
		// Update existing config
		existingConfig.BankName = req.BankName
		existingConfig.BankBLZ = req.BankBLZ
		existingConfig.UserID = req.UserID
		existingConfig.AccountNumber = req.AccountNumber
		existingConfig.EncryptedPIN = encryptedPIN
		existingConfig.FinTSURL = req.FinTSURL
		existingConfig.SyncEnabled = req.SyncEnabled

		if err := h.configRepo.Update(ctx, existingConfig); err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update configuration")
			return
		}

		response.JSON(w, http.StatusOK, BankingConfigResponse{
			ID:            existingConfig.ID.String(),
			BankName:      existingConfig.BankName,
			BankBLZ:       existingConfig.BankBLZ,
			UserID:        existingConfig.UserID,
			AccountNumber: existingConfig.AccountNumber,
			FinTSURL:      existingConfig.FinTSURL,
			SyncEnabled:   existingConfig.SyncEnabled,
			IsConfigured:  existingConfig.IsConfigured(),
		})
		return
	}

	// Create new config
	newConfig := &domain.BankingConfig{
		ID:            uuid.New(),
		BankName:      req.BankName,
		BankBLZ:       req.BankBLZ,
		UserID:        req.UserID,
		AccountNumber: req.AccountNumber,
		EncryptedPIN:  encryptedPIN,
		FinTSURL:      req.FinTSURL,
		SyncEnabled:   req.SyncEnabled,
	}

	if err := h.configRepo.Create(ctx, newConfig); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create configuration")
		return
	}

	response.JSON(w, http.StatusCreated, BankingConfigResponse{
		ID:            newConfig.ID.String(),
		BankName:      newConfig.BankName,
		BankBLZ:       newConfig.BankBLZ,
		UserID:        newConfig.UserID,
		AccountNumber: newConfig.AccountNumber,
		FinTSURL:      newConfig.FinTSURL,
		SyncEnabled:   newConfig.SyncEnabled,
		IsConfigured:  newConfig.IsConfigured(),
	})
}

// DeleteConfig godoc
// @Summary Delete banking configuration
// @Description Removes the banking configuration and stops automatic sync
// @Tags banking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} response.ErrorResponse
// @Router /banking/config [delete]
func (h *BankingHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.configRepo.Delete(ctx); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete configuration")
		return
	}

	response.NoContent(w)
}

// GetSyncStatus godoc
// @Summary Get synchronization status
// @Description Returns the current sync status including last sync time and transaction count
// @Tags banking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.SyncStatus
// @Failure 401 {object} response.ErrorResponse
// @Router /banking/sync/status [get]
func (h *BankingHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.bankingSvc.GetStatus(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get sync status")
		return
	}

	response.JSON(w, http.StatusOK, status)
}

// TriggerSync godoc
// @Summary Trigger manual synchronization
// @Description Starts a manual sync with the bank. This endpoint can be called via cron job.
// @Tags banking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SyncResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /banking/sync [post]
func (h *BankingHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	result, err := h.bankingSvc.Sync(ctx)
	if err != nil {
		// Check if it's a cron job request (has special header or token)
		cronToken := r.Header.Get("X-Cron-Token")
		envCronToken := os.Getenv("CRON_API_TOKEN")

		if cronToken != "" && cronToken == envCronToken {
			// For cron requests, return 200 even on error so the cron doesn't retry
			response.JSON(w, http.StatusOK, SyncResponse{
				Success:    false,
				Errors:     append(result.Errors, err.Error()),
				LastSyncAt: result.LastSyncAt.Format("2006-01-02T15:04:05Z"),
			})
			return
		}

		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, SyncResponse{
		Success:              true,
		TransactionsFetched:  result.TransactionsFetched,
		TransactionsImported: result.TransactionsImported,
		TransactionsSkipped:  result.TransactionsSkipped,
		Errors:               result.Errors,
		LastSyncAt:           result.LastSyncAt.Format("2006-01-02T15:04:05Z"),
	})
}

// TestConnection godoc
// @Summary Test bank connection
// @Description Tests the connection to the bank without importing any data
// @Tags banking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /banking/test-connection [post]
func (h *BankingHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.bankingSvc.TestConnection(ctx); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Connection to bank successful",
	})
}
