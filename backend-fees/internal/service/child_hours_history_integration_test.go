package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// The hours exposed on domain.Child are derived from the history tables. These tests pin
// that behavior after removing the denormalized fees.children hour columns.
func TestChildHours_DerivedFromHistoryWithoutWrites(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)

	today := time.Now().Truncate(24 * time.Hour)
	entryDate := today.AddDate(-1, 0, 0)

	child := &domain.Child{
		ID:           uuid.New(),
		MemberNumber: "THRS1",
		FirstName:    "Test",
		LastName:     "Kind",
		BirthDate:    today.AddDate(-2, 0, 0),
		EntryDate:    entryDate,
		LegalHours:   intPtr(30),
		CareHours:    intPtr(30),
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := childRepo.Create(ctx, child); err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	loaded, err := childRepo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if loaded.CareHours == nil || *loaded.CareHours != 30 {
		t.Fatalf("expected initial care hours 30, got %v", loaded.CareHours)
	}
	if loaded.LegalHours == nil || *loaded.LegalHours != 30 {
		t.Fatalf("expected initial legal hours 30, got %v", loaded.LegalHours)
	}

	// A period starting today becomes effective without any further write to the child row.
	if err := childRepo.UpsertCareHoursHistory(ctx, child.ID, intPtr(40), today); err != nil {
		t.Fatalf("UpsertCareHoursHistory (today) failed: %v", err)
	}
	// A period starting tomorrow must not leak into the currently effective value.
	if err := childRepo.UpsertCareHoursHistory(ctx, child.ID, intPtr(45), today.AddDate(0, 0, 1)); err != nil {
		t.Fatalf("UpsertCareHoursHistory (tomorrow) failed: %v", err)
	}

	loaded, err = childRepo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if loaded.CareHours == nil || *loaded.CareHours != 40 {
		t.Fatalf("expected currently effective care hours 40, got %v", loaded.CareHours)
	}

	// The same derived value must be used by every read path.
	byMemberNumber, err := childRepo.GetByMemberNumber(ctx, child.MemberNumber)
	if err != nil {
		t.Fatalf("GetByMemberNumber failed: %v", err)
	}
	if byMemberNumber.CareHours == nil || *byMemberNumber.CareHours != 40 {
		t.Fatalf("GetByMemberNumber: expected care hours 40, got %v", byMemberNumber.CareHours)
	}

	byIDs, err := childRepo.GetByIDs(ctx, []uuid.UUID{child.ID})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}
	if got := byIDs[child.ID]; got == nil || got.CareHours == nil || *got.CareHours != 40 {
		t.Fatalf("GetByIDs: expected care hours 40, got %v", got)
	}
}

// A child that has not started yet has no currently effective period, so the derived value
// is empty while the future period is still recorded and used for fee calculation.
func TestChildHours_FuturePeriodIsNotCurrentButDrivesFees(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)

	today := time.Now().Truncate(24 * time.Hour)
	entryDate := today.AddDate(0, 0, 5)

	child := &domain.Child{
		ID:           uuid.New(),
		MemberNumber: "THRS2",
		FirstName:    "Test",
		LastName:     "Kind",
		BirthDate:    today.AddDate(-2, 0, 0),
		EntryDate:    entryDate,
		CareHours:    intPtr(35),
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := childRepo.Create(ctx, child); err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	loaded, err := childRepo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if loaded.CareHours != nil {
		t.Fatalf("expected no currently effective care hours, got %v", *loaded.CareHours)
	}

	history, err := childRepo.ListCareHoursHistory(ctx, child.ID)
	if err != nil {
		t.Fatalf("ListCareHoursHistory failed: %v", err)
	}
	if len(history) != 1 || history[0].CareHours == nil || *history[0].CareHours != 35 {
		t.Fatalf("expected one recorded period with 35 hours, got %+v", history)
	}

	// Fee calculation must use the recorded period, not a default.
	feeService := service.NewFeeService(
		repository.NewPostgresFeeRepository(testDB),
		childRepo,
		repository.NewPostgresHouseholdRepository(testDB),
		repository.NewPostgresMatchRepository(testDB),
		repository.NewPostgresTransactionRepository(testDB),
	)
	month := int(entryDate.Month())
	hours := feeService.ResolveCareHours(ctx, loaded, entryDate.Year(), &month)
	if hours != 35 {
		t.Fatalf("expected fee calculation to use 35 hours, got %d", hours)
	}
}

// Children whose hours only start in the future are fully documented and must not be
// reported as missing master data.
func TestChildHours_WarningsUseHistoryNotCurrentPeriod(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	parentRepo := repository.NewPostgresParentRepository(testDB)

	today := time.Now().Truncate(24 * time.Hour)
	entryDate := today.AddDate(0, 0, 5)

	child := &domain.Child{
		ID:           uuid.New(),
		MemberNumber: "THRS3",
		FirstName:    "Test",
		LastName:     "Kind",
		BirthDate:    today.AddDate(-2, 0, 0),
		EntryDate:    entryDate,
		LegalHours:   intPtr(35),
		CareHours:    intPtr(35),
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := childRepo.Create(ctx, child); err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	parent := &domain.Parent{
		ID:                    uuid.New(),
		FirstName:             "Test",
		LastName:              "Elternteil",
		Email:                 stringPtr("hours-warning@example.test"),
		AnnualHouseholdIncome: float64Ptr(40000),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := parentRepo.Create(ctx, parent); err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}
	if err := childRepo.LinkParent(ctx, child.ID, parent.ID, true); err != nil {
		t.Fatalf("failed to link parent: %v", err)
	}

	warned, _, err := childRepo.List(ctx, false, false, true, false, child.MemberNumber, "", "", 0, 50)
	if err != nil {
		t.Fatalf("List(hasWarnings) failed: %v", err)
	}
	for _, c := range warned {
		if c.ID == child.ID {
			t.Fatalf("child with future hours periods must not be reported as missing hours")
		}
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}
