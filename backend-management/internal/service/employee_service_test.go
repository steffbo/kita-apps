package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWorkdaysDistributesWeeklyHoursByRemainder(t *testing.T) {
	tests := []struct {
		name        string
		weeklyHours float64
		wantMinutes []int
	}{
		{
			name:        "31 hours",
			weeklyHours: 31,
			wantMinutes: []int{360, 360, 360, 360, 420},
		},
		{
			name:        "37 hours",
			weeklyHours: 37,
			wantMinutes: []int{420, 420, 420, 480, 480},
		},
		{
			name:        "20 hours",
			weeklyHours: 20,
			wantMinutes: []int{240, 240, 240, 240, 240},
		},
		{
			name:        "40 hours",
			weeklyHours: 40,
			wantMinutes: []int{480, 480, 480, 480, 480},
		},
		{
			name:        "33 hours",
			weeklyHours: 33,
			wantMinutes: []int{360, 360, 420, 420, 420},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workdays := defaultWorkdays(12, tt.weeklyHours)
			gotMinutes := make([]int, 0, len(workdays))
			for _, workday := range workdays {
				assert.Equal(t, int64(12), workday.ContractID)
				gotMinutes = append(gotMinutes, workday.PlannedMinutes)
			}
			assert.Equal(t, tt.wantMinutes, gotMinutes)
		})
	}
}
