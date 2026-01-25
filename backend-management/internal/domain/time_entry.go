package domain

import "time"

// TimeEntryType defines the type of time entry.
type TimeEntryType string

const (
	TimeEntryTypeWork         TimeEntryType = "WORK"
	TimeEntryTypeVacation     TimeEntryType = "VACATION"
	TimeEntryTypeSick         TimeEntryType = "SICK"
	TimeEntryTypeSpecialLeave TimeEntryType = "SPECIAL_LEAVE"
	TimeEntryTypeTraining     TimeEntryType = "TRAINING"
	TimeEntryTypeEvent        TimeEntryType = "EVENT"
)

// TimeEntry represents a recorded time entry.
type TimeEntry struct {
	ID           int64         `db:"id"`
	EmployeeID   int64         `db:"employee_id"`
	Date         time.Time     `db:"date"`
	ClockIn      time.Time     `db:"clock_in"`
	ClockOut     *time.Time    `db:"clock_out"`
	BreakMinutes int           `db:"break_minutes"`
	EntryType    TimeEntryType `db:"entry_type"`
	Notes        *string       `db:"notes"`
	EditedBy     *int64        `db:"edited_by"`
	EditedAt     *time.Time    `db:"edited_at"`
	EditReason   *string       `db:"edit_reason"`
	CreatedAt    time.Time     `db:"created_at"`
	Employee     *Employee     `db:"-"`
}
