package domain

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestClassifyKitaYear(t *testing.T) {
	// Birth dates mirror the Stichtagsmeldung of 1 September 2026 (Kita year 2026/27).
	stichtag := day(2026, time.September, 1)
	tests := []struct {
		birth time.Time
		want  KitaYearGroup
	}{
		{day(2024, time.September, 7), KitaYearGroupU3},
		{day(2023, time.September, 2), KitaYearGroupU3},
		{day(2023, time.September, 1), KitaYearGroupOtherUe3}, // third birthday on the Stichtag
		{day(2022, time.October, 4), KitaYearGroupOtherUe3},
		{day(2022, time.October, 1), KitaYearGroupSecondToLast}, // completes 6 on 30 Sept 2028
		{day(2021, time.October, 7), KitaYearGroupSecondToLast},
		{day(2021, time.September, 23), KitaYearGroupLast},
		{day(2020, time.October, 16), KitaYearGroupLast},
		{day(2020, time.October, 1), KitaYearGroupDeferred},
		{day(2020, time.July, 24), KitaYearGroupDeferred},
	}
	for _, tt := range tests {
		if got := ClassifyKitaYear(tt.birth, stichtag); got != tt.want {
			t.Errorf("born %s: got %s, want %s", tt.birth.Format("2006-01-02"), got, tt.want)
		}
	}

	// Before 1 August the previous Kita year applies.
	if got := ClassifyKitaYear(day(2021, time.March, 27), day(2027, time.March, 1)); got != KitaYearGroupLast {
		t.Errorf("spring Stichtag: got %s, want LAST_YEAR", got)
	}
}

func TestJugendamtReportAdd(t *testing.T) {
	r := JugendamtReport{Stichtag: day(2026, time.September, 1)}
	for _, birth := range []time.Time{
		day(2024, time.September, 7), day(2022, time.October, 4), day(2021, time.October, 7),
		day(2020, time.October, 16), day(2020, time.July, 24),
	} {
		r.Add(JugendamtChild{BirthDate: birth})
	}
	if r.Total != 5 || r.U3 != 1 || r.Ue3 != 4 || r.OtherUe3 != 1 || r.SecondToLastYear != 1 ||
		r.LastYear != 2 || r.Deferred != 1 {
		t.Fatalf("unexpected counts: %+v", r)
	}
}

func TestStichtagForJugendamt(t *testing.T) {
	tests := []struct{ today, want time.Time }{
		{day(2026, time.October, 31), day(2026, time.December, 1)},
		{day(2026, time.September, 1), day(2026, time.September, 1)},
		{day(2026, time.September, 8), day(2026, time.September, 1)},
		{day(2026, time.October, 1), day(2026, time.September, 1)},
		{day(2026, time.October, 2), day(2026, time.December, 1)},
		{day(2026, time.December, 20), day(2026, time.December, 1)},
		{day(2027, time.January, 15), day(2027, time.March, 1)},
		{day(2027, time.February, 28), day(2027, time.March, 1)},
	}
	for _, tt := range tests {
		if got := StichtagForJugendamt(tt.today); !got.Equal(tt.want) {
			t.Errorf("today %s: got %s, want %s", tt.today.Format("2006-01-02"),
				got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
		}
	}
}
