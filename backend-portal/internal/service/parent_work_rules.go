package service

import "time"

type ParentWorkRequirementInput struct {
	EntryDate             time.Time
	ExitDate              *time.Time
	ExemptFromParentWork  bool
	RequiredHoursOverride *int
}

func RequiredParentWorkHours(kitaYearStartYear int, input ParentWorkRequirementInput) int {
	if input.ExemptFromParentWork {
		return 0
	}
	if input.RequiredHoursOverride != nil {
		if *input.RequiredHoursOverride < 0 {
			return 0
		}
		return *input.RequiredHoursOverride
	}

	yearStart, yearEnd := KitaYearRange(kitaYearStartYear)
	if !overlaps(input.EntryDate, input.ExitDate, yearStart, yearEnd) {
		return 0
	}

	starts := []time.Time{
		time.Date(kitaYearStartYear, time.August, 1, 0, 0, 0, 0, time.UTC),
		time.Date(kitaYearStartYear, time.December, 1, 0, 0, 0, 0, time.UTC),
		time.Date(kitaYearStartYear+1, time.April, 1, 0, 0, 0, 0, time.UTC),
	}
	ends := []time.Time{
		time.Date(kitaYearStartYear, time.November, 30, 0, 0, 0, 0, time.UTC),
		time.Date(kitaYearStartYear+1, time.March, 31, 0, 0, 0, 0, time.UTC),
		time.Date(kitaYearStartYear+1, time.July, 31, 0, 0, 0, 0, time.UTC),
	}

	activeTerms := 0
	for i := range starts {
		if overlaps(input.EntryDate, input.ExitDate, starts[i], ends[i]) {
			activeTerms++
		}
	}

	if activeTerms == 0 {
		return 0
	}
	return activeTerms * 3
}

func KitaYearRange(startYear int) (time.Time, time.Time) {
	return time.Date(startYear, time.August, 1, 0, 0, 0, 0, time.UTC),
		time.Date(startYear+1, time.July, 31, 0, 0, 0, 0, time.UTC)
}

func overlaps(activeStart time.Time, activeEnd *time.Time, rangeStart, rangeEnd time.Time) bool {
	activeStart = dateOnly(activeStart)
	if activeStart.After(rangeEnd) {
		return false
	}
	if activeEnd == nil {
		return true
	}

	end := dateOnly(*activeEnd)
	return !end.Before(rangeStart)
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
