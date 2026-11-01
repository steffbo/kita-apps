package domain

import (
	"testing"
	"time"
)

func TestCalculateInstallments(t *testing.T) {
	date := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	final := date("2026-12-31")
	for _, tc := range []struct {
		name             string
		paid, charge     float64
		final            *time.Time
		today            string
		amounts, covered []float64
		statuses         []string
		state            string
		rest             float64
	}{
		{"FIFO surplus", 150, 0, nil, "2026-03-01", []float64{100, 100, 50}, []float64{100, 50, 0}, []string{"PAID", "OVERDUE", "OPEN"}, "OVERDUE", 100},
		{"partial future", 150, 0, nil, "2026-02-01", []float64{100, 100, 50}, []float64{100, 50, 0}, []string{"PAID", "PARTIAL", "OPEN"}, "ACTIVE", 100},
		{"due today open", 0, 0, nil, "2026-01-31", []float64{100, 100, 50}, []float64{0, 0, 0}, []string{"OPEN", "OPEN", "OPEN"}, "ACTIVE", 250},
		{"final replaces overdue", 150, 0, &final, "2026-11-01", []float64{100, 150}, []float64{100, 50}, []string{"PAID", "PARTIAL"}, "ACTIVE", 100},
		{"final overdue", 150, 0, &final, "2027-01-01", []float64{100, 150}, []float64{100, 50}, []string{"PAID", "OVERDUE"}, "OVERDUE", 100},
		{"charges remainder", 0, 5, nil, "2026-01-01", []float64{100, 100, 55}, []float64{0, 0, 0}, []string{"OPEN", "OPEN", "OPEN"}, "ACTIVE", 255},
		{"overpayment", 300, 0, &final, "2027-01-01", []float64{100, 100, 50}, []float64{100, 100, 50}, []string{"PAID", "PAID", "PAID"}, "DONE", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := InstallmentPlan{TotalAmount: 250, InstallmentAmount: 100, IntervalMonths: 1, FirstDueDate: date("2026-01-31"), FinalDueDate: tc.final, Status: "ACTIVE"}
			summary, rates := CalculateInstallments(p, []InstallmentPayment{{Amount: tc.paid}}, []InstallmentCharge{{Amount: tc.charge}}, date(tc.today))
			if summary.Rest != tc.rest || summary.State != tc.state || len(rates) != len(tc.amounts) {
				t.Fatalf("summary %+v rates %+v", summary, rates)
			}
			for i, v := range rates {
				if v.Amount != tc.amounts[i] || v.Paid != tc.covered[i] || v.Status != tc.statuses[i] {
					t.Fatalf("rate %d: %+v", i, v)
				}
			}
			if tc.final == nil && rates[1].DueDate != date("2026-02-28") {
				t.Fatal("month clamp", rates)
			}
			p.Status = "CLOSED"
			summary, _ = CalculateInstallments(p, nil, nil, date(tc.today))
			if summary.State != "CLOSED" {
				t.Fatal(summary)
			}
		})
	}
	for _, interval := range []int{1, 2, 3, 6, 12} {
		p := InstallmentPlan{TotalAmount: 300, InstallmentAmount: 100, IntervalMonths: interval, FirstDueDate: date("2024-01-31")}
		_, rates := CalculateInstallments(p, nil, nil, date("2024-01-01"))
		if interval == 1 && (rates[1].DueDate != date("2024-02-29") || rates[2].DueDate != date("2024-03-31")) {
			t.Fatal(rates)
		}
	}
}
