package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"sort"
	"strings"
	"time"
)

// PreviewReminderCase builds the final email preview without side effects.
func (s *ReminderService) PreviewReminderCase(ctx context.Context, householdID uuid.UUID, req *ReminderCaseRequest) (*ReminderCasePreview, error) {
	plan, err := s.prepareCasePlan(ctx, householdID, req)
	if err != nil {
		return nil, err
	}

	paymentSettings, err := s.GetPaymentSettings(ctx)
	if err != nil {
		return nil, err
	}

	subject, body := buildFamilyMixedReminderEmail(plan.stage, plan.runDate, plan.deadline, plan.firstNames, plan.items, paymentSettings)
	if strings.TrimSpace(req.Subject) != "" {
		subject = req.Subject
	}
	if strings.TrimSpace(req.Body) != "" {
		body = req.Body
	}

	preview := &ReminderCasePreview{
		HouseholdID:         householdID,
		HouseholdName:       plan.householdName,
		Recipients:          plan.recipients,
		Subject:             subject,
		Body:                body,
		Deadline:            plan.deadline,
		TotalAmount:         plan.totalAmount,
		IncludeQR:           plan.includeQR,
		SelectedFees:        plan.selectedFees,
		PlannedReminderFees: plan.plannedFees,
		Stage:               plan.stage,
		Warnings:            plan.warnings,
	}

	if plan.includeQR {
		qrData, qrErr := s.buildReminderQRCode(paymentSettings, plan.runDate, plan.householdName, plan.items)
		if qrErr != nil {
			log.Warn().Err(qrErr).Str("household", plan.householdName).Msg("Failed to generate payment QR code, continuing without QR")
		} else if qrData != nil {
			preview.QRImageDataURL = &qrData.DataURL
			preview.QRPayload = qrData.Payload
		}
	}

	return preview, nil
}

// SendReminderCase validates the plan again, sends the email and persists
// reminder fees. It returns a CaseConflictError when the state changed since
// the preview (fees paid or closed, reminder fees created after the preview,
// including a concurrent send detected via the unique reminder-fee index).
// previewedAt is mandatory: without it the concurrency guard could be bypassed.
func (s *ReminderService) SendReminderCase(ctx context.Context, householdID uuid.UUID, req *ReminderCaseRequest, sentBy *uuid.UUID) (*ReminderCaseSendResult, error) {
	if req.PreviewedAt == nil {
		return nil, ErrInvalidInput
	}
	plan, err := s.prepareCasePlan(ctx, householdID, req)
	if err != nil {
		return nil, err
	}

	if s.emailSender == nil || !s.emailSender.IsEnabled() {
		return nil, ErrEmailDisabled
	}
	if len(plan.recipients) == 0 {
		return nil, &CaseConflictError{Reason: "family has no valid email address"}
	}

	// Persist planned reminder fees. The unique index on reminder fees makes
	// a concurrent send of the same base fee fail on insert. On any later
	// failure the fees created by this request are removed again, so a retry
	// can plan them once more.
	createdAt := s.now().UTC()
	createdFeeIDs := make([]uuid.UUID, 0, len(plan.plannedFees))
	for _, planned := range plan.plannedFees {
		reminder := &domain.FeeExpectation{
			ID:            uuid.New(),
			ChildID:       plan.childIDByFee[planned.BaseFeeID],
			HouseholdID:   &householdID,
			FeeType:       domain.FeeTypeReminder,
			Year:          createdAt.Year(),
			Month:         nil,
			Amount:        planned.Amount,
			DueDate:       planned.DueDate,
			CreatedAt:     createdAt,
			ReminderForID: &planned.BaseFeeID,
		}
		if err := s.feeRepo.Create(ctx, reminder); err != nil {
			s.deleteFeesBestEffort(ctx, createdFeeIDs)
			if isUniqueReminderFeeViolation(err) {
				return nil, &CaseConflictError{FeeIDs: []uuid.UUID{planned.BaseFeeID}, Reason: "reminder fees were created after the preview"}
			}
			return nil, err
		}
		createdFeeIDs = append(createdFeeIDs, reminder.ID)
	}

	paymentSettings, err := s.GetPaymentSettings(ctx)
	if err != nil {
		s.deleteFeesBestEffort(ctx, createdFeeIDs)
		return nil, err
	}

	subject, body := buildFamilyMixedReminderEmail(plan.stage, plan.runDate, plan.deadline, plan.firstNames, plan.items, paymentSettings)
	if strings.TrimSpace(req.Subject) != "" {
		subject = req.Subject
	}
	if strings.TrimSpace(req.Body) != "" {
		body = req.Body
	}

	var qrData *reminderQRCodeData
	if plan.includeQR {
		data, qrErr := s.buildReminderQRCode(paymentSettings, plan.runDate, plan.householdName, plan.items)
		if qrErr != nil {
			log.Warn().Err(qrErr).Str("household", plan.householdName).Msg("Failed to generate payment QR code, continuing without QR")
		} else {
			qrData = data
		}
	}

	if qrData != nil {
		htmlBody := buildReminderEmailHTML(body, reminderEmailQRCodeCID)
		if err := s.emailSender.SendTextAndHTMLEmailMulti(plan.recipients, subject, body, htmlBody, reminderEmailQRCodeCID, qrData.PNG); err != nil {
			s.deleteFeesBestEffort(ctx, createdFeeIDs)
			return nil, err
		}
	} else {
		if err := s.emailSender.SendTextEmailMulti(plan.recipients, subject, body); err != nil {
			s.deleteFeesBestEffort(ctx, createdFeeIDs)
			return nil, err
		}
	}

	if err := s.logCaseEmail(ctx, householdID, plan, subject, body, qrData != nil, sentBy); err != nil {
		return nil, err
	}

	return &ReminderCaseSendResult{
		HouseholdID:         householdID,
		SentTo:              plan.recipients,
		Subject:             subject,
		Deadline:            plan.deadline,
		Stage:               plan.stage,
		CreatedReminderFees: ensurePlannedFeesSlice(plan.plannedFees),
	}, nil
}

