package domain

import (
	"testing"
	"time"
)

func TestIsUnderThreeForEntireMonth(t *testing.T) {
	// Per German law (§ 188 Abs. 2 BGB), the 3rd year of life is completed on the day
	// BEFORE the 3rd birthday. Example: Child born Oct 1st completes 3rd year on Sept 30th,
	// so September is already fee-free.
	tests := []struct {
		name      string
		birthDate time.Time
		year      int
		month     time.Month
		expected  bool
	}{
		// Key edge case from § 188 Abs. 2 BGB:
		// Birthday on Oct 1st -> completes 3rd year on Sept 30th -> September is fee-free
		{
			name:      "Birthday Oct 1st - September is fee-free (§188 BGB)",
			birthDate: time.Date(2022, time.October, 1, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.September,
			expected:  false,
		},
		{
			name:      "Birthday Oct 1st - August still has fee",
			birthDate: time.Date(2022, time.October, 1, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.August,
			expected:  true,
		},
		// Regular cases
		{
			name:      "Child turns 3 on Jan 2nd - completes 3rd year Jan 1st - no fee for January",
			birthDate: time.Date(2022, time.January, 2, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  false,
		},
		{
			name:      "Child turns 3 on Jan 1st - completes 3rd year Dec 31st previous year - no fee for January",
			birthDate: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  false, // Child already completed 3rd year on Dec 31st 2024
		},
		{
			name:      "Child turns 3 on Jan 1st - December of previous year is fee-free",
			birthDate: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			year:      2024,
			month:     time.December,
			expected:  false,
		},
		{
			name:      "Child turns 3 on Jan 15th - no fee for January",
			birthDate: time.Date(2022, time.January, 15, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  false,
		},
		{
			name:      "Child turns 3 on Feb 1st - completes Jan 31st - no fee for January",
			birthDate: time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  false,
		},
		{
			name:      "Child turns 3 on Feb 2nd - completes Feb 1st - fee for January",
			birthDate: time.Date(2022, time.February, 2, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  true,
		},
		{
			name:      "Child turns 3 on Feb 2nd - no fee for February",
			birthDate: time.Date(2022, time.February, 2, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.February,
			expected:  false,
		},
		{
			name:      "Child already turned 3 last year - no fee",
			birthDate: time.Date(2021, time.June, 15, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  false,
		},
		{
			name:      "Child turns 3 next year - fee is due",
			birthDate: time.Date(2023, time.June, 15, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.January,
			expected:  true,
		},
		{
			name:      "Child turns 3 in December - fee due for October",
			birthDate: time.Date(2022, time.December, 15, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.October,
			expected:  true,
		},
		{
			name:      "Child turns 3 in December - no fee for December",
			birthDate: time.Date(2022, time.December, 15, 0, 0, 0, 0, time.UTC),
			year:      2025,
			month:     time.December,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := &Child{
				BirthDate: tt.birthDate,
			}
			result := child.IsUnderThreeForEntireMonth(tt.year, tt.month)
			if result != tt.expected {
				t.Errorf("IsUnderThreeForEntireMonth(%d, %v) = %v, expected %v (birthDate: %v, 3rd birthday: %v)",
					tt.year, tt.month, result, tt.expected, tt.birthDate, tt.birthDate.AddDate(3, 0, 0))
			}
		})
	}
}

func TestIsUnderThree(t *testing.T) {
	tests := []struct {
		name      string
		birthDate time.Time
		checkDate time.Time
		expected  bool
	}{
		{
			name:      "Day before 3rd birthday",
			birthDate: time.Date(2022, time.January, 15, 0, 0, 0, 0, time.UTC),
			checkDate: time.Date(2025, time.January, 14, 0, 0, 0, 0, time.UTC),
			expected:  true,
		},
		{
			name:      "On 3rd birthday",
			birthDate: time.Date(2022, time.January, 15, 0, 0, 0, 0, time.UTC),
			checkDate: time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC),
			expected:  false,
		},
		{
			name:      "Day after 3rd birthday",
			birthDate: time.Date(2022, time.January, 15, 0, 0, 0, 0, time.UTC),
			checkDate: time.Date(2025, time.January, 16, 0, 0, 0, 0, time.UTC),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := &Child{
				BirthDate: tt.birthDate,
			}
			result := child.IsUnderThree(tt.checkDate)
			if result != tt.expected {
				t.Errorf("IsUnderThree(%v) = %v, expected %v", tt.checkDate, result, tt.expected)
			}
		})
	}
}
