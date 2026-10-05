package service

import (
	"context"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ReminderStage is the kind of a family reminder mail: a Zahlungserinnerung
// (initial) or, when it charges Mahngebühren, a Mahnung (final).
type ReminderStage string

const (
	ReminderStageInitial ReminderStage = "initial"
	ReminderStageFinal   ReminderStage = "final"
)

// ReminderEmailSender defines the required email behavior.
type ReminderEmailSender interface {
	SendTextEmailMulti(to []string, subject, body string) error
	SendTextAndHTMLEmailMulti(to []string, subject, textBody, htmlBody string, inlineImageCID string, inlineImagePNG []byte) error
	IsEnabled() bool
}

// ReminderService is the shared reminder core for all fee-type scopes
// (Food/Childcare and Membership).
type ReminderService struct {
	feeRepo       repository.FeeRepository
	childRepo     repository.ChildRepository
	householdRepo repository.HouseholdRepository
	settingsRepo  repository.SettingsRepository
	emailLogRepo  repository.EmailLogRepository
	emailSender   ReminderEmailSender
	now           func() time.Time
}

// EmailLogFilter filters the sent-email log.
type EmailLogFilter = repository.EmailLogFilter

// ListEmailLogs returns a page of sent emails.
func (s *ReminderService) ListEmailLogs(ctx context.Context, offset, limit int, filter EmailLogFilter) ([]domain.EmailLog, int64, error) {
	return s.emailLogRepo.List(ctx, offset, limit, filter)
}

// NewReminderService creates a new reminder service.
func NewReminderService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	householdRepo repository.HouseholdRepository,
	settingsRepo repository.SettingsRepository,
	emailLogRepo repository.EmailLogRepository,
	emailSender ReminderEmailSender,
) *ReminderService {
	return &ReminderService{
		feeRepo:       feeRepo,
		childRepo:     childRepo,
		householdRepo: householdRepo,
		settingsRepo:  settingsRepo,
		emailLogRepo:  emailLogRepo,
		emailSender:   emailSender,
		now:           time.Now,
	}
}
