package domain

import "time"

// ScheduleEntryType defines the type of schedule entry.
type ScheduleEntryType string

const (
	ScheduleEntryTypeWork         ScheduleEntryType = "WORK"
	ScheduleEntryTypeVacation     ScheduleEntryType = "VACATION"
	ScheduleEntryTypeSick         ScheduleEntryType = "SICK"
	ScheduleEntryTypeSpecialLeave ScheduleEntryType = "SPECIAL_LEAVE"
	ScheduleEntryTypeTraining     ScheduleEntryType = "TRAINING"
	ScheduleEntryTypeEvent        ScheduleEntryType = "EVENT"
)

// ScheduleEntry represents a scheduled work entry.
type ScheduleEntry struct {
	ID           int64             `db:"id"`
	EmployeeID   int64             `db:"employee_id"`
	Date         time.Time         `db:"date"`
	StartTime    *time.Time        `db:"start_time"`
	EndTime      *time.Time        `db:"end_time"`
	BreakMinutes int               `db:"break_minutes"`
	GroupID      *int64            `db:"group_id"`
	EntryType    ScheduleEntryType `db:"entry_type"`
	Notes        *string           `db:"notes"`
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
	Employee     *Employee         `db:"-"`
	Group        *Group            `db:"-"`
}
