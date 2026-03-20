package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChildCareHoursHistory represents a care hours period for a child.
type ChildCareHoursHistory struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ChildID        uuid.UUID  `json:"childId" db:"child_id"`
	CareHours      *int       `json:"careHours" db:"care_hours"`
	EffectiveFrom  time.Time  `json:"effectiveFrom" db:"effective_from"`
	EffectiveUntil *time.Time `json:"effectiveUntil,omitempty" db:"effective_until"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
}

// ChildLegalHoursHistory represents a legal entitlement period for a child.
type ChildLegalHoursHistory struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ChildID        uuid.UUID  `json:"childId" db:"child_id"`
	LegalHours     *int       `json:"legalHours" db:"legal_hours"`
	EffectiveFrom  time.Time  `json:"effectiveFrom" db:"effective_from"`
	EffectiveUntil *time.Time `json:"effectiveUntil,omitempty" db:"effective_until"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
}
