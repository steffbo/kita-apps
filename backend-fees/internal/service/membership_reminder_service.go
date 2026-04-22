package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// MembershipReminderService handles membership fee reminder emails.
type MembershipReminderService struct {
	feeRepo       repository.FeeRepository
	childRepo     repository.ChildRepository
	householdRepo repository.HouseholdRepository
	settingsRepo  repository.SettingsRepository
	emailLogRepo  repository.EmailLogRepository
	emailSender   ReminderEmailSender
	now           func() time.Time
}

// NewMembershipReminderService creates a new membership reminder service.
func NewMembershipReminderService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	householdRepo repository.HouseholdRepository,
	settingsRepo repository.SettingsRepository,
	emailLogRepo repository.EmailLogRepository,
	emailSender ReminderEmailSender,
) *MembershipReminderService {
	return &MembershipReminderService{
		feeRepo:       feeRepo,
		childRepo:     childRepo,
		householdRepo: householdRepo,
		settingsRepo:  settingsRepo,
		emailLogRepo:  emailLogRepo,
		emailSender:   emailSender,
		now:           time.Now,
	}
}

// Run executes membership reminder logic for the given date and stage.
// deadline overrides the due date shown in the email; if nil, 31.03.<runYear> is used.
func (s *MembershipReminderService) Run(
	ctx context.Context,
	runDate time.Time,
	stage ReminderStage,
	sentBy *uuid.UUID,
	dryRun bool,
	deadline *time.Time,
	selectedHouseholdIDs []uuid.UUID,
) (*ReminderRunResult, error) {
	if stage != ReminderStageInitial && stage != ReminderStageFinal {
		return nil, ErrInvalidInput
	}

	feeTypes := []domain.FeeType{domain.FeeTypeMembership}
	dueOnOrBefore := endOfDayUTC(runDate)

	fees, err := s.feeRepo.ListUnpaidByTypesDueOnOrBefore(ctx, feeTypes, dueOnOrBefore)
	if err != nil {
		return nil, err
	}

	if stage == ReminderStageInitial {
		fees = filterOutReminderFees(fees)
	}

	result := &ReminderRunResult{
		Stage:  stage,
		Date:   runDate,
		DryRun: dryRun,
	}

	if len(fees) == 0 {
		result.Message = "no unpaid membership fees for this period"
		return result, nil
	}

	if stage == ReminderStageFinal {
		toRemind, err := s.feeRepo.ListUnpaidWithoutReminderByTypesDueOnOrBefore(ctx, feeTypes, dueOnOrBefore)
		if err != nil {
			return nil, err
		}
		result.RemindersCreated = len(toRemind)
		fees = append(fees, syntheticMembershipReminderFees(toRemind, membershipReminderDueDate(runDate), s.now().UTC())...)
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
					Amount:        domain.MembershipReminderFeeAmount,
					DueDate:       membershipReminderDueDate(runDate),
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

	helper := &ReminderService{
		feeRepo:       s.feeRepo,
		childRepo:     s.childRepo,
		householdRepo: s.householdRepo,
		settingsRepo:  s.settingsRepo,
	}

	items, children, err := helper.buildItemsWithChildren(ctx, fees)
	if err != nil {
		return nil, err
	}

	householdGroups, err := helper.groupByHousehold(items, children)
	if err != nil {
		return nil, err
	}
	householdGroups = filterHouseholdGroupsBySelection(householdGroups, selectedHouseholdIDs)

	result.FamiliesProcessed = len(householdGroups)
	paymentSettings, err := helper.GetPaymentSettings(ctx)
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
			log.Warn().Str("household", group.householdName).Msg("No valid parent emails for membership reminder, skipping family")
			result.FamiliesSkippedNoEmail++
			result.Warnings = append(result.Warnings, ReminderWarning{
				HouseholdName: group.householdName,
				Reason:        "keine gültige E-Mail-Adresse",
			})
			continue
		}

		firstNames := parentFirstNames(parents)
		subject, body := buildFamilyMembershipReminderEmail(stage, runDate, firstNames, group.items, deadline, paymentSettings)
		qrData, qrErr := helper.buildReminderQRCode(paymentSettings, runDate, group.householdName, group.items)
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

func filterOutReminderFees(fees []domain.FeeExpectation) []domain.FeeExpectation {
	filtered := make([]domain.FeeExpectation, 0, len(fees))
	for _, fee := range fees {
		if fee.FeeType == domain.FeeTypeReminder {
			continue
		}
		filtered = append(filtered, fee)
	}
	return filtered
}

func endOfDayUTC(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC)
}

func membershipReminderDueDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC)
}

func defaultMembershipReminderDeadline(runDate time.Time) time.Time {
	return time.Date(runDate.Year(), time.March, 31, 0, 0, 0, 0, time.UTC)
}

