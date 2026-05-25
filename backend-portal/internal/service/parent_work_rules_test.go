package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequiredParentWorkHours(t *testing.T) {
	date := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	ptrDate := func(value time.Time) *time.Time {
		return &value
	}
	ptrInt := func(value int) *int {
		return &value
	}

	tests := []struct {
		name  string
		input ParentWorkRequirementInput
		want  int
	}{
		{
			name: "full kita year counts all three terms",
			input: ParentWorkRequirementInput{
				EntryDate: date(2025, time.August, 1),
				ExitDate:  ptrDate(date(2026, time.July, 31)),
			},
			want: 9,
		},
		{
			name: "partial year counts started terms",
			input: ParentWorkRequirementInput{
				EntryDate: date(2025, time.September, 15),
				ExitDate:  ptrDate(date(2026, time.February, 10)),
			},
			want: 6,
		},
		{
			name: "single active day still counts the active term",
			input: ParentWorkRequirementInput{
				EntryDate: date(2026, time.June, 15),
				ExitDate:  ptrDate(date(2026, time.June, 15)),
			},
			want: 3,
		},
		{
			name: "child outside kita year has no required hours",
			input: ParentWorkRequirementInput{
				EntryDate: date(2026, time.August, 1),
			},
			want: 0,
		},
		{
			name: "board exemption takes precedence",
			input: ParentWorkRequirementInput{
				EntryDate:            date(2025, time.August, 1),
				ExemptFromParentWork: true,
			},
			want: 0,
		},
		{
			name: "leitung override replaces calculated hours",
			input: ParentWorkRequirementInput{
				EntryDate:             date(2025, time.August, 1),
				RequiredHoursOverride: ptrInt(4),
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiredParentWorkHours(2025, tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
