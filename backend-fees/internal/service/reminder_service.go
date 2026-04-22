package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
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
	HouseholdID    string
	HouseholdName  string
	Recipients     []string
	Subject        string
	Body           string
	QRImageDataURL *string
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
// deadline overrides the payment deadline shown in the email; if nil, the 10th of the run month is used.
func (s *ReminderService) Run(ctx context.Context, runDate time.Time, stage ReminderStage, sentBy *uuid.UUID, dryRun bool, deadline *time.Time, selectedHouseholdIDs []uuid.UUID) (*ReminderRunResult, error) {
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

	fees, err := s.feeRepo.ListUnpaidUpToMonthAndTypes(ctx, year, month, feeTypes)
	if err != nil {
		return nil, err
	}

	result := &ReminderRunResult{
		Stage:  stage,
		Date:   runDate,
		DryRun: dryRun,
	}

	if len(fees) == 0 {
		result.Message = "no unpaid fees for this period"
		return result, nil
	}

	if stage == ReminderStageFinal {
		toRemind, err := s.feeRepo.ListUnpaidWithoutReminderByMonthAndTypes(ctx, year, month, feeTypes)
		if err != nil {
			return nil, err
		}
		result.RemindersCreated = len(toRemind)
		fees = append(fees, syntheticReminderFees(toRemind, reminderDueDate(runDate), s.now().UTC())...)
		if !dryRun {
			createdAt := s.now().UTC()
			for _, fee := range toRemind {
				reminder := &domain.FeeExpectation{
					ID:            uuid.New(),
					ChildID:       fee.ChildID,
					HouseholdID:   fee.HouseholdID,
					FeeType:       domain.FeeTypeReminder,
					Year:          createdAt.Year(),
					Month:         nil,
					Amount:        domain.ReminderFeeAmount,
					DueDate:       reminderDueDate(runDate),
					CreatedAt:     createdAt,
					ReminderForID: &fee.ID,
				}
				if err := s.feeRepo.Create(ctx, reminder); err != nil {
					return nil, err
				}
			}
		}
	}

	result.UnpaidCount = len(fees)

	items, children, err := s.buildItemsWithChildren(ctx, fees)
	if err != nil {
		return nil, err
	}

	// Group items by household
	householdGroups, err := s.groupByHousehold(items, children)
	if err != nil {
		return nil, err
	}
	householdGroups = filterHouseholdGroupsBySelection(householdGroups, selectedHouseholdIDs)

	result.FamiliesProcessed = len(householdGroups)
	paymentSettings, err := s.GetPaymentSettings(ctx)
	if err != nil {
		return nil, err
	}

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
		subject, body := buildFamilyReminderEmail(stage, runDate, firstNames, group.items, deadline)
		qrData, qrErr := s.buildReminderQRCode(paymentSettings, runDate, group.householdName, group.items)
		if qrErr != nil {
			log.Warn().Err(qrErr).Str("household", group.householdName).Msg("Failed to generate payment QR code, continuing without QR")
		}

		var qrImageDataURL *string
		if qrData != nil {
			qrImageDataURL = &qrData.DataURL
		}

		if dryRun {
			result.Previews = append(result.Previews, ReminderPreview{
				HouseholdID:    group.householdID.String(),
				HouseholdName:  group.householdName,
				Recipients:     recipients,
				Subject:        subject,
				Body:           body,
				QRImageDataURL: qrImageDataURL,
			})
			result.FamiliesEmailed++
			continue
		}

		if s.emailSender == nil || !s.emailSender.IsEnabled() {
			result.Message = "email service disabled"
			return result, nil
		}

		if qrData != nil {
			htmlBody := buildReminderEmailHTML(body, reminderEmailQRCodeCID)
			if err := s.emailSender.SendTextAndHTMLEmailMulti(
				recipients,
				subject,
				body,
				htmlBody,
				reminderEmailQRCodeCID,
				qrData.PNG,
			); err != nil {
				return nil, err
			}
		} else {
			if err := s.emailSender.SendTextEmailMulti(recipients, subject, body); err != nil {
				return nil, err
			}
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

func filterHouseholdGroupsBySelection(groups []householdGroup, selectedHouseholdIDs []uuid.UUID) []householdGroup {
	if len(selectedHouseholdIDs) == 0 {
		return groups
	}

	selected := make(map[uuid.UUID]struct{}, len(selectedHouseholdIDs))
	for _, householdID := range selectedHouseholdIDs {
		selected[householdID] = struct{}{}
	}

	filtered := make([]householdGroup, 0, len(groups))
	for _, group := range groups {
		if _, ok := selected[group.householdID]; ok {
			filtered = append(filtered, group)
		}
	}
	return filtered
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
	BaseFeeType  *domain.FeeType
	BaseYear     int
	BaseMonth    int
}

func (s *ReminderService) buildItemsWithChildren(ctx context.Context, fees []domain.FeeExpectation) ([]reminderItem, map[uuid.UUID]*domain.Child, error) {
	childIDs := make([]uuid.UUID, 0, len(fees))
	seen := make(map[uuid.UUID]bool, len(fees))
	reminderBaseIDs := make([]uuid.UUID, 0, len(fees))
	seenBaseIDs := make(map[uuid.UUID]bool, len(fees))
	for _, fee := range fees {
		if !seen[fee.ChildID] {
			seen[fee.ChildID] = true
			childIDs = append(childIDs, fee.ChildID)
		}
		if fee.FeeType == domain.FeeTypeReminder && fee.ReminderForID != nil && !seenBaseIDs[*fee.ReminderForID] {
			seenBaseIDs[*fee.ReminderForID] = true
			reminderBaseIDs = append(reminderBaseIDs, *fee.ReminderForID)
		}
	}

	children, err := s.childRepo.GetByIDs(ctx, childIDs)
	if err != nil {
		return nil, nil, err
	}

	baseFees := make(map[uuid.UUID]*domain.FeeExpectation)
	if len(reminderBaseIDs) > 0 {
		baseFees, err = s.feeRepo.GetByIDs(ctx, reminderBaseIDs)
		if err != nil {
			return nil, nil, err
		}
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
		item := reminderItem{
			FeeID:        fee.ID,
			ChildID:      fee.ChildID,
			ChildName:    childName,
			MemberNumber: memberNumber,
			FeeType:      fee.FeeType,
			Amount:       fee.Amount,
			Year:         fee.Year,
			Month:        month,
			DueDate:      fee.DueDate,
		}
		if fee.FeeType == domain.FeeTypeReminder && fee.ReminderForID != nil {
			if baseFee, ok := baseFees[*fee.ReminderForID]; ok && baseFee != nil {
				baseFeeType := baseFee.FeeType
				item.BaseFeeType = &baseFeeType
				item.BaseYear = baseFee.Year
				if baseFee.Month != nil {
					item.BaseMonth = *baseFee.Month
				}
			}
		}
		items = append(items, item)
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

func syntheticReminderFees(baseFees []domain.FeeExpectation, dueDate time.Time, createdAt time.Time) []domain.FeeExpectation {
	reminders := make([]domain.FeeExpectation, 0, len(baseFees))
	for _, baseFee := range baseFees {
		baseID := baseFee.ID
		reminders = append(reminders, domain.FeeExpectation{
			ID:            uuid.New(),
			ChildID:       baseFee.ChildID,
			HouseholdID:   baseFee.HouseholdID,
			FeeType:       domain.FeeTypeReminder,
			Year:          createdAt.Year(),
			Month:         nil,
			Amount:        domain.ReminderFeeAmount,
			DueDate:       dueDate,
			CreatedAt:     createdAt,
			ReminderForID: &baseID,
		})
	}
	return reminders
}

// buildFamilyReminderEmail builds a parent-facing email for a single family.
// deadlineOverride sets a custom payment deadline; if nil, defaults to 7 days after runDate.
func buildFamilyReminderEmail(stage ReminderStage, runDate time.Time, parentFirstNames []string, items []reminderItem, deadlineOverride *time.Time) (string, string) {
	monthName := germanMonthName(int(runDate.Month()))
	year := runDate.Year()
	isFinal := stage == ReminderStageFinal

	var subject string
	if isFinal {
		subject = fmt.Sprintf("Kita Mahnung %s %d", monthName, year)
	} else {
		subject = fmt.Sprintf("Kita Zahlungserinnerung %s %d", monthName, year)
	}

	greeting := "Hallo"
	if len(parentFirstNames) > 0 {
		greeting = "Hallo " + strings.Join(parentFirstNames, " und ")
	}

	// Deadline: use override if provided, otherwise 7 days after the run date
	var dl time.Time
	if deadlineOverride != nil {
		dl = *deadlineOverride
	} else {
		dl = defaultReminderDeadline(runDate)
	}
	deadlineStr := dl.Format("02.01.2006")

	var builder strings.Builder
	builder.WriteString(greeting + ",\n\n")

	if len(items) == 1 {
		item := items[0]
		memberHint := ""
		if item.MemberNumber != "" {
			memberHint = fmt.Sprintf(" (Mitgliedsnr. %s)", item.MemberNumber)
		}
		if isFinal {
			builder.WriteString(fmt.Sprintf("für %s%s ist folgender offener Beitrag vermerkt:\n\n", item.ChildName, memberHint))
		} else {
			builder.WriteString(fmt.Sprintf("für %s%s ist folgender Beitrag offen:\n\n", item.ChildName, memberHint))
		}
		builder.WriteString(reminderLine(item, false) + "\n")
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	} else {
		if isFinal {
			builder.WriteString("für eure Familie sind folgende offene Beiträge vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie sind folgende Beiträge offen:\n\n")
		}
		for _, item := range items {
			builder.WriteString("- " + reminderLine(item, true) + "\n")
		}
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Beiträge spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Beiträge bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	}

	builder.WriteString("Empfänger: Knirpsenstadt e.V.\n")
	builder.WriteString("IBAN: DE33 3702 0500 0003 3214 00\n")
	builder.WriteString("BIC: BFSWDE33XXX\n\n")
	builder.WriteString("Wichtig: Bitte gebt als Empfänger genau \"Knirpsenstadt e.V.\" an, damit das Matching bei eurer Bank korrekt funktioniert.\n\n")
	if isFinal {
		builder.WriteString(fmt.Sprintf("Dies ist eine Mahnung. Bitte begleicht die offenen Beiträge spätestens bis zum %s.\n\n", deadlineStr))
		builder.WriteString("Falls ihr die Zahlung bereits veranlasst habt, betrachtet diese Nachricht bitte als gegenstandslos.\n\n")
	} else {
		builder.WriteString(fmt.Sprintf("Falls die Zahlung bis zum %s nicht eingegangen ist, wird leider automatisch eine Mahngebühr fällig.\n\n", deadlineStr))
	}
	builder.WriteString("Vielen Dank!\n\n")
	builder.WriteString("Freundliche Grüße\n")
	builder.WriteString("Knirpsenstadt Beitrag\n\n")
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
	case domain.FeeTypeMembership:
		return "Vereinsbeitrag"
	case domain.FeeTypeReminder:
		return "Mahngebühr"
	default:
		return string(feeType)
	}
}

func reminderLine(item reminderItem, includeChild bool) string {
	memberHint := ""
	if item.MemberNumber != "" {
		memberHint = fmt.Sprintf(" (Mitgliedsnr. %s)", item.MemberNumber)
	}
	prefix := ""
	if includeChild {
		prefix = fmt.Sprintf("%s%s: ", item.ChildName, memberHint)
	}

	label := feeTypeLabel(item.FeeType)
	switch item.FeeType {
	case domain.FeeTypeReminder:
		if item.BaseFeeType != nil && item.BaseMonth > 0 {
			return fmt.Sprintf("%sMahngebühr für %s %s/%d — %s",
				prefix,
				feeTypeLabel(*item.BaseFeeType),
				germanMonthName(item.BaseMonth),
				item.BaseYear,
				formatCurrencyEUR(item.Amount),
			)
		}
		if item.BaseFeeType != nil {
			return fmt.Sprintf("%sMahngebühr für %s — %s",
				prefix,
				feeTypeLabel(*item.BaseFeeType),
				formatCurrencyEUR(item.Amount),
			)
		}
		return fmt.Sprintf("%s%s — %s",
			prefix,
			label,
			formatCurrencyEUR(item.Amount),
		)
	default:
		if item.Month > 0 {
			return fmt.Sprintf("%s%s %s/%d — %s",
				prefix,
				label,
				germanMonthName(item.Month),
				item.Year,
				formatCurrencyEUR(item.Amount),
			)
		}
		return fmt.Sprintf("%s%s — %s",
			prefix,
			label,
			formatCurrencyEUR(item.Amount),
		)
	}
}

func defaultReminderDeadline(runDate time.Time) time.Time {
	base := time.Date(runDate.Year(), runDate.Month(), runDate.Day(), 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, 7)
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

const reminderEmailQRCodeCID = "payment-qr-code"

func buildReminderEmailHTML(textBody string, qrImageCID string) string {
	var builder strings.Builder
	builder.WriteString("<!doctype html><html><body style=\"font-family:Arial,sans-serif;line-height:1.5;color:#111;\">")

	lines := strings.Split(textBody, "\n")
	for idx, line := range lines {
		if idx > 0 {
			builder.WriteString("<br>")
		}
		builder.WriteString(html.EscapeString(line))
	}

	if strings.TrimSpace(qrImageCID) != "" {
		builder.WriteString("<hr style=\"margin:24px 0;border:none;border-top:1px solid #ddd;\">")
		builder.WriteString("<p><strong>QR-Code fuer die Ueberweisung:</strong></p>")
		builder.WriteString("<img alt=\"SEPA Zahlungs-QR\" src=\"cid:")
		builder.WriteString(html.EscapeString(qrImageCID))
		builder.WriteString("\" style=\"display:block;max-width:280px;width:100%;height:auto;border:1px solid #ddd;padding:8px;\">")
	}

	builder.WriteString("</body></html>")
	return builder.String()
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
