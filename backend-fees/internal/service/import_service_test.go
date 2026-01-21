package service_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// testDB holds the test database connection
var testDB *sqlx.DB

// TestMain sets up and tears down the test database
func TestMain(m *testing.M) {
	// Get database URL from environment or use default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable"
	}

	var err error
	testDB, err = sqlx.Connect("postgres", dbURL)
	if err != nil {
		panic("Failed to connect to test database: " + err.Error())
	}
	defer testDB.Close()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// cleanupTestData removes test data created during tests
func cleanupTestData(t *testing.T) {
	t.Helper()

	// Clean up in reverse order of dependencies
	testDB.Exec("DELETE FROM fees.payment_matches WHERE transaction_id IN (SELECT id FROM fees.bank_transactions WHERE payer_iban LIKE 'TEST%')")
	testDB.Exec("DELETE FROM fees.bank_transactions WHERE payer_iban LIKE 'TEST%'")
	testDB.Exec("DELETE FROM fees.known_ibans WHERE iban LIKE 'TEST%'")
	testDB.Exec("DELETE FROM fees.fee_expectations WHERE child_id IN (SELECT id FROM fees.children WHERE member_number LIKE 'TEST%')")
	testDB.Exec("DELETE FROM fees.child_parents WHERE child_id IN (SELECT id FROM fees.children WHERE member_number LIKE 'TEST%')")
	testDB.Exec("DELETE FROM fees.children WHERE member_number LIKE 'TEST%'")
}

// createTestChild creates a child for testing
func createTestChild(t *testing.T, childRepo repository.ChildRepository) *domain.Child {
	t.Helper()

	child := &domain.Child{
		ID:           uuid.New(),
		MemberNumber: "TEST" + time.Now().Format("150405"),
		FirstName:    "Test",
		LastName:     "Kind",
		BirthDate:    time.Now().AddDate(-2, 0, 0), // 2 years old
		EntryDate:    time.Now().AddDate(-1, 0, 0),
		IsActive:     true,
	}

	err := childRepo.Create(context.Background(), child)
	if err != nil {
		t.Fatalf("Failed to create test child: %v", err)
	}

	return child
}

// createTestFee creates a fee expectation for testing
func createTestFee(t *testing.T, feeRepo repository.FeeRepository, childID uuid.UUID, feeType domain.FeeType, amount float64) *domain.FeeExpectation {
	t.Helper()

	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   childID,
		FeeType:   feeType,
		Year:      time.Now().Year(),
		Month:     intPtr(int(time.Now().Month())),
		Amount:    amount,
		DueDate:   time.Now().AddDate(0, 0, 5),
		CreatedAt: time.Now(),
	}

	err := feeRepo.Create(context.Background(), fee)
	if err != nil {
		t.Fatalf("Failed to create test fee: %v", err)
	}

	return fee
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

// TestImportService_BlacklistFiltering tests that blacklisted IBANs are filtered during import
func TestImportService_BlacklistFiltering(t *testing.T) {
	cleanupTestData(t)
	defer cleanupTestData(t)

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

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
	cleanupTestData(t)
	defer cleanupTestData(t)

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create test data
	child := createTestChild(t, childRepo)
	fee := createTestFee(t, feeRepo, child.ID, domain.FeeTypeFood, 45.40)

	// Create a transaction
	tx := &domain.BankTransaction{
		ID:          uuid.New(),
		BookingDate: time.Now(),
		ValueDate:   time.Now(),
		PayerName:   stringPtr("Test Payer"),
		PayerIBAN:   stringPtr("TESTDE999888777666"),
		Description: stringPtr(child.FirstName + " " + child.LastName + " " + child.MemberNumber),
		Amount:      45.40,
		Currency:    "EUR",
		ImportedAt:  time.Now(),
	}
	err := txRepo.Create(context.Background(), tx)
	if err != nil {
		t.Fatalf("Failed to create test transaction: %v", err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

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
	cleanupTestData(t)
	defer cleanupTestData(t)

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create multiple transactions with same IBAN
	testIBAN := "TESTDE111222333444"
	for i := 0; i < 3; i++ {
		tx := &domain.BankTransaction{
			ID:          uuid.New(),
			BookingDate: time.Now().AddDate(0, 0, -i),
			ValueDate:   time.Now().AddDate(0, 0, -i),
			PayerName:   stringPtr("Unwanted Payer"),
			PayerIBAN:   stringPtr(testIBAN),
			Description: stringPtr("Random payment"),
			Amount:      float64(100 + i*10),
			Currency:    "EUR",
			ImportedAt:  time.Now(),
		}
		err := txRepo.Create(context.Background(), tx)
		if err != nil {
			t.Fatalf("Failed to create test transaction %d: %v", i, err)
		}
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

	// Get one of the transactions
	transactions, _, err := txRepo.ListUnmatched(context.Background(), 0, 100)
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
	transactions, _, err = txRepo.ListUnmatched(context.Background(), 0, 100)
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
	cleanupTestData(t)
	defer cleanupTestData(t)

	// Clean up any existing unmatched transactions first
	testDB.Exec("DELETE FROM fees.bank_transactions WHERE id NOT IN (SELECT transaction_id FROM fees.payment_matches)")

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child := createTestChild(t, childRepo)

	// Create unmatched transaction (before fee exists)
	tx := &domain.BankTransaction{
		ID:          uuid.New(),
		BookingDate: time.Now(),
		ValueDate:   time.Now(),
		PayerName:   stringPtr("Test Payer"),
		PayerIBAN:   stringPtr("TESTDE555666777888"),
		Description: stringPtr(child.FirstName + " " + child.LastName + " " + child.MemberNumber + " Essensgeld"),
		Amount:      45.40,
		Currency:    "EUR",
		ImportedAt:  time.Now(),
	}
	err := txRepo.Create(context.Background(), tx)
	if err != nil {
		t.Fatalf("Failed to create test transaction: %v", err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

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
	createTestFee(t, feeRepo, child.ID, domain.FeeTypeFood, 45.40)

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
	cleanupTestData(t)
	defer cleanupTestData(t)

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
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

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
	cleanupTestData(t)
	defer cleanupTestData(t)

	// Initialize repos
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)

	// Create child
	child := createTestChild(t, childRepo)

	// Add a trusted IBAN
	trustedIBAN := "TESTDE777888999000"
	err := knownIBANRepo.Create(context.Background(), &domain.KnownIBAN{
		IBAN:      trustedIBAN,
		PayerName: stringPtr("Trusted Payer"),
		Status:    domain.KnownIBANStatusTrusted,
		Reason:    stringPtr("Test trusted"),
	})
	if err != nil {
		t.Fatalf("Failed to create trusted IBAN: %v", err)
	}

	// Initialize service
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

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

// TestImportService_GetBlacklistAndTrusted tests listing blacklisted and trusted IBANs
func TestImportService_GetBlacklistAndTrusted(t *testing.T) {
	cleanupTestData(t)
	defer cleanupTestData(t)

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
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)

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
