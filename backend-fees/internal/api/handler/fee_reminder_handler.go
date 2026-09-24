package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// ReminderWarningResponse represents a family skipped due to missing emails.
type ReminderWarningResponse struct {
	HouseholdName string `json:"householdName" example:"Müller"`
	Reason        string `json:"reason" example:"keine gültige E-Mail-Adresse"`
} //@name ReminderWarningResponse

// ReminderPreviewResponse holds the preview for a single family email.
type ReminderPreviewResponse struct {
	HouseholdID    string   `json:"householdId" example:"550e8400-e29b-41d4-a716-446655440000"`
	HouseholdName  string   `json:"householdName" example:"Schmidt"`
	Recipients     []string `json:"recipients" example:"[\"anna@example.com\"]"`
	Subject        string   `json:"subject" example:"Kita Zahlungserinnerung April 2026"`
	Body           string   `json:"body" example:"Hallo Anna,..."`
	QRImageDataURL *string  `json:"qrImageDataUrl,omitempty" binding:"optional"`
	QRPayload      string   `json:"qrPayload,omitempty" example:"BCD\n002\n1\nSCT..." binding:"optional"`
} //@name ReminderPreviewResponse

// ReminderRunOverrideRequest replaces the generated subject and/or body for one household.
// @Description Per-household override for generated reminder emails; empty fields keep the generated value
type ReminderRunOverrideRequest struct {
	Subject string `json:"subject,omitempty" example:"Kita Zahlungserinnerung" binding:"optional"`
	Body    string `json:"body,omitempty" example:"Hallo Anna,..." binding:"optional"`
} //@name ReminderRunOverrideRequest

// ReminderRunRequestBody is the optional JSON body for reminder run endpoints.
// @Description Optional run behaviour: disable the QR code attachment and override generated email content per household
type ReminderRunRequestBody struct {
	IncludeQR *bool                                 `json:"includeQR,omitempty" example:"true" binding:"optional"`
	Overrides map[string]ReminderRunOverrideRequest `json:"overrides,omitempty" binding:"optional"`
} //@name ReminderRunRequestBody

type ReminderPaymentSettingsPayload struct {
	RecipientName string `json:"recipientName" example:"Knirpsenstadt e.V."`
	IBAN          string `json:"iban" example:"DE33370205000003321400"`
	BIC           string `json:"bic,omitempty" example:"BFSWDE33XXX" binding:"optional"`
}

// ReminderRunResponse represents the result of a reminder run.
// @Description Ergebnis einer Erinnerungs-/Mahnungsprüfung
type ReminderRunResponse struct {
	Stage                  string                    `json:"stage" example:"initial" enums:"auto,initial,final,none"`
	Date                   string                    `json:"date" example:"2026-02-05"`
	DryRun                 bool                      `json:"dryRun" example:"false"`
	UnpaidCount            int                       `json:"unpaidCount" example:"12"`
	FamiliesProcessed      int                       `json:"familiesProcessed" example:"6"`
	FamiliesEmailed        int                       `json:"familiesEmailed" example:"5"`
	FamiliesSkippedNoEmail int                       `json:"familiesSkippedNoEmail" example:"1"`
	RemindersCreated       int                       `json:"remindersCreated" example:"8"`
	EmailSent              bool                      `json:"emailSent" example:"true"`
	Warnings               []ReminderWarningResponse `json:"warnings,omitempty" binding:"optional"`
	Previews               []ReminderPreviewResponse `json:"previews,omitempty" binding:"optional"`
	Message                string                    `json:"message,omitempty" example:"no unpaid fees for this period" binding:"optional"`

	// Deprecated: kept for backward compat
	Recipient       string `json:"recipient,omitempty" binding:"optional"`
	ReminderCreated int    `json:"reminderCreated,omitempty" binding:"optional"`
} //@name ReminderRunResponse

// ReminderSettingsResponse represents reminder settings.
// @Description Reminder settings
type ReminderSettingsResponse struct {
	AutoEnabled bool                           `json:"autoEnabled" example:"false"`
	Payment     ReminderPaymentSettingsPayload `json:"payment"`
} //@name ReminderSettingsResponse

// UpdateReminderSettingsRequest represents request body for reminder settings.
// @Description Reminder settings update
type UpdateReminderSettingsRequest struct {
	AutoEnabled bool                            `json:"autoEnabled" example:"true"`
	Payment     *ReminderPaymentSettingsPayload `json:"payment,omitempty" binding:"optional"`
} //@name UpdateReminderSettingsRequest

