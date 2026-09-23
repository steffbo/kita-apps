package domain_test

import (
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestCentsAvoidsFloatDrift(t *testing.T) {
	// 0.1 + 0.2 != 0.3 in float64; in cents it is exact.
	if got := domain.SumCents(0.1, 0.2); got != 30 {
		t.Fatalf("SumCents(0.1, 0.2) = %d, want 30", got)
	}
	if got := domain.SumCents(45.40, 45.40, 45.40); got != 13620 {
		t.Fatalf("SumCents(3x45.40) = %d, want 13620", got)
	}
	if got := domain.Cents(171.765); got != 17177 {
		t.Fatalf("Cents(171.765) = %d, want 17177", got)
	}
}

func TestIsPaid(t *testing.T) {
	tests := []struct {
		matched, amount float64
		want            bool
	}{
		{45.40, 45.40, true},
		{45.39, 45.40, true},  // one cent tolerance
		{45.38, 45.40, false}, // two cents missing
		{50.00, 45.40, true},
		{0, 0, true},
		{0.1 + 0.2, 0.3, true},
	}
	for _, tt := range tests {
		if got := domain.IsPaid(tt.matched, tt.amount); got != tt.want {
			t.Errorf("IsPaid(%v, %v) = %v, want %v", tt.matched, tt.amount, got, tt.want)
		}
	}
}
