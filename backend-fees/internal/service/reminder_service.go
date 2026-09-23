package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ReminderStage defines which reminder phase to run.
type ReminderStage string

const (
	ReminderStageAuto    ReminderStage = "auto"
	ReminderStageInitial ReminderStage = "initial"
	ReminderStageFinal   ReminderStage = "final"
	ReminderStageNone    ReminderStage = "none"
)

// ReminderWarning describes a family that was skipped.
type ReminderWarning struct {
	HouseholdName string
	Reason        string
}

// ReminderPreview holds the preview data for a single family email.
type ReminderPreview struct {
	HouseholdID    string
	HouseholdName  string
	Recipients     []string
	Subject        string
	Body           string
	QRImageDataURL *string
	QRPayload      string
}

// ReminderOverride replaces the generated subject and/or body for a household.
// Empty fields keep the generated value.
type ReminderOverride struct {
	Subject string
	Body    string
}

// ReminderRunOptions carries optional per-run behaviour: disabling the QR code
// attachment and overriding generated email content per household.
type ReminderRunOptions struct {
	IncludeQR *bool
	Overrides map[uuid.UUID]ReminderOverride
}

// ReminderRunResult holds the outcome of a reminder run.
type ReminderRunResult struct {
	Stage                  ReminderStage
	Date                   time.Time
	UnpaidCount            int
	FamiliesProcessed      int
	FamiliesEmailed        int
	FamiliesSkippedNoEmail int
	RemindersCreated       int
	EmailSent              bool
	DryRun                 bool
	Message                string
	Warnings               []ReminderWarning
	Previews               []ReminderPreview

	// Kept for backward compat
	Recipient string
}

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

// ParseReminderStage parses a stage string into a ReminderStage.
func ParseReminderStage(stage string) (ReminderStage, error) {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "":
		return ReminderStageAuto, nil
	case string(ReminderStageAuto):
		return ReminderStageAuto, nil
	case string(ReminderStageInitial), "first":
		return ReminderStageInitial, nil
	case string(ReminderStageFinal), "second":
		return ReminderStageFinal, nil
	default:
		return "", ErrInvalidInput
	}
}

var reminderScopeFoodChildcare = reminderScope{
	feeTypes:           []domain.FeeType{domain.FeeTypeFood, domain.FeeTypeChildcare},
	membership:         false,
	reminderFeeAmount:  domain.ReminderFeeAmount,
	reminderFeeDueDate: reminderDueDate,
	buildEmail:         buildFamilyReminderEmail,
	initialEmailType:   domain.EmailLogTypeReminderInitial,
	finalEmailType:     domain.EmailLogTypeReminderFinal,
	noFeesMessage:      "no unpaid fees for this period",
}

var reminderScopeMembership = reminderScope{
	feeTypes:           []domain.FeeType{domain.FeeTypeMembership},
	membership:         true,
	reminderFeeAmount:  domain.MembershipReminderFeeAmount,
	reminderFeeDueDate: membershipReminderFeeDueDate,
	buildEmail:         buildFamilyMembershipReminderEmail,
	initialEmailType:   domain.EmailLogTypeMembershipReminderInitial,
	finalEmailType:     domain.EmailLogTypeMembershipReminderFinal,
	noFeesMessage:      "no unpaid membership fees for this period",
}

// Run executes reminder logic for Food/Childcare fees for the given date and stage.
// deadline overrides the payment deadline shown in the email; if nil, 7 days after the run date are used.
func (s *ReminderService) Run(ctx context.Context, runDate time.Time, stage ReminderStage, sentBy *uuid.UUID, dryRun bool, deadline *time.Time, selectedHouseholdIDs []uuid.UUID, options *ReminderRunOptions) (*ReminderRunResult, error) {
	if stage == ReminderStageAuto {
		autoEnabled, err := s.GetAutoEnabled(ctx)
		if err != nil {
			return nil, err
		}
		if !autoEnabled {
			return &ReminderRunResult{
				Stage:   ReminderStageNone,
				Date:    runDate,
				DryRun:  dryRun,
				Message: "auto reminders disabled",
			}, nil
		}
		stage = stageFromDate(runDate)
		if stage == ReminderStageNone {
			return &ReminderRunResult{
				Stage:   ReminderStageNone,
				Date:    runDate,
				DryRun:  dryRun,
				Message: "no reminder stage for this date",
			}, nil
		}
	}

	return s.runScope(ctx, reminderScopeFoodChildcare, runDate, stage, sentBy, dryRun, deadline, selectedHouseholdIDs, options)
}

// RunMembership executes reminder logic for Membership fees for the given date and stage.
// deadline overrides the payment deadline shown in the email; if nil, 7 days after the run date are used.
func (s *ReminderService) RunMembership(ctx context.Context, runDate time.Time, stage ReminderStage, sentBy *uuid.UUID, dryRun bool, deadline *time.Time, selectedHouseholdIDs []uuid.UUID, options *ReminderRunOptions) (*ReminderRunResult, error) {
	return s.runScope(ctx, reminderScopeMembership, runDate, stage, sentBy, dryRun, deadline, selectedHouseholdIDs, options)
}

func stageFromDate(date time.Time) ReminderStage {
	switch date.Day() {
	case 5:
		return ReminderStageInitial
	case 10:
		return ReminderStageFinal
	default:
		return ReminderStageNone
	}
}

func reminderDueDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 15, 23, 59, 59, 0, time.UTC)
}

func membershipReminderFeeDueDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC)
}

func (s *ReminderService) GetAutoEnabled(ctx context.Context) (bool, error) {
	if s.settingsRepo == nil {
		return false, nil
	}
	setting, err := s.settingsRepo.Get(ctx, settingReminderAutoEnabled)
	if err != nil {
		if err == repository.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return strings.ToLower(setting.Value) == "true", nil
}

func (s *ReminderService) SetAutoEnabled(ctx context.Context, enabled bool) error {
	if s.settingsRepo == nil {
		return ErrInvalidInput
	}
	value := "false"
	if enabled {
		value = "true"
	}
	return s.settingsRepo.Upsert(ctx, &domain.AppSetting{
		Key:   settingReminderAutoEnabled,
		Value: value,
	})
}
