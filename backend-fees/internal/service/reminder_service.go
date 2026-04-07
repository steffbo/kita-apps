package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

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
	HouseholdName string
	Recipients    []string
	Subject       string
	Body          string
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
	IsEnabled() bool
}

// ReminderService handles scheduled payment reminders.
type ReminderService struct {
	feeRepo       repository.FeeRepository
	childRepo     repository.ChildRepository
	householdRepo repository.HouseholdRepository
	settingsRepo  repository.SettingsRepository
	emailLogRepo  repository.EmailLogRepository
	emailSender   ReminderEmailSender
	now           func() time.Time
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

// Run executes reminder logic for the given date and stage.
func (s *ReminderService) Run(ctx context.Context, runDate time.Time, stage ReminderStage, sentBy *uuid.UUID, dryRun bool) (*ReminderRunResult, error) {
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
		UnpaidCount: len(fees),
		DryRun:      dryRun,
	}

	if len(fees) == 0 {
		result.Message = "no unpaid fees for this period"
		return result, nil
	}

	items, children, err := s.buildItemsWithChildren(ctx, fees)
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

	// Group items by household
	householdGroups, err := s.groupByHousehold(items, children)
	if err != nil {
		return nil, err
	}

	result.FamiliesProcessed = len(householdGroups)

	for _, group := range householdGroups {
		parents, err := s.householdRepo.GetParents(ctx, group.householdID)
		if err != nil {
			return nil, err
		}

		recipients := collectEmails(parents)
		if len(recipients) == 0 {
			log.Warn().Str("household", group.householdName).Msg("No valid parent emails, skipping family")
			result.FamiliesSkippedNoEmail++
			result.Warnings = append(result.Warnings, ReminderWarning{
				HouseholdName: group.householdName,
				Reason:        "keine gültige E-Mail-Adresse",
			})
			continue
		}

		firstNames := parentFirstNames(parents)
		subject, body := buildFamilyReminderEmail(stage, runDate, firstNames, group.items)

		if dryRun {
			result.Previews = append(result.Previews, ReminderPreview{
				HouseholdName: group.householdName,
				Recipients:    recipients,
				Subject:       subject,
				Body:          body,
			})
			result.FamiliesEmailed++
			continue
		}

		if s.emailSender == nil || !s.emailSender.IsEnabled() {
			result.Message = "email service disabled"
			return result, nil
		}

		if err := s.emailSender.SendTextEmailMulti(recipients, subject, body); err != nil {
			return nil, err
		}

		toEmail := strings.Join(recipients, ", ")
		if err := s.logEmail(ctx, stage, runDate, toEmail, subject, body, group.items, result.RemindersCreated, sentBy); err != nil {
			return nil, err
		}

		result.FamiliesEmailed++
		result.EmailSent = true
	}

	if dryRun {
		result.Message = "dry run: no emails sent and no reminders created"
	}

	return result, nil
}

// householdGroup holds items grouped under a single household.
type householdGroup struct {
	householdID   uuid.UUID
	householdName string
	items         []reminderItem
}

// groupByHousehold groups reminder items by the household of their child.
// Children without a household are logged and skipped.
func (s *ReminderService) groupByHousehold(items []reminderItem, children map[uuid.UUID]*domain.Child) ([]householdGroup, error) {
	groupMap := make(map[uuid.UUID]*householdGroup)
	var order []uuid.UUID

	for _, item := range items {
		child, ok := children[item.ChildID]
		if !ok || child == nil || child.HouseholdID == nil {
			log.Error().Str("childName", item.ChildName).Msg("Child has no household, skipping")
			continue
		}
		hid := *child.HouseholdID
		if _, exists := groupMap[hid]; !exists {
			householdName := child.LastName
			groupMap[hid] = &householdGroup{
				householdID:   hid,
				householdName: householdName,
			}
			order = append(order, hid)
		}
		groupMap[hid].items = append(groupMap[hid].items, item)
	}

	result := make([]householdGroup, 0, len(order))
	for _, hid := range order {
		result = append(result, *groupMap[hid])
	}
	return result, nil
}

// collectEmails returns deduplicated non-empty parent email addresses.
func collectEmails(parents []domain.Parent) []string {
	seen := make(map[string]bool)
	var result []string
	for _, p := range parents {
		if p.Email == nil || *p.Email == "" {
			continue
		}
		e := *p.Email
		if !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}
	return result
}

// parentFirstNames returns the first names of all parents.
func parentFirstNames(parents []domain.Parent) []string {
	names := make([]string, 0, len(parents))
	for _, p := range parents {
		if p.FirstName != "" {
			names = append(names, p.FirstName)
		}
	}
	return names
}

