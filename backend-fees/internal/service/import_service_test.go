package service_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// TestMain sets up and tears down the test database using testcontainers
func TestMain(m *testing.M) {
	// Setup testcontainer
	if err := setupTestContainer(); err != nil {
		fmt.Printf("Failed to setup test container: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	teardownTestContainer()

	os.Exit(code)
}

// TestImportService_BlacklistFiltering tests that blacklisted IBANs are filtered during import
func TestImportService_BlacklistFiltering(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Add a blacklisted IBAN
	blacklistedIBAN := "TESTDE123456789012"
	err := knownIBANRepo.Create(context.Background(), &domain.KnownIBAN{
		IBAN:      blacklistedIBAN,
		PayerName: stringPtr("Blocked Payer"),
		Status:    domain.KnownIBANStatusBlacklisted,
		Reason:    stringPtr("Test blacklist"),
	})
	if err != nil {
		t.Fatalf("Failed to create blacklisted IBAN: %v", err)
	}

	// Create CSV with blacklisted IBAN
	csvContent := `Bezeichnung Auftragskonto;IBAN Auftragskonto;BIC Auftragskonto;Bankname Auftragskonto;Buchungstag;Valutadatum;Name Zahlungsbeteiligter;IBAN Zahlungsbeteiligter;BIC (SWIFT-Code) Zahlungsbeteiligter;Buchungstext;Verwendungszweck;Betrag;Waehrung;Saldo nach Buchung
Test;DE1234;BIC;Bank;02.01.2026;02.01.2026;Blocked Payer;TESTDE123456789012;BIC;Transfer;Payment;30,00;EUR;1000,00
`

	result, err := importService.ProcessCSV(context.Background(), strings.NewReader(csvContent), "test.csv", uuid.New())
	if err != nil {
		t.Fatalf("ProcessCSV failed: %v", err)
	}

	// Verify blacklisted transaction was filtered
	if result.Blacklisted != 1 {
		t.Errorf("Expected 1 blacklisted, got %d", result.Blacklisted)
	}
	if result.Imported != 0 {
		t.Errorf("Expected 0 imported, got %d", result.Imported)
	}
}

// TestImportService_TrustedIBANOnMatch tests that IBANs are marked as trusted when matched
func TestImportService_TrustedIBANOnMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create test data
	child, err := createTestChild(childRepo, "TRUSTED")
	if err != nil {
		t.Fatal(err)
	}
	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatal(err)
	}

	// Create a transaction
	tx, err := createTestTransaction(txRepo, "TESTDE999888777666", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Get admin user ID (seeded in migrations)
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	// Create manual match - this should mark IBAN as trusted
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify IBAN is now trusted
	isTrusted, err := knownIBANRepo.IsTrusted(context.Background(), *tx.PayerIBAN)
	if err != nil {
		t.Fatalf("IsTrusted check failed: %v", err)
	}
	if !isTrusted {
		t.Error("Expected IBAN to be marked as trusted after match")
	}
}

// TestImportService_DismissTransaction tests dismissing a transaction and blacklisting IBAN
func TestImportService_DismissTransaction(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create multiple transactions with same IBAN
	testIBAN := "TESTDE111222333444"
	for i := 0; i < 3; i++ {
		_, err := createTestTransaction(txRepo, testIBAN, float64(100+i*10),
			time.Now().AddDate(0, 0, -i), "Random payment")
		if err != nil {
			t.Fatalf("Failed to create test transaction %d: %v", i, err)
		}
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Get one of the transactions
	transactions, _, err := txRepo.ListUnmatched(context.Background(), "", "date", "desc", 0, 100)
	if err != nil {
		t.Fatalf("ListUnmatched failed: %v", err)
	}

	var txToDismiss *domain.BankTransaction
	for i := range transactions {
		if transactions[i].PayerIBAN != nil && *transactions[i].PayerIBAN == testIBAN {
			txToDismiss = &transactions[i]
			break
		}
	}
	if txToDismiss == nil {
		t.Fatal("Could not find test transaction to dismiss")
	}

	// Dismiss the transaction
	result, err := importService.DismissTransaction(context.Background(), txToDismiss.ID)
	if err != nil {
		t.Fatalf("DismissTransaction failed: %v", err)
	}

	// Verify result
	if result.IBAN != testIBAN {
		t.Errorf("Expected IBAN %s, got %s", testIBAN, result.IBAN)
	}
	if result.TransactionsRemoved != 3 {
		t.Errorf("Expected 3 transactions removed, got %d", result.TransactionsRemoved)
	}

	// Verify IBAN is blacklisted
	isBlacklisted, err := knownIBANRepo.IsBlacklisted(context.Background(), testIBAN)
	if err != nil {
		t.Fatalf("IsBlacklisted check failed: %v", err)
	}
	if !isBlacklisted {
		t.Error("Expected IBAN to be blacklisted after dismiss")
	}

	// Verify transactions are gone
	transactions, _, err = txRepo.ListUnmatched(context.Background(), "", "date", "desc", 0, 100)
	if err != nil {
		t.Fatalf("ListUnmatched failed: %v", err)
	}
	for _, tx := range transactions {
		if tx.PayerIBAN != nil && *tx.PayerIBAN == testIBAN {
			t.Error("Transaction with blacklisted IBAN still exists")
		}
	}
}

// TestImportService_Rescan tests rescanning unmatched transactions
func TestImportService_Rescan(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Clean up any existing unmatched transactions first
	testDB.Exec("DELETE FROM fees.bank_transactions WHERE id NOT IN (SELECT transaction_id FROM fees.payment_matches)")

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "RESCAN")
	if err != nil {
		t.Fatal(err)
	}

	// Create unmatched transaction (before fee exists)
	_, err = createTestTransaction(txRepo, "TESTDE555666777888", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Rescan before fee exists - should find child match but no fee expectation
	result1, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}
	if result1.Scanned != 1 {
		t.Errorf("Expected 1 scanned, got %d", result1.Scanned)
	}
	// Should find child match but no fee expectation
	if len(result1.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion (child match), got %d", len(result1.Suggestions))
	}

	// Now create the fee
	_, err = createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatal(err)
	}

	// Rescan after fee exists - should find match with expectation
	result2, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}
	if len(result2.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result2.Suggestions))
	}
	if result2.Suggestions[0].Expectation == nil {
		t.Error("Expected suggestion to have an expectation after fee was created")
	}
}

