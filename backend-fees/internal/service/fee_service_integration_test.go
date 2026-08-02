package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestFeeService_Generate_UsesEnrollmentPeriodForMonthlyFees(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	feeService := service.NewFeeService(feeRepo, childRepo, nil, nil, nil)

	createChild := func(suffix string, entryDate time.Time, exitDate *time.Time, active bool) *domain.Child {
		now := time.Now()
		child := &domain.Child{
			ID:           uuid.New(),
			MemberNumber: fmt.Sprintf("TGEN%s", suffix),
			FirstName:    "Test",
			LastName:     "Generation",
			BirthDate:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			EntryDate:    entryDate,
			ExitDate:     exitDate,
			IsActive:     active,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := childRepo.Create(ctx, child); err != nil {
			t.Fatalf("create child %s: %v", suffix, err)
		}
		return child
	}

	july31 := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	august15 := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	exitedBeforeAugust := createChild("01", time.Date(2021, 8, 1, 0, 0, 0, 0, time.UTC), &july31, true)
	startsAfterAugust := createChild("02", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), nil, true)
	inactiveButEnrolled := createChild("03", time.Date(2021, 8, 1, 0, 0, 0, 0, time.UTC), nil, false)
	exitsDuringAugust := createChild("04", time.Date(2021, 8, 1, 0, 0, 0, 0, time.UTC), &august15, true)

	month := 8
	result, err := feeService.Generate(ctx, 2026, &month)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.Created != 2 || result.Skipped != 0 {
		t.Fatalf("expected 2 created and 0 skipped, got %d created and %d skipped", result.Created, result.Skipped)
	}

	assertFeeCount := func(child *domain.Child, expected int) {
		fees, err := feeRepo.GetForChild(ctx, child.ID, nil)
		if err != nil {
			t.Fatalf("get fees for %s: %v", child.MemberNumber, err)
		}
		if len(fees) != expected {
			t.Errorf("expected %d fees for %s, got %d", expected, child.MemberNumber, len(fees))
		}
	}
	assertFeeCount(exitedBeforeAugust, 0)
	assertFeeCount(startsAfterAugust, 0)
	assertFeeCount(inactiveButEnrolled, 1)
	assertFeeCount(exitsDuringAugust, 1)
}

func TestFeeService_GetByID_PartialMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)

	child, err := createTestChild(childRepo, "PM")
	if err != nil {
		t.Fatal(err)
	}

	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatal(err)
	}

	tx, err := createTestTransaction(txRepo, "TESTDE444555666777", 20.00, time.Now(),
		child.FirstName+" "+child.LastName+" Essensgeld "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	match := &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        20.00,
		MatchType:     domain.MatchTypeManual,
		MatchedAt:     time.Now(),
	}
	if err := matchRepo.Create(context.Background(), match); err != nil {
		t.Fatalf("matchRepo.Create failed: %v", err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo)
	loaded, err := feeService.GetByID(context.Background(), fee.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if loaded.MatchedAmount != 20.00 {
		t.Fatalf("Expected matchedAmount 20.00, got %.2f", loaded.MatchedAmount)
	}
	if loaded.Remaining < 25.39 || loaded.Remaining > 25.41 {
		t.Fatalf("Expected remaining ~25.40, got %.2f", loaded.Remaining)
	}
	if loaded.IsPaid {
		t.Fatalf("Expected fee to be unpaid for partial match")
	}
	if len(loaded.PartialMatches) != 1 {
		t.Fatalf("Expected 1 partial match, got %d", len(loaded.PartialMatches))
	}
}

func TestFeeService_GetByID_FullMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)

	child, err := createTestChild(childRepo, "FM")
	if err != nil {
		t.Fatal(err)
	}

	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatal(err)
	}

	tx, err := createTestTransaction(txRepo, "TESTDE888999000111", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" Essensgeld "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	match := &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        45.40,
		MatchType:     domain.MatchTypeAuto,
		MatchedAt:     time.Now(),
	}
	if err := matchRepo.Create(context.Background(), match); err != nil {
		t.Fatalf("matchRepo.Create failed: %v", err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo)
	loaded, err := feeService.GetByID(context.Background(), fee.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if loaded.MatchedAmount < 45.39 || loaded.MatchedAmount > 45.41 {
		t.Fatalf("Expected matchedAmount 45.40, got %.2f", loaded.MatchedAmount)
	}
	if loaded.Remaining != 0 {
		t.Fatalf("Expected remaining 0.00, got %.2f", loaded.Remaining)
	}
	if !loaded.IsPaid {
		t.Fatalf("Expected fee to be paid")
	}
}

func TestFeeService_SyncChildcareExpectationsFrom_AdjustsPaidIncrease(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)

	child, err := createTestChild(childRepo, "SI")
	if err != nil {
		t.Fatal(err)
	}
	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeChildcare, 120, 2026, 6)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := createTestTransaction(txRepo, "TESTDE111222333444", 120, time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC), "Platzgeld")
	if err != nil {
		t.Fatal(err)
	}
	if err := matchRepo.Create(context.Background(), &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        120,
		MatchType:     domain.MatchTypeManual,
		MatchedAt:     time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo)
	exitDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	result, err := feeService.SyncChildcareExpectationsFrom(context.Background(), child.ID, child.HouseholdID, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 150, &exitDate)
	if err != nil {
		t.Fatalf("SyncChildcareExpectationsFrom failed: %v", err)
	}

	if result.Updated != 1 {
		t.Fatalf("expected 1 updated fee, got %d", result.Updated)
	}
	if result.DeltaOpen < 30-0.01 || result.DeltaOpen > 30+0.01 {
		t.Fatalf("expected deltaOpen 30.00, got %.2f", result.DeltaOpen)
	}
	loaded, err := feeService.GetByID(context.Background(), fee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Amount != 150 {
		t.Fatalf("expected amount 150.00, got %.2f", loaded.Amount)
	}
	if loaded.Remaining < 30-0.01 || loaded.Remaining > 30+0.01 {
		t.Fatalf("expected remaining 30.00, got %.2f", loaded.Remaining)
	}
}

func TestFeeService_SyncChildcareExpectationsFrom_ReportsCreditReviewForDecrease(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)

	child, err := createTestChild(childRepo, "SD")
	if err != nil {
		t.Fatal(err)
	}
	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeChildcare, 120, 2026, 6)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := createTestTransaction(txRepo, "TESTDE555666777888", 120, time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC), "Platzgeld")
	if err != nil {
		t.Fatal(err)
	}
	if err := matchRepo.Create(context.Background(), &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        120,
		MatchType:     domain.MatchTypeManual,
		MatchedAt:     time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo)
	exitDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	result, err := feeService.SyncChildcareExpectationsFrom(context.Background(), child.ID, child.HouseholdID, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 100, &exitDate)
	if err != nil {
		t.Fatalf("SyncChildcareExpectationsFrom failed: %v", err)
	}

	if len(result.CreditReviewRequired) != 1 {
		t.Fatalf("expected one credit review item, got %d", len(result.CreditReviewRequired))
	}
	if result.CreditReviewRequired[0].CreditAmount < 20-0.01 || result.CreditReviewRequired[0].CreditAmount > 20+0.01 {
		t.Fatalf("expected credit amount 20.00, got %.2f", result.CreditReviewRequired[0].CreditAmount)
	}
	loaded, err := feeService.GetByID(context.Background(), fee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Amount != 120 {
		t.Fatalf("expected original amount to remain 120.00, got %.2f", loaded.Amount)
	}
}
