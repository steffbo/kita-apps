package service

import (
	"testing"
	"time"
)

func TestCalculateEffectiveFromMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "14th applies same month",
			input:    time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "15th applies following month",
			input:    time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEffectiveFromMonth(tt.input)
			if !got.Equal(tt.expected) {
				t.Fatalf("expected %s, got %s", tt.expected.Format("2006-01-02"), got.Format("2006-01-02"))
			}
		})
	}
}
