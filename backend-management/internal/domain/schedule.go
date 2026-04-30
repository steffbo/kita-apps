package domain

import "time"

// ScheduleEntryType defines the type of schedule entry.
type ScheduleEntryType string
type ShiftKind string
type ScheduleRequestType string
type ScheduleRequestStatus string

const (
	ScheduleEntryTypeWork         ScheduleEntryType = "WORK"
	ScheduleEntryTypeVacation     ScheduleEntryType = "VACATION"
	ScheduleEntryTypeSick         ScheduleEntryType = "SICK"
	ScheduleEntryTypeChildSick    ScheduleEntryType = "CHILD_SICK"
	ScheduleEntryTypeRecoveryDay  ScheduleEntryType = "RECOVERY_DAY"
	ScheduleEntryTypeSpecialLeave ScheduleEntryType = "SPECIAL_LEAVE"
	ScheduleEntryTypeTraining     ScheduleEntryType = "TRAINING"
	ScheduleEntryTypeEvent        ScheduleEntryType = "EVENT"
)

const (
	ShiftKindEarly  ShiftKind = "EARLY"
	ShiftKindLate   ShiftKind = "LATE"
	ShiftKindManual ShiftKind = "MANUAL"
)

const (
	ScheduleRequestTypeWish        ScheduleRequestType = "WISH"
	ScheduleRequestTypeAppointment ScheduleRequestType = "APPOINTMENT"
)

const (
	ScheduleRequestStatusOpen ScheduleRequestStatus = "OPEN"
	ScheduleRequestStatusDone ScheduleRequestStatus = "DONE"
)

// ScheduleEntrySegment represents a structured group assignment within a work shift.
type ScheduleEntrySegment struct {
	ID              int64     `db:"id"`
	ScheduleEntryID int64     `db:"schedule_entry_id"`
	GroupID         int64     `db:"group_id"`
	StartTime       time.Time `db:"start_time"`
	EndTime         time.Time `db:"end_time"`
	Notes           *string   `db:"notes"`
	SortOrder       int       `db:"sort_order"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	Group           *Group    `db:"-"`
}

// ScheduleEntry represents a scheduled work entry.
type ScheduleEntry struct {
	ID              int64                  `db:"id"`
	EmployeeID      int64                  `db:"employee_id"`
	Date            time.Time              `db:"date"`
	StartTime       *time.Time             `db:"start_time"`
	EndTime         *time.Time             `db:"end_time"`
	BreakMinutes    int                    `db:"break_minutes"`
	GroupID         *int64                 `db:"group_id"`
	EntryType       ScheduleEntryType      `db:"entry_type"`
	ShiftKind       ShiftKind              `db:"shift_kind"`
	Notes           *string                `db:"notes"`
	CreatedAt       time.Time              `db:"created_at"`
	UpdatedAt       time.Time              `db:"updated_at"`
	Employee        *Employee              `db:"-"`
	Group           *Group                 `db:"-"`
	Segments        []ScheduleEntrySegment `db:"-"`
	SegmentsChanged bool                   `db:"-"`
}

// ScheduleRequest represents a non-working wish or appointment request.
type ScheduleRequest struct {
	ID          int64                 `db:"id"`
	EmployeeID  int64                 `db:"employee_id"`
	Date        time.Time             `db:"date"`
	StartTime   *time.Time            `db:"start_time"`
	EndTime     *time.Time            `db:"end_time"`
	RequestType ScheduleRequestType   `db:"request_type"`
	Text        string                `db:"text"`
	Status      ScheduleRequestStatus `db:"status"`
	CreatedAt   time.Time             `db:"created_at"`
	UpdatedAt   time.Time             `db:"updated_at"`
	Employee    *Employee             `db:"-"`
}
