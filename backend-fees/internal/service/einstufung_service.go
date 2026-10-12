package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// EinstufungService handles Einstufung-related business logic.
type EinstufungService struct {
	einstufungRepo repository.EinstufungRepository
	householdRepo  repository.HouseholdRepository
	childRepo      repository.ChildRepository
	feeService     *FeeService
	txm            *repository.TxManager
}

// NewEinstufungService creates a new Einstufung service. txm may be nil in tests.
func NewEinstufungService(
	einstufungRepo repository.EinstufungRepository,
	householdRepo repository.HouseholdRepository,
	childRepo repository.ChildRepository,
	feeService *FeeService,
	txm *repository.TxManager,
) *EinstufungService {
	return &EinstufungService{
		einstufungRepo: einstufungRepo,
		householdRepo:  householdRepo,
		childRepo:      childRepo,
		feeService:     feeService,
		txm:            txm,
	}
}

// CreateInput defines input for creating an Einstufung.
type CreateEinstufungInput struct {
	ChildID           uuid.UUID                         `json:"childId"`
	Year              int                               `json:"year"`
	IncomeCalculation domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	// IncomeBasis may be empty; it is then derived from HighestRateVoluntary.
	IncomeBasis          domain.IncomeBasis `json:"incomeBasis"`
	HighestRateVoluntary bool               `json:"highestRateVoluntary"`
	ChildrenCount        int                `json:"childrenCount"`
	Provisional          bool               `json:"provisional"`
	Notes                string             `json:"notes"`
}

// CreateFollowUpEinstufungInput defines input for creating a follow-up Einstufung.
type CreateFollowUpEinstufungInput struct {
	ChangeDate           time.Time                         `json:"changeDate"`
	IncomeCalculation    domain.HouseholdIncomeCalculation `json:"incomeCalculation"`
	IncomeBasis          domain.IncomeBasis                `json:"incomeBasis"`
	HighestRateVoluntary bool                              `json:"highestRateVoluntary"`
	ChildrenCount        int                               `json:"childrenCount"`
	Provisional          bool                              `json:"provisional"`
	Notes                string                            `json:"notes"`
}

// CreateFollowUpEinstufungResult includes the created Einstufung and automatic fee changes.
type CreateFollowUpEinstufungResult struct {
	Einstufung         *domain.Einstufung              `json:"einstufung" binding:"optional"`
	ExpectationChanges *ChildcareExpectationSyncResult `json:"expectationChanges" binding:"optional"`
}

// UpdateInput defines input for updating an Einstufung.
type UpdateEinstufungInput struct {
	IncomeCalculation    *domain.HouseholdIncomeCalculation `json:"incomeCalculation,omitempty" binding:"optional"`
	IncomeBasis          *domain.IncomeBasis                `json:"incomeBasis,omitempty" binding:"optional"`
	HighestRateVoluntary *bool                              `json:"highestRateVoluntary,omitempty" binding:"optional"`
	ChildrenCount        *int                               `json:"childrenCount,omitempty" binding:"optional"`
	Provisional          *bool                              `json:"provisional,omitempty" binding:"optional"`
	ValidFrom            *time.Time                         `json:"validFrom,omitempty" binding:"optional"`
	Notes                *string                            `json:"notes,omitempty" binding:"optional"`
}

// PreviewEinstufungInput describes an unsaved Einstufung form. EinstufungID
// previews an edit, SourceEinstufungID a follow-up, otherwise a new Einstufung
// for ChildID and Year.
type PreviewEinstufungInput struct {
	EinstufungID       *uuid.UUID
	SourceEinstufungID *uuid.UUID
	ChildID            uuid.UUID
	Year               int
	ChangeDate         *time.Time
	IncomeCalculation  domain.HouseholdIncomeCalculation
	IncomeBasis        domain.IncomeBasis
	ChildrenCount      int
	Provisional        bool
}

// EinstufungPreview is the computed, unsaved result of an Einstufung form.
type EinstufungPreview struct {
	Einstufung *domain.Einstufung
	Income     IncomeBreakdown
	// Warnings need a confirmation before saving.
	Warnings []string
	// Notices are informational.
	Notices []string
}

