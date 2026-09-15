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

// ReminderCaseFeeStatus describes the workflow state of a single open fee.
type ReminderCaseFeeStatus string

const (
	FeeStatusNeverContacted    ReminderCaseFeeStatus = "never_contacted"
	FeeStatusWaiting           ReminderCaseFeeStatus = "waiting"
	FeeStatusActionableInitial ReminderCaseFeeStatus = "actionable_initial"
	FeeStatusActionableFinal   ReminderCaseFeeStatus = "actionable_final"
	FeeStatusHistoryUnknown    ReminderCaseFeeStatus = "history_unknown"
)

// Scopes for ListReminderCases.
const (
	ReminderCasesScopeActionable = "actionable"
	ReminderCasesScopeAll        = "all"
)

// ReminderCaseFee is a single open fee inside a family case.
type ReminderCaseFee struct {
	FeeID        uuid.UUID             `json:"feeId"`
	ChildID      uuid.UUID             `json:"childId"`
	ChildName    string                `json:"childName"`
	MemberNumber string                `json:"memberNumber,omitempty"`
	FeeType      domain.FeeType        `json:"feeType"`
	Year         int                   `json:"year"`
	Month        int                   `json:"month"`
	DueDate      time.Time             `json:"dueDate"`
	Amount       float64               `json:"amount"`
	Remaining    float64               `json:"remaining"`
	Status       ReminderCaseFeeStatus `json:"status"`
	ActionableAt time.Time             `json:"actionableAt"`
	LastContact  *FeeContact           `json:"lastContact,omitempty"`
	HasReminder  bool                  `json:"hasReminder"`
}

// ReminderCase is the family-level working item.
type ReminderCase struct {
	HouseholdID    uuid.UUID         `json:"householdId"`
	HouseholdName  string            `json:"householdName"`
	Recipients     []string          `json:"recipients"`
	TotalRemaining float64           `json:"totalRemaining"`
	NextActionAt   time.Time         `json:"nextActionAt"`
	Fees           []ReminderCaseFee `json:"fees"`
}

// ReminderCasesResult is the working list payload.
type ReminderCasesResult struct {
	AsOf  string         `json:"asOf"`
	Scope string         `json:"scope"`
	Cases []ReminderCase `json:"cases"`
}

// ReminderCaseRequest is the shared request for preview and send.
type ReminderCaseRequest struct {
	Stage       ReminderStage `json:"stage"`
	RunDate     time.Time     `json:"runDate"`
	FeeIDs      []uuid.UUID   `json:"feeIds"`
	IncludeQR   *bool         `json:"includeQR,omitempty"`
	Subject     string        `json:"subject,omitempty"`
	Body        string        `json:"body,omitempty"`
	PreviewedAt *time.Time    `json:"previewedAt,omitempty"`
}

// ReminderCasePlannedFee is a reminder fee that would be created on send.
type ReminderCasePlannedFee struct {
	BaseFeeID   uuid.UUID      `json:"baseFeeId"`
	BaseFeeType domain.FeeType `json:"baseFeeType"`
	BaseLabel   string         `json:"baseLabel"`
	Amount      float64        `json:"amount"`
	DueDate     time.Time      `json:"dueDate"`
}

// ReminderCasePreview is the final email preview for one family.
type ReminderCasePreview struct {
	HouseholdID         uuid.UUID                `json:"householdId"`
	HouseholdName       string                   `json:"householdName"`
	Recipients          []string                 `json:"recipients"`
	Subject             string                   `json:"subject"`
	Body                string                   `json:"body"`
	Deadline            time.Time                `json:"deadline"`
	TotalAmount         float64                  `json:"totalAmount"`
	IncludeQR           bool                     `json:"includeQR"`
	QRImageDataURL      *string                  `json:"qrImageDataUrl,omitempty"`
	QRPayload           string                   `json:"qrPayload,omitempty"`
	SelectedFees        []ReminderCaseFee        `json:"selectedFees"`
	PlannedReminderFees []ReminderCasePlannedFee `json:"plannedReminderFees"`
	RecommendedStage    ReminderStage            `json:"recommendedStage"`
	Warnings            []string                 `json:"warnings,omitempty"`
}

