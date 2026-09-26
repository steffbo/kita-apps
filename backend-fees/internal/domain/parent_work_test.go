package domain

import (
	"testing"
	"time"
)

func pwDate(value string) time.Time { d, _ := time.Parse("2006-01-02", value); return d }
func pwChild(start, end string) ParentWorkChild {
	v := ParentWorkChild{EntryDate: pwDate(start)}
	if end != "" {
		d := pwDate(end)
		v.ExitDate = &d
	}
	return v
}
func TestCalculateParentWork(t *testing.T) {
	rule := &ParentWorkRule{HoursPerChildMinutes: 540, MissingHourRateCents: 3000,
		MaxCarryOverMinutes: 180}
	term := BoardTerm{MemberName: "Ada Beispiel", Office: "Vorsitzende", StartDate: pwDate("2025-12-01")}
	override := &ParentWorkOverride{RequiredMinutes: 240, Reason: "Vereinbarung"}
	cases := []struct {
		name                                                         string
		year                                                         int
		rule                                                         *ParentWorkRule
		children                                                     []ParentWorkChild
		terms                                                        []BoardTerm
		override                                                     *ParentWorkOverride
		entries                                                      []ParentWorkEntry
		carry                                                        int
		calculated, required, done, carryOut, open, amount, tertials int
	}{
		{name: "full year", year: 2025, rule: rule, children: []ParentWorkChild{pwChild("2025-08-01", "")},
			calculated: 540, required: 540, open: 540, amount: 27000, tertials: 3},
		{name: "January entry", year: 2025, rule: rule, children: []ParentWorkChild{pwChild("2026-01-15", "")},
			calculated: 360, required: 360, open: 360, amount: 18000, tertials: 2},
		{name: "one day", year: 2025, rule: rule, children: []ParentWorkChild{pwChild("2026-07-31", "2026-07-31")},
			calculated: 180, required: 180, open: 180, amount: 9000, tertials: 1},
		{name: "two children", year: 2025, rule: rule, children: []ParentWorkChild{
			pwChild("2025-08-01", ""), pwChild("2025-08-01", "")},
			calculated: 1080, required: 1080, open: 1080, amount: 54000, tertials: 3},
		{name: "board six months", year: 2025, rule: rule, children: []ParentWorkChild{pwChild("2025-08-01", "")},
			terms: []BoardTerm{term}, tertials: 3},
		{name: "one-day board term at year end", year: 2025, rule: rule,
			children: []ParentWorkChild{pwChild("2025-08-01", "")},
			terms: []BoardTerm{{MemberName: "Ada Beispiel", Office: "Vorsitzende",
				StartDate: pwDate("2026-07-31"), EndDate: func() *time.Time {
					d := pwDate("2026-07-31")
					return &d
				}()}}, tertials: 3},
		{name: "override beats board", year: 2025, rule: rule,
			children: []ParentWorkChild{pwChild("2025-08-01", "")}, terms: []BoardTerm{term},
			override: override, required: 240, open: 240, amount: 12000, tertials: 3},
		{name: "carry capped", year: 2025, rule: rule, entries: []ParentWorkEntry{
			{WorkDate: pwDate("2025-09-01"), DurationMinutes: 300, Status: "APPROVED"}},
			done: 300, carryOut: 180},
		{name: "carry cannot repeat", year: 2025, rule: rule, carry: 180},
		{name: "ninety minutes open", year: 2025, rule: rule,
			children: []ParentWorkChild{pwChild("2025-08-01", "")}, entries: []ParentWorkEntry{
				{WorkDate: pwDate("2025-09-01"), DurationMinutes: 450, Status: "APPROVED"}},
			calculated: 540, required: 540, done: 450, open: 90, amount: 4500, tertials: 3},
		{name: "July 31 exit", year: 2025, rule: rule,
			children:   []ParentWorkChild{pwChild("2025-08-01", "2026-07-31")},
			calculated: 540, required: 540, open: 540, amount: 27000, tertials: 3},
		{name: "April 1 entry", year: 2025, rule: rule,
			children:   []ParentWorkChild{pwChild("2026-04-01", "")},
			calculated: 180, required: 180, open: 180, amount: 9000, tertials: 1},
		{name: "no rules", year: 2024, children: []ParentWorkChild{pwChild("2024-08-01", "")}},
		{name: "round cents", year: 2025, rule: &ParentWorkRule{HoursPerChildMinutes: 1,
			MissingHourRateCents: 1}, children: []ParentWorkChild{pwChild("2025-08-01", "")},
			calculated: 1, required: 1, open: 1, amount: 0, tertials: 3},
		{name: "round up", year: 2025, rule: &ParentWorkRule{HoursPerChildMinutes: 31,
			MissingHourRateCents: 1}, children: []ParentWorkChild{pwChild("2025-08-01", "")},
			calculated: 31, required: 31, open: 31, amount: 1, tertials: 3},
		{name: "voided excluded", year: 2025, rule: rule, entries: []ParentWorkEntry{
			{WorkDate: pwDate("2025-09-01"), DurationMinutes: 300, Status: "VOIDED"}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateParentWork(tt.year, tt.rule, tt.children, tt.terms, tt.override, tt.entries, tt.carry)
			if got.CalculatedMinutes != tt.calculated || got.RequiredMinutes != tt.required ||
				got.DoneMinutes != tt.done || got.CarryOutMinutes != tt.carryOut ||
				got.OpenMinutes != tt.open || got.MissingAmountCents != tt.amount {
				t.Fatalf("account = %+v", got)
			}
			if len(got.Children) > 0 && got.Children[0].Tertials != tt.tertials {
				t.Fatalf("tertials = %d", got.Children[0].Tertials)
			}
		})
	}
}