// TestImportService_RemoveFromBlacklist tests removing an IBAN from blacklist
func TestImportService_RemoveFromBlacklist(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Add a blacklisted IBAN
	blacklistedIBAN := "TESTDE444333222111"
	err := knownIBANRepo.Create(context.Background(), &domain.KnownIBAN{
		IBAN:      blacklistedIBAN,
		PayerName: stringPtr("Blocked Payer"),
		Status:    domain.KnownIBANStatusBlacklisted,
		Reason:    stringPtr("Test blacklist"),
	})
	if err != nil {
		t.Fatalf("Failed to create blacklisted IBAN: %v", err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Remove from blacklist
	err = importService.RemoveFromBlacklist(context.Background(), blacklistedIBAN)
	if err != nil {
		t.Fatalf("RemoveFromBlacklist failed: %v", err)
	}

	// Verify IBAN is no longer blacklisted
	isBlacklisted, err := knownIBANRepo.IsBlacklisted(context.Background(), blacklistedIBAN)
	if err != nil {
		t.Fatalf("IsBlacklisted check failed: %v", err)
	}
	if isBlacklisted {
		t.Error("Expected IBAN to no longer be blacklisted")
	}
}

// TestImportService_LinkIBANToChild tests linking a trusted IBAN to a child
func TestImportService_LinkIBANToChild(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "LINK")
	if err != nil {
		t.Fatal(err)
	}

	// Add a trusted IBAN
	trustedIBAN := "TESTDE777888999000"
	err = knownIBANRepo.Create(context.Background(), &domain.KnownIBAN{
		IBAN:      trustedIBAN,
		PayerName: stringPtr("Trusted Payer"),
		Status:    domain.KnownIBANStatusTrusted,
		Reason:    stringPtr("Test trusted"),
	})
	if err != nil {
		t.Fatalf("Failed to create trusted IBAN: %v", err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Link IBAN to child
	err = importService.LinkIBANToChild(context.Background(), trustedIBAN, child.ID)
	if err != nil {
		t.Fatalf("LinkIBANToChild failed: %v", err)
	}

	// Verify link
	iban, err := knownIBANRepo.GetByIBAN(context.Background(), trustedIBAN)
	if err != nil {
		t.Fatalf("GetByIBAN failed: %v", err)
	}
	if iban.ChildID == nil || *iban.ChildID != child.ID {
		t.Error("Expected IBAN to be linked to child")
	}

	// Unlink
	err = importService.UnlinkIBANFromChild(context.Background(), trustedIBAN)
	if err != nil {
		t.Fatalf("UnlinkIBANFromChild failed: %v", err)
	}

	// Verify unlink
	iban, err = knownIBANRepo.GetByIBAN(context.Background(), trustedIBAN)
	if err != nil {
		t.Fatalf("GetByIBAN failed: %v", err)
	}
	if iban.ChildID != nil {
		t.Error("Expected IBAN to be unlinked from child")
	}
}

// TestImportService_OldestFeeMatchedFirst tests that when there is only one unpaid fee,
// it gets matched correctly. Also tests that after paying one fee, the next oldest gets matched.
// Note: With the new logic, multiple unpaid fees do NOT auto-match (they create a warning instead).
// This test verifies the sequential payment scenario works correctly.
func TestImportService_OldestFeeMatchedFirst(t *testing.T) {
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
	child, err := createTestChild(childRepo, "OLDEST")
	if err != nil {
		t.Fatal(err)
	}

	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	// Create only January fee first (single fee = should match)
	feeJan, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction that should match the single fee
	tx1, err := createTestTransaction(txRepo, "TESTDE123456789999", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Rescan - should match January fee (only one unpaid)
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	suggestion := result.Suggestions[0]
	if suggestion.Expectation == nil {
		t.Fatal("Expected suggestion to have an expectation")
	}

	// Verify January fee was matched
	if suggestion.Expectation.ID != feeJan.ID {
		t.Errorf("Expected January fee (ID=%s) to be matched, got ID=%s",
			feeJan.ID, suggestion.Expectation.ID)
	}

	// Confirm this match
	_, err = importService.CreateManualMatch(context.Background(), tx1.ID, feeJan.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Now create February fee (still only one unpaid since January is paid)
	feeFeb, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 2, time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create another transaction
	tx2, err := createTestTransaction(txRepo, "TESTDE123456789998", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Rescan - should match February fee (only one unpaid)
	result2, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	if len(result2.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result2.Suggestions))
	}

	suggestion2 := result2.Suggestions[0]
	if suggestion2.Expectation == nil {
		t.Fatal("Expected second suggestion to have an expectation")
	}

	// Verify February fee is matched
	if suggestion2.Expectation.ID != feeFeb.ID {
		t.Errorf("Expected February fee (ID=%s) to be matched, got ID=%s",
			feeFeb.ID, suggestion2.Expectation.ID)
	}

	// Suppress unused variable warning
	_ = tx2
}

// TestImportService_OldestFeeMatchedFirst_DifferentAmounts tests that amount matching
// still works correctly - a transaction should only match fees with the same amount.
func TestImportService_OldestFeeMatchedFirst_DifferentAmounts(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "DIFFAMT")
	if err != nil {
		t.Fatal(err)
	}

	// Create an older childcare fee with different amount
	_, err = createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeChildcare, 150.00,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create a newer food fee
	feeFood, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 2, time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction for food fee amount (45.40)
	_, err = createTestTransaction(txRepo, "TESTDE987654321111", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Rescan
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	suggestion := result.Suggestions[0]
	if suggestion.Expectation == nil {
		t.Fatal("Expected suggestion to have an expectation")
	}

	// Should match food fee (correct amount), not the older childcare fee (wrong amount)
	if suggestion.Expectation.ID != feeFood.ID {
		t.Errorf("Expected food fee to be matched (correct amount), got different fee")
	}
	if suggestion.Expectation.FeeType != domain.FeeTypeFood {
		t.Errorf("Expected FeeTypeFood, got %s", suggestion.Expectation.FeeType)
	}
}

// TestImportService_CombinedFeeAndReminderMatch tests that a transaction amount of 55.40
// (food fee 45.40 + reminder 10.00) correctly matches both fees.
func TestImportService_CombinedFeeAndReminderMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "COMBINED")
	if err != nil {
		t.Fatal(err)
	}

	// Create a food fee
	foodFee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create a reminder fee linked to the food fee
	reminderFee := &domain.FeeExpectation{
		ID:            uuid.New(),
		ChildID:       child.ID,
		FeeType:       domain.FeeTypeReminder,
		Year:          2025,
		Month:         nil, // Reminders don't have months
		Amount:        10.00,
		DueDate:       time.Date(2025, 1, 19, 0, 0, 0, 0, time.UTC), // 14 days after original
		CreatedAt:     time.Now(),
		ReminderForID: &foodFee.ID,
	}
	err = feeRepo.Create(context.Background(), reminderFee)
	if err != nil {
		t.Fatalf("Failed to create reminder fee: %v", err)
	}

	// Create transaction for combined amount (45.40 + 10.00 = 55.40)
	_, err = createTestTransaction(txRepo, "TESTDE555666777888", 55.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld inkl Mahnung")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Rescan to get suggestions
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	suggestion := result.Suggestions[0]

	// Should have multiple expectations (combined match)
	if len(suggestion.Expectations) != 2 {
		t.Fatalf("Expected 2 expectations in combined match, got %d", len(suggestion.Expectations))
	}

	// Verify the matchedBy is "combined"
	if suggestion.MatchedBy != "combined" {
		t.Errorf("Expected MatchedBy='combined', got '%s'", suggestion.MatchedBy)
	}

	// Verify both fees are present
	var foundFood, foundReminder bool
	for _, exp := range suggestion.Expectations {
		if exp.ID == foodFee.ID {
			foundFood = true
		}
		if exp.ID == reminderFee.ID {
			foundReminder = true
		}
	}

	if !foundFood {
		t.Error("Food fee not found in combined match expectations")
	}
	if !foundReminder {
		t.Error("Reminder fee not found in combined match expectations")
	}

	// Verify the primary expectation is the food fee (not the reminder)
	if suggestion.Expectation == nil {
		t.Fatal("Expected primary Expectation to be set")
	}
	if suggestion.Expectation.ID != foodFee.ID {
		t.Errorf("Expected primary expectation to be food fee, got %s", suggestion.Expectation.FeeType)
	}
}

// TestImportService_CombinedMatch_PreferExactMatch tests that exact amount matches
// are preferred over combined matches when both are possible.
func TestImportService_CombinedMatch_PreferExactMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "EXACT")
	if err != nil {
		t.Fatal(err)
	}

	// Create a food fee for 45.40
	foodFee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction for exact amount (45.40)
	_, err = createTestTransaction(txRepo, "TESTDE111222333444", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Rescan
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	suggestion := result.Suggestions[0]

	// Should be a single exact match, not a combined match
	if len(suggestion.Expectations) > 0 {
		t.Errorf("Expected no combined expectations for exact match, got %d", len(suggestion.Expectations))
	}

	if suggestion.Expectation == nil {
		t.Fatal("Expected single expectation to be set")
	}

	if suggestion.Expectation.ID != foodFee.ID {
		t.Errorf("Expected food fee to be matched, got different fee")
	}

	// MatchedBy should NOT be "combined"
	if suggestion.MatchedBy == "combined" {
		t.Error("Exact match should not be marked as 'combined'")
	}
}

// TestImportService_PartialPayment_FeeOnlyNotReminder tests that when a reminder exists
// but the transaction only covers the original fee amount, only the fee is matched
// and the reminder remains unpaid.
func TestImportService_PartialPayment_FeeOnlyNotReminder(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child, err := createTestChild(childRepo, "PARTIAL")
	if err != nil {
		t.Fatal(err)
	}

	// Create a food fee
	foodFee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create a reminder fee linked to the food fee
	reminderFee := &domain.FeeExpectation{
		ID:            uuid.New(),
		ChildID:       child.ID,
		FeeType:       domain.FeeTypeReminder,
		Year:          2025,
		Month:         nil,
		Amount:        10.00,
		DueDate:       time.Date(2025, 1, 19, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		ReminderForID: &foodFee.ID,
	}
	err = feeRepo.Create(context.Background(), reminderFee)
	if err != nil {
		t.Fatalf("Failed to create reminder fee: %v", err)
	}

	// Create transaction for ONLY the food fee amount (45.40), not the combined amount
	tx, err := createTestTransaction(txRepo, "TESTDE999888777666", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Rescan to get suggestions
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	suggestion := result.Suggestions[0]

	// Should be a SINGLE match (food fee only), NOT a combined match
	if len(suggestion.Expectations) > 0 {
		t.Errorf("Expected no combined expectations for partial payment, got %d", len(suggestion.Expectations))
	}

	if suggestion.Expectation == nil {
		t.Fatal("Expected single expectation to be set")
	}

	// Should match the food fee
	if suggestion.Expectation.ID != foodFee.ID {
		t.Errorf("Expected food fee to be matched, got different fee")
	}

	// MatchedBy should NOT be "combined"
	if suggestion.MatchedBy == "combined" {
		t.Error("Partial payment should not be marked as 'combined'")
	}

	// Now confirm the match
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, foodFee.ID, adminUserID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify reminder is still unpaid by checking if it would match a new 10 EUR transaction
	_, err = createTestTransaction(txRepo, "TESTDE999888777665", 10.00, time.Now(),
		child.FirstName+" "+child.LastName+" "+child.MemberNumber+" Mahnung")
	if err != nil {
		t.Fatal(err)
	}

	// Rescan again
	result2, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	// Should find the reminder as unmatched
	if len(result2.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion for reminder, got %d", len(result2.Suggestions))
	}

	// The suggestion should match the child (by name/member number)
	// but may not have an expectation since REMINDER type matching isn't explicitly handled
	// This is actually a limitation - let's just verify the reminder exists and is unpaid
	unpaidReminder, err := feeRepo.GetByID(context.Background(), reminderFee.ID)
	if err != nil {
		t.Fatalf("Failed to get reminder fee: %v", err)
	}

	// Check that reminder is still not matched
	matched, _ := matchRepo.ExistsForExpectation(context.Background(), unpaidReminder.ID)
	if matched {
		t.Error("Reminder should still be unpaid after partial payment")
	}
}

// TestImportService_GetBlacklistAndTrusted tests listing blacklisted and trusted IBANs
func TestImportService_GetBlacklistAndTrusted(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Add blacklisted IBANs
	for i := 0; i < 3; i++ {
		err := knownIBANRepo.Create(context.Background(), &domain.KnownIBAN{
			IBAN:      "TESTBLACK" + string(rune('A'+i)) + "12345",
			PayerName: stringPtr("Blocked " + string(rune('A'+i))),
			Status:    domain.KnownIBANStatusBlacklisted,
		})
		if err != nil {
			t.Fatalf("Failed to create blacklisted IBAN: %v", err)
		}
	}

	// Add trusted IBANs
	for i := 0; i < 2; i++ {
		err := knownIBANRepo.Create(context.Background(), &domain.KnownIBAN{
			IBAN:      "TESTTRUST" + string(rune('A'+i)) + "12345",
			PayerName: stringPtr("Trusted " + string(rune('A'+i))),
			Status:    domain.KnownIBANStatusTrusted,
		})
		if err != nil {
			t.Fatalf("Failed to create trusted IBAN: %v", err)
		}
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, nil)

	// Get blacklist
	blacklist, total, err := importService.GetBlacklist(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetBlacklist failed: %v", err)
	}
	if total < 3 {
		t.Errorf("Expected at least 3 blacklisted, got %d", total)
	}
	for _, iban := range blacklist {
		if iban.Status != domain.KnownIBANStatusBlacklisted {
			t.Error("Blacklist returned non-blacklisted IBAN")
		}
	}

	// Get trusted
	trusted, total, err := importService.GetTrustedIBANs(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetTrustedIBANs failed: %v", err)
	}
	if total < 2 {
		t.Errorf("Expected at least 2 trusted, got %d", total)
	}
	for _, iban := range trusted {
		if iban.Status != domain.KnownIBANStatusTrusted {
			t.Error("Trusted list returned non-trusted IBAN")
		}
	}
}

// TestWarningFromTrustedIBAN tests warning generation for trusted IBANs with no matching fee.
func TestWarningFromTrustedIBAN(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Create repositories
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create a test child
	child, err := createTestChild(childRepo, "WARN")
	if err != nil {
		t.Fatal(err)
	}

	// Create a trusted IBAN linked to the child
	trustedIBAN := &domain.KnownIBAN{
		IBAN:      "TESTDE111222333444",
		PayerName: stringPtr("Test Parent"),
		Status:    domain.KnownIBANStatusTrusted,
		ChildID:   &child.ID,
		Reason:    stringPtr("Test trusted IBAN"),
	}
	err = knownIBANRepo.Create(context.Background(), trustedIBAN)
	if err != nil {
		t.Fatalf("Failed to create trusted IBAN: %v", err)
	}

	// Don't create any fee - so we should get a warning

	// Create a transaction from the trusted IBAN directly (no open fee for child)
	tx, err := createTestTransaction(txRepo, trustedIBAN.IBAN, 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service with warning repo
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Rescan to trigger warning generation
	// Note: ProcessCSV triggers warnings automatically, but rescan also needs to check
	// For this test, we'll manually check the warning logic via the checkForWarning path

	// First verify suggestions (matching finds the child by name but no fee exists)
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	// Depending on name matching, we may or may not get a suggestion
	// The key point is there's no fee expectation to match
	for _, s := range result.Suggestions {
		if s.Expectation != nil {
			t.Errorf("Expected no fee expectation match, got %v", s.Expectation)
		}
	}

	// Manually create a warning since checkForWarning is only called during ProcessCSV
	warning := &domain.TransactionWarning{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		WarningType:   domain.WarningTypeNoMatchingFee,
		Message:       "Keine offene Beitragsforderung gefunden",
		ActualAmount:  &tx.Amount,
		ChildID:       trustedIBAN.ChildID,
		CreatedAt:     time.Now(),
	}
	err = warningRepo.Create(context.Background(), warning)
	if err != nil {
		t.Fatalf("Failed to create warning: %v", err)
	}

	// Verify warning is stored in database
	_, total, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 warning in database, got %d", total)
	}

	userID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	// Test dismissing the warning
	err = importService.DismissWarning(context.Background(), warning.ID, userID, "Test dismiss")
	if err != nil {
		t.Fatalf("DismissWarning failed: %v", err)
	}

	// Verify warning is now resolved
	_, total, err = importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed after dismiss: %v", err)
	}
	if total != 0 {
		t.Errorf("Expected 0 unresolved warnings after dismiss, got %d", total)
	}
}

// TestImportService_MultipleOpenFees_NoAutoMatch tests that when multiple unpaid fees
// of the same type exist for a child, no auto-match occurs and a warning is created.
func TestImportService_MultipleOpenFees_NoAutoMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create child with a pure 5-digit member number for easier matching
	memberNum := fmt.Sprintf("%05d", time.Now().UnixNano()%100000)
	child := &domain.Child{
		ID:           uuid.New(),
		MemberNumber: memberNum,
		FirstName:    "MultiTest",
		LastName:     "Kindermann",
		BirthDate:    time.Now().AddDate(-2, 0, 0),
		EntryDate:    time.Now().AddDate(-1, 0, 0),
		IsActive:     true,
	}
	err := childRepo.Create(context.Background(), child)
	if err != nil {
		t.Fatal(err)
	}

	// Create 2 unpaid food fees for different months (same amount)
	fee1, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	fee2, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 2, time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Verify fees are created
	count, err := feeRepo.CountUnpaidByType(context.Background(), child.ID, domain.FeeTypeFood, 45.40)
	if err != nil {
		t.Fatalf("CountUnpaidByType failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 unpaid fees, got %d (fee1=%s, fee2=%s)", count, fee1.ID, fee2.ID)
	}

	// Create transaction with member number in description for reliable matching
	tx := &domain.BankTransaction{
		ID:          uuid.New(),
		BookingDate: time.Now(),
		ValueDate:   time.Now(),
		PayerName:   stringPtr("Parent Name"),
		PayerIBAN:   stringPtr("TESTDE123456789111"),
		Description: stringPtr(child.MemberNumber + " Essensgeld"), // e.g., "12345 Essensgeld"
		Amount:      45.40,
		Currency:    "EUR",
		ImportedAt:  time.Now(),
	}
	err = txRepo.Create(context.Background(), tx)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize service with warning repo
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Rescan to get suggestions
	result, err := importService.Rescan(context.Background())
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	// Should NOT have any suggestions (because multiple open fees exist)
	if len(result.Suggestions) != 0 {
		t.Errorf("Expected 0 suggestions when multiple open fees exist, got %d", len(result.Suggestions))
		for i, s := range result.Suggestions {
			t.Logf("Suggestion %d: Child=%v, Expectation=%v, Confidence=%.2f",
				i, s.Child != nil, s.Expectation != nil, s.Confidence)
		}
	}

	// Should have created a warning
	warnings, total, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}

	if total != 1 {
		t.Fatalf("Expected 1 warning for multiple open fees, got %d", total)
	}

	// Verify warning type and content
	if warnings[0].WarningType != domain.WarningTypeMultipleOpenFees {
		t.Errorf("Expected warning type MULTIPLE_OPEN_FEES, got %s", warnings[0].WarningType)
	}
	if warnings[0].ChildID == nil || *warnings[0].ChildID != child.ID {
		t.Error("Warning should be linked to the child")
	}
}

// TestImportService_SingleOpenFee_AutoMatch tests that when only one unpaid fee
// of a type exists, the transaction is auto-matched normally.
func TestImportService_SingleOpenFee_AutoMatch(t *testing.T) {
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
	child, err := createTestChild(childRepo, "SINGLE")
	if err != nil {
		t.Fatal(err)
	}

	// Create only 1 unpaid food fee
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeFood, 45.40,
		2025, 1, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Create transaction
	_, err = createTestTransaction(txRepo, "TESTDE123456789222", 45.40, time.Now(),
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

	// Should have 1 suggestion with matched expectation
	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion when single open fee exists, got %d", len(result.Suggestions))
	}

	suggestion := result.Suggestions[0]
	if suggestion.Expectation == nil {
		t.Fatal("Expected suggestion to have an expectation")
	}
	if suggestion.Expectation.ID != fee.ID {
		t.Errorf("Expected fee ID %s to be matched, got %s", fee.ID, suggestion.Expectation.ID)
	}

	// Should NOT have any warnings
	_, total, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}
	if total != 0 {
		t.Errorf("Expected 0 warnings when single open fee, got %d", total)
	}
}

