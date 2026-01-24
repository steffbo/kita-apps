package service

import (
	"time"
)

const (
	dateLayout     = "2006-01-02"
	timeLayout     = "15:04"
	timeLayoutSecs = "15:04:05"
)

// ParseDate parses a YYYY-MM-DD date string.
func ParseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, value)
}

// ParseTime parses a time string in HH:mm or HH:mm:ss format.
func ParseTime(value string) (time.Time, error) {
	if len(value) == len(timeLayout) {
		parsed, err := time.Parse(timeLayout, value)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(2000, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC), nil
	}
	parsed, err := time.Parse(timeLayoutSecs, value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(2000, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC), nil
}

// ParseDateTime parses an RFC3339 timestamp.
func ParseDateTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}
