package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// A fee due on 1 Oct is overdue once 2 Oct has started in Berlin, even though
// it is still 1 Oct in UTC.
func TestFeeOverview_OverdueFollowsBerlinCalendar(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)

	child, err := createTestChild(childRepo, "BERLIN")
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	dueDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if _, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40, 2026, 10, dueDate); err != nil {
		t.Fatalf("create fee: %v", err)
	}

	tests := []struct {
		name        string
		instant     time.Time
		wantOverdue int
	}{
		{name: "due day in Berlin", instant: time.Date(2026, 10, 1, 21, 0, 0, 0, time.UTC), wantOverdue: 0},
		{name: "next day in Berlin, still due day in UTC", instant: time.Date(2026, 10, 1, 22, 30, 0, 0, time.UTC), wantOverdue: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := util.SetClock(func() time.Time { return tt.instant })
			defer restore()

			overview, err := feeRepo.GetOverview(ctx, 2026)
			if err != nil {
				t.Fatalf("GetOverview: %v", err)
			}
			if overview.TotalOverdue != tt.wantOverdue {
				t.Fatalf("TotalOverdue = %d, want %d", overview.TotalOverdue, tt.wantOverdue)
			}
		})
	}
}
