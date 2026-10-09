package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"sort"
	"strings"
	"time"
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
	MemberNumber string                `json:"memberNumber,omitempty" binding:"optional"`
	FeeType      domain.FeeType        `json:"feeType"`
	Year         int                   `json:"year"`
	Month        int                   `json:"month"`
	DueDate      time.Time             `json:"dueDate"`
	Amount       float64               `json:"amount"`
	Remaining    float64               `json:"remaining"`
	Status       ReminderCaseFeeStatus `json:"status"`
	ActionableAt time.Time             `json:"actionableAt"`
	LastContact  *FeeContact           `json:"lastContact,omitempty" binding:"optional"`
	HasReminder  bool                  `json:"hasReminder"`
	// ReminderFeeDue: a Mahngebühr is due by the rules (reminded before, the
	// reminder deadline has passed, no Mahngebühr yet). It is only created
	// when the fee is listed in ReminderCaseRequest.ReminderFeeIDs.
	ReminderFeeDue bool `json:"reminderFeeDue"`
	// ReminderForID and ReminderFor name the base fee of a Mahngebühr.
	ReminderForID *uuid.UUID     `json:"reminderForId,omitempty" binding:"optional"`
	ReminderFor   *domain.FeeRef `json:"reminderFor,omitempty" binding:"optional"`
	// ClubMember is the club member a membership fee (or its Mahngebühr)
	// belongs to; membership fees are owed per member, not per child.
	ClubMember *ReminderCaseMember `json:"clubMember,omitempty" binding:"optional"`
}

// ReminderCaseMember identifies the club member of a membership fee.
type ReminderCaseMember struct {
	ID           uuid.UUID `json:"id"`
	MemberNumber string    `json:"memberNumber"`
	Name         string    `json:"name"`
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
	RunDate time.Time   `json:"runDate"`
	FeeIDs  []uuid.UUID `json:"feeIds"`
	// ReminderFeeIDs are the selected fees that get a Mahngebühr on send.
	// Each must be part of FeeIDs and have ReminderFeeDue set. With at least
	// one, the mail is a Mahnung, otherwise a Zahlungserinnerung.
	ReminderFeeIDs []uuid.UUID `json:"reminderFeeIds,omitempty" binding:"optional"`
	IncludeQR      *bool       `json:"includeQR,omitempty" binding:"optional"`
	Subject        string      `json:"subject,omitempty" binding:"optional"`
	Body           string      `json:"body,omitempty" binding:"optional"`
	PreviewedAt    *time.Time  `json:"previewedAt,omitempty" binding:"optional"`
}

