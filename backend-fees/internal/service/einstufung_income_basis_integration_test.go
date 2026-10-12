package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

type einstufungFixture struct {
	ctx            context.Context
	childRepo      *repository.PostgresChildRepository
	householdRepo  *repository.PostgresHouseholdRepository
	einstufungRepo *repository.PostgresEinstufungRepository
	fees           *service.FeeService
	service        *service.EinstufungService
}

func newEinstufungFixture(t *testing.T) *einstufungFixture {
	t.Helper()
	requireTestDatabase(t)
	cleanupTestData()
	t.Cleanup(cleanupTestData)

	childRepo := repository.NewPostgresChildRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	einstufungRepo := repository.NewPostgresEinstufungRepository(testDB)
	feeService := service.NewFeeService(
		repository.NewPostgresFeeRepository(testDB), childRepo, householdRepo,
		repository.NewPostgresMatchRepository(testDB), repository.NewPostgresTransactionRepository(testDB),
		repository.NewPostgresFeeScheduleRepository(testDB),
	)
	feeService.UseEinstufungen(einstufungRepo)
	return &einstufungFixture{
		ctx:            context.Background(),
		childRepo:      childRepo,
		householdRepo:  householdRepo,
		einstufungRepo: einstufungRepo,
		fees:           feeService,
		service: service.NewEinstufungService(einstufungRepo, householdRepo, childRepo, feeService,
			repository.NewTxManager(testDB)),
	}
}