// ReminderCaseSendResult reports a successful send.
type ReminderCaseSendResult struct {
	HouseholdID         uuid.UUID                `json:"householdId"`
	SentTo              []string                 `json:"sentTo"`
	Subject             string                   `json:"subject"`
	Deadline            time.Time                `json:"deadline"`
	Stage               ReminderStage            `json:"stage"`
	CreatedReminderFees []ReminderCasePlannedFee `json:"createdReminderFees"`
}

// CaseConflictError marks a stale preview state; affected fee IDs are attached.
type CaseConflictError struct {
	FeeIDs []uuid.UUID
	Reason string
}

func (e *CaseConflictError) Error() string {
	return e.Reason
}

// ListReminderCases builds the family-based working list.
func (s *ReminderService) ListReminderCases(ctx context.Context, asOf time.Time, scope string) (*ReminderCasesResult, error) {
	if scope != ReminderCasesScopeActionable && scope != ReminderCasesScopeAll {
		return nil, ErrInvalidInput
	}

	cutoff, err := s.GetHistoryReliableFrom(ctx)
	if err != nil {
		return nil, err
	}

	households, err := s.householdRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	asOfStart := startOfDayUTC(asOf)
	cases := make([]ReminderCase, 0)
	for _, household := range households {
		fees, err := s.loadCaseFees(ctx, household.ID, asOfStart, cutoff)
		if err != nil {
			return nil, err
		}
		if len(fees) == 0 {
			continue
		}
		if scope == ReminderCasesScopeActionable && !hasActionableFee(fees) {
			continue
		}

		parents, err := s.householdRepo.GetParents(ctx, household.ID)
		if err != nil {
			return nil, err
		}

		total := 0.0
		nextAction := time.Time{}
		for _, fee := range fees {
			total += fee.Remaining
			if nextAction.IsZero() || fee.ActionableAt.Before(nextAction) {
				nextAction = fee.ActionableAt
			}
		}

		cases = append(cases, ReminderCase{
			HouseholdID:    household.ID,
			HouseholdName:  household.Name,
			Recipients:     collectEmails(parents),
			TotalRemaining: roundCent(total),
			NextActionAt:   nextAction,
			Fees:           fees,
		})
	}

	return &ReminderCasesResult{
		AsOf:  asOfStart.Format("2006-01-02"),
		Scope: scope,
		Cases: cases,
	}, nil
}

// loadCaseFees resolves the open fees of a household with workflow status.
func (s *ReminderService) loadCaseFees(ctx context.Context, householdID uuid.UUID, asOfStart, cutoff time.Time) ([]ReminderCaseFee, error) {
	rows, err := s.feeRepo.ListOpenByHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	baseIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if row.FeeType == domain.FeeTypeReminder && row.ReminderForID != nil {
			baseIDs = append(baseIDs, *row.ReminderForID)
		}
	}
	hasReminder, err := s.feeRepo.GetOpenReminderBaseIDs(ctx, baseIDs)
	if err != nil {
		return nil, err
	}
	// Reminder fees themselves always report an existing reminder.
	for _, row := range rows {
		if row.FeeType == domain.FeeTypeReminder {
			hasReminder[row.ID] = true
		}
	}

	contacts, err := s.ListFeeContacts(ctx, householdID)
	if err != nil {
		return nil, err
	}

	childIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		childIDs = append(childIDs, row.ChildID)
	}
	children, err := s.childRepo.GetByIDs(ctx, childIDs)
	if err != nil {
		return nil, err
	}

	fees := make([]ReminderCaseFee, 0, len(rows))
	for _, row := range rows {
		childName := "Unbekanntes Kind"
		memberNumber := ""
		if child, ok := children[row.ChildID]; ok && child != nil {
			childName = child.FirstName
			memberNumber = child.MemberNumber
		}
		month := 0
		if row.Month != nil {
			month = *row.Month
		}

		remaining := row.Amount - row.Matched
		if remaining < 0 {
			remaining = 0
		}

		fee := ReminderCaseFee{
			FeeID:        row.ID,
			ChildID:      row.ChildID,
			ChildName:    childName,
			MemberNumber: memberNumber,
			FeeType:      row.FeeType,
			Year:         row.Year,
			Month:        month,
			DueDate:      row.DueDate,
			Amount:       row.Amount,
			Remaining:    roundCent(remaining),
			HasReminder:  hasReminder[row.ID],
		}
		fee.Status, fee.ActionableAt = feeWorkflowStatus(row.CreatedAt, row.DueDate, contacts[row.ID], cutoff, asOfStart)
		if contact, ok := contacts[row.ID]; ok {
			contactCopy := contact
			fee.LastContact = &contactCopy
		}
		fees = append(fees, fee)
	}

	sort.SliceStable(fees, func(i, j int) bool {
		if fees[i].DueDate.Equal(fees[j].DueDate) {
			return fees[i].FeeID.String() < fees[j].FeeID.String()
		}
		return fees[i].DueDate.Before(fees[j].DueDate)
	})

	return fees, nil
}