// IncomeBreakdown shows how the household income was derived.
type IncomeBreakdown struct {
	Parent1NetIncome         float64
	Parent2NetIncome         float64
	Parent1FeeRelevantIncome float64
	Parent2FeeRelevantIncome float64
	// Applied WK-Pauschale after the cap at the gross wages.
	Parent1AdvertisingCosts float64
	Parent2AdvertisingCosts float64
	HouseholdFeeIncome      float64
	HouseholdFullIncome     float64
}

// CalculateIncomeBreakdown derives the income figures of an income calculation sheet.
func CalculateIncomeBreakdown(calc domain.HouseholdIncomeCalculation) IncomeBreakdown {
	return IncomeBreakdown{
		Parent1NetIncome:         calc.Parent1.CalculateNetIncome(),
		Parent2NetIncome:         calc.Parent2.CalculateNetIncome(),
		Parent1FeeRelevantIncome: calc.Parent1.CalculateFeeRelevantIncome(),
		Parent2FeeRelevantIncome: calc.Parent2.CalculateFeeRelevantIncome(),
		Parent1AdvertisingCosts:  calc.Parent1.AppliedAdvertisingCosts(),
		Parent2AdvertisingCosts:  calc.Parent2.AppliedAdvertisingCosts(),
		HouseholdFeeIncome:       calc.CalculateAnnualNetIncome(),
		HouseholdFullIncome:      calc.CalculateFullNetIncome(),
	}
}

// einstufungDraft is an Einstufung with the records needed to compute it.
type einstufungDraft struct {
	einstufung *domain.Einstufung
	child      *domain.Child
	household  *domain.Household
	// source is the closed predecessor of a new follow-up.
	source *domain.Einstufung
}

