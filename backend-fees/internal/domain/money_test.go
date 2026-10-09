package domain_test

import (
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestCents(t *testing.T) {
	if got := domain.Cents(171.765); got != 17177 {
		t.Fatalf("Cents(171.765) = %d, want 17177", got)
	}
}

func TestEuros(t *testing.T) {
	if got := domain.Euros(17177); got != 171.77 {
		t.Fatalf("Euros(17177) = %v, want 171.77", got)
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
