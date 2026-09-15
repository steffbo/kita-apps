package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EmailLogType represents the kind of email that was sent.
type EmailLogType string

const (
	EmailLogTypeReminderInitial           EmailLogType = "REMINDER_INITIAL"
	EmailLogTypeReminderFinal             EmailLogType = "REMINDER_FINAL"
	EmailLogTypeMembershipReminderInitial EmailLogType = "MEMBERSHIP_REMINDER_INITIAL"
	EmailLogTypeMembershipReminderFinal   EmailLogType = "MEMBERSHIP_REMINDER_FINAL"
	EmailLogTypePasswordReset             EmailLogType = "PASSWORD_RESET"
)

// EmailLog represents a sent email entry.
type EmailLog struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	SentAt      time.Time        `json:"sentAt" db:"sent_at"`
	ToEmail     string           `json:"toEmail" db:"to_email"`
	Subject     string           `json:"subject" db:"subject"`
	Body        *string          `json:"body,omitempty" db:"body"`
	EmailType   EmailLogType     `json:"emailType" db:"email_type"`
	Payload     *json.RawMessage `json:"payload,omitempty" db:"payload"`
	SentBy      *uuid.UUID       `json:"sentBy,omitempty" db:"sent_by"`
	HouseholdID *uuid.UUID       `json:"householdId,omitempty" db:"household_id"`
}

// FeeIDsFromPayload extracts the feeIds array from a reminder email log
// payload. Returns an empty slice when the payload is missing or malformed.
func (l EmailLog) FeeIDsFromPayload() []uuid.UUID {
	if l.Payload == nil {
		return nil
	}
	var payload struct {
		FeeIDs []uuid.UUID `json:"feeIds"`
	}
	if err := json.Unmarshal(*l.Payload, &payload); err != nil {
		return nil
	}
	return payload.FeeIDs
}

// StageFromPayload extracts the reminder stage from a log payload.
func (l EmailLog) StageFromPayload() (string, bool) {
	if l.Payload == nil {
		return "", false
	}
	var payload struct {
		Stage string `json:"stage"`
	}
	if err := json.Unmarshal(*l.Payload, &payload); err != nil || payload.Stage == "" {
		return "", false
	}
	return payload.Stage, true
}
