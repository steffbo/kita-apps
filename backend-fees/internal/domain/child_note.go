package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChildNote represents a free-text note attached to a child.
type ChildNote struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ChildID   uuid.UUID `json:"childId" db:"child_id"`
	Text      string    `json:"text" db:"text"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	// ChildName is only populated for the global notes list.
	ChildName *string `json:"childName,omitempty" db:"-" binding:"optional"`
}
