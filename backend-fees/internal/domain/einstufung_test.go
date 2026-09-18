package domain

import (
	"testing"
	"time"
)

func TestGenerateMonthlyTableProratesAdmissionAfter15th(t *testing.T) {
	e := &Einstufung{
		ValidFrom:           time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC),
		EffectiveFromMonth:  time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek:    30,
		CareType:            ChildAgeTypeKrippe,
		MonthlyChildcareFee: 277.60,
		MonthlyFoodFee:      45.40,
		AnnualMembershipFee: 30,
	}
	child := &Child{EntryDate: time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)}

	rows := e.GenerateMonthlyTable(child)
	if len(rows) == 0 {
		t.Fatal("expected monthly rows")
	}
	if rows[0].ChildcareFee != 138.80 || rows[0].FoodFee != 22.70 {
		t.Fatalf("first row = childcare %.2f, food %.2f; want 138.80 and 22.70", rows[0].ChildcareFee, rows[0].FoodFee)
	}
	if rows[1].ChildcareFee != 277.60 || rows[1].FoodFee != 45.40 {
		t.Fatalf("second row = childcare %.2f, food %.2f; want full monthly amounts", rows[1].ChildcareFee, rows[1].FoodFee)
	}
}
