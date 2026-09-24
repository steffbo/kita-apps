package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// EmailLogResponse represents an email log entry.
// @Description Email log entry
type EmailLogResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SentAt      string  `json:"sentAt" example:"2026-02-05T10:15:00Z"`
	ToEmail     string  `json:"toEmail" example:"admin@knirpsenstadt.de"`
	Subject     string  `json:"subject" example:"Zahlungserinnerung Essens- und Platzgeld Februar 2026"`
	Body        *string `json:"body,omitempty" example:"Hallo,..." binding:"optional"`
	EmailType   string  `json:"emailType" example:"REMINDER_INITIAL"`
	SentBy      *string `json:"sentBy,omitempty" example:"550e8400-e29b-41d4-a716-446655440001" binding:"optional"`
	HouseholdID *string `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002" binding:"optional"`
} //@name EmailLogResponse

// EmailLogListResponse represents a paginated list of email logs.
// @Description Paginated list of email logs
type EmailLogListResponse struct {
	Data       []EmailLogResponse `json:"data"`
	Total      int                `json:"total" example:"100"`
	Page       int                `json:"page" example:"1"`
	PerPage    int                `json:"perPage" example:"20"`
	TotalPages int                `json:"totalPages" example:"5"`
} //@name EmailLogListResponse

// GetEmailLogs handles GET /fees/email-logs
// @Summary List email logs
// @Description Get a filtered, sorted and paginated list of sent email logs
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param emailType query string false "Filter by email type" Enums(REMINDER_INITIAL, REMINDER_FINAL, MEMBERSHIP_REMINDER_INITIAL, MEMBERSHIP_REMINDER_FINAL, PASSWORD_RESET)
// @Param householdId query string false "Filter by household UUID (family chronology)"
// @Param search query string false "Search in recipient and subject"
// @Param sortDir query string false "Sort by sent_at direction" Enums(asc, desc) default(desc)
// @Success 200 {object} EmailLogListResponse "Paginated list of email logs"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/email-logs [get]
func (h *FeeHandler) GetEmailLogs(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	filter := service.EmailLogFilter{
		SortDir: request.GetQueryString(r, "sortDir", "desc"),
		Search:  request.GetQueryString(r, "search", ""),
	}
	if typeRaw := strings.TrimSpace(request.GetQueryString(r, "emailType", "")); typeRaw != "" {
		filter.EmailType = &typeRaw
	}
	if householdRaw := strings.TrimSpace(request.GetQueryString(r, "householdId", "")); householdRaw != "" {
		householdID, err := uuid.Parse(householdRaw)
		if err != nil {
			response.BadRequest(w, "invalid householdId format")
			return
		}
		filter.HouseholdID = &householdID
	}

	logs, total, err := h.reminderService.ListEmailLogs(r.Context(), pagination.Offset, pagination.PerPage, filter)
	if err != nil {
		response.InternalError(w, "failed to list email logs")
		return
	}

	resp := make([]EmailLogResponse, 0, len(logs))
	for _, entry := range logs {
		var sentBy *string
		if entry.SentBy != nil {
			value := entry.SentBy.String()
			sentBy = &value
		}
		var householdID *string
		if entry.HouseholdID != nil {
			value := entry.HouseholdID.String()
			householdID = &value
		}
		resp = append(resp, EmailLogResponse{
			ID:          entry.ID.String(),
			SentAt:      entry.SentAt.Format(time.RFC3339),
			ToEmail:     entry.ToEmail,
			Subject:     entry.Subject,
			Body:        entry.Body,
			EmailType:   string(entry.EmailType),
			SentBy:      sentBy,
			HouseholdID: householdID,
		})
	}

	response.Paginated(w, resp, total, pagination.Page, pagination.PerPage)
}
