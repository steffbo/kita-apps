package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// childcareFeeGrid covers all brackets, boundaries, care hours (incl. out of
// range), sibling counts and flags.
func childcareFeeGrid() []domain.ChildcareFeeInput {
	incomes := []float64{0, 19999.99, 20000.00, 20000.01, 34999.99, 35000.00, 35000.01, 40000.00, 40000.01,
		45000.00, 45000.01, 50000.00, 50000.01, 54999.99, 55000.00, 55000.01, 60000, 1_000_000}
	for income := 0.0; income <= 70000; income += 250 {
		incomes = append(incomes, income)
	}
	hours := []int{0, 25, 30, 32, 35, 38, 40, 45, 50, 55, 60}
	var inputs []domain.ChildcareFeeInput
	for _, ageType := range []domain.ChildAgeType{domain.ChildAgeTypeKrippe, domain.ChildAgeTypeKindergarten} {
		for _, income := range incomes {
			for _, h := range hours {
				for siblings := 0; siblings <= 8; siblings++ {
					for _, highest := range []bool{false, true} {
						for _, foster := range []bool{false, true} {
							inputs = append(inputs, domain.ChildcareFeeInput{
								ChildAgeType: ageType, NetIncome: income, SiblingsCount: siblings,
								CareHours: h, HighestRate: highest, FosterFamily: foster,
							})
						}
					}
				}
			}
		}
	}
	return inputs
}

func assertSameChildcareFee(t *testing.T, calc func(domain.ChildcareFeeInput) *domain.ChildcareFeeResult) {
	t.Helper()
	failures := 0
	for _, in := range childcareFeeGrid() {
		want := service.LegacyCalculateChildcareFee(in)
		got := calc(in)
		if got.Fee != want.Fee || got.BaseFee != want.BaseFee || got.Rule != want.Rule ||
			got.DiscountFactor != want.DiscountFactor || got.DiscountPercent != want.DiscountPercent ||
			got.ShowEntlastung != want.ShowEntlastung || len(got.Notes) != len(want.Notes) {
			t.Errorf("input %+v:\n got  %+v\n want %+v", in, got, want)
			failures++
		} else {
			for i := range want.Notes {
				if got.Notes[i] != want.Notes[i] {
					t.Errorf("input %+v: note %d = %q, want %q", in, i, got.Notes[i], want.Notes[i])
					failures++
				}
			}
		}
		if failures > 10 {
			t.Fatal("too many differences")
		}
	}
}

// The fee schedule seeded by migration 000034 must reproduce the formerly
// hard-coded calculation exactly, for every date before any new version.
func TestSeededFeeSchedule_MatchesLegacyCalculation(t *testing.T) {
	repo := repository.NewPostgresFeeScheduleRepository(testDB)
	for _, date := range []time.Time{
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	} {
		schedule, err := repo.GetAt(context.Background(), date)
		if err != nil {
			t.Fatalf("GetAt(%s): %v", date.Format("2006-01-02"), err)
		}
		if err := schedule.Config.Validate(); err != nil {
			t.Fatalf("seeded config invalid: %v", err)
		}
		if schedule.Config.MonthlyFoodFee != 45.40 || schedule.Config.AnnualMembershipFee != 30.00 {
			t.Fatalf("food/membership = %v/%v, want 45.40/30.00", schedule.Config.MonthlyFoodFee, schedule.Config.AnnualMembershipFee)
		}
		assertSameChildcareFee(t, schedule.Config.CalculateChildcareFee)
	}
}