// feeWorkflowStatus derives the per-fee status and the next action date.
// asOfStart is the first moment of the reference day; a fee is overdue when
// its due date lies before that day.
func feeWorkflowStatus(createdAt, dueDate time.Time, contact FeeContact, cutoff, asOfStart time.Time) (ReminderCaseFeeStatus, time.Time) {
	overdue := dueDate.Before(asOfStart)

	if !contact.LastContactAt.IsZero() {
		deadline := defaultReminderDeadline(contact.RunDate)
		if !deadline.Before(asOfStart) {
			return FeeStatusWaiting, deadline
		}
		return FeeStatusActionableFinal, deadline
	}

	if !overdue {
		return FeeStatusNeverContacted, dueDate
	}
	if !cutoff.IsZero() && !createdAt.After(cutoff) {
		return FeeStatusHistoryUnknown, dueDate
	}
	return FeeStatusActionableInitial, dueDate
}

func hasActionableFee(fees []ReminderCaseFee) bool {
	for _, fee := range fees {
		switch fee.Status {
		case FeeStatusActionableInitial, FeeStatusActionableFinal, FeeStatusHistoryUnknown:
			return true
		}
	}
	return false
}

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
		RecommendedStage:    plan.recommendedStage,
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
// the preview.
func (s *ReminderService) SendReminderCase(ctx context.Context, householdID uuid.UUID, req *ReminderCaseRequest, sentBy *uuid.UUID) (*ReminderCaseSendResult, error) {
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

	// Persist planned reminder fees.
	createdAt := s.now().UTC()
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
			return nil, err
		}
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
			return nil, err
		}
	} else {
		if err := s.emailSender.SendTextEmailMulti(plan.recipients, subject, body); err != nil {
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
		CreatedReminderFees: plan.plannedFees,
	}, nil
}

// casePlan is the validated internal state shared by preview and send.
type casePlan struct {
	stage            ReminderStage
	runDate          time.Time
	deadline         time.Time
	includeQR        bool
	householdName    string
	recipients       []string
	firstNames       []string
	selectedFees     []ReminderCaseFee
	items            []reminderItem
	plannedFees      []ReminderCasePlannedFee
	childIDByFee     map[uuid.UUID]uuid.UUID
	recommendedStage ReminderStage
	warnings         []string
	totalAmount      float64
}

