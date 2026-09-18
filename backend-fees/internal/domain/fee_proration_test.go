package domain

import (
	"testing"
	"time"
)

func TestContributionAmountForMonth(t *testing.T) {
	tests := []struct {
		name      string
		entryDate time.Time
		year      int
		month     time.Month
		amount    float64
		want      float64
	}{
		{"entry through 15th is full", time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC), 2026, time.August, 312.30, 312.30},
		{"entry after 15th is half", time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC), 2026, time.August, 312.30, 156.15},
		{"odd cent is rounded to cents", time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC), 2026, time.August, 45.41, 22.71},
		{"later month is full", time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC), 2026, time.September, 312.30, 312.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContributionAmountForMonth(tt.amount, tt.entryDate, tt.year, tt.month); got != tt.want {
				t.Fatalf("ContributionAmountForMonth() = %.2f, want %.2f", got, tt.want)
			}
		})
	}
}