func syntheticMembershipReminderFees(baseFees []domain.FeeExpectation, dueDate time.Time, createdAt time.Time) []domain.FeeExpectation {
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
			Amount:        domain.MembershipReminderFeeAmount,
			DueDate:       dueDate,
			CreatedAt:     createdAt,
			ReminderForID: &baseID,
		})
	}
	return reminders
}

func buildFamilyMembershipReminderEmail(
	stage ReminderStage,
	runDate time.Time,
	parentFirstNames []string,
	items []reminderItem,
	deadlineOverride *time.Time,
	paymentSettings ReminderPaymentSettings,
) (string, string) {
	year := runDate.Year()
	isFinal := stage == ReminderStageFinal

	subject := fmt.Sprintf("Kita Zahlungserinnerung Vereinsbeitrag %d", year)
	if isFinal {
		subject = fmt.Sprintf("Kita Mahnung Vereinsbeitrag %d", year)
	}

	greeting := "Hallo"
	if len(parentFirstNames) > 0 {
		greeting = "Hallo " + strings.Join(parentFirstNames, " und ")
	}

	var dl time.Time
	if deadlineOverride != nil {
		dl = *deadlineOverride
	} else {
		dl = defaultMembershipReminderDeadline(runDate)
	}
	deadlineStr := dl.Format("02.01.2006")

	var builder strings.Builder
	builder.WriteString(greeting + ",\n\n")

	if len(items) == 1 {
		if isFinal {
			builder.WriteString("für eure Familie ist folgender offener Vereinsbeitrag vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie ist folgender Vereinsbeitrag offen:\n\n")
		}
		builder.WriteString(membershipReminderLine(items[0]) + "\n")
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	} else {
		if isFinal {
			builder.WriteString("für eure Familie sind folgende offene Vereinsbeiträge vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie sind folgende Vereinsbeiträge offen:\n\n")
		}
		for _, item := range items {
			builder.WriteString("- " + membershipReminderLine(item) + "\n")
		}
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Vereinsbeiträge spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Vereinsbeiträge bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	}

	effectivePaymentSettings := applyLegacyReminderPaymentDefaults(paymentSettings)
	builder.WriteString(fmt.Sprintf("Empfänger: %s\n", effectivePaymentSettings.RecipientName))
	builder.WriteString(fmt.Sprintf("IBAN: %s\n", formatIBANForEmail(effectivePaymentSettings.IBAN)))
	if effectivePaymentSettings.BIC != "" {
		builder.WriteString(fmt.Sprintf("BIC: %s\n", effectivePaymentSettings.BIC))
	}
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Wichtig: Bitte gebt als Empfänger genau \"%s\" an, damit das Matching bei eurer Bank korrekt funktioniert.\n\n", effectivePaymentSettings.RecipientName))
	if isFinal {
		builder.WriteString(fmt.Sprintf("Dies ist eine Mahnung. Bitte begleicht die offenen Vereinsbeiträge spätestens bis zum %s.\n\n", deadlineStr))
		builder.WriteString("Falls ihr die Zahlung bereits veranlasst habt, betrachtet diese Nachricht bitte als gegenstandslos.\n\n")
	} else {
		builder.WriteString("Falls ihr die Zahlung bereits veranlasst habt, betrachtet diese Nachricht bitte als gegenstandslos.\n\n")
	}
	builder.WriteString("Vielen Dank!\n\n")
	builder.WriteString("Freundliche Grüße\n")
	builder.WriteString("Knirpsenstadt Beitrag\n\n")
	builder.WriteString("---\n")
	builder.WriteString("Diese E-Mail wurde automatisch erstellt. Fehler sind nicht ausgeschlossen — bei Fragen wendet euch gerne direkt an uns.\n")

	return subject, builder.String()
}

func membershipReminderLine(item reminderItem) string {
	switch item.FeeType {
	case domain.FeeTypeReminder:
		if item.BaseFeeType != nil {
			if item.BaseYear > 0 {
				return fmt.Sprintf("Mahngebühr für %s %d — %s", feeTypeLabel(*item.BaseFeeType), item.BaseYear, formatCurrencyEUR(item.Amount))
			}
			return fmt.Sprintf("Mahngebühr für %s — %s", feeTypeLabel(*item.BaseFeeType), formatCurrencyEUR(item.Amount))
		}
		return fmt.Sprintf("Mahngebühr — %s", formatCurrencyEUR(item.Amount))
	default:
		if item.Year > 0 {
			return fmt.Sprintf("%s %d — %s", feeTypeLabel(item.FeeType), item.Year, formatCurrencyEUR(item.Amount))
		}
		return fmt.Sprintf("%s — %s", feeTypeLabel(item.FeeType), formatCurrencyEUR(item.Amount))
	}
}

func (s *MembershipReminderService) logEmail(
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
	emailType := domain.EmailLogTypeMembershipReminderInitial
	if stage == ReminderStageFinal {
		emailType = domain.EmailLogTypeMembershipReminderFinal
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
