package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	ParentWorkStatusSubmitted = "SUBMITTED"
	ParentWorkStatusApproved  = "APPROVED"
	ParentWorkStatusRejected  = "REJECTED"
	ParentWorkStatusVoided    = "VOIDED"
)

// ParentWorkUnassignedChild is a cared-for child without a household.
type ParentWorkUnassignedChild struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	EntryDate time.Time  `json:"-" db:"entry_date"`
	ExitDate  *time.Time `json:"-" db:"exit_date"`
}

// ParentWorkRule defines the contribution rules valid from a given date.
type ParentWorkRule struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	ValidFrom            time.Time `json:"validFrom" db:"valid_from"`
	HoursPerChildMinutes int       `json:"hoursPerChildMinutes" db:"hours_per_child_minutes"`
	MissingHourRateCents int       `json:"missingHourRateCents" db:"missing_hour_rate_cents"`
	MaxCarryOverMinutes  int       `json:"maxCarryOverMinutes" db:"max_carry_over_minutes"`
	CreatedAt            time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time `json:"updatedAt" db:"updated_at"`
}

// ParentWorkChild is a child linked to a household for account calculations.
type ParentWorkChild struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	HouseholdID uuid.UUID  `json:"-" db:"household_id"`
	Name        string     `json:"name" db:"name"`
	FirstName   string     `json:"firstName" db:"first_name"`
	LastName    string     `json:"lastName" db:"last_name"`
	EntryDate   time.Time  `json:"entryDate" db:"entry_date"`
	ExitDate    *time.Time `json:"exitDate,omitempty" db:"exit_date" binding:"optional"`
	Tertials    int        `json:"tertials" db:"-"`
}

// ParentWorkMember is a club member linked to a household.
type ParentWorkMember struct {
	ID          uuid.UUID `json:"id" db:"id"`
	HouseholdID uuid.UUID `json:"-" db:"household_id"`
	Name        string    `json:"name" db:"name"`
}

// ParentWorkParent is a parent linked to a household.
type ParentWorkParent struct {
	HouseholdID uuid.UUID `json:"-" db:"household_id"`
	Name        string    `json:"name" db:"name"`
}

// BoardTerm records a club board appointment.
type BoardTerm struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	MemberID      uuid.UUID  `json:"memberId" db:"member_id"`
	MemberName    string     `json:"memberName" db:"member_name"`
	HouseholdID   *uuid.UUID `json:"householdId,omitempty" db:"household_id" binding:"optional"`
	HouseholdName *string    `json:"householdName,omitempty" db:"household_name" binding:"optional"`
	Office        string     `json:"office" db:"office"`
	StartDate     time.Time  `json:"startDate" db:"start_date"`
	EndDate       *time.Time `json:"endDate,omitempty" db:"end_date" binding:"optional"`
	Note          *string    `json:"note,omitempty" db:"note" binding:"optional"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
}

// ParentWorkEntry records work credited to a household.
type ParentWorkEntry struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	HouseholdID     uuid.UUID  `json:"householdId" db:"household_id"`
	WorkDate        time.Time  `json:"workDate" db:"work_date"`
	DurationMinutes int        `json:"durationMinutes" db:"duration_minutes"`
	Occasion        string     `json:"occasion" db:"occasion"`
	MemberName      *string    `json:"memberName,omitempty" db:"member_name" binding:"optional"`
	ChildName       *string    `json:"childName,omitempty" db:"child_name" binding:"optional"`
	Status          string     `json:"status" db:"status"`
	VoidReason      *string    `json:"voidReason,omitempty" db:"void_reason" binding:"optional"`
	Source          string     `json:"source" db:"source"`
	CreatedBy       *uuid.UUID `json:"createdBy,omitempty" db:"created_by" binding:"optional"`
	UpdatedBy       *uuid.UUID `json:"updatedBy,omitempty" db:"updated_by" binding:"optional"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" db:"updated_at"`
}

// ParentWorkOverride replaces a household's calculated annual requirement.
type ParentWorkOverride struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	HouseholdID     uuid.UUID  `json:"householdId" db:"household_id"`
	KitaYear        int        `json:"kitaYear" db:"kita_year"`
	RequiredMinutes int        `json:"requiredMinutes" db:"required_minutes"`
	Reason          string     `json:"reason" db:"reason"`
	CreatedBy       *uuid.UUID `json:"createdBy,omitempty" db:"created_by" binding:"optional"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" db:"updated_at"`
}

