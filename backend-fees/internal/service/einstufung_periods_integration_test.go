package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
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
		MonthlyFoodFee:       domain.FoodFeeAmount,
		AnnualMembershipFee:  domain.MembershipFeeAmount,
		FeeRule:              "Test",
		DiscountFactor:       1,
	}
}
