package util

import (
	"testing"
	"time"
)

func TestTodayFollowsBerlinCalendar(t *testing.T) {
	tests := []struct {
		name     string
		instant  time.Time
		wantDate string
		wantYear int
	}{
		// 22:30 UTC on 30 Sep is already 1 Oct in Berlin (CEST, UTC+2).
		{name: "summer month boundary", instant: time.Date(2026, 9, 30, 22, 30, 0, 0, time.UTC), wantDate: "2026-10-01", wantYear: 2026},
		// 23:30 UTC on 31 Dec is already New Year in Berlin (CET, UTC+1).
		{name: "year boundary", instant: time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC), wantDate: "2027-01-01", wantYear: 2027},
		{name: "same day", instant: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), wantDate: "2026-03-15", wantYear: 2026},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := SetClock(func() time.Time { return tt.instant })
			defer restore()

			today := Today()
			if got := today.Format("2006-01-02"); got != tt.wantDate {
				t.Errorf("Today() = %s, want %s", got, tt.wantDate)
			}
			if today.Location() != time.UTC || today.Hour() != 0 {
				t.Errorf("Today() = %v, want midnight UTC", today)
			}
			if got := Now().Year(); got != tt.wantYear {
				t.Errorf("Now().Year() = %d, want %d", got, tt.wantYear)
			}
		})
	}
}
