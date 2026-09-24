package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

func TestFeeScheduleService_Rules(t *testing.T) {
	ctx := context.Background()
	restore := util.SetClock(func() time.Time { return time.Date(2031, 5, 10, 12, 0, 0, 0, time.UTC) })
	defer restore()

	repo := repository.NewPostgresFeeScheduleRepository(testDB)
	svc := service.NewFeeScheduleService(repo)
	defer testDB.Exec(`DELETE FROM fees.fee_schedules WHERE valid_from > '2025-01-01'`)

	seeded, err := repo.GetAt(ctx, time.Date(2031, 5, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	input := func(validFrom time.Time) service.FeeScheduleInput {
		return service.FeeScheduleInput{ValidFrom: validFrom, Name: "Test", Config: seeded.Config}
	}

	for name, validFrom := range map[string]time.Time{
		"past":               time.Date(2031, 5, 1, 0, 0, 0, 0, time.UTC),
		"not first of month": time.Date(2031, 7, 15, 0, 0, 0, 0, time.UTC),
	} {
		if _, err := svc.Create(ctx, input(validFrom)); !errors.Is(err, service.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}

	invalid := input(time.Date(2031, 8, 1, 0, 0, 0, 0, time.UTC))
	invalid.Config.SatzungTable = nil
	if _, err := svc.Create(ctx, invalid); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("empty table: err = %v, want ErrInvalidInput", err)
	}

	planned, err := svc.Create(ctx, input(time.Date(2031, 8, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("create planned: %v", err)
	}
	if _, err := svc.Create(ctx, input(time.Date(2031, 8, 1, 0, 0, 0, 0, time.UTC))); !errors.Is(err, service.ErrConflict) {
		t.Errorf("duplicate date: err = %v, want ErrConflict", err)
	}
	if _, err := svc.Update(ctx, seeded.ID, input(time.Date(2031, 9, 1, 0, 0, 0, 0, time.UTC))); !errors.Is(err, service.ErrConflict) {
		t.Errorf("update active version: err = %v, want ErrConflict", err)
	}
	if err := svc.Delete(ctx, seeded.ID); !errors.Is(err, service.ErrConflict) {
		t.Errorf("delete active version: err = %v, want ErrConflict", err)
	}
	if err := svc.Delete(ctx, planned.ID); err != nil {
		t.Errorf("delete planned version: %v", err)
	}
}

// A new version changes generated fees from its start month on, and leaves
// earlier months untouched.
func TestFeeSchedule_NewVersionAppliesFromValidFrom(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()
	ctx := context.Background()
	restore := util.SetClock(func() time.Time { return time.Date(2031, 5, 10, 12, 0, 0, 0, time.UTC) })
	defer restore()

	scheduleRepo := repository.NewPostgresFeeScheduleRepository(testDB)
	defer testDB.Exec(`DELETE FROM fees.fee_schedules WHERE valid_from > '2025-01-01'`)
	seeded, err := scheduleRepo.GetAt(ctx, time.Date(2031, 5, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	cfg := seeded.Config
	cfg.MonthlyFoodFee = 50.00
	if _, err := service.NewFeeScheduleService(scheduleRepo).Create(ctx, service.FeeScheduleInput{
		ValidFrom: time.Date(2031, 8, 1, 0, 0, 0, 0, time.UTC), Name: "Essensgeld 2031", Config: cfg,
	}); err != nil {
		t.Fatal(err)
	}

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	child, err := createTestChild(childRepo, "FS")
	if err != nil {
		t.Fatal(err)
	}
	child.BirthDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) // no childcare fee
	child.EntryDate = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := childRepo.Update(ctx, child); err != nil {
		t.Fatal(err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, nil, nil, nil, scheduleRepo)
	for month, want := range map[int]float64{7: 45.40, 8: 50.00} {
		m := month
		if _, err := feeService.Generate(ctx, 2031, &m); err != nil {
			t.Fatalf("generate %d: %v", month, err)
		}
		var amount float64
		if err := testDB.Get(&amount, `SELECT amount FROM fees.fee_expectations WHERE child_id = $1 AND fee_type = $2 AND month = $3`,
			child.ID, domain.FeeTypeFood, month); err != nil {
			t.Fatalf("load food fee %d: %v", month, err)
		}
		if amount != want {
			t.Errorf("food fee %d/2031 = %v, want %v", month, amount, want)
		}
	}
}
