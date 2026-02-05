package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EmailLogType represents the kind of email that was sent.
type EmailLogType string

const (
	EmailLogTypeReminderInitial EmailLogType = "REMINDER_INITIAL"
	EmailLogTypeReminderFinal   EmailLogType = "REMINDER_FINAL"
	EmailLogTypePasswordReset   EmailLogType = "PASSWORD_RESET"
)

// EmailLog represents a sent email entry.
type EmailLog struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	SentAt    time.Time        `json:"sentAt" db:"sent_at"`
	ToEmail   string           `json:"toEmail" db:"to_email"`
	Subject   string           `json:"subject" db:"subject"`
	Body      *string          `json:"body,omitempty" db:"body"`
	EmailType EmailLogType     `json:"emailType" db:"email_type"`
	Payload   *json.RawMessage `json:"payload,omitempty" db:"payload"`
	SentBy    *uuid.UUID       `json:"sentBy,omitempty" db:"sent_by"`
}