// Create creates a new Einstufung, computing the fee from the income basis,
// mirroring it on the household and aligning generated CHILDCARE expectations.
func (s *EinstufungService) Create(ctx context.Context, input CreateEinstufungInput) (*domain.Einstufung, *ChildcareExpectationSyncResult, error) {
	draft, err := s.draftNew(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var changes *ChildcareExpectationSyncResult
	err = s.withTx(ctx, func(ctx context.Context) error {
		if err := s.einstufungRepo.Create(ctx, draft.einstufung); err != nil {
			return mapEinstufungRepoError(err)
		}
		if err := s.applyToHousehold(ctx, draft.household, draft.einstufung); err != nil {
			return err
		}
		changes, err = s.syncFees(ctx, draft.einstufung)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return draft.einstufung, changes, nil
}

// CreateFollowUp creates a follow-up Einstufung, closes the source period, and syncs CHILDCARE expectations.
func (s *EinstufungService) CreateFollowUp(ctx context.Context, sourceID uuid.UUID, input CreateFollowUpEinstufungInput) (*CreateFollowUpEinstufungResult, error) {
	draft, err := s.draftFollowUp(ctx, sourceID, input)
	if err != nil {
		return nil, err
	}
	e := draft.einstufung
	var changes *ChildcareExpectationSyncResult
	err = s.withTx(ctx, func(ctx context.Context) error {
		sourceValidUntil := e.EffectiveFromMonth.AddDate(0, 0, -1)
		if err := s.einstufungRepo.CreateFollowUp(ctx, draft.source.ID, sourceValidUntil, e); err != nil {
			return mapEinstufungRepoError(err)
		}
		if err := s.applyToHousehold(ctx, draft.household, e); err != nil {
			return err
		}
		changes, err = s.syncFees(ctx, e)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &CreateFollowUpEinstufungResult{
		Einstufung:         e,
		ExpectationChanges: changes,
	}, nil
}

// Update recalculates and updates an existing Einstufung and aligns generated
// CHILDCARE expectations of its period.
func (s *EinstufungService) Update(ctx context.Context, id uuid.UUID, input UpdateEinstufungInput) (*domain.Einstufung, *ChildcareExpectationSyncResult, error) {
	draft, err := s.draftUpdate(ctx, id, input)
	if err != nil {
		return nil, nil, err
	}
	var changes *ChildcareExpectationSyncResult
	err = s.withTx(ctx, func(ctx context.Context) error {
		if err := s.einstufungRepo.Update(ctx, draft.einstufung); err != nil {
			return mapEinstufungRepoError(err)
		}
		if err := s.applyToHousehold(ctx, draft.household, draft.einstufung); err != nil {
			return err
		}
		changes, err = s.syncFees(ctx, draft.einstufung)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return draft.einstufung, changes, nil
}

// syncFees fills the monthly table of e and aligns the generated CHILDCARE
// expectations of its months with it. Months from the third birthday on owe no
// childcare fee. e needs Child and Household.
func (s *EinstufungService) syncFees(ctx context.Context, e *domain.Einstufung) (*ChildcareExpectationSyncResult, error) {
	s.populateMonthlyTable(ctx, e)
	if e.MonthlyTable == nil {
		return nil, fmt.Errorf("Monatstabelle der Einstufung konnte nicht berechnet werden")
	}
	months := make([]ChildcareMonth, 0, len(e.MonthlyTable))
	for _, row := range e.MonthlyTable {
		amount := 0.0
		if row.CareType == formatEinstufungCareType(domain.ChildAgeTypeKrippe) {
			amount = row.ChildcareFee
		}
		months = append(months, ChildcareMonth{Year: row.Year, Month: row.Month, Amount: amount})
	}
	return s.feeService.SyncChildcareExpectations(ctx, e.ChildID, e.Child.HouseholdID, months)
}

// Preview computes an Einstufung form like Create, CreateFollowUp or Update
// would, without saving anything.
func (s *EinstufungService) Preview(ctx context.Context, input PreviewEinstufungInput) (*EinstufungPreview, error) {
	var draft *einstufungDraft
	var err error
	switch {
	case input.EinstufungID != nil:
		calc := input.IncomeCalculation
		basis := input.IncomeBasis
		childrenCount := input.ChildrenCount
		provisional := input.Provisional
		draft, err = s.draftUpdate(ctx, *input.EinstufungID, UpdateEinstufungInput{
			IncomeCalculation: &calc,
			IncomeBasis:       &basis,
			ChildrenCount:     &childrenCount,
			Provisional:       &provisional,
			ValidFrom:         input.ChangeDate,
		})
	case input.SourceEinstufungID != nil:
		if input.ChangeDate == nil {
			return nil, newValidationError("Bitte ein Änderungsdatum angeben.")
		}
		draft, err = s.draftFollowUp(ctx, *input.SourceEinstufungID, CreateFollowUpEinstufungInput{
			ChangeDate:        *input.ChangeDate,
			IncomeCalculation: input.IncomeCalculation,
			IncomeBasis:       input.IncomeBasis,
			ChildrenCount:     input.ChildrenCount,
			Provisional:       input.Provisional,
		})
	default:
		draft, err = s.draftNew(ctx, CreateEinstufungInput{
			ChildID:           input.ChildID,
			Year:              input.Year,
			IncomeCalculation: input.IncomeCalculation,
			IncomeBasis:       input.IncomeBasis,
			ChildrenCount:     input.ChildrenCount,
			Provisional:       input.Provisional,
		})
	}
	if err != nil {
		return nil, err
	}

	e := draft.einstufung
	s.populateMonthlyTable(ctx, e)
	preview := &EinstufungPreview{
		Einstufung: e,
		Income:     CalculateIncomeBreakdown(input.IncomeCalculation),
		Warnings:   []string{},
		Notices:    []string{},
	}
	if e.IncomeBasis == domain.IncomeBasisIncome && e.CareType == domain.ChildAgeTypeKrippe {
		if !input.IncomeCalculation.HasIncomeData() {
			preview.Warnings = append(preview.Warnings,
				"Keine Einkommensangaben eingetragen: Das Einkommen wird mit 0 € angesetzt. "+
					"Ohne Nachweise bitte „Höchstsatz“ wählen oder die Einstufung als vorläufig markieren.")
		} else if *e.AnnualNetIncome < 0 {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf(
				"Das berechnete Einkommen ist negativ (%s). Bitte die Angaben prüfen.", formatCurrencyEUR(*e.AnnualNetIncome)))
		}
	}
	if e.CareType == domain.ChildAgeTypeKindergarten && e.IncomeBasis != domain.IncomeBasisNotRequired {
		preview.Notices = append(preview.Notices,
			"Das Kind ist ab Beginn Kindergartenkind, die Betreuung ist beitragsfrei. "+
				"Einkommensangaben sind nicht erforderlich.")
	}
	if e.IncomeBasis == domain.IncomeBasisIncome {
		for i, parent := range []domain.IncomeDetails{input.IncomeCalculation.Parent1, input.IncomeCalculation.Parent2} {
			if parent.AdvertisingCosts > 0 && parent.AppliedAdvertisingCosts() < parent.AdvertisingCosts {
				preview.Notices = append(preview.Notices, fmt.Sprintf(
					"Elternteil %d: Werbungskosten-Pauschale auf das Bruttoeinkommen begrenzt (%s statt %s).",
					i+1, formatCurrencyEUR(parent.AppliedAdvertisingCosts()), formatCurrencyEUR(parent.AdvertisingCosts)))
			}
		}
	}
	return preview, nil
}

// draftNew builds an unsaved initial Einstufung. It starts on January 1 of the
// year or on the child's entry date.
func (s *EinstufungService) draftNew(ctx context.Context, input CreateEinstufungInput) (*einstufungDraft, error) {
	child, household, err := s.childWithHousehold(ctx, input.ChildID)
	if err != nil {
		return nil, err
	}

	classificationYear := input.Year
	if classificationYear == 0 {
		classificationYear = child.EntryDate.Year()
	}
	changeDate := time.Date(classificationYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	effectiveFromMonth := changeDate
	if child.EntryDate.After(changeDate) {
		changeDate = child.EntryDate
		effectiveFromMonth = firstDayOfMonth(child.EntryDate)
	}

	childrenCount := input.ChildrenCount
	if childrenCount == 0 {
		childrenCount = 0
		for _, c := range household.Children {
			if c.IsEnrolledAt(effectiveFromMonth) {
				childrenCount++
			}
		}
		if childrenCount == 0 {
			childrenCount = 1
		}
	}

	basis, err := resolveIncomeBasis(input.IncomeBasis, input.HighestRateVoluntary)
	if err != nil {
		return nil, err
	}
	draft := &einstufungDraft{
		child:     child,
		household: household,
		einstufung: &domain.Einstufung{
			ID:                 uuid.New(),
			ChildID:            child.ID,
			HouseholdID:        household.ID,
			Year:               effectiveFromMonth.Year(),
			ValidFrom:          effectiveFromMonth,
			ChangeDate:         &changeDate,
			EffectiveFromMonth: effectiveFromMonth,
			IncomeCalculation:  input.IncomeCalculation,
			IncomeBasis:        basis,
			ChildrenCount:      childrenCount,
			Provisional:        input.Provisional,
			Notes:              input.Notes,
		},
	}
	if err := s.calculate(ctx, draft); err != nil {
		return nil, err
	}
	draft.einstufung.AnnualMembershipFee = s.membershipFee(ctx, effectiveFromMonth)
	return draft, nil
}

// draftFollowUp builds an unsaved follow-up of sourceID. The change date is
// subject to the 14th/15th cut-off rule.
func (s *EinstufungService) draftFollowUp(ctx context.Context, sourceID uuid.UUID, input CreateFollowUpEinstufungInput) (*einstufungDraft, error) {
	source, err := s.einstufungRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, ErrNotFound
	}

	effectiveFromMonth := CalculateEffectiveFromMonth(input.ChangeDate)
	if !source.EffectiveFromMonth.IsZero() && !effectiveFromMonth.After(source.EffectiveFromMonth) {
		return nil, newValidationError("Die Folgeeinstufung muss nach dem Beginn der bisherigen Einstufung (%s) wirksam werden.",
			formatMonthYear(source.EffectiveFromMonth))
	}
	if source.ValidUntil != nil && effectiveFromMonth.After(*source.ValidUntil) {
		return nil, newValidationError("Die bisherige Einstufung endet bereits am %s.", source.ValidUntil.Format("02.01.2006"))
	}

	child, household, err := s.childWithHousehold(ctx, source.ChildID)
	if err != nil {
		return nil, err
	}

	childrenCount := input.ChildrenCount
	if childrenCount == 0 {
		childrenCount = source.ChildrenCount
	}
	basis, err := resolveIncomeBasis(input.IncomeBasis, input.HighestRateVoluntary)
	if err != nil {
		return nil, err
	}
	changeDate := input.ChangeDate
	draft := &einstufungDraft{
		child:     child,
		household: household,
		source:    source,
		einstufung: &domain.Einstufung{
			ID:                 uuid.New(),
			ChildID:            source.ChildID,
			HouseholdID:        source.HouseholdID,
			Year:               effectiveFromMonth.Year(),
			ValidFrom:          effectiveFromMonth,
			SourceEinstufungID: &source.ID,
			ChangeDate:         &changeDate,
			EffectiveFromMonth: effectiveFromMonth,
			IncomeCalculation:  input.IncomeCalculation,
			IncomeBasis:        basis,
			ChildrenCount:      childrenCount,
			Provisional:        input.Provisional,
			Notes:              input.Notes,
		},
	}
	if err := s.calculate(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}

// draftUpdate applies input to a stored Einstufung and recalculates it.
func (s *EinstufungService) draftUpdate(ctx context.Context, id uuid.UUID, input UpdateEinstufungInput) (*einstufungDraft, error) {
	existing, err := s.einstufungRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if input.IncomeCalculation != nil {
		existing.IncomeCalculation = *input.IncomeCalculation
	}
	switch {
	case input.IncomeBasis != nil && *input.IncomeBasis != "":
		existing.IncomeBasis = *input.IncomeBasis
	case input.HighestRateVoluntary != nil && *input.HighestRateVoluntary:
		existing.IncomeBasis = domain.IncomeBasisHighestRate
	case input.HighestRateVoluntary != nil && existing.IncomeBasis == domain.IncomeBasisHighestRate:
		existing.IncomeBasis = domain.IncomeBasisIncome
	}
	if !existing.IncomeBasis.Valid() {
		return nil, newValidationError("Unbekannte Einkommensgrundlage „%s“.", existing.IncomeBasis)
	}
	if input.ChildrenCount != nil {
		existing.ChildrenCount = *input.ChildrenCount
	}
	if input.Provisional != nil {
		existing.Provisional = *input.Provisional
	}
	if input.ValidFrom != nil && existing.SourceEinstufungID != nil {
		changeDate := *input.ValidFrom
		existing.ChangeDate = &changeDate
		existing.EffectiveFromMonth = CalculateEffectiveFromMonth(changeDate)
		existing.ValidFrom = existing.EffectiveFromMonth
		existing.Year = existing.EffectiveFromMonth.Year()
	}
	if input.Notes != nil {
		existing.Notes = *input.Notes
	}
	if existing.ChildrenCount < 1 {
		return nil, newValidationError("Die Anzahl der Kinder muss mindestens 1 sein.")
	}

	child, err := s.childRepo.GetByID(ctx, existing.ChildID)
	if err != nil {
		return nil, ErrNotFound
	}
	household, err := s.householdRepo.GetWithMembers(ctx, existing.HouseholdID)
	if err != nil {
		return nil, ErrNotFound
	}
	draft := &einstufungDraft{einstufung: existing, child: child, household: household}
	if err := s.calculate(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}

// calculate derives care type, care hours, income and fees of a draft from its
// income basis, child data and the fee schedule valid in the first month.
func (s *EinstufungService) calculate(ctx context.Context, d *einstufungDraft) error {
	e := d.einstufung
	start := e.EffectiveFromMonth

	e.CareType = domain.ChildAgeTypeKrippe
	if !d.child.IsUnderThreeForEntireMonth(start.Year(), start.Month()) {
		e.CareType = domain.ChildAgeTypeKindergarten
	}
	if e.IncomeBasis == domain.IncomeBasisNotRequired && e.CareType != domain.ChildAgeTypeKindergarten {
		return newValidationError("„Einkommen nicht erforderlich“ gilt nur für Kinder, die ab %s Kindergartenkinder sind. "+
			"Für Krippenkinder bitte Einkommen oder Höchstsatz wählen.", formatMonthYear(start))
	}
	if e.ChildrenCount < 1 {
		return newValidationError("Die Anzahl der Kinder muss mindestens 1 sein.")
	}

	// Care hours are child data and are always resolved from the history.
	startMonth := int(start.Month())
	e.CareHoursPerWeek = s.feeService.ResolveCareHours(ctx, d.child, start.Year(), &startMonth)

	e.HighestRateVoluntary = e.IncomeBasis == domain.IncomeBasisHighestRate
	e.AnnualNetIncome = nil
	if e.IncomeBasis == domain.IncomeBasisIncome {
		income := e.IncomeCalculation.CalculateAnnualNetIncome()
		e.AnnualNetIncome = &income
	} else {
		// Income is not part of the decision; do not keep stale form values.
		e.IncomeCalculation = domain.HouseholdIncomeCalculation{}
	}

	schedule, err := s.feeService.ScheduleAt(ctx, start)
	if err != nil {
		return err
	}
	feeResult := schedule.Config.CalculateChildcareFee(einstufungFeeInput(e, d.household, e.CareType, e.CareHoursPerWeek))
	e.MonthlyChildcareFee = feeResult.Fee
	e.MonthlyFoodFee = schedule.Config.MonthlyFoodFee
	e.FeeRule = feeResult.Rule
	e.DiscountPercent = feeResult.DiscountPercent
	e.DiscountFactor = feeResult.DiscountFactor
	e.BaseFee = feeResult.BaseFee

	e.Child = d.child
	e.Household = d.household
	return nil
}

// einstufungFeeInput is the fee calculation input of e for one month.
func einstufungFeeInput(e *domain.Einstufung, household *domain.Household, careType domain.ChildAgeType, careHours int) domain.ChildcareFeeInput {
	input := domain.ChildcareFeeInput{
		ChildAgeType:  careType,
		SiblingsCount: e.ChildrenCount,
		CareHours:     careHours,
		HighestRate:   e.IncomeBasis == domain.IncomeBasisHighestRate || e.HighestRateVoluntary,
		FosterFamily:  household != nil && household.IncomeStatus == domain.IncomeStatusFosterFamily,
	}
	if e.AnnualNetIncome != nil {
		input.NetIncome = *e.AnnualNetIncome
	}
	return input
}

func (s *EinstufungService) membershipFee(ctx context.Context, month time.Time) float64 {
	schedule, err := s.feeService.ScheduleAt(ctx, month)
	if err != nil {
		return 0
	}
	return schedule.Config.AnnualMembershipFee
}

func (s *EinstufungService) childWithHousehold(ctx context.Context, childID uuid.UUID) (*domain.Child, *domain.Household, error) {
	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	if child.HouseholdID == nil {
		return nil, nil, newValidationError("%s %s ist keinem Haushalt zugeordnet.", child.FirstName, child.LastName)
	}
	household, err := s.householdRepo.GetWithMembers(ctx, *child.HouseholdID)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	return child, household, nil
}

// applyToHousehold mirrors an open Einstufung on its household. The household
// status and income drive fee generation for months without an Einstufung.
// Closed (historic) Einstufungen leave the household alone; foster families
// keep their status.
func (s *EinstufungService) applyToHousehold(ctx context.Context, household *domain.Household, e *domain.Einstufung) error {
	if e.ValidUntil != nil {
		return nil
	}
	childrenCount := e.ChildrenCount
	household.ChildrenCountForFees = &childrenCount
	if household.IncomeStatus != domain.IncomeStatusFosterFamily {
		switch e.IncomeBasis {
		case domain.IncomeBasisIncome:
			income := *e.AnnualNetIncome
			household.IncomeStatus = domain.IncomeStatusProvided
			household.AnnualHouseholdIncome = &income
		case domain.IncomeBasisHighestRate:
			household.IncomeStatus = domain.IncomeStatusMaxAccepted
			household.AnnualHouseholdIncome = nil
		case domain.IncomeBasisNotRequired:
			// Siblings may still need the household income; only replace an unknown status.
			switch household.IncomeStatus {
			case domain.IncomeStatusUnknown, domain.IncomeStatusPending, domain.IncomeStatusNotRequired:
				household.IncomeStatus = domain.IncomeStatusNotRequired
				household.AnnualHouseholdIncome = nil
			}
		}
	}
	return s.householdRepo.Update(ctx, household)
}

func (s *EinstufungService) withTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.txm.WithTx(ctx, fn)
}

// resolveIncomeBasis accepts the explicit basis or derives it from the
// deprecated highestRateVoluntary flag.
func resolveIncomeBasis(basis domain.IncomeBasis, highestRate bool) (domain.IncomeBasis, error) {
	if basis == "" {
		if highestRate {
			return domain.IncomeBasisHighestRate, nil
		}
		return domain.IncomeBasisIncome, nil
	}
	if !basis.Valid() {
		return "", newValidationError("Unbekannte Einkommensgrundlage „%s“.", basis)
	}
	return basis, nil
}

func mapEinstufungRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrEinstufungOverlap):
		return newValidationError("Für dieses Kind gibt es in diesem Zeitraum bereits eine Einstufung. " +
			"Bitte stattdessen eine Folgeeinstufung erstellen.")
	case errors.Is(err, repository.ErrEinstufungHasSuccessor):
		return newValidationError("Die Einstufung hat eine Folgeeinstufung. Bitte zuerst die Folgeeinstufung löschen.")
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	}
	return err
}

func formatMonthYear(t time.Time) string {
	return fmt.Sprintf("%s %d", germanMonthName(int(t.Month())), t.Year())
}

// GetByID returns an Einstufung by ID with loaded relations.
func (s *EinstufungService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Einstufung, error) {
	e, err := s.einstufungRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	s.loadRelations(ctx, e)
	return e, nil
}

// GetByChildAndYear returns the Einstufung for a child in a given year.
func (s *EinstufungService) GetByChildAndYear(ctx context.Context, childID uuid.UUID, year int) (*domain.Einstufung, error) {
	e, err := s.einstufungRepo.GetByChildAndYear(ctx, childID, year)
	if err != nil {
		return nil, ErrNotFound
	}
	s.loadRelations(ctx, e)
	return e, nil
}

// ListByHousehold returns all Einstufungen for a household.
func (s *EinstufungService) ListByHousehold(ctx context.Context, householdID uuid.UUID) ([]domain.Einstufung, error) {
	results, err := s.einstufungRepo.ListByHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	for i := range results {
		s.loadRelations(ctx, &results[i])
	}
	return results, nil
}

// ListByYear returns the Einstufungen of a year matching the filter, with pagination.
func (s *EinstufungService) ListByYear(ctx context.Context, filter repository.EinstufungListFilter, offset, limit int) ([]domain.Einstufung, int64, error) {
	if !slices.Contains(repository.EinstufungListStatuses, filter.Status) {
		return nil, 0, newValidationError("Unbekannter Status „%s“.", filter.Status)
	}
	results, total, err := s.einstufungRepo.ListByYear(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	for i := range results {
		s.loadRelations(ctx, &results[i])
	}
	return results, total, nil
}

// MissingEinstufung is a child that owes childcare fees in Month but has no
// Einstufung for it.
type MissingEinstufung struct {
	ChildID       uuid.UUID
	FirstName     string
	LastName      string
	MemberNumber  string
	HouseholdName *string
	EntryDate     time.Time
	Month         time.Time
	// PreviousUntil is the end of the child's last Einstufung, if any.
	PreviousUntil *time.Time
}

// missingLookahead is how far ahead children starting later are listed.
const missingLookahead = 3

// ListMissing lists U3 children without an Einstufung for the current month,
// or for their entry month when they start within the next three months.
func (s *EinstufungService) ListMissing(ctx context.Context) ([]MissingEinstufung, error) {
	currentMonth := firstDayOfMonth(util.Today())
	rows, err := s.einstufungRepo.ListWithoutEinstufung(ctx, currentMonth, currentMonth.AddDate(0, missingLookahead+1, 0))
	if err != nil {
		return nil, err
	}
	out := []MissingEinstufung{}
	for _, row := range rows {
		child := domain.Child{BirthDate: row.BirthDate}
		if !child.IsUnderThreeForEntireMonth(row.ReferenceMonth.Year(), row.ReferenceMonth.Month()) {
			continue
		}
		out = append(out, MissingEinstufung{ChildID: row.ChildID, FirstName: row.FirstName, LastName: row.LastName,
			MemberNumber: row.MemberNumber, HouseholdName: row.HouseholdName, EntryDate: row.EntryDate,
			Month: row.ReferenceMonth, PreviousUntil: row.LatestValidTo})
	}
	return out, nil
}

// Delete deletes an Einstufung. Deleting a follow-up reopens its source, which
// then again describes the household.
func (s *EinstufungService) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.einstufungRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	return s.withTx(ctx, func(ctx context.Context) error {
		if err := s.einstufungRepo.Delete(ctx, id); err != nil {
			return mapEinstufungRepoError(err)
		}
		if existing.SourceEinstufungID == nil || existing.ValidUntil != nil {
			return nil
		}
		source, err := s.einstufungRepo.GetByID(ctx, *existing.SourceEinstufungID)
		if err != nil {
			return mapEinstufungRepoError(err)
		}
		household, err := s.householdRepo.GetByID(ctx, source.HouseholdID)
		if err != nil {
			return ErrNotFound
		}
		if err := s.applyToHousehold(ctx, household, source); err != nil {
			return err
		}
		// The reopened source covers the follow-up's months again.
		s.loadRelations(ctx, source)
		changes, err := s.syncFees(ctx, source)
		if err != nil {
			return err
		}
		if len(changes.CreditReviewRequired) > 0 {
			log.Warn().Str("einstufungId", source.ID.String()).Int("months", len(changes.CreditReviewRequired)).
				Msg("deleting follow-up Einstufung leaves paid months for credit review")
		}
		return nil
	})
}

// GetLatestForChild returns the most recent Einstufung for a child.
func (s *EinstufungService) GetLatestForChild(ctx context.Context, childID uuid.UUID) (*domain.Einstufung, error) {
	e, err := s.einstufungRepo.GetLatestForChild(ctx, childID)
	if err != nil {
		return nil, ErrNotFound
	}
	s.loadRelations(ctx, e)
	return e, nil
}

// CalculateEffectiveFromMonth applies the 14th/15th monthly cut-off rule.
func CalculateEffectiveFromMonth(changeDate time.Time) time.Time {
	monthStart := firstDayOfMonth(changeDate)
	if changeDate.Day() <= 14 {
		return monthStart
	}
	return monthStart.AddDate(0, 1, 0)
}

// loadRelations loads child and household relations for an Einstufung.
func (s *EinstufungService) loadRelations(ctx context.Context, e *domain.Einstufung) {
	if child, err := s.childRepo.GetByID(ctx, e.ChildID); err == nil {
		e.Child = child
	}
	if household, err := s.householdRepo.GetWithMembers(ctx, e.HouseholdID); err == nil {
		e.Household = household
	}
	s.populateMonthlyTable(ctx, e)
}

// populateMonthlyTable derives the contribution timeline from child data. Care-hour
// changes never require another Einstufung; only a new income decision does.
func (s *EinstufungService) populateMonthlyTable(ctx context.Context, e *domain.Einstufung) {
	if e.Child == nil || e.Household == nil {
		return
	}

	start := firstDayOfMonth(e.EffectiveFromMonth)
	if e.EffectiveFromMonth.IsZero() {
		start = firstDayOfMonth(e.ValidFrom)
	}
	end := start
	if e.ValidUntil != nil {
		end = firstDayOfMonth(*e.ValidUntil)
	} else if e.Child.ExitDate != nil {
		end = firstDayOfMonth(*e.Child.ExitDate)
	} else {
		feeFreeMonth := firstDayOfMonth(e.Child.BirthDate.AddDate(3, 0, 0))
		if feeFreeMonth.After(end) {
			end = feeFreeMonth
		}
	}

	schedules, err := s.feeService.Schedules(ctx)
	if err != nil {
		log.Warn().Err(err).Str("einstufungId", e.ID.String()).Msg("cannot load fee schedules for monthly table")
		return
	}

	rows := make([]domain.EinstufungMonthRow, 0)
	for current := start; !current.After(end); current = current.AddDate(0, 1, 0) {
		schedule, err := schedules.At(current)
		if err != nil {
			log.Warn().Err(err).Str("einstufungId", e.ID.String()).Msg("no fee schedule for month")
			return
		}
		month := int(current.Month())
		careHours := s.feeService.ResolveCareHours(ctx, e.Child, current.Year(), &month)
		careType := domain.ChildAgeTypeKrippe
		if !e.Child.IsUnderThreeForEntireMonth(current.Year(), current.Month()) {
			careType = domain.ChildAgeTypeKindergarten
		}
		feeResult := schedule.Config.CalculateChildcareFee(einstufungFeeInput(e, e.Household, careType, careHours))
		row := domain.EinstufungMonthRow{
			Month:            month,
			Year:             current.Year(),
			CareHoursPerWeek: careHours,
			CareType:         formatEinstufungCareType(careType),
			ChildcareFee:     domain.ContributionAmountForMonth(feeResult.Fee, e.Child.EntryDate, current.Year(), current.Month()),
			FoodFee:          domain.ContributionAmountForMonth(schedule.Config.MonthlyFoodFee, e.Child.EntryDate, current.Year(), current.Month()),
		}
		if current.Equal(start) {
			row.MembershipFee = e.AnnualMembershipFee
		}
		rows = append(rows, row)
	}
	e.MonthlyTable = rows
}

func formatEinstufungCareType(careType domain.ChildAgeType) string {
	if careType == domain.ChildAgeTypeKindergarten {
		return "Kindergarten"
	}
	return "Krippe"
}
