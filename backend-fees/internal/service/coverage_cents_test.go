package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// 45.40 + 69.40 is 114.80000000000001 in float64; a single payment of 114.80
// used to leave the month "partial" with a balance of 1e-14.
func TestCalculateMonthCoverage_ExactInCents(t *testing.T) {
	childID := uuid.New()
	month := 9
	fees := []domain.FeeExpectation{
		{ID: uuid.New(), ChildID: childID, FeeType: domain.FeeTypeFood, Year: 2026, Month: &month, Amount: 45.40},
		{ID: uuid.New(), ChildID: childID, FeeType: domain.FeeTypeChildcare, Year: 2026, Month: &month, Amount: 69.40},
	}
	txs := []domain.BankTransaction{
		{ID: uuid.New(), BookingDate: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), Amount: 114.80},
	}

	coverage := (&CoverageService{}).calculateMonthCoverage(childID, 2026, month, fees, txs)

	if coverage.Status != domain.CoverageStatusCovered {
		t.Fatalf("status = %s, want %s (balance %v)", coverage.Status, domain.CoverageStatusCovered, coverage.Balance)
	}
	if coverage.Balance != 0 || coverage.ExpectedTotal != 114.80 || coverage.ReceivedTotal != 114.80 {
		t.Fatalf("expected/received/balance = %v/%v/%v", coverage.ExpectedTotal, coverage.ReceivedTotal, coverage.Balance)
	}
}