func ensurePlannedFeesSlice(fees []ReminderCasePlannedFee) []ReminderCasePlannedFee {
	if fees == nil {
		return make([]ReminderCasePlannedFee, 0)
	}
	return fees
}

// deleteFeesBestEffort removes reminder fees created by a failed send so no
// orphaned reminder fees remain. Failures are logged but do not mask the
// original error.
func (s *ReminderService) deleteFeesBestEffort(ctx context.Context, ids []uuid.UUID) {
	for _, id := range ids {
		if err := s.feeRepo.Delete(ctx, id); err != nil {
			log.Error().Err(err).Str("feeId", id.String()).Msg("Failed to remove reminder fee after failed send")
		}
	}
}

// isUniqueReminderFeeViolation reports whether the error is a violation of
// the unique reminder-per-base-fee index (concurrent send).
func isUniqueReminderFeeViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505" && pqErr.Constraint == "uq_fee_expectations_reminder_for_id"
	}
	return false
}

// casePlan is the validated internal state shared by preview and send.
type casePlan struct {
	stage         ReminderStage
	runDate       time.Time
	deadline      time.Time
	includeQR     bool
	householdName string
	recipients    []string
	firstNames    []string
	selectedFees  []ReminderCaseFee
	items         []reminderItem
	plannedFees   []ReminderCasePlannedFee
	childIDByFee  map[uuid.UUID]uuid.UUID
	warnings      []string
	totalAmount   float64
}