type reminderItem struct {
	FeeID        uuid.UUID
	ChildID      uuid.UUID
	ChildName    string
	MemberNumber string
	FeeType      domain.FeeType
	Amount       float64
	Year         int
	Month        int
	DueDate      time.Time
}

func (s *ReminderService) buildItemsWithChildren(ctx context.Context, fees []domain.FeeExpectation) ([]reminderItem, map[uuid.UUID]*domain.Child, error) {
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
		return nil, nil, err
	}

	items := make([]reminderItem, 0, len(fees))
	for _, fee := range fees {
		childName := "Unbekanntes Kind"
		memberNumber := ""
		if child, ok := children[fee.ChildID]; ok && child != nil {
			childName = child.FirstName
			memberNumber = child.MemberNumber
		}
		month := 0
		if fee.Month != nil {
			month = *fee.Month
		}
		items = append(items, reminderItem{
			FeeID:        fee.ID,
			ChildID:      fee.ChildID,
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

	return items, children, nil
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

// buildFamilyReminderEmail builds a parent-facing email for a single family.
func buildFamilyReminderEmail(stage ReminderStage, runDate time.Time, parentFirstNames []string, items []reminderItem) (string, string) {
	monthName := germanMonthName(int(runDate.Month()))
	year := runDate.Year()

	var subject string
	if stage == ReminderStageFinal {
		subject = fmt.Sprintf("Kita Mahnung %s %d", monthName, year)
	} else {
		subject = fmt.Sprintf("Kita Zahlungserinnerung %s %d", monthName, year)
	}

	greeting := "Hallo"
	if len(parentFirstNames) > 0 {
		greeting = "Hallo " + strings.Join(parentFirstNames, " und ")
	}

	// Deadline: 10th of the current month
	deadline := time.Date(runDate.Year(), runDate.Month(), 10, 0, 0, 0, 0, time.UTC)
	deadlineStr := deadline.Format("02.01.2006")

	var builder strings.Builder
	builder.WriteString(greeting + ",\n\n")

	if len(items) == 1 {
		item := items[0]
		feeLabel := feeTypeLabel(item.FeeType)
		itemMonth := germanMonthName(item.Month)
		amount := formatCurrencyEUR(item.Amount)
		memberHint := ""
		if item.MemberNumber != "" {
			memberHint = fmt.Sprintf(" (Mitgliedsnr. %s)", item.MemberNumber)
		}
		builder.WriteString(fmt.Sprintf("für %s%s ist noch ein Beitrag offen:\n\n", item.ChildName, memberHint))
		builder.WriteString(fmt.Sprintf("%s %s/%d — %s\n", feeLabel, itemMonth, item.Year, amount))
		builder.WriteString(fmt.Sprintf("\nBitte überweist den offenen Beitrag von %s bis zum %s auf folgendes Konto:\n\n", amount, deadlineStr))
	} else {
		builder.WriteString("für eure Familie sind noch folgende Beiträge offen:\n\n")
		for _, item := range items {
			itemMonth := germanMonthName(item.Month)
			feeLabel := feeTypeLabel(item.FeeType)
			amount := formatCurrencyEUR(item.Amount)
			memberHint := ""
			if item.MemberNumber != "" {
				memberHint = fmt.Sprintf(" (Mitgliedsnr. %s)", item.MemberNumber)
			}
			builder.WriteString(fmt.Sprintf("- %s%s: %s %s/%d — %s\n",
				item.ChildName,
				memberHint,
				feeLabel,
				itemMonth,
				item.Year,
				amount,
			))
		}
		builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Beiträge jeweils einzeln bis zum %s auf folgendes Konto:\n\n", deadlineStr))
	}

	builder.WriteString("Empfänger: Knirpsenstadt e.V.\n")
	builder.WriteString("IBAN: DE33 3702 0500 0003 3214 00\n")
	builder.WriteString("BIC: BFSWDE33XXX\n\n")
	builder.WriteString("Wichtig: Bitte gebt als Empfänger genau \"Knirpsenstadt e.V.\" an, damit das Matching bei eurer Bank korrekt funktioniert.\n\n")
	builder.WriteString(fmt.Sprintf("Falls die Zahlung bis zum %s nicht eingegangen ist, wird leider automatisch eine Mahngebühr fällig.\n\n", deadlineStr))
	builder.WriteString("Vielen Dank!\n\n")
	builder.WriteString("---\n")
	builder.WriteString("Diese E-Mail wurde automatisch erstellt. Fehler sind nicht ausgeschlossen — bei Fragen wendet euch gerne direkt an uns.\n")

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
	toEmail string,
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
		ToEmail:   toEmail,
		Subject:   subject,
		Body:      &bodyCopy,
		EmailType: emailType,
		Payload:   &rawPayload,
		SentBy:    sentBy,
	})
}