// ReminderCasePlannedFee is a reminder fee that would be created on send.
// BaseLabel names the base fee with period and person, e.g.
// "Essensgeld September 2026 (Haily)".
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
	QRImageDataURL      *string                  `json:"qrImageDataUrl,omitempty" binding:"optional"`
	QRPayload           string                   `json:"qrPayload,omitempty" binding:"optional"`
	SelectedFees        []ReminderCaseFee        `json:"selectedFees"`
	PlannedReminderFees []ReminderCasePlannedFee `json:"plannedReminderFees"`
	// Stage is derived: final (Mahnung) when Mahngebühren are planned.
	Stage    ReminderStage `json:"stage"`
	Warnings []string      `json:"warnings,omitempty" binding:"optional"`
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

		var totalCents int64
		nextAction := time.Time{}
		for _, fee := range fees {
			totalCents += domain.Cents(fee.Remaining)
			if nextAction.IsZero() || fee.ActionableAt.Before(nextAction) {
				nextAction = fee.ActionableAt
			}
		}

		cases = append(cases, ReminderCase{
			HouseholdID:    household.ID,
			HouseholdName:  household.Name,
			Recipients:     collectEmails(parents),
			TotalRemaining: domain.Euros(totalCents),
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

	// Base fees with any Mahngebühr, paid or open: a fee gets at most one.
	baseIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if row.FeeType != domain.FeeTypeReminder {
			baseIDs = append(baseIDs, row.ID)
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
		if row.FeeType == domain.FeeTypeReminder && row.ReminderForID != nil && row.BaseFeeType != nil {
			baseID := *row.ReminderForID
			fee.ReminderForID = &baseID
			fee.ReminderFor = &domain.FeeRef{FeeType: *row.BaseFeeType, Year: derefInt(row.BaseYear), Month: row.BaseMonth}
		}
		if row.ClubMemberID != nil {
			fee.ClubMember = &ReminderCaseMember{
				ID:           *row.ClubMemberID,
				MemberNumber: derefString(row.ClubMemberNumber),
				Name:         strings.TrimSpace(derefString(row.ClubMemberFirstName) + " " + derefString(row.ClubMemberLastName)),
			}
		}
		fee.Status, fee.ActionableAt = feeWorkflowStatus(row.CreatedAt, row.DueDate, contacts[row.ID], cutoff, asOfStart)
		fee.ReminderFeeDue = fee.FeeType != domain.FeeTypeReminder && !fee.HasReminder &&
			fee.Status == FeeStatusActionableFinal && reminderFeeAmountFor(fee.FeeType) > 0
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

	return placeRemindersAfterBase(fees), nil
}

// placeRemindersAfterBase moves each Mahngebühr directly behind its open base
// fee so the pair reads together; reminders without an open base keep their
// due-date position.
func placeRemindersAfterBase(fees []ReminderCaseFee) []ReminderCaseFee {
	open := make(map[uuid.UUID]bool, len(fees))
	for _, fee := range fees {
		open[fee.FeeID] = true
	}
	attached := make(map[uuid.UUID][]ReminderCaseFee)
	for _, fee := range fees {
		if fee.ReminderForID != nil && open[*fee.ReminderForID] {
			attached[*fee.ReminderForID] = append(attached[*fee.ReminderForID], fee)
		}
	}
	ordered := make([]ReminderCaseFee, 0, len(fees))
	for _, fee := range fees {
		if fee.ReminderForID != nil && open[*fee.ReminderForID] {
			continue
		}
		ordered = append(ordered, fee)
		ordered = append(ordered, attached[fee.FeeID]...)
	}
	return ordered
}

// caseFeeLabel names a fee with its period, e.g. "Essensgeld September 2026",
// "Vereinsbeitrag 2026" or "Mahngebühr für Essensgeld September 2026".
func caseFeeLabel(fee ReminderCaseFee) string {
	if fee.FeeType == domain.FeeTypeReminder && fee.ReminderFor != nil {
		return "Mahngebühr für " + feeRefLabel(fee.ReminderFor.FeeType, fee.ReminderFor.Year, derefInt(fee.ReminderFor.Month))
	}
	if fee.FeeType == domain.FeeTypeReminder {
		return feeTypeLabel(fee.FeeType)
	}
	return feeRefLabel(fee.FeeType, fee.Year, fee.Month)
}

// caseFeePerson is the person a fee belongs to: the club member for
// membership fees, otherwise the child.
func caseFeePerson(fee ReminderCaseFee) string {
	if fee.ClubMember != nil && fee.ClubMember.Name != "" {
		return fee.ClubMember.Name
	}
	return fee.ChildName
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// feeWorkflowStatus derives the per-fee status and the next action date.
// asOfStart is the first moment of the reference day; a fee is overdue when
// its due date lies before that day. Not-yet-due fees stay never_contacted
// regardless of earlier contacts: actionability is defined by the due date.
func feeWorkflowStatus(createdAt, dueDate time.Time, contact FeeContact, cutoff, asOfStart time.Time) (ReminderCaseFeeStatus, time.Time) {
	overdue := dueDate.Before(asOfStart)

	if !overdue {
		return FeeStatusNeverContacted, dueDate
	}

	if !contact.LastContactAt.IsZero() {
		deadline := defaultReminderDeadline(contact.RunDate)
		if !deadline.Before(asOfStart) {
			return FeeStatusWaiting, deadline
		}
		return FeeStatusActionableFinal, deadline
	}

	// Fees created on or before the reliability cutoff date have no
	// assignable history; compare calendar days, not exact timestamps.
	if !cutoff.IsZero() && !startOfDayUTC(createdAt).After(cutoff) {
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

func reminderFeeAmountFor(feeType domain.FeeType) float64 {
	if feeType == domain.FeeTypeMembership {
		return domain.MembershipReminderFeeAmount
	}
	return domain.ReminderFeeAmount
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