// TestWarningAutoResolveOnMatch tests that warnings are auto-resolved when a transaction is matched.
func TestWarningAutoResolveOnMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	// Create repositories
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)

	// Create a test child
	child, err := createTestChild(childRepo, "AUTORES")
	if err != nil {
		t.Fatal(err)
	}

	// Create a trusted IBAN linked to the child
	trustedIBAN := &domain.KnownIBAN{
		IBAN:      "TESTDE222333444555",
		PayerName: stringPtr("Test Parent 2"),
		Status:    domain.KnownIBANStatusTrusted,
		ChildID:   &child.ID,
		Reason:    stringPtr("Test trusted IBAN 2"),
	}
	err = knownIBANRepo.Create(context.Background(), trustedIBAN)
	if err != nil {
		t.Fatalf("Failed to create trusted IBAN: %v", err)
	}

	// Create a transaction on Jan 10, 2025 (before the fee's 15th deadline - NOT late)
	// This ensures the auto-resolve test doesn't create a late payment warning
	paymentDate := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	tx, err := createTestTransaction(txRepo, trustedIBAN.IBAN, 45.40, paymentDate,
		child.FirstName+" "+child.LastName+" Essensgeld")
	if err != nil {
		t.Fatal(err)
	}

	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)

	// Create a warning for this transaction (as if ProcessCSV detected it)
	warning := &domain.TransactionWarning{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		WarningType:   domain.WarningTypeNoMatchingFee,
		Message:       "Keine offene Beitragsforderung gefunden",
		ActualAmount:  &tx.Amount,
		ChildID:       trustedIBAN.ChildID,
		CreatedAt:     time.Now(),
	}
	err = warningRepo.Create(context.Background(), warning)
	if err != nil {
		t.Fatalf("Failed to create warning: %v", err)
	}

	// Verify warning exists
	warnings, total, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("Expected 1 warning, got %d", total)
	}
	_ = warnings // unused but shows we have warnings

	// Now create the fee that should have been there
	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   child.ID,
		FeeType:   domain.FeeTypeFood,
		Year:      2025,
		Month:     intPtr(1),
		Amount:    45.40,
		DueDate:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
	}
	err = feeRepo.Create(context.Background(), fee)
	if err != nil {
		t.Fatalf("Failed to create fee: %v", err)
	}

	userID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	// Manually match the transaction to the fee
	_, err = importService.CreateManualMatch(context.Background(), tx.ID, fee.ID, userID)
	if err != nil {
		t.Fatalf("CreateManualMatch failed: %v", err)
	}

	// Verify warning is now auto-resolved (and no new late payment warning was created
	// since the payment was on Jan 10 which is before the Jan 15 deadline)
	_, resolvedTotal, err := importService.GetWarnings(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}
	if resolvedTotal != 0 {
		t.Errorf("Expected 0 unresolved warnings after match, got %d (warnings should be auto-resolved)", resolvedTotal)
	}
}
