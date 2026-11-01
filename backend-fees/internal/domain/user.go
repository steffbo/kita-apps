package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents an authenticated user in the system.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	FirstName    *string   `json:"firstName,omitempty" db:"first_name" binding:"optional"`
	LastName     *string   `json:"lastName,omitempty" db:"last_name" binding:"optional"`
	// Role is the active role: tokens and thus all access checks use only this one.
	// It stays stored as the last one used, and is always one of Roles.
	Role              UserRole       `json:"role" db:"role"`
	Roles             pq.StringArray `json:"roles" db:"roles"`
	ParentID          *uuid.UUID     `json:"parentId" db:"parent_id" binding:"optional"`
	ParentName        *string        `json:"parentName" db:"parent_name" binding:"optional"`
	IsActive          bool           `json:"isActive" db:"is_active"`
	InvitationPending bool           `json:"invitationPending" db:"invitation_pending"`
	CreatedAt         time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time      `json:"updatedAt" db:"updated_at"`
}

// HasRole reports whether the role is assigned to the account, active or not.
func (u *User) HasRole(role UserRole) bool {
	return slices.Contains(u.Roles, string(role))
}

// UserRole defines the access level of a user.
type UserRole string

const (
	UserRoleAdmin UserRole = "ADMIN"
	// UserRoleBoard is for active board members: read-only access to
	// everything the staff sees, incl. fees, income and bank data, until write
	// access can be audited. Enforced by middleware.ReadOnly.
	UserRoleBoard UserRole = "BOARD"
	// UserRoleUser identifies the automated CSV import in the request context
	// (see middleware.ImportAuthMiddleware). fees.users does not allow it, so no
	// account can hold it.
	UserRoleUser       UserRole = "USER"
	UserRoleParentWork UserRole = "PARENT_WORK"
	// UserRoleLeitung manages children, parents, households, members and
	// Elternstunden, but never sees fee, income or bank data.
	UserRoleLeitung UserRole = "LEITUNG"
	UserRolePARENT  UserRole = "PARENT"
)

// AssignableRoles lists the roles an account can hold, most privileged first.
var AssignableRoles = []UserRole{
	UserRoleAdmin, UserRoleBoard, UserRoleLeitung, UserRoleParentWork, UserRolePARENT,
}

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}
