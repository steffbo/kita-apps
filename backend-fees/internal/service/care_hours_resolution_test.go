package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func hoursPtr(v int) *int {
	return &v
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestCareHoursAt(t *testing.T) {
	childID := uuid.New()

	tests := []struct {
		name      string
		history   []domain.ChildCareHoursHistory
		reference time.Time
		wantHours int
		wantOK    bool
	}{
		{
			name:      "no history",
			history:   nil,
			reference: date(2026, time.August, 1),
			wantOK:    false,
		},
		{
			name: "open period covers reference",
			history: []domain.ChildCareHoursHistory{
				{ChildID: childID, CareHours: hoursPtr(35), EffectiveFrom: date(2026, time.August, 1)},
			},
			reference: date(2026, time.September, 1),
			wantHours: 35,
			wantOK:    true,
		},
		{
			name: "upcoming period is used for a child that has not started yet",
			history: []domain.ChildCareHoursHistory{
				{ChildID: childID, CareHours: hoursPtr(35), EffectiveFrom: date(2026, time.August, 1)},
			},
			reference: date(2026, time.July, 27),
			wantHours: 35,
			wantOK:    true,
		},
		{
			name: "closest upcoming period wins",
			history: []domain.ChildCareHoursHistory{
				{ChildID: childID, CareHours: hoursPtr(45), EffectiveFrom: date(2027, time.January, 1)},
				{ChildID: childID, CareHours: hoursPtr(35), EffectiveFrom: date(2026, time.August, 1)},
			},
			reference: date(2026, time.July, 27),
			wantHours: 35,
			wantOK:    true,
		},
		{
			name: "historic period is not used after it ended",
			history: []domain.ChildCareHoursHistory{
				{ChildID: childID, CareHours: hoursPtr(40), EffectiveFrom: date(2026, time.March, 1)},
				{
					ChildID:        childID,
					CareHours:      hoursPtr(30),
					EffectiveFrom:  date(2026, time.January, 1),
					EffectiveUntil: timePtr(date(2026, time.February, 28)),
				},
			},
			reference: date(2026, time.May, 1),
			wantHours: 40,
			wantOK:    true,
		},
		{
			name: "period effective in the billed month wins over later period",
			history: []domain.ChildCareHoursHistory{
				{ChildID: childID, CareHours: hoursPtr(45), EffectiveFrom: date(2026, time.October, 1)},
				{
					ChildID:        childID,
					CareHours:      hoursPtr(35),
					EffectiveFrom:  date(2026, time.August, 1),
					EffectiveUntil: timePtr(date(2026, time.September, 30)),
				},
			},
			reference: date(2026, time.September, 1),
			wantHours: 35,
			wantOK:    true,
		},
		{
			name: "entries without hours are ignored",
			history: []domain.ChildCareHoursHistory{
				{ChildID: childID, CareHours: nil, EffectiveFrom: date(2026, time.August, 1)},
			},
			reference: date(2026, time.August, 1),
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours, ok := careHoursAt(tt.history, tt.reference)
			if ok != tt.wantOK {
				t.Fatalf("careHoursAt ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && hours != tt.wantHours {
				t.Fatalf("careHoursAt hours = %d, want %d", hours, tt.wantHours)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
