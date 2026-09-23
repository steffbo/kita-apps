package util

import (
	"time"
	_ "time/tzdata" // embed zone data so Europe/Berlin resolves without OS tzdata
)

// Berlin is the business time zone. Months, due dates and "today" follow the
// calendar in Germany, independent of the server's TZ.
var Berlin = mustLoadLocation("Europe/Berlin")

// now is swapped in tests via SetClock.
var now = time.Now

// Now returns the current wall-clock time in Europe/Berlin. Use it for
// timestamps and to derive the current year or month.
func Now() time.Time {
	return now().In(Berlin)
}

// Today returns the current Berlin calendar date as midnight UTC, the same
// representation DATE columns have after scanning. Use it whenever a value is
// compared with or stored as a date (entry/exit dates, due dates, history
// periods, as-of dates).
func Today() time.Time {
	return DateOf(Now())
}

// DateOf returns the calendar date of t (in t's own location) as midnight UTC.
func DateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// SetClock replaces the clock for tests and returns a function restoring it.
func SetClock(fn func() time.Time) (restore func()) {
	previous := now
	now = fn
	return func() { now = previous }
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