// child creates a household with the given income status and one child.
func (f *einstufungFixture) child(t *testing.T, member string, status domain.IncomeStatus, birth, entry time.Time) (*domain.Child, *domain.Household) {
	t.Helper()
	household := &domain.Household{
		ID:               uuid.New(),
		Name:             "TEST Einkommensgrundlage " + member,
		IncomeStatus:     status,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := f.householdRepo.Create(f.ctx, household); err != nil {
		t.Fatal(err)
	}
	child := &domain.Child{
		ID:           uuid.New(),
		HouseholdID:  &household.ID,
		MemberNumber: member,
		FirstName:    "Test",
		LastName:     member,
		BirthDate:    birth,
		EntryDate:    entry,
		CareHours:    intPtr(45),
		IsActive:     true,
	}
	if err := f.childRepo.Create(f.ctx, child); err != nil {
		t.Fatal(err)
	}
	return child, household
}

func (f *einstufungFixture) household(t *testing.T, id uuid.UUID) *domain.Household {
	t.Helper()
	household, err := f.householdRepo.GetByID(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return household
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func requireValidationError(t *testing.T, err error, contains string) {
	t.Helper()
	var validation *service.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if !strings.Contains(validation.Message, contains) {
		t.Fatalf("message = %q, want it to contain %q", validation.Message, contains)
	}
}

func TestEinstufungIncomeBasis_NotRequiredOnlyForKindergartenChildren(t *testing.T) {
	f := newEinstufungFixture(t)

	krippe, _ := f.child(t, "TIBK", domain.IncomeStatusUnknown, date(2025, time.March, 1), date(2026, time.January, 1))
	_, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: krippe.ID, Year: 2026, IncomeBasis: domain.IncomeBasisNotRequired, ChildrenCount: 1,
	})
	requireValidationError(t, err, "Kindergartenkinder")

	kita, household := f.child(t, "TIBG", domain.IncomeStatusUnknown, date(2021, time.March, 1), date(2026, time.January, 1))
	created, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: kita.ID, Year: 2026, IncomeBasis: domain.IncomeBasisNotRequired, ChildrenCount: 1,
		IncomeCalculation: domain.HouseholdIncomeCalculation{Parent1: domain.IncomeDetails{AdvertisingCosts: 1500}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.AnnualNetIncome != nil || created.MonthlyChildcareFee != 0 || created.HighestRateVoluntary {
		t.Fatalf("created = income %v fee %.2f highest %v, want no income, free, not highest",
			created.AnnualNetIncome, created.MonthlyChildcareFee, created.HighestRateVoluntary)
	}
	if created.IncomeCalculation.HasIncomeData() || created.IncomeCalculation.Parent1.AdvertisingCosts != 0 {
		t.Fatalf("income calculation kept for NOT_REQUIRED: %+v", created.IncomeCalculation)
	}
	reloaded := f.household(t, household.ID)
	if reloaded.IncomeStatus != domain.IncomeStatusNotRequired || reloaded.AnnualHouseholdIncome != nil {
		t.Fatalf("household = %s/%v, want NOT_REQUIRED without income", reloaded.IncomeStatus, reloaded.AnnualHouseholdIncome)
	}
}

func TestEinstufungIncomeBasis_HouseholdMirrorsOpenEinstufung(t *testing.T) {
	f := newEinstufungFixture(t)
	child, household := f.child(t, "TIBH", domain.IncomeStatusUnknown, date(2025, time.March, 1), date(2026, time.January, 1))

	created, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.HighestRateVoluntary || created.AnnualNetIncome != nil {
		t.Fatalf("highest rate = %v income %v", created.HighestRateVoluntary, created.AnnualNetIncome)
	}
	reloaded := f.household(t, household.ID)
	if reloaded.IncomeStatus != domain.IncomeStatusMaxAccepted || reloaded.AnnualHouseholdIncome != nil ||
		reloaded.ChildrenCountForFees == nil || *reloaded.ChildrenCountForFees != 2 {
		t.Fatalf("household = %s/%v/%v, want MAX_ACCEPTED without income, 2 children",
			reloaded.IncomeStatus, reloaded.AnnualHouseholdIncome, reloaded.ChildrenCountForFees)
	}

	// Switching to income: PROVIDED with the computed income; the WK-Pauschale is capped at the wages.
	basis := domain.IncomeBasisIncome
	calc := domain.HouseholdIncomeCalculation{
		Parent1: domain.IncomeDetails{GrossIncome: 40000, Tax: 5000, AdvertisingCosts: 1230},
		Parent2: domain.IncomeDetails{AdvertisingCosts: 1230, ParentalBenefit: 6000},
	}
	updated, _, err := f.service.Update(f.ctx, created.ID, service.UpdateEinstufungInput{IncomeBasis: &basis, IncomeCalculation: &calc})
	if err != nil {
		t.Fatal(err)
	}
	if updated.HighestRateVoluntary || updated.AnnualNetIncome == nil || *updated.AnnualNetIncome != 39770 {
		t.Fatalf("updated = highest %v income %v, want 39770", updated.HighestRateVoluntary, updated.AnnualNetIncome)
	}
	reloaded = f.household(t, household.ID)
	if reloaded.IncomeStatus != domain.IncomeStatusProvided || reloaded.AnnualHouseholdIncome == nil ||
		*reloaded.AnnualHouseholdIncome != 39770 {
		t.Fatalf("household = %s/%v, want PROVIDED 39770", reloaded.IncomeStatus, reloaded.AnnualHouseholdIncome)
	}

	// A follow-up closes the first Einstufung; editing the closed one leaves the household alone.
	followUp, err := f.service.CreateFollowUp(f.ctx, created.ID, service.CreateFollowUpEinstufungInput{
		ChangeDate: date(2026, time.July, 1), IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	notes := "nachträglich korrigiert"
	if _, _, err := f.service.Update(f.ctx, created.ID, service.UpdateEinstufungInput{Notes: &notes}); err != nil {
		t.Fatal(err)
	}
	reloaded = f.household(t, household.ID)
	if reloaded.IncomeStatus != domain.IncomeStatusMaxAccepted {
		t.Fatalf("household after editing the closed Einstufung = %s, want MAX_ACCEPTED", reloaded.IncomeStatus)
	}

	// Deleting the follow-up reopens the source, which describes the household again.
	if err := f.service.Delete(f.ctx, created.ID); err == nil {
		t.Fatal("deleting an Einstufung with a follow-up must fail")
	} else {
		requireValidationError(t, err, "Folgeeinstufung")
	}
	if err := f.service.Delete(f.ctx, followUp.Einstufung.ID); err != nil {
		t.Fatal(err)
	}
	reloaded = f.household(t, household.ID)
	if reloaded.IncomeStatus != domain.IncomeStatusProvided || reloaded.AnnualHouseholdIncome == nil ||
		*reloaded.AnnualHouseholdIncome != 39770 {
		t.Fatalf("household after deleting the follow-up = %s/%v, want PROVIDED 39770",
			reloaded.IncomeStatus, reloaded.AnnualHouseholdIncome)
	}
}

func TestEinstufungIncomeBasis_FosterFamilyAndSiblingStatusKept(t *testing.T) {
	f := newEinstufungFixture(t)

	foster, fosterHousehold := f.child(t, "TIBF", domain.IncomeStatusFosterFamily, date(2025, time.March, 1), date(2026, time.January, 1))
	if _, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: foster.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if status := f.household(t, fosterHousehold.ID).IncomeStatus; status != domain.IncomeStatusFosterFamily {
		t.Fatalf("foster household status = %s", status)
	}

	// A Kindergarten child of a household with proven income keeps PROVIDED for its siblings.
	older, household := f.child(t, "TIBS", domain.IncomeStatusProvided, date(2021, time.March, 1), date(2026, time.January, 1))
	income := 42000.0
	household.AnnualHouseholdIncome = &income
	if err := f.householdRepo.Update(f.ctx, household); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: older.ID, Year: 2026, IncomeBasis: domain.IncomeBasisNotRequired, ChildrenCount: 2,
	}); err != nil {
		t.Fatal(err)
	}
	reloaded := f.household(t, household.ID)
	if reloaded.IncomeStatus != domain.IncomeStatusProvided || reloaded.AnnualHouseholdIncome == nil ||
		*reloaded.AnnualHouseholdIncome != 42000 {
		t.Fatalf("household = %s/%v, want PROVIDED 42000", reloaded.IncomeStatus, reloaded.AnnualHouseholdIncome)
	}
}

func TestEinstufungPreview_ComputesWithoutSaving(t *testing.T) {
	f := newEinstufungFixture(t)
	child, household := f.child(t, "TIBP", domain.IncomeStatusUnknown, date(2025, time.March, 1), date(2026, time.August, 17))

	preview, err := f.service.Preview(f.ctx, service.PreviewEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisIncome, ChildrenCount: 1,
		IncomeCalculation: domain.HouseholdIncomeCalculation{Parent1: domain.IncomeDetails{AdvertisingCosts: 1230}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Warnings) != 1 || !strings.Contains(preview.Warnings[0], "Keine Einkommensangaben") {
		t.Fatalf("warnings = %v, want the missing-income warning", preview.Warnings)
	}
	e := preview.Einstufung
	if !e.EffectiveFromMonth.Equal(date(2026, time.August, 1)) || e.AnnualNetIncome == nil || *e.AnnualNetIncome != 0 {
		t.Fatalf("preview = from %s income %v, want August 2026 with income 0", e.EffectiveFromMonth, e.AnnualNetIncome)
	}
	if len(e.MonthlyTable) == 0 || e.MonthlyTable[0].ChildcareFee != domain.ContributionAmountForMonth(e.MonthlyChildcareFee, child.EntryDate, 2026, time.August) {
		t.Fatalf("monthly table = %+v", e.MonthlyTable)
	}
	if preview.Income.Parent1AdvertisingCosts != 0 {
		t.Fatalf("applied WK-Pauschale without wages = %.2f, want 0", preview.Income.Parent1AdvertisingCosts)
	}
	saved, err := f.einstufungRepo.ListByHousehold(f.ctx, household.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 0 || f.household(t, household.ID).IncomeStatus != domain.IncomeStatusUnknown {
		t.Fatalf("preview saved data: %d Einstufungen", len(saved))
	}

	// The saved Einstufung matches the preview; a second one for the same period is rejected.
	created, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisIncome, ChildrenCount: 1,
		IncomeCalculation: domain.HouseholdIncomeCalculation{Parent1: domain.IncomeDetails{AdvertisingCosts: 1230}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.MonthlyChildcareFee != e.MonthlyChildcareFee || created.FeeRule != e.FeeRule {
		t.Fatalf("created %.2f/%s, preview %.2f/%s", created.MonthlyChildcareFee, created.FeeRule, e.MonthlyChildcareFee, e.FeeRule)
	}
	_, _, err = f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 1,
	})
	requireValidationError(t, err, "bereits eine Einstufung")

	// Follow-up preview: the change date must lie after the start of the source.
	_, err = f.service.Preview(f.ctx, service.PreviewEinstufungInput{
		SourceEinstufungID: &created.ID, ChangeDate: timePtr(date(2026, time.August, 3)),
		IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 1,
	})
	requireValidationError(t, err, "nach dem Beginn")
}

func timePtr(t time.Time) *time.Time { return &t }
