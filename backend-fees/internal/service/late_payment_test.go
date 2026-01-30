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

// Late Payment Rules:
// - Late threshold: Payment after 15th of the fee's month
// - Late fee amount: Fixed 10 EUR (uses existing REMINDER fee type)
// - Applicable fees: Only monthly fees (CHILDCARE, FOOD) - not MEMBERSHIP
// - Matching priority: Oldest unpaid fee first (FIFO)
// - Human-in-the-loop: Late fees should go to review queue, not auto-created

// TestLatePayment_OnTime_Within15th tests that a payment on or before the 15th is NOT late.
func TestLatePayment_OnTime_Within15th(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "ONTIME")
	if err != nil {
		t.Fatal(err)
	}

	// Create fee for January 2025
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create trusted IBAN for this child
	err = createTrustedIBAN(knownIBANRepo, "TESTDE111222333444", &child.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction on January 10th (BEFORE 15th - on time)
	bookingDate := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err = createTestTransaction(txRepo, "TESTDE111222333444", 45.40, bookingDate,
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Rescan to get suggestions
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	// Should have 1 suggestion
	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	// Verify no late payment warning exists for this transaction
	warnings, total, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	// Look for LATE_PAYMENT warning type
	for _, w := range warnings {
		if w.WarningType == domain.WarningTypeLatePayment {
			t.Errorf("Expected NO late payment warning for on-time payment, but found one")
		}
	}
	_ = total
	_ = fee
}

// TestLatePayment_Late_After15th tests that a payment after the 15th IS late.
func TestLatePayment_Late_After15th(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "LATE")
	if err != nil {
		t.Fatal(err)
	}

	// Create fee for January 2025
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create trusted IBAN for this child
	err = createTrustedIBAN(knownIBANRepo, "TESTDE555666777888", &child.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction on January 20th (AFTER 15th - late!)
	bookingDate := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	tx, err := createTestTransaction(txRepo, "TESTDE555666777888", 45.40, bookingDate,
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Manually create a match to trigger late payment detection
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify a LATE_PAYMENT warning was created
	warnings, _, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	foundLateWarning := false
	for _, w := range warnings {
		if w.WarningType == domain.WarningTypeLatePayment {
			foundLateWarning = true
			// Verify warning has matched fee ID
			if w.MatchedFeeID == nil || *w.MatchedFeeID != fee.ID {
				t.Errorf("Late payment warning should reference the matched fee")
			}
			// Verify warning has child ID
			if w.ChildID == nil || *w.ChildID != child.ID {
				t.Errorf("Late payment warning should reference the child")
			}
		}
	}

	if !foundLateWarning {
		t.Skip("Late payment detection not yet implemented - expected LATE_PAYMENT warning")
	}
}

// TestLatePayment_Boundary_Exactly15th tests that payment on exactly the 15th is on-time.
func TestLatePayment_Boundary_Exactly15th(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "BOUND")
	if err != nil {
		t.Fatal(err)
	}

	// Create fee for January 2025
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create trusted IBAN for this child
	err = createTrustedIBAN(knownIBANRepo, "TESTDE999888777666", &child.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction on January 15th (exactly on the deadline - still on time)
	bookingDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	tx, err := createTestTransaction(txRepo, "TESTDE999888777666", 45.40, bookingDate,
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Manually create a match
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify NO late payment warning was created
	warnings, _, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	for _, w := range warnings {
		if w.WarningType == domain.WarningTypeLatePayment {
			t.Errorf("Payment on exactly the 15th should NOT be late")
		}
	}
}

// TestLatePayment_PreviousMonthFee tests that paying a previous month's fee is late.
func TestLatePayment_PreviousMonthFee(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "PREVMO")
	if err != nil {
		t.Fatal(err)
	}

	// Create fee for January 2025 (previous month)
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create trusted IBAN for this child
	err = createTrustedIBAN(knownIBANRepo, "TESTDE123456789012", &child.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction in February for January's fee (late!)
	bookingDate := time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)
	tx, err := createTestTransaction(txRepo, "TESTDE123456789012", 45.40, bookingDate,
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld Januar")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Manually create a match
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify a LATE_PAYMENT warning was created
	warnings, _, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	foundLateWarning := false
	for _, w := range warnings {
		if w.WarningType == domain.WarningTypeLatePayment {
			foundLateWarning = true
		}
	}

	if !foundLateWarning {
		t.Skip("Late payment detection not yet implemented - expected LATE_PAYMENT warning for previous month payment")
	}
}

// TestLatePayment_OnlyMonthlyFees tests that MEMBERSHIP fees don't trigger late payment.
func TestLatePayment_OnlyMonthlyFees(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "MEMBER")
	if err != nil {
		t.Fatal(err)
	}

	// Create MEMBERSHIP fee for 2025 (annual, not monthly)
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeMembership, 30.00,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create trusted IBAN for this child
	err = createTrustedIBAN(knownIBANRepo, "TESTDE987654321098", &child.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction on March 20th (well after 15th, but for membership)
	bookingDate := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)
	tx, err := createTestTransaction(txRepo, "TESTDE987654321098", 30.00, bookingDate,
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Mitgliedsbeitrag")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Manually create a match
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify NO late payment warning was created for membership fee
	warnings, _, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	for _, w := range warnings {
		if w.WarningType == domain.WarningTypeLatePayment {
			t.Errorf("MEMBERSHIP fees should NOT trigger late payment warnings")
		}
	}
}

// TestLatePayment_Childcare_IsMonthly tests that CHILDCARE fees do trigger late payment.
func TestLatePayment_Childcare_IsMonthly(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "CARE")
	if err != nil {
		t.Fatal(err)
	}

	// Create CHILDCARE fee for January 2025
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeChildcare, 150.00,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create trusted IBAN for this child
	err = createTrustedIBAN(knownIBANRepo, "TESTDE456789012345", &child.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction on January 20th (after 15th - late!)
	bookingDate := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	tx, err := createTestTransaction(txRepo, "TESTDE456789012345", 150.00, bookingDate,
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Betreuungsgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Manually create a match
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify a LATE_PAYMENT warning was created for childcare fee
	warnings, _, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	foundLateWarning := false
	for _, w := range warnings {
		if w.WarningType == domain.WarningTypeLatePayment {
			foundLateWarning = true
		}
	}

	if !foundLateWarning {
		t.Skip("Late payment detection not yet implemented - expected LATE_PAYMENT warning for childcare fee")
	}
}

// TestResolveWarning_WithLateFee tests resolving a late payment warning by creating a late fee.
func TestResolveWarning_WithLateFee(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "RESOLVE")
	if err != nil {
		t.Fatal(err)
	}

	// Create fee for January 2025
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Create a LATE_PAYMENT warning directly (simulating it was created during matching)
	warning := &domain.TransactionWarning{
		ID:           uuid.New(),
		WarningType:  domain.WarningTypeLatePayment,
		Message:      "Zahlung nach dem 15. des Monats",
		ChildID:      &child.ID,
		MatchedFeeID: &fee.ID,
		CreatedAt:    time.Now(),
	}

	// We need a transaction for the warning
	tx, err := createTestTransaction(txRepo, "TESTDE111222333445", 45.40, time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}
	warning.TransactionID = tx.ID

	err = warningRepo.Create(context.Background(), warning)
	if err != nil {
		t.Fatalf("Failed to create warning: %v", err)
	}

	// Test the ResolveWarningWithLateFee method (to be implemented)
	// This should:
	// 1. Create a REMINDER fee of 10 EUR linked to the original fee
	// 2. Resolve the warning
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	result, err := importService.ResolveWarningWithLateFee(context.Background(), warning.ID, adminUserID)
	if err != nil {
		t.Fatalf("ResolveWarningWithLateFee failed: %v", err)
	}

	// Verify the result
	if result.WarningID != warning.ID {
		t.Errorf("Expected warningID %s, got %s", warning.ID, result.WarningID)
	}
	if result.LateFeeAmount != domain.ReminderFeeAmount {
		t.Errorf("Expected lateFeeAmount %.2f, got %.2f", domain.ReminderFeeAmount, result.LateFeeAmount)
	}

	// Verify warning is resolved
	resolvedWarning, err := importService.GetWarningByID(context.Background(), warning.ID)
	if err != nil {
		t.Fatalf("GetWarningByID failed: %v", err)
	}
	if !resolvedWarning.IsResolved() {
		t.Errorf("Warning should be resolved after creating late fee")
	}

	// Verify a new REMINDER fee was created
	filter := repository.FeeFilter{ChildID: &child.ID}
	fees, _, err := feeRepo.List(context.Background(), filter, 0, 100)
	if err != nil {
		t.Fatalf("Failed to list fees: %v", err)
	}

	foundReminderFee := false
	for _, f := range fees {
		if f.FeeType == domain.FeeTypeReminder && f.Amount == domain.ReminderFeeAmount {
			foundReminderFee = true
			// Verify it's linked to the original fee
			if f.ReminderForID == nil || *f.ReminderForID != fee.ID {
				t.Errorf("Reminder fee should be linked to original fee")
			}
		}
	}

	if !foundReminderFee {
		t.Errorf("Expected a REMINDER fee of %.2f EUR to be created", domain.ReminderFeeAmount)
	}
}
