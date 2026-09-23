package domain

import "math"

// Amounts are stored as NUMERIC(…,2) and carried as float64 euros. Comparisons
// and sums are done in integer cents to avoid float drift.

// PaymentToleranceCents is how much may be missing for a fee to still count as
// paid. Mirrors the SQL condition `matched >= amount - 0.01` in the repositories.
const PaymentToleranceCents = 1

// Cents converts a euro amount to integer cents.
func Cents(eur float64) int64 {
	return int64(math.Round(eur * 100))
}

// Euros converts integer cents to a euro amount.
func Euros(cents int64) float64 {
	return float64(cents) / 100
}

// SumCents adds euro amounts exactly and returns the total in cents.
func SumCents(amounts ...float64) int64 {
	var total int64
	for _, a := range amounts {
		total += Cents(a)
	}
	return total
}

// IsPaid reports whether matched covers amount within PaymentToleranceCents.
func IsPaid(matched, amount float64) bool {
	return Cents(matched) >= Cents(amount)-PaymentToleranceCents
}
