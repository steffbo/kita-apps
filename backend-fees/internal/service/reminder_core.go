package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// reminderScope describes a fee-type family processed by the shared reminder
// core: how unpaid fees are selected, which reminder fees are created and how
// the family email is built.
type reminderScope struct {
	feeTypes           []domain.FeeType
	membership         bool
	reminderFeeAmount  float64
	reminderFeeDueDate func(time.Time) time.Time
	buildEmail         func(ReminderStage, time.Time, []string, []reminderItem, *time.Time, ReminderPaymentSettings) (string, string)
	initialEmailType   domain.EmailLogType
	finalEmailType     domain.EmailLogType
	noFeesMessage      string
}

// runScope executes the shared reminder pipeline for the given scope.
// deadline overrides the payment deadline shown in the email; if nil, the
// scope default applies (7 days after the run date).
func (s *ReminderService) runScope(
	ctx context.Context,
	scope reminderScope,
	runDate time.Time,
	stage ReminderStage,
	sentBy *uuid.UUID,
	dryRun bool,
	deadline *time.Time,
	selectedHouseholdIDs []uuid.UUID,
	options *ReminderRunOptions,
) (*ReminderRunResult, error) {
	if stage != ReminderStageInitial && stage != ReminderStageFinal {
		return nil, ErrInvalidInput
	}

	year := runDate.Year()
	month := int(runDate.Month())
	dueOnOrBefore := endOfDayUTC(runDate)

	var fees []domain.FeeExpectation
	var err error
	if scope.membership {
		fees, err = s.feeRepo.ListUnpaidByTypesDueOnOrBefore(ctx, scope.feeTypes, dueOnOrBefore)
		if err != nil {
			return nil, err
		}
		if stage == ReminderStageInitial {
			fees = filterOutReminderFees(fees)
		}
	} else {
		fees, err = s.feeRepo.ListUnpaidUpToMonthAndTypes(ctx, year, month, scope.feeTypes)
		if err != nil {
			return nil, err
		}
	}

	result := &ReminderRunResult{
		Stage:  stage,
		Date:   runDate,
		DryRun: dryRun,
	}

	if len(fees) == 0 {
		result.Message = scope.noFeesMessage
		return result, nil
	}

	if stage == ReminderStageFinal {
		var toRemind []domain.FeeExpectation
		if scope.membership {
			toRemind, err = s.feeRepo.ListUnpaidWithoutReminderByTypesDueOnOrBefore(ctx, scope.feeTypes, dueOnOrBefore)
		} else {
			toRemind, err = s.feeRepo.ListUnpaidWithoutReminderByMonthAndTypes(ctx, year, month, scope.feeTypes)
		}
		if err != nil {
			return nil, err
		}
		result.RemindersCreated = len(toRemind)
		fees = append(fees, syntheticReminderFeesForScope(toRemind, scope, runDate, s.now().UTC())...)
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
					Amount:        scope.reminderFeeAmount,
					DueDate:       scope.reminderFeeDueDate(runDate),
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
		subject, body := scope.buildEmail(stage, runDate, firstNames, group.items, deadline, paymentSettings)
		subject, body = applyReminderOverrides(subject, body, options, group.householdID)
		qrData, qrErr := s.buildReminderQRCode(paymentSettings, runDate, group.householdName, group.items)
		if qrErr != nil {
			log.Warn().Err(qrErr).Str("household", group.householdName).Msg("Failed to generate payment QR code, continuing without QR")
		}
		if qrData != nil && reminderQRDisabled(options) {
			qrData = nil
		}

		var qrImageDataURL *string
		var qrPayload string
		if qrData != nil {
			qrImageDataURL = &qrData.DataURL
			qrPayload = qrData.Payload
		}

		if dryRun {
			result.Previews = append(result.Previews, ReminderPreview{
				HouseholdID:    group.householdID.String(),
				HouseholdName:  group.householdName,
				Recipients:     recipients,
				Subject:        subject,
				Body:           body,
				QRImageDataURL: qrImageDataURL,
				QRPayload:      qrPayload,
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
		if err := s.logScopeEmail(ctx, scope, stage, runDate, toEmail, subject, body, group.items, result.RemindersCreated, sentBy); err != nil {
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

// applyReminderOverrides replaces the generated subject and/or body when an
// override exists for the household. Empty override fields keep the generated
// value.
func applyReminderOverrides(
	subject, body string,
	options *ReminderRunOptions,
	householdID uuid.UUID,
) (string, string) {
	if options == nil || len(options.Overrides) == 0 {
		return subject, body
	}
	override, ok := options.Overrides[householdID]
	if !ok {
		return subject, body
	}
	if strings.TrimSpace(override.Subject) != "" {
		subject = override.Subject
	}
	if strings.TrimSpace(override.Body) != "" {
		body = override.Body
	}
	return subject, body
}

// reminderQRDisabled reports whether QR code generation/attachment is switched
// off for this run.
func reminderQRDisabled(options *ReminderRunOptions) bool {
	return options != nil && options.IncludeQR != nil && !*options.IncludeQR
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

// collectEmails returns deduplicated non-empty parent email addresses. The
// result is never nil so it marshals to an empty JSON array.
func collectEmails(parents []domain.Parent) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
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

func syntheticReminderFeesForScope(baseFees []domain.FeeExpectation, scope reminderScope, runDate time.Time, createdAt time.Time) []domain.FeeExpectation {
	dueDate := scope.reminderFeeDueDate(runDate)
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
			Amount:        scope.reminderFeeAmount,
			DueDate:       dueDate,
			CreatedAt:     createdAt,
			ReminderForID: &baseID,
		})
	}
	return reminders
}

func (s *ReminderService) logScopeEmail(
	ctx context.Context,
	scope reminderScope,
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
	emailType := scope.initialEmailType
	if stage == ReminderStageFinal {
		emailType = scope.finalEmailType
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