// ParentWorkAccount contains the calculated annual balance of a household.
type ParentWorkAccount struct {
	HouseholdID        uuid.UUID         `json:"householdId"`
	HouseholdName      string            `json:"householdName"`
	Children           []ParentWorkChild `json:"children"`
	RequiredMinutes    int               `json:"requiredMinutes"`
	CalculatedMinutes  int               `json:"calculatedMinutes"`
	OverrideMinutes    *int              `json:"overrideMinutes,omitempty" binding:"optional"`
	OverrideReason     *string           `json:"overrideReason,omitempty" binding:"optional"`
	ExemptReason       *string           `json:"exemptReason,omitempty" binding:"optional"`
	DoneMinutes        int               `json:"doneMinutes"`
	CarryInMinutes     int               `json:"carryInMinutes"`
	OpenMinutes        int               `json:"openMinutes"`
	CarryOutMinutes    int               `json:"carryOutMinutes"`
	MissingAmountCents int               `json:"missingAmountCents"`
	EntryCount         int               `json:"entryCount"`
}

// ParentWorkYearStart returns the first day of a Kita year.
func ParentWorkYearStart(year int) time.Time {
	return time.Date(year, 8, 1, 0, 0, 0, 0, time.UTC)
}

// ParentWorkKitaYear returns the start year of the Kita year containing date.
func ParentWorkKitaYear(date time.Time) int {
	if date.Month() < time.August {
		return date.Year() - 1
	}
	return date.Year()
}

func overlaps(start time.Time, end *time.Time, from, until time.Time) bool {
	return !start.After(until) && (end == nil || !end.Before(from))
}

// CalculateParentWork computes one family account from in-memory source rows.
func CalculateParentWork(year int, rule *ParentWorkRule, children []ParentWorkChild,
	terms []BoardTerm, override *ParentWorkOverride, entries []ParentWorkEntry,
	carryIn int) ParentWorkAccount {
	result := ParentWorkAccount{Children: []ParentWorkChild{}}
	if rule == nil {
		return result
	}
	starts := []time.Time{ParentWorkYearStart(year),
		time.Date(year, 12, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year+1, 4, 1, 0, 0, 0, 0, time.UTC)}
	ends := []time.Time{time.Date(year, 11, 30, 0, 0, 0, 0, time.UTC),
		time.Date(year+1, 3, 31, 0, 0, 0, 0, time.UTC),
		time.Date(year+1, 7, 31, 0, 0, 0, 0, time.UTC)}
	for _, child := range children {
		child.Tertials = 0
		for i := range starts {
			if overlaps(child.EntryDate, child.ExitDate, starts[i], ends[i]) {
				child.Tertials++
			}
		}
		if child.Tertials > 0 {
			result.Children = append(result.Children, child)
			result.CalculatedMinutes += rule.HoursPerChildMinutes * child.Tertials / 3
		}
	}
	for _, term := range terms {
		if overlaps(term.StartDate, term.EndDate, starts[0], ends[2]) {
			reason := fmt.Sprintf("Vorstand: %s (%s)", term.MemberName, term.Office)
			result.ExemptReason = &reason
			result.CalculatedMinutes = 0
			break
		}
	}
	result.RequiredMinutes = result.CalculatedMinutes
	if override != nil {
		result.OverrideMinutes = &override.RequiredMinutes
		result.OverrideReason = &override.Reason
		result.RequiredMinutes = override.RequiredMinutes
	}
	result.CarryInMinutes = carryIn
	for _, entry := range entries {
		if !entry.WorkDate.Before(starts[0]) && !entry.WorkDate.After(ends[2]) {
			result.EntryCount++
			if entry.Status == ParentWorkStatusApproved {
				result.DoneMinutes += entry.DurationMinutes
			}
		}
	}
	result.OpenMinutes = max(0, result.RequiredMinutes-result.DoneMinutes-carryIn)
	result.CarryOutMinutes = min(rule.MaxCarryOverMinutes,
		max(0, result.DoneMinutes+carryIn-result.RequiredMinutes), result.DoneMinutes)
	result.MissingAmountCents = (result.OpenMinutes*rule.MissingHourRateCents + 30) / 60
	return result
}