func parseSelectedHouseholdIDs(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	var selectedHouseholdIDs []uuid.UUID
	if selectedRaw := strings.TrimSpace(request.GetQueryString(r, "selectedHouseholdIds", "")); selectedRaw != "" {
		parts := strings.Split(selectedRaw, ",")
		selectedHouseholdIDs = make([]uuid.UUID, 0, len(parts))
		for _, part := range parts {
			idStr := strings.TrimSpace(part)
			if idStr == "" {
				continue
			}
			parsed, err := uuid.Parse(idStr)
			if err != nil {
				response.BadRequest(w, "invalid selectedHouseholdIds value")
				return nil, false
			}
			selectedHouseholdIDs = append(selectedHouseholdIDs, parsed)
		}
	}
	return selectedHouseholdIDs, true
}

func reminderRunResponseFromResult(result *service.ReminderRunResult) ReminderRunResponse {
	resp := ReminderRunResponse{
		Stage:                  string(result.Stage),
		Date:                   result.Date.Format("2006-01-02"),
		DryRun:                 result.DryRun,
		UnpaidCount:            result.UnpaidCount,
		FamiliesProcessed:      result.FamiliesProcessed,
		FamiliesEmailed:        result.FamiliesEmailed,
		FamiliesSkippedNoEmail: result.FamiliesSkippedNoEmail,
		RemindersCreated:       result.RemindersCreated,
		EmailSent:              result.EmailSent,
		Message:                result.Message,
	}

	for _, warn := range result.Warnings {
		resp.Warnings = append(resp.Warnings, ReminderWarningResponse{
			HouseholdName: warn.HouseholdName,
			Reason:        warn.Reason,
		})
	}

	for _, prev := range result.Previews {
		resp.Previews = append(resp.Previews, ReminderPreviewResponse{
			HouseholdID:    prev.HouseholdID,
			HouseholdName:  prev.HouseholdName,
			Recipients:     prev.Recipients,
			Subject:        prev.Subject,
			Body:           prev.Body,
			QRImageDataURL: prev.QRImageDataURL,
			QRPayload:      prev.QRPayload,
		})
	}

	return resp
}

// parseReminderRunOptions reads the optional JSON body of reminder run
// endpoints. An empty body yields nil options.
func parseReminderRunOptions(r *http.Request, w http.ResponseWriter) (*service.ReminderRunOptions, bool) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		response.BadRequest(w, "failed to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(bodyBytes))) == 0 {
		return nil, true
	}

	var body ReminderRunRequestBody
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		response.BadRequest(w, "invalid request body")
		return nil, false
	}

	options := &service.ReminderRunOptions{
		IncludeQR: body.IncludeQR,
	}
	if len(body.Overrides) > 0 {
		overrides := make(map[uuid.UUID]service.ReminderOverride, len(body.Overrides))
		for householdIDRaw, override := range body.Overrides {
			householdID, err := uuid.Parse(householdIDRaw)
			if err != nil {
				response.BadRequest(w, "invalid overrides key (expected household UUID)")
				return nil, false
			}
			overrides[householdID] = service.ReminderOverride{
				Subject: override.Subject,
				Body:    override.Body,
			}
		}
		options.Overrides = overrides
	}
	return options, true
}

// RunReminders handles POST /fees/reminders/run
// @Summary Run payment reminder checks
// @Description Sends reminder emails for unpaid Food/Childcare fees and optionally creates reminder fees
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param date query string false "Run date (YYYY-MM-DD, defaults to today)"
// @Param stage query string false "Stage: auto, initial, final" Enums(auto, initial, final)
// @Param dryRun query bool false "If true, don't send emails or create reminders"
// @Param selectedHouseholdIds query string false "Optional comma-separated household IDs to process"
// @Success 200 {object} ReminderRunResponse "Reminder run result"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/reminders/run [post]
func (h *FeeHandler) RunReminders(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	runDate := util.Now()
	if dateStr := request.GetQueryString(r, "date", ""); dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(w, "invalid date format (expected YYYY-MM-DD)")
			return
		}
		runDate = parsed
	}

	stageRaw := strings.ToLower(request.GetQueryString(r, "stage", "auto"))
	stage, err := service.ParseReminderStage(stageRaw)
	if err != nil {
		response.BadRequest(w, "invalid stage (expected auto, initial, final)")
		return
	}

	dryRun := false
	if dryRunPtr := request.GetQueryBool(r, "dryRun"); dryRunPtr != nil {
		dryRun = *dryRunPtr
	}

	var deadline *time.Time
	if deadlineStr := request.GetQueryString(r, "deadline", ""); deadlineStr != "" {
		parsed, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			response.BadRequest(w, "invalid deadline format (expected YYYY-MM-DD)")
			return
		}
		deadline = &parsed
	}

	var sentBy *uuid.UUID
	if userCtx.UserID != "" {
		if parsed, err := uuid.Parse(userCtx.UserID); err == nil {
			sentBy = &parsed
		}
	}

	selectedHouseholdIDs, ok := parseSelectedHouseholdIDs(w, r)
	if !ok {
		return
	}

	runOptions, ok := parseReminderRunOptions(r, w)
	if !ok {
		return
	}

	result, err := h.reminderService.Run(r.Context(), runDate, stage, sentBy, dryRun, deadline, selectedHouseholdIDs, runOptions)
	if err != nil {
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid request")
			return
		}
		response.InternalError(w, "failed to run reminders")
		return
	}

	response.Success(w, reminderRunResponseFromResult(result))
}

