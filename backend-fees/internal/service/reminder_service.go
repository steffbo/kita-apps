package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// ReminderRunResult holds the outcome of a reminder run.
type ReminderRunResult struct {
	Stage            ReminderStage
	Date             time.Time
	Recipient        string
	UnpaidCount      int
	RemindersCreated int
	EmailSent        bool
	DryRun           bool
	Message          string
}

// ReminderEmailSender defines the required email behavior.
type ReminderEmailSender interface {
	SendTextEmail(to, subject, body string) error
	IsEnabled() bool
}

// ReminderService handles scheduled payment reminders.
type ReminderService struct {
	feeRepo      repository.FeeRepository
	childRepo    repository.ChildRepository
	settingsRepo repository.SettingsRepository
	emailLogRepo repository.EmailLogRepository
	emailSender  ReminderEmailSender
	now          func() time.Time
}

// NewReminderService creates a new reminder service.
func NewReminderService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	settingsRepo repository.SettingsRepository,
	emailLogRepo repository.EmailLogRepository,
	emailSender ReminderEmailSender,
) *ReminderService {
	return &ReminderService{
		feeRepo:      feeRepo,
		childRepo:    childRepo,
		settingsRepo: settingsRepo,
		emailLogRepo: emailLogRepo,
		emailSender:  emailSender,
		now:          time.Now,
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

// Run executes reminder logic for the given date and stage.
func (s *ReminderService) Run(ctx context.Context, runDate time.Time, stage ReminderStage, recipient string, sentBy *uuid.UUID, dryRun bool) (*ReminderRunResult, error) {
	if recipient == "" {
		return nil, ErrInvalidInput
	}

	if stage == ReminderStageAuto {
		autoEnabled, err := s.GetAutoEnabled(ctx)
		if err != nil {
			return nil, err
		}
		if !autoEnabled {
			return &ReminderRunResult{
				Stage:     ReminderStageNone,
				Date:      runDate,
				Recipient: recipient,
				DryRun:    dryRun,
				Message:   "auto reminders disabled",
			}, nil
		}
		stage = stageFromDate(runDate)
		if stage == ReminderStageNone {
			return &ReminderRunResult{
				Stage:     ReminderStageNone,
				Date:      runDate,
				Recipient: recipient,
				DryRun:    dryRun,
				Message:   "no reminder stage for this date",
			}, nil
		}
	}

	if stage != ReminderStageInitial && stage != ReminderStageFinal {
		return nil, ErrInvalidInput
	}

	year := runDate.Year()
	month := int(runDate.Month())
	feeTypes := []domain.FeeType{domain.FeeTypeFood, domain.FeeTypeChildcare}

	fees, err := s.feeRepo.ListUnpaidByMonthAndTypes(ctx, year, month, feeTypes)
	if err != nil {
		return nil, err
	}

	result := &ReminderRunResult{
		Stage:       stage,
		Date:        runDate,
		Recipient:   recipient,
		UnpaidCount: len(fees),
		DryRun:      dryRun,
	}

	if len(fees) == 0 {
		result.Message = "no unpaid fees for this period"
		return result, nil
	}

	items, err := s.buildItems(ctx, fees)
	if err != nil {
		return nil, err
	}

	if stage == ReminderStageFinal {
		toRemind, err := s.feeRepo.ListUnpaidWithoutReminderByMonthAndTypes(ctx, year, month, feeTypes)
		if err != nil {
			return nil, err
		}
		result.RemindersCreated = len(toRemind)
		if !dryRun {
			dueDate := reminderDueDate(runDate)
			createdAt := s.now().UTC()
			for _, fee := range toRemind {
				reminder := &domain.FeeExpectation{
					ID:            uuid.New(),
					ChildID:       fee.ChildID,
					FeeType:       domain.FeeTypeReminder,
					Year:          createdAt.Year(),
					Month:         nil,
					Amount:        domain.ReminderFeeAmount,
					DueDate:       dueDate,
					CreatedAt:     createdAt,
					ReminderForID: &fee.ID,
				}
				if err := s.feeRepo.Create(ctx, reminder); err != nil {
					return nil, err
				}
			}
		}
	}

	if dryRun {
		result.Message = "dry run: no emails sent and no reminders created"
		return result, nil
	}

	if s.emailSender == nil || !s.emailSender.IsEnabled() {
		result.Message = "email service disabled"
		return result, nil
	}

	subject, body := buildReminderEmail(stage, runDate, items, result.RemindersCreated)
	if err := s.emailSender.SendTextEmail(recipient, subject, body); err != nil {
		return nil, err
	}
	if err := s.logEmail(ctx, stage, runDate, recipient, subject, body, items, result.RemindersCreated, sentBy); err != nil {
		return nil, err
	}
	result.EmailSent = true
	return result, nil
}

type reminderItem struct {
	FeeID        uuid.UUID
	ChildName    string
	MemberNumber string
	FeeType      domain.FeeType
	Amount       float64
	Year         int
	Month        int
	DueDate      time.Time
}

func (s *ReminderService) buildItems(ctx context.Context, fees []domain.FeeExpectation) ([]reminderItem, error) {
	childIDs := make([]uuid.UUID, 0, len(fees))
	seen := make(map[uuid.UUID]bool, len(fees))
	for _, fee := range fees {
		if !seen[fee.ChildID] {
			seen[fee.ChildID] = true
			childIDs = append(childIDs, fee.ChildID)
		}
	}

	children, err := s.childRepo.GetByIDs(ctx, childIDs)
	if err != nil {
		return nil, err
	}

	items := make([]reminderItem, 0, len(fees))
	for _, fee := range fees {
		childName := "Unbekanntes Kind"
		memberNumber := ""
		if child, ok := children[fee.ChildID]; ok && child != nil {
			childName = child.FullName()
			memberNumber = child.MemberNumber
		}
		month := 0
		if fee.Month != nil {
			month = *fee.Month
		}
		items = append(items, reminderItem{
			FeeID:        fee.ID,
			ChildName:    childName,
			MemberNumber: memberNumber,
			FeeType:      fee.FeeType,
			Amount:       fee.Amount,
			Year:         fee.Year,
			Month:        month,
			DueDate:      fee.DueDate,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ChildName == items[j].ChildName {
			if items[i].Year == items[j].Year {
				return items[i].Month < items[j].Month
			}
			return items[i].Year < items[j].Year
		}
		return items[i].ChildName < items[j].ChildName
	})

	return items, nil
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

func buildReminderEmail(stage ReminderStage, runDate time.Time, items []reminderItem, remindersCreated int) (string, string) {
	monthName := germanMonthName(int(runDate.Month()))
	year := runDate.Year()
	dateLabel := runDate.Format("02.01.2006")
	var subject string

	if stage == ReminderStageFinal {
		subject = fmt.Sprintf("Mahnung Essens- und Platzgeld %s %d", monthName, year)
	} else {
		subject = fmt.Sprintf("Zahlungserinnerung Essens- und Platzgeld %s %d", monthName, year)
	}

	var builder strings.Builder
	builder.WriteString("Hallo,\n\n")
	if stage == ReminderStageFinal {
		builder.WriteString(fmt.Sprintf("bis zum %s sind folgende Essens- oder Platzgelder weiterhin offen.\n", dateLabel))
		builder.WriteString(fmt.Sprintf("Es wurde eine Mahngebuehr erstellt und eine Frist bis zum %s gesetzt.\n\n", reminderDueDate(runDate).Format("02.01.2006")))
		builder.WriteString(fmt.Sprintf("Erstellte Mahngebuehren: %d\n\n", remindersCreated))
	} else {
		builder.WriteString(fmt.Sprintf("bis zum %s sind folgende Essens- oder Platzgelder noch nicht als bezahlt markiert:\n\n", dateLabel))
	}

	for _, item := range items {
		itemMonth := germanMonthName(item.Month)
		feeLabel := feeTypeLabel(item.FeeType)
		amount := formatCurrencyEUR(item.Amount)
		member := ""
		if item.MemberNumber != "" {
			member = fmt.Sprintf(" (%s)", item.MemberNumber)
		}
		builder.WriteString(fmt.Sprintf("- %s%s - %s %s %d: %s (faellig %s)\n",
			item.ChildName,
			member,
			feeLabel,
			itemMonth,
			item.Year,
			amount,
			item.DueDate.Format("02.01.2006"),
		))
	}

	builder.WriteString("\nViele Gruesse\nKnirpsenstadt Beitraege\n")
	return subject, builder.String()
}

func feeTypeLabel(feeType domain.FeeType) string {
	switch feeType {
	case domain.FeeTypeFood:
		return "Essensgeld"
	case domain.FeeTypeChildcare:
		return "Platzgeld"
	default:
		return string(feeType)
	}
}

func germanMonthName(month int) string {
	switch month {
	case 1:
		return "Januar"
	case 2:
		return "Februar"
	case 3:
		return "Maerz"
	case 4:
		return "April"
	case 5:
		return "Mai"
	case 6:
		return "Juni"
	case 7:
		return "Juli"
	case 8:
		return "August"
	case 9:
		return "September"
	case 10:
		return "Oktober"
	case 11:
		return "November"
	case 12:
		return "Dezember"
	default:
		return ""
	}
}

func formatCurrencyEUR(amount float64) string {
	value := fmt.Sprintf("%.2f", amount)
	value = strings.ReplaceAll(value, ".", ",")
	return value + " EUR"
}

func (s *ReminderService) GetAutoEnabled(ctx context.Context) (bool, error) {
	if s.settingsRepo == nil {
		return false, nil
	}
	setting, err := s.settingsRepo.Get(ctx, "reminder_auto_enabled")
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
		Key:   "reminder_auto_enabled",
		Value: value,
	})
}

func (s *ReminderService) logEmail(
	ctx context.Context,
	stage ReminderStage,
	runDate time.Time,
	recipient string,
	subject string,
	body string,
	items []reminderItem,
	remindersCreated int,
	sentBy *uuid.UUID,
) error {
	if s.emailLogRepo == nil {
		return nil
	}

	feeIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		feeIDs = append(feeIDs, item.FeeID)
	}

	payload := struct {
		Stage            ReminderStage `json:"stage"`
		RunDate          string        `json:"runDate"`
		UnpaidCount      int           `json:"unpaidCount"`
		RemindersCreated int           `json:"remindersCreated"`
		FeeIDs           []uuid.UUID   `json:"feeIds"`
	}{
		Stage:            stage,
		RunDate:          runDate.Format("2006-01-02"),
		UnpaidCount:      len(items),
		RemindersCreated: remindersCreated,
		FeeIDs:           feeIDs,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rawPayload := json.RawMessage(payloadBytes)

	bodyCopy := body
	emailType := domain.EmailLogTypeReminderInitial
	if stage == ReminderStageFinal {
		emailType = domain.EmailLogTypeReminderFinal
	}

	return s.emailLogRepo.Create(ctx, &domain.EmailLog{
		ID:        uuid.New(),
		SentAt:    s.now().UTC(),
		ToEmail:   recipient,
		Subject:   subject,
		Body:      &bodyCopy,
		EmailType: emailType,
		Payload:   &rawPayload,
		SentBy:    sentBy,
	})
}