// prepareCasePlan loads and validates everything preview and send need.
// Concurrency conflicts (paid fees, foreign fees, reminder fees created after
// the preview) surface as CaseConflictError.
func (s *ReminderService) prepareCasePlan(ctx context.Context, householdID uuid.UUID, req *ReminderCaseRequest) (*casePlan, error) {
	if req == nil || (req.Stage != ReminderStageInitial && req.Stage != ReminderStageFinal) {
		return nil, ErrInvalidInput
	}
	if len(req.FeeIDs) == 0 {
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

	// Planned reminder fees: one per selected base fee without an existing
	// reminder fee. Reminder fees themselves never get reminder fees.
	existingReminders, err := s.feeRepo.GetOpenReminderBaseIDs(ctx, baseIDs)
	if err != nil {
		return nil, err
	}
	var planned []ReminderCasePlannedFee
	for _, fee := range selected {
		if fee.FeeType == domain.FeeTypeReminder || existingReminders[fee.FeeID] {
			continue
		}
		amount := reminderFeeAmountFor(fee.FeeType)
		if amount <= 0 {
			continue
		}
		planned = append(planned, ReminderCasePlannedFee{
			BaseFeeID:   fee.FeeID,
			BaseFeeType: fee.FeeType,
			BaseLabel:   feeTypeLabel(fee.FeeType),
			Amount:      amount,
			DueDate:     deadline,
		})
	}

	// Mail/QR items: selected fees with remaining amounts plus planned fees.
	items := make([]reminderItem, 0, len(selected)+len(planned))
	total := 0.0
	for _, fee := range selected {
		total += fee.Remaining
		items = append(items, reminderItem{
			FeeID:        fee.FeeID,
			ChildID:      fee.ChildID,
			ChildName:    fee.ChildName,
			MemberNumber: fee.MemberNumber,
			FeeType:      fee.FeeType,
			Amount:       fee.Remaining,
			Year:         fee.Year,
			Month:        fee.Month,
			DueDate:      fee.DueDate,
		})
	}
	for _, plannedFee := range planned {
		total += plannedFee.Amount
		baseFee := feeByID[plannedFee.BaseFeeID]
		baseType := plannedFee.BaseFeeType
		items = append(items, reminderItem{
			ChildID:     baseFee.ChildID,
			ChildName:   baseFee.ChildName,
			FeeType:     domain.FeeTypeReminder,
			Amount:      plannedFee.Amount,
			Year:        runDate.Year(),
			BaseFeeType: &baseType,
			BaseYear:    baseFee.Year,
			BaseMonth:   baseFee.Month,
			DueDate:     plannedFee.DueDate,
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
		stage:            req.Stage,
		runDate:          runDate,
		deadline:         deadline,
		includeQR:        includeQR,
		householdName:    household.Name,
		recipients:       collectEmails(parents),
		firstNames:       parentFirstNames(parents),
		selectedFees:     selected,
		items:            items,
		plannedFees:      planned,
		childIDByFee:     childIDByFee,
		recommendedStage: recommendedStageFor(selected),
		warnings:         buildCaseWarnings(req.Stage, selected, existingReminders),
		totalAmount:      roundCent(total),
	}, nil
}

func reminderFeeAmountFor(feeType domain.FeeType) float64 {
	if feeType == domain.FeeTypeMembership {
		return domain.MembershipReminderFeeAmount
	}
	return domain.ReminderFeeAmount
}

// recommendedStageFor returns the stage recommendation for a fee selection.
// Mixed stages or unknown history produce no recommendation.
func recommendedStageFor(selected []ReminderCaseFee) ReminderStage {
	if len(selected) == 0 {
		return ReminderStageNone
	}
	allInitial := true
	allFinal := true
	for _, fee := range selected {
		if fee.Status != FeeStatusActionableInitial {
			allInitial = false
		}
		if fee.Status != FeeStatusActionableFinal {
			allFinal = false
		}
	}
	if allInitial {
		return ReminderStageInitial
	}
	if allFinal {
		return ReminderStageFinal
	}
	return ReminderStageNone
}

// buildCaseWarnings creates concrete warnings for deviations from the
// recommended flow.
func buildCaseWarnings(stage ReminderStage, selected []ReminderCaseFee, existingReminders map[uuid.UUID]bool) []string {
	var warnings []string
	for _, fee := range selected {
		label := fmt.Sprintf("%s: %s", fee.ChildName, feeTypeLabel(fee.FeeType))
		switch fee.Status {
		case FeeStatusActionableInitial:
			if stage == ReminderStageFinal {
				warnings = append(warnings, fmt.Sprintf("%s wurde noch nicht erinnert", label))
			}
		case FeeStatusWaiting:
			if fee.LastContact != nil {
				deadline := defaultReminderDeadline(fee.LastContact.RunDate)
				warnings = append(warnings, fmt.Sprintf("%s wurde erst am %s kontaktiert, die Frist läuft bis %s", label, fee.LastContact.RunDate.Format("02.01.2006"), deadline.Format("02.01.2006")))
			}
		case FeeStatusHistoryUnknown:
			warnings = append(warnings, fmt.Sprintf("Für %s gibt es keine zuordenbare Historie", label))
		}
		if fee.FeeType != domain.FeeTypeReminder && existingReminders[fee.FeeID] {
			warnings = append(warnings, fmt.Sprintf("Für %s existiert bereits eine Mahngebühr", label))
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

func startOfDayUTC(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func roundCent(value float64) float64 {
	if value < 0 {
		return -float64(int(-value*100+0.5)) / 100
	}
	return float64(int(value*100+0.5)) / 100
}