// prepareCasePlan loads and validates everything preview and send need.
// Concurrency conflicts (paid fees, foreign fees, reminder fees created after
// the preview) surface as CaseConflictError.
func (s *ReminderService) prepareCasePlan(ctx context.Context, householdID uuid.UUID, req *ReminderCaseRequest) (*casePlan, error) {
	if req == nil || len(req.FeeIDs) == 0 {
		return nil, ErrInvalidInput
	}
	seen := make(map[uuid.UUID]struct{}, len(req.FeeIDs))
	feeIDs := make([]uuid.UUID, 0, len(req.FeeIDs))
	for _, id := range req.FeeIDs {
		if _, dup := seen[id]; dup {
			return nil, ErrInvalidInput
		}
		seen[id] = struct{}{}
		feeIDs = append(feeIDs, id)
	}
	reminderFor := make(map[uuid.UUID]bool, len(req.ReminderFeeIDs))
	for _, id := range req.ReminderFeeIDs {
		if _, selected := seen[id]; !selected || reminderFor[id] {
			return nil, ErrInvalidInput
		}
		reminderFor[id] = true
	}

	runDate := req.RunDate
	if runDate.IsZero() {
		runDate = s.now()
	}
	runDate = startOfDayUTC(runDate)
	deadline := defaultReminderDeadline(runDate)

	rows, err := s.feeRepo.ListOpenByHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	openByFee := make(map[uuid.UUID]repository.OpenFeeRow, len(rows))
	for _, row := range rows {
		openByFee[row.ID] = row
	}

	// Foreign or meanwhile paid fees are conflicts.
	var missing []uuid.UUID
	for _, id := range feeIDs {
		if _, ok := openByFee[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, &CaseConflictError{FeeIDs: missing, Reason: "selected fees are no longer open or do not belong to this household"}
	}

	household, err := s.householdRepo.GetByID(ctx, householdID)
	if err != nil {
		return nil, err
	}
	parents, err := s.householdRepo.GetParents(ctx, householdID)
	if err != nil {
		return nil, err
	}

	cutoff, err := s.GetHistoryReliableFrom(ctx)
	if err != nil {
		return nil, err
	}

	fees, err := s.loadCaseFees(ctx, householdID, runDate, cutoff)
	if err != nil {
		return nil, err
	}
	feeByID := make(map[uuid.UUID]ReminderCaseFee, len(fees))
	for _, fee := range fees {
		feeByID[fee.FeeID] = fee
	}

	selected := make([]ReminderCaseFee, 0, len(feeIDs))
	baseIDs := make([]uuid.UUID, 0, len(feeIDs))
	childIDByFee := make(map[uuid.UUID]uuid.UUID)
	for _, id := range feeIDs {
		fee := feeByID[id]
		selected = append(selected, fee)
		childIDByFee[fee.FeeID] = fee.ChildID
		if fee.FeeType != domain.FeeTypeReminder {
			baseIDs = append(baseIDs, fee.FeeID)
		}
	}

	// Reminder fees created after the preview indicate a concurrent send.
	if req.PreviewedAt != nil && len(baseIDs) > 0 {
		recentReminders, err := s.feeRepo.GetReminderBaseIDsCreatedAfter(ctx, baseIDs, *req.PreviewedAt)
		if err != nil {
			return nil, err
		}
		var conflicted []uuid.UUID
		for _, id := range baseIDs {
			if recentReminders[id] {
				conflicted = append(conflicted, id)
			}
		}
		if len(conflicted) > 0 {
			return nil, &CaseConflictError{FeeIDs: conflicted, Reason: "reminder fees were created after the preview"}
		}
	}

	existingReminders, err := s.feeRepo.GetOpenReminderBaseIDs(ctx, baseIDs)
	if err != nil {
		return nil, err
	}

	// Planned reminder fees: only for the fees the user ticked. Each must be
	// due by the rules (reminded before, deadline passed, no Mahngebühr yet);
	// a fee that lost this state since the preview is a conflict. Nothing is
	// charged automatically, so reminders can be repeated as a courtesy.
	planned := make([]ReminderCasePlannedFee, 0)
	var notDue []uuid.UUID
	for _, fee := range selected {
		if !reminderFor[fee.FeeID] {
			continue
		}
		if !fee.ReminderFeeDue || existingReminders[fee.FeeID] {
			notDue = append(notDue, fee.FeeID)
			continue
		}
		planned = append(planned, ReminderCasePlannedFee{
			BaseFeeID:   fee.FeeID,
			BaseFeeType: fee.FeeType,
			BaseLabel:   fmt.Sprintf("%s (%s)", caseFeeLabel(fee), caseFeePerson(fee)),
			Amount:      reminderFeeAmountFor(fee.FeeType),
			DueDate:     deadline,
		})
	}
	if len(notDue) > 0 {
		return nil, &CaseConflictError{FeeIDs: notDue, Reason: "a reminder fee is not due for the selected fees"}
	}
	stage := ReminderStageInitial
	if len(planned) > 0 {
		stage = ReminderStageFinal
	}

	// Mail/QR items: selected fees with remaining amounts plus planned fees.
	items := make([]reminderItem, 0, len(selected)+len(planned))
	var totalCents int64
	for _, fee := range selected {
		totalCents += domain.Cents(fee.Remaining)
		item := reminderItem{
			FeeID:         fee.FeeID,
			ChildID:       fee.ChildID,
			ChildName:     fee.ChildName,
			MemberNumber:  fee.MemberNumber,
			FeeType:       fee.FeeType,
			Amount:        fee.Remaining,
			Year:          fee.Year,
			Month:         fee.Month,
			DueDate:       fee.DueDate,
			ReminderForID: fee.ReminderForID,
			ClubMember:    fee.ClubMember,
		}
		if fee.ReminderFor != nil {
			baseType := fee.ReminderFor.FeeType
			item.BaseFeeType = &baseType
			item.BaseYear = fee.ReminderFor.Year
			item.BaseMonth = derefInt(fee.ReminderFor.Month)
		}
		items = append(items, item)
	}
	for _, plannedFee := range planned {
		totalCents += domain.Cents(plannedFee.Amount)
		baseFee := feeByID[plannedFee.BaseFeeID]
		baseType := plannedFee.BaseFeeType
		baseID := plannedFee.BaseFeeID
		items = append(items, reminderItem{
			ChildID:       baseFee.ChildID,
			ChildName:     baseFee.ChildName,
			MemberNumber:  baseFee.MemberNumber,
			FeeType:       domain.FeeTypeReminder,
			Amount:        plannedFee.Amount,
			Year:          runDate.Year(),
			BaseFeeType:   &baseType,
			BaseYear:      baseFee.Year,
			BaseMonth:     baseFee.Month,
			DueDate:       plannedFee.DueDate,
			ReminderForID: &baseID,
			ClubMember:    baseFee.ClubMember,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ChildName != items[j].ChildName {
			return items[i].ChildName < items[j].ChildName
		}
		if items[i].Year != items[j].Year {
			return items[i].Year < items[j].Year
		}
		return items[i].Month < items[j].Month
	})

	includeQR := req.IncludeQR == nil || *req.IncludeQR

	return &casePlan{
		stage:         stage,
		runDate:       runDate,
		deadline:      deadline,
		includeQR:     includeQR,
		householdName: household.Name,
		recipients:    collectEmails(parents),
		firstNames:    parentFirstNames(parents),
		selectedFees:  selected,
		items:         items,
		plannedFees:   planned,
		childIDByFee:  childIDByFee,
		warnings:      buildCaseWarnings(selected, reminderFor),
		totalAmount:   domain.Euros(totalCents),
	}, nil
}

// buildCaseWarnings names what the user should know before sending: due
// Mahngebühren that are not charged, fees whose reminder deadline is still
// running, and fees without assignable history (no Mahngebühr possible).
func buildCaseWarnings(selected []ReminderCaseFee, reminderFor map[uuid.UUID]bool) []string {
	warnings := make([]string, 0)
	for _, fee := range selected {
		label := fmt.Sprintf("%s: %s", caseFeePerson(fee), caseFeeLabel(fee))
		if fee.ReminderFeeDue && !reminderFor[fee.FeeID] {
			warnings = append(warnings, fmt.Sprintf("%s – Mahngebühr laut Regeln fällig, wird nicht erhoben", label))
		}
		switch fee.Status {
		case FeeStatusWaiting:
			if fee.LastContact != nil {
				deadline := defaultReminderDeadline(fee.LastContact.RunDate)
				warnings = append(warnings, fmt.Sprintf("%s wurde erst am %s erinnert, die Frist läuft bis %s", label, fee.LastContact.RunDate.Format("02.01.2006"), deadline.Format("02.01.2006")))
			}
		case FeeStatusHistoryUnknown:
			warnings = append(warnings, fmt.Sprintf("Für %s gibt es keine zuordenbare Historie – eine Mahngebühr ist erst nach einer Erinnerung möglich", label))
		}
	}
	return warnings
}

func (s *ReminderService) logCaseEmail(
	ctx context.Context,
	householdID uuid.UUID,
	plan *casePlan,
	subject string,
	body string,
	qrIncluded bool,
	sentBy *uuid.UUID,
) error {
	if s.emailLogRepo == nil {
		return nil
	}

	feeIDs := make([]uuid.UUID, 0, len(plan.selectedFees))
	for _, fee := range plan.selectedFees {
		feeIDs = append(feeIDs, fee.FeeID)
	}

	created := make([]map[string]any, 0, len(plan.plannedFees))
	for _, planned := range plan.plannedFees {
		created = append(created, map[string]any{
			"baseFeeId":   planned.BaseFeeID,
			"baseFeeType": planned.BaseFeeType,
			"amount":      planned.Amount,
		})
	}

	payload := struct {
		Stage            ReminderStage    `json:"stage"`
		RunDate          string           `json:"runDate"`
		Deadline         string           `json:"deadline"`
		HouseholdID      uuid.UUID        `json:"householdId"`
		FeeIDs           []uuid.UUID      `json:"feeIds"`
		QRIncluded       bool             `json:"qrIncluded"`
		RemindersCreated []map[string]any `json:"remindersCreated"`
	}{
		Stage:            plan.stage,
		RunDate:          plan.runDate.Format("2006-01-02"),
		Deadline:         plan.deadline.Format("2006-01-02"),
		HouseholdID:      householdID,
		FeeIDs:           feeIDs,
		QRIncluded:       qrIncluded,
		RemindersCreated: created,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	rawPayload := json.RawMessage(payloadBytes)

	bodyCopy := body
	emailType := domain.EmailLogTypeReminderInitial
	if plan.stage == ReminderStageFinal {
		emailType = domain.EmailLogTypeReminderFinal
	}

	return s.emailLogRepo.Create(ctx, &domain.EmailLog{
		ID:          uuid.New(),
		SentAt:      s.now().UTC(),
		ToEmail:     strings.Join(plan.recipients, ", "),
		Subject:     subject,
		Body:        &bodyCopy,
		EmailType:   emailType,
		Payload:     &rawPayload,
		SentBy:      sentBy,
		HouseholdID: &householdID,
	})
}
