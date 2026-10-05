package handler

import (
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

type ReminderPaymentSettingsPayload struct {
	RecipientName string `json:"recipientName" example:"Knirpsenstadt e.V."`
	IBAN          string `json:"iban" example:"DE33370205000003321400"`
	BIC           string `json:"bic,omitempty" example:"BFSWDE33XXX" binding:"optional"`
}

// ReminderSettingsResponse represents reminder settings.
// @Description Reminder settings (payment data for mails and QR codes)
type ReminderSettingsResponse struct {
	Payment ReminderPaymentSettingsPayload `json:"payment"`
} //@name ReminderSettingsResponse

// UpdateReminderSettingsRequest represents request body for reminder settings.
// @Description Reminder settings update
type UpdateReminderSettingsRequest struct {
	Payment *ReminderPaymentSettingsPayload `json:"payment,omitempty" binding:"optional"`
} //@name UpdateReminderSettingsRequest

// GetReminderSettings handles GET /fees/reminders/settings
// @Summary Get reminder settings
// @Description Returns reminder settings
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ReminderSettingsResponse "Reminder settings"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/reminders/settings [get]
func (h *FeeHandler) GetReminderSettings(w http.ResponseWriter, r *http.Request) {
	paymentSettings, err := h.reminderService.GetPaymentSettings(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load reminder settings")
		return
	}

	response.Success(w, ReminderSettingsResponse{
		Payment: ReminderPaymentSettingsPayload{
			RecipientName: paymentSettings.RecipientName,
			IBAN:          paymentSettings.IBAN,
			BIC:           paymentSettings.BIC,
		},
	})
}

// UpdateReminderSettings handles PUT /fees/reminders/settings
// @Summary Update reminder settings
// @Description Updates reminder settings
// @Tags Fees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateReminderSettingsRequest true "Reminder settings"
// @Success 200 {object} ReminderSettingsResponse "Updated reminder settings"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/reminders/settings [put]
func (h *FeeHandler) UpdateReminderSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateReminderSettingsRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Payment != nil {
		err := h.reminderService.SetPaymentSettings(r.Context(), service.ReminderPaymentSettings{
			RecipientName: req.Payment.RecipientName,
			IBAN:          req.Payment.IBAN,
			BIC:           req.Payment.BIC,
		})
		if err != nil {
			response.InternalError(w, "failed to update reminder settings")
			return
		}
	}

	paymentSettings, err := h.reminderService.GetPaymentSettings(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load reminder settings")
		return
	}

	response.Success(w, ReminderSettingsResponse{
		Payment: ReminderPaymentSettingsPayload{
			RecipientName: paymentSettings.RecipientName,
			IBAN:          paymentSettings.IBAN,
			BIC:           paymentSettings.BIC,
		},
	})
}
