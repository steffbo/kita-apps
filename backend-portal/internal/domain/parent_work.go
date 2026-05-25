package domain

import (
	"time"

	"github.com/google/uuid"
)

type ParentWorkEntryStatus string

const (
	ParentWorkStatusSubmitted ParentWorkEntryStatus = "SUBMITTED"
	ParentWorkStatusApproved  ParentWorkEntryStatus = "APPROVED"
	ParentWorkStatusRejected  ParentWorkEntryStatus = "REJECTED"
	ParentWorkStatusVoided    ParentWorkEntryStatus = "VOIDED"
)

type SyncedChild struct {
	ID             uuid.UUID
	SourceSystem   string
	SourceID       string
	HouseholdID    uuid.UUID
	FirstName      string
	LastName       string
	BirthDate      time.Time
	EntryDate      time.Time
	ExitDate       *time.Time
	GroupName      *string
	IsActive       bool
	LastSyncedAt   time.Time
	SourceChecksum *string
}

type ParentWorkEntry struct {
	ID                    uuid.UUID
	ChildID               uuid.UUID
	SubmittedBy           uuid.UUID
	WorkDate              time.Time
	DurationHours         float64
	ApprovedDurationHours *float64
	Category              string
	Description           string
	Status                ParentWorkEntryStatus
	ReviewNote            *string
	ReviewedBy            *uuid.UUID
	ReviewedAt            *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
