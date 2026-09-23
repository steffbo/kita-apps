package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestEinstufungPeriods_AllowConsecutiveRejectOverlapping(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	einstufungRepo := repository.NewPostgresEinstufungRepository(testDB)

	child, err := createTestChild(childRepo, "EP")
	if err != nil {
		t.Fatal(err)
	}
	household := &domain.Household{
		ID:               uuid.New(),
		Name:             "TEST Einstufung Periods",
		IncomeStatus:     domain.IncomeStatusProvided,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := householdRepo.Create(ctx, household); err != nil {
		t.Fatal(err)
	}
	child.HouseholdID = &household.ID
	if err := childRepo.Update(ctx, child); err != nil {
		t.Fatal(err)
	}

	firstStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	firstEnd := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	first := testEinstufung(child.ID, household.ID, firstStart)
	first.ValidUntil = &firstEnd
	if err := einstufungRepo.Create(ctx, first); err != nil {
		t.Fatalf("expected first period to be created: %v", err)
	}

	secondStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	second := testEinstufung(child.ID, household.ID, secondStart)
	if err := einstufungRepo.Create(ctx, second); err != nil {
		t.Fatalf("expected consecutive period to be created: %v", err)
	}

	overlapStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	overlapping := testEinstufung(child.ID, household.ID, overlapStart)
	if err := einstufungRepo.Create(ctx, overlapping); err == nil {
		t.Fatalf("expected overlapping period to be rejected")
	}
}

func TestEinstufungPeriods_UpdateFollowUpMovesSourceBoundary(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	einstufungRepo := repository.NewPostgresEinstufungRepository(testDB)

	child, err := createTestChild(childRepo, "EPU")
	if err != nil {
		t.Fatal(err)
	}
	household := &domain.Household{
		ID:               uuid.New(),
		Name:             "TEST Einstufung Update",
		IncomeStatus:     domain.IncomeStatusProvided,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := householdRepo.Create(ctx, household); err != nil {
		t.Fatal(err)
	}
	child.HouseholdID = &household.ID
	if err := childRepo.Update(ctx, child); err != nil {
		t.Fatal(err)
	}

	sourceStart := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	source := testEinstufung(child.ID, household.ID, sourceStart)
	if err := einstufungRepo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}

	followUpStart := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	followUp := testEinstufung(child.ID, household.ID, followUpStart)
	followUp.SourceEinstufungID = &source.ID
	if err := einstufungRepo.CreateFollowUp(ctx, source.ID, followUpStart.AddDate(0, 0, -1), followUp); err != nil {
		t.Fatal(err)
	}

	movedStart := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	followUp.ValidFrom = movedStart
	followUp.EffectiveFromMonth = movedStart
	followUp.ChangeDate = &movedStart
	if err := einstufungRepo.Update(ctx, followUp); err != nil {
		t.Fatalf("move follow-up: %v", err)
	}

	reloadedSource, err := einstufungRepo.GetByID(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedSource.ValidUntil == nil || !reloadedSource.ValidUntil.Equal(time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("source valid_until = %v, want 2026-09-30", reloadedSource.ValidUntil)
	}
}

func TestEinstufungPeriods_DeleteFollowUpReopensSource(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	einstufungRepo := repository.NewPostgresEinstufungRepository(testDB)

	child, err := createTestChild(childRepo, "EPD")
	if err != nil {
		t.Fatal(err)
	}
	household := &domain.Household{
		ID:               uuid.New(),
		Name:             "TEST Einstufung Delete",
		IncomeStatus:     domain.IncomeStatusProvided,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := householdRepo.Create(ctx, household); err != nil {
		t.Fatal(err)
	}
	child.HouseholdID = &household.ID
	if err := childRepo.Update(ctx, child); err != nil {
		t.Fatal(err)
	}

	source := testEinstufung(child.ID, household.ID, time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))
	if err := einstufungRepo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	followUpStart := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	followUp := testEinstufung(child.ID, household.ID, followUpStart)
	followUp.SourceEinstufungID = &source.ID
	if err := einstufungRepo.CreateFollowUp(ctx, source.ID, followUpStart.AddDate(0, 0, -1), followUp); err != nil {
		t.Fatal(err)
	}

	if err := einstufungRepo.Delete(ctx, followUp.ID); err != nil {
		t.Fatalf("delete follow-up: %v", err)
	}
	reloadedSource, err := einstufungRepo.GetByID(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedSource.ValidUntil != nil {
		t.Fatalf("source valid_until = %v, want open period", reloadedSource.ValidUntil)
	}
}

func TestEinstufungMonthlyTableUsesChildCareHoursHistory(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	einstufungRepo := repository.NewPostgresEinstufungRepository(testDB)
	feeService := service.NewFeeService(nil, childRepo, householdRepo, nil, nil, repository.NewPostgresFeeScheduleRepository(testDB))
	einstufungService := service.NewEinstufungService(einstufungRepo, householdRepo, childRepo, feeService)

	household := &domain.Household{
		ID:               uuid.New(),
		Name:             "TEST Einstufung Timeline",
		IncomeStatus:     domain.IncomeStatusMaxAccepted,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := householdRepo.Create(ctx, household); err != nil {
		t.Fatal(err)
	}
	child := &domain.Child{
		ID:           uuid.New(),
		HouseholdID:  &household.ID,
		MemberNumber: "TEPT",
		FirstName:    "Test",
		LastName:     "Timeline",
		BirthDate:    time.Date(2025, time.June, 10, 0, 0, 0, 0, time.UTC),
		EntryDate:    time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC),
		CareHours:    intPtr(30),
		IsActive:     true,
	}
	if err := childRepo.Create(ctx, child); err != nil {
		t.Fatal(err)
	}
	if err := childRepo.UpsertCareHoursHistory(ctx, child.ID, intPtr(40), time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	e := testEinstufung(child.ID, household.ID, time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))
	e.HighestRateVoluntary = true
	e.ChildrenCount = 2
	if err := einstufungRepo.Create(ctx, e); err != nil {
		t.Fatal(err)
	}
	loaded, err := einstufungService.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}

	assertRow := func(year, month, hours int, childcare, food float64) {
		t.Helper()
		for _, row := range loaded.MonthlyTable {
			if row.Year == year && row.Month == month {
				if row.CareHoursPerWeek != hours || row.ChildcareFee != childcare || row.FoodFee != food {
					t.Fatalf("%d-%02d row = %dh %.2f/%.2f, want %dh %.2f/%.2f", year, month, row.CareHoursPerWeek, row.ChildcareFee, row.FoodFee, hours, childcare, food)
				}
				return
			}
		}
		t.Fatalf("missing row for %d-%02d", year, month)
	}
	assertRow(2026, 8, 30, 124.92, 22.70)
	assertRow(2026, 9, 30, 249.84, 45.40)
	assertRow(2026, 10, 40, 312.30, 45.40)
	assertRow(2028, 6, 40, 0, 45.40)
}

func testEinstufung(childID uuid.UUID, householdID uuid.UUID, start time.Time) *domain.Einstufung {
	changeDate := start
	return &domain.Einstufung{
		ID:                   uuid.New(),
		ChildID:              childID,
		HouseholdID:          householdID,
		Year:                 start.Year(),
		ValidFrom:            start,
		ChangeDate:           &changeDate,
		EffectiveFromMonth:   start,
		IncomeCalculation:    domain.HouseholdIncomeCalculation{},
		AnnualNetIncome:      50000,
		HighestRateVoluntary: false,
		CareHoursPerWeek:     45,
		CareType:             domain.ChildAgeTypeKrippe,
		ChildrenCount:        1,
		MonthlyChildcareFee:  120,
		MonthlyFoodFee:       45.40,
		AnnualMembershipFee:  30.00,
		FeeRule:              "Test",
		DiscountFactor:       1,
	}
}