// RunMembershipReminders handles POST /fees/membership-reminders/run
// @Summary Run membership reminder checks
// @Description Sends reminder emails for unpaid Membership fees and optionally creates 5 EUR reminder fees
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param date query string false "Run date (YYYY-MM-DD, defaults to today)"
// @Param stage query string false "Stage: initial, final" Enums(initial, final)
// @Param dryRun query bool false "If true, don't send emails or create reminders"
// @Param deadline query string false "Payment deadline (YYYY-MM-DD), defaults to 31.03.<year>"
// @Param selectedHouseholdIds query string false "Optional comma-separated household IDs to process"
// @Success 200 {object} ReminderRunResponse "Reminder run result"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/membership-reminders/run [post]
func (h *FeeHandler) RunMembershipReminders(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	runDate := util.Now()
	if dateStr := request.GetQueryString(r, "date", ""); dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(w, "invalid date format (expected YYYY-MM-DD)")
			return
		}
		runDate = parsed
	}

	stageRaw := strings.ToLower(strings.TrimSpace(request.GetQueryString(r, "stage", "initial")))
	stage, err := service.ParseReminderStage(stageRaw)
	if err != nil || (stage != service.ReminderStageInitial && stage != service.ReminderStageFinal) {
		response.BadRequest(w, "invalid stage (expected initial, final)")
		return
	}

	dryRun := false
	if dryRunPtr := request.GetQueryBool(r, "dryRun"); dryRunPtr != nil {
		dryRun = *dryRunPtr
	}

	var deadline *time.Time
	if deadlineStr := request.GetQueryString(r, "deadline", ""); deadlineStr != "" {
		parsed, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			response.BadRequest(w, "invalid deadline format (expected YYYY-MM-DD)")
			return
		}
		deadline = &parsed
	}

	var sentBy *uuid.UUID
	if userCtx.UserID != "" {
		if parsed, err := uuid.Parse(userCtx.UserID); err == nil {
			sentBy = &parsed
		}
	}

	selectedHouseholdIDs, ok := parseSelectedHouseholdIDs(w, r)
	if !ok {
		return
	}

	runOptions, ok := parseReminderRunOptions(r, w)
	if !ok {
		return
	}

	result, err := h.reminderService.RunMembership(r.Context(), runDate, stage, sentBy, dryRun, deadline, selectedHouseholdIDs, runOptions)
	if err != nil {
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid request")
			return
		}
		response.InternalError(w, "failed to run membership reminders")
		return
	}

	response.Success(w, reminderRunResponseFromResult(result))
}

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
	autoEnabled, err := h.reminderService.GetAutoEnabled(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load reminder settings")
		return
	}

	paymentSettings, err := h.reminderService.GetPaymentSettings(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load reminder settings")
		return
	}

	response.Success(w, ReminderSettingsResponse{
		AutoEnabled: autoEnabled,
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

	if err := h.reminderService.SetAutoEnabled(r.Context(), req.AutoEnabled); err != nil {
		response.InternalError(w, "failed to update reminder settings")
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
		AutoEnabled: req.AutoEnabled,
		Payment: ReminderPaymentSettingsPayload{
			RecipientName: paymentSettings.RecipientName,
			IBAN:          paymentSettings.IBAN,
			BIC:           paymentSettings.BIC,
		},
	})
}
