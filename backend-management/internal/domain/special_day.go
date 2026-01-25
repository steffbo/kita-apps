package domain

import "time"

// SpecialDayType defines the type of special day.
type SpecialDayType string

const (
	SpecialDayTypeHoliday SpecialDayType = "HOLIDAY"
	SpecialDayTypeClosure SpecialDayType = "CLOSURE"
	SpecialDayTypeTeamDay SpecialDayType = "TEAM_DAY"
	SpecialDayTypeEvent   SpecialDayType = "EVENT"
)

// SpecialDay represents holidays, closures, and events.
type SpecialDay struct {
	ID         int64          `db:"id"`
	Date       time.Time      `db:"date"`
	EndDate    *time.Time     `db:"end_date"`
	Name       string         `db:"name"`
	DayType    SpecialDayType `db:"day_type"`
	AffectsAll bool           `db:"affects_all"`
	Notes      *string        `db:"notes"`
	CreatedAt  time.Time      `db:"created_at"`
}
