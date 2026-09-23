package domain

// Test-only access to unexported calculation helpers.
var (
	HoursToIndex    = hoursToIndex
	FindRateInTable = findRateInTable
)

func AverageSatzungRate(c FeeScheduleConfig, hours int) float64 { return c.averageSatzungRate(hours) }

func SiblingDiscountFactor(c FeeScheduleConfig, siblings int) float64 {
	return c.siblingDiscountFactor(siblings)
}
