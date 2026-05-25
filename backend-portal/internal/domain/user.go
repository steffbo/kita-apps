package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleAdmin    UserRole = "ADMIN"
	UserRoleBeitrag  UserRole = "BEITRAG"
	UserRoleVorstand UserRole = "VORSTAND"
	UserRoleLeitung  UserRole = "LEITUNG"
	UserRoleTeam     UserRole = "TEAM"
	UserRoleParent   UserRole = "PARENT"
)

type UserStatus string

const (
	UserStatusInvited  UserStatus = "INVITED"
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusDisabled UserStatus = "DISABLED"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash *string
	FirstName    *string
	LastName     *string
	Status       UserStatus
	Roles        []UserRole
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
