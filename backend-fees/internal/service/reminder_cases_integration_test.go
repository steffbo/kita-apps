package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// fakeReminderSender records sends without SMTP.
type fakeReminderSender struct {
	sent    int
	enabled bool
}

func (f *fakeReminderSender) SendTextEmailMulti(to []string, subject, body string) error {
	f.sent++
	return nil
}

func (f *fakeReminderSender) SendTextAndHTMLEmailMulti(to []string, subject, textBody, htmlBody string, inlineImageCID string, inlineImagePNG []byte) error {
	f.sent++
	return nil
}

func (f *fakeReminderSender) IsEnabled() bool { return f.enabled }

func newReminderCaseService(emailSender *fakeReminderSender) *service.ReminderService {
	return service.NewReminderService(
		repository.NewPostgresFeeRepository(testDB),
		repository.NewPostgresChildRepository(testDB),
		repository.NewPostgresHouseholdRepository(testDB),
		repository.NewPostgresSettingsRepository(testDB),
		repository.NewPostgresEmailLogRepository(testDB),
		emailSender,
	)
}

func createCaseHousehold(t *testing.T, householdRepo repository.HouseholdRepository, name string) *domain.Household {
	return createHistoryHousehold(t, householdRepo, name)
}

func createCaseParent(t *testing.T, parentRepo repository.ParentRepository, householdID uuid.UUID, firstName string) *domain.Parent {
	t.Helper()
	email := "parents@example.test"
	parent := &domain.Parent{
		ID:           uuid.New(),
		HouseholdID:  &householdID,
		FirstName:    firstName,
		LastName:     "Test",
		Email:        &email,
		IncomeStatus: domain.IncomeStatusUnknown,
	}
	if err := parentRepo.Create(context.Background(), parent); err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}
	return parent
}

func createCaseFee(t *testing.T, feeRepo repository.FeeRepository, childID, householdID uuid.UUID, feeType domain.FeeType, amount float64, year int, month *int, dueDate time.Time) *domain.FeeExpectation {
	t.Helper()
	fee := &domain.FeeExpectation{
		ID:          uuid.New(),
		ChildID:     childID,
		HouseholdID: &householdID,
		FeeType:     feeType,
		Year:        year,
		Month:       month,
		Amount:      amount,
		DueDate:     dueDate,
		CreatedAt:   time.Now().UTC(),
	}
	if err := feeRepo.Create(context.Background(), fee); err != nil {
		t.Fatalf("failed to create fee: %v", err)
	}
	return fee
}

func matchFeeFully(t *testing.T, fee *domain.FeeExpectation) {
	t.Helper()
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	tx, err := createTestTransaction(txRepo, "TESTIBAN123456", fee.Amount, time.Now().AddDate(0, 0, -1), "payment")
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}
	if err := matchRepo.Create(context.Background(), &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        fee.Amount,
		MatchedAt:     time.Now().UTC(),
		MatchType:     domain.MatchTypeAuto,
	}); err != nil {
		t.Fatalf("failed to create payment match: %v", err)
	}
}

func matchFeePartially(t *testing.T, fee *domain.FeeExpectation, amount float64) {
	t.Helper()
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	tx, err := createTestTransaction(txRepo, "TESTIBAN123456", amount, time.Now().AddDate(0, 0, -1), "partial payment")
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}
	if err := matchRepo.Create(context.Background(), &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        amount,
		MatchedAt:     time.Now().UTC(),
		MatchType:     domain.MatchTypeAuto,
	}); err != nil {
		t.Fatalf("failed to create payment match: %v", err)
	}
}

func insertContactLog(t *testing.T, emailLogRepo repository.EmailLogRepository, householdID uuid.UUID, feeIDs []uuid.UUID, stage string, runDate time.Time) {
	t.Helper()
	payload := map[string]any{
		"stage":   stage,
		"runDate": runDate.Format("2006-01-02"),
		"feeIds":  feeIDs,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := json.RawMessage(bytes)
	err = emailLogRepo.Create(context.Background(), &domain.EmailLog{
		ID:          uuid.New(),
		SentAt:      runDate.Add(12 * time.Hour),
		ToEmail:     "parents@example.test",
		Subject:     "Kita Test",
		EmailType:   domain.EmailLogTypeReminderInitial,
		Payload:     &raw,
		HouseholdID: &householdID,
	})
	if err != nil {
		t.Fatalf("failed to create log: %v", err)
	}
}

func TestReminderCases_GroupingStatusesAndScope(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{})

	household := createCaseHousehold(t, householdRepo, "TEST Cases Grouped")
	child, err := createTestChild(childRepo, "CG")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	overdueFood := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	futureChildcare := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeChildcare, 100, 2026, ptrInt(11), time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))

	// Household with only paid fees must not appear.
	paidHousehold := createCaseHousehold(t, householdRepo, "TEST Cases Paid")
	paidChild, err := createTestChild(childRepo, "CP")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	paidFee := createCaseFee(t, feeRepo, paidChild.ID, paidHousehold.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	matchFeeFully(t, paidFee)

	asOf := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	result, err := reminderService.ListReminderCases(ctx, asOf, service.ReminderCasesScopeAll)
	if err != nil {
		t.Fatalf("ListReminderCases failed: %v", err)
	}

	var grouped *service.ReminderCase
	for i := range result.Cases {
		if result.Cases[i].HouseholdID == household.ID {
			grouped = &result.Cases[i]
		}
	}
	if grouped == nil {
		t.Fatalf("expected household in cases")
	}
	if len(grouped.Fees) != 2 {
		t.Fatalf("expected 2 fees, got %d", len(grouped.Fees))
	}
	if grouped.TotalRemaining != 145.40 {
		t.Fatalf("unexpected total remaining: %v", grouped.TotalRemaining)
	}

	feeByID := map[uuid.UUID]service.ReminderCaseFee{}
	for _, fee := range grouped.Fees {
		feeByID[fee.FeeID] = fee
	}
	if got := feeByID[overdueFood.ID]; got.Status != service.FeeStatusActionableInitial {
		t.Fatalf("expected overdue food actionable_initial, got %s", got.Status)
	}
	if got := feeByID[futureChildcare.ID]; got.Status != service.FeeStatusNeverContacted {
		t.Fatalf("expected future childcare never_contacted, got %s", got.Status)
	}
	if !grouped.NextActionAt.Equal(time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected next action date: %v", grouped.NextActionAt)
	}

	for i := range result.Cases {
		if result.Cases[i].HouseholdID == paidHousehold.ID {
			t.Fatalf("paid household must not appear in cases")
		}
	}

	actionable, err := reminderService.ListReminderCases(ctx, asOf, service.ReminderCasesScopeActionable)
	if err != nil {
		t.Fatalf("ListReminderCases actionable failed: %v", err)
	}
	for i := range actionable.Cases {
		if actionable.Cases[i].HouseholdID == household.ID {
			return
		}
	}
	t.Fatalf("expected household in actionable scope")
}

func TestReminderCases_WaitingHiddenFromActionable(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{})

	household := createCaseHousehold(t, householdRepo, "TEST Cases Waiting")
	child, err := createTestChild(childRepo, "CW")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	fee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	contactRunDate := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	insertContactLog(t, emailLogRepo, household.ID, []uuid.UUID{fee.ID}, "initial", contactRunDate)

	asOf := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	actionable, err := reminderService.ListReminderCases(ctx, asOf, service.ReminderCasesScopeActionable)
	if err != nil {
		t.Fatalf("ListReminderCases failed: %v", err)
	}
	for i := range actionable.Cases {
		if actionable.Cases[i].HouseholdID == household.ID {
			t.Fatalf("waiting family must not appear in actionable scope")
		}
	}

	all, err := reminderService.ListReminderCases(ctx, asOf, service.ReminderCasesScopeAll)
	if err != nil {
		t.Fatalf("ListReminderCases all failed: %v", err)
	}
	var found *service.ReminderCase
	for i := range all.Cases {
		if all.Cases[i].HouseholdID == household.ID {
			found = &all.Cases[i]
		}
	}
	if found == nil {
		t.Fatalf("expected household in all scope")
	}
	if found.Fees[0].Status != service.FeeStatusWaiting {
		t.Fatalf("expected waiting status, got %s", found.Fees[0].Status)
	}
	if !found.Fees[0].ActionableAt.Equal(time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected actionableAt = deadline 19.09., got %v", found.Fees[0].ActionableAt)
	}

	// After the deadline expired the fee becomes actionable_final.
	expired, err := reminderService.ListReminderCases(ctx, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), service.ReminderCasesScopeActionable)
	if err != nil {
		t.Fatalf("ListReminderCases expired failed: %v", err)
	}
	for i := range expired.Cases {
		if expired.Cases[i].HouseholdID == household.ID {
			if expired.Cases[i].Fees[0].Status != service.FeeStatusActionableFinal {
				t.Fatalf("expected actionable_final after deadline, got %s", expired.Cases[i].Fees[0].Status)
			}
			return
		}
	}
	t.Fatalf("expected household in actionable scope after deadline")
}

func TestReminderCases_PartialPaymentRemaining(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{})

	household := createCaseHousehold(t, householdRepo, "TEST Cases Partial")
	child, err := createTestChild(childRepo, "CPT")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	fee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	matchFeePartially(t, fee, 20.00)

	result, err := reminderService.ListReminderCases(context.Background(), time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), service.ReminderCasesScopeActionable)
	if err != nil {
		t.Fatalf("ListReminderCases failed: %v", err)
	}
	var found *service.ReminderCase
	for i := range result.Cases {
		if result.Cases[i].HouseholdID == household.ID {
			found = &result.Cases[i]
		}
	}
	if found == nil {
		t.Fatalf("expected household in cases")
	}
	if found.Fees[0].Remaining != 25.40 {
		t.Fatalf("expected remaining 25.40, got %v", found.Fees[0].Remaining)
	}
	if found.TotalRemaining != 25.40 {
		t.Fatalf("expected total 25.40, got %v", found.TotalRemaining)
	}
}

func TestReminderCase_Preview_MixedTypesPlansFees(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{})

	household := createCaseHousehold(t, householdRepo, "TEST Preview Mixed")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Anna")
	child, err := createTestChild(childRepo, "PM")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	foodFee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	membershipFee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeMembership, 30, 2026, nil, time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC))

	runDate := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	preview, err := reminderService.PreviewReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageFinal,
		RunDate: runDate,
		FeeIDs:  []uuid.UUID{foodFee.ID, membershipFee.ID},
	})
	if err != nil {
		t.Fatalf("PreviewReminderCase failed: %v", err)
	}

	if preview.Subject != "Kita Mahnung: offene Beiträge" {
		t.Fatalf("unexpected subject: %s", preview.Subject)
	}
	if !preview.Deadline.Equal(time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected deadline runDate+7, got %v", preview.Deadline)
	}
	if len(preview.PlannedReminderFees) != 2 {
		t.Fatalf("expected 2 planned reminder fees, got %d", len(preview.PlannedReminderFees))
	}
	amounts := map[uuid.UUID]float64{}
	for _, planned := range preview.PlannedReminderFees {
		amounts[planned.BaseFeeID] = planned.Amount
	}
	if amounts[foodFee.ID] != 10.0 {
		t.Fatalf("expected 10 EUR reminder fee for food, got %v", amounts[foodFee.ID])
	}
	if amounts[membershipFee.ID] != 5.0 {
		t.Fatalf("expected 5 EUR reminder fee for membership, got %v", amounts[membershipFee.ID])
	}
	if preview.TotalAmount != 90.40 {
		t.Fatalf("expected total 90.40 (45.40+30+15), got %v", preview.TotalAmount)
	}
	if preview.Body == "" || preview.Recipients == nil {
		t.Fatalf("expected mail preview content")
	}
	if preview.RecommendedStage != service.ReminderStageInitial {
		t.Fatalf("expected recommendation initial for never-contacted fees, got %s", preview.RecommendedStage)
	}
	foundWarning := false
	for _, warning := range preview.Warnings {
		if contains(warning, "wurde noch nicht erinnert") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("expected not-yet-reminded warning, got %v", preview.Warnings)
	}
}

func TestReminderCase_Send_CreatesFeesLogsAndPreventsDuplicates(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)
	sender := &fakeReminderSender{enabled: true}
	reminderService := newReminderCaseService(sender)

	household := createCaseHousehold(t, householdRepo, "TEST Send Once")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Ben")
	child, err := createTestChild(childRepo, "SO")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	foodFee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	membershipFee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeMembership, 30, 2026, nil, time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC))

	runDate := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	result, err := reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageFinal,
		RunDate: runDate,
		FeeIDs:  []uuid.UUID{foodFee.ID, membershipFee.ID},
	}, nil)
	if err != nil {
		t.Fatalf("SendReminderCase failed: %v", err)
	}
	if sender.sent != 1 {
		t.Fatalf("expected 1 email, got %d", sender.sent)
	}
	if len(result.CreatedReminderFees) != 2 {
		t.Fatalf("expected 2 created reminder fees, got %d", len(result.CreatedReminderFees))
	}

	logs, err := emailLogRepo.ListByHouseholdAndTypes(ctx, household.ID, []domain.EmailLogType{domain.EmailLogTypeReminderFinal})
	if err != nil {
		t.Fatalf("ListByHouseholdAndTypes failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 final reminder log, got %d", len(logs))
	}
	if logs[0].HouseholdID == nil || *logs[0].HouseholdID != household.ID {
		t.Fatalf("expected log household_id set")
	}

	// Second send (repeat dunning): still sends, but no duplicate reminder fees.
	repeat, err := reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageFinal,
		RunDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		FeeIDs:  []uuid.UUID{foodFee.ID, membershipFee.ID},
	}, nil)
	if err != nil {
		t.Fatalf("repeat SendReminderCase failed: %v", err)
	}
	if len(repeat.CreatedReminderFees) != 0 {
		t.Fatalf("expected no duplicate reminder fees, got %d", len(repeat.CreatedReminderFees))
	}
}

func TestReminderCase_Send_ConflictsAndValidation(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)
	sender := &fakeReminderSender{enabled: true}
	reminderService := newReminderCaseService(sender)

	householdA := createCaseHousehold(t, householdRepo, "TEST Conflict A")
	householdB := createCaseHousehold(t, householdRepo, "TEST Conflict B")
	childA, err := createTestChild(childRepo, "CA")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	childB, err := createTestChild(childRepo, "CB")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	feeA := createCaseFee(t, feeRepo, childA.ID, householdA.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	feeB := createCaseFee(t, feeRepo, childB.ID, householdB.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	// Foreign fee id → conflict.
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageInitial,
		RunDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:  []uuid.UUID{feeB.ID},
	}, nil)
	assertConflict(t, err, []uuid.UUID{feeB.ID})

	// Paid fee → conflict.
	matchFeeFully(t, feeA)
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageInitial,
		RunDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:  []uuid.UUID{feeA.ID},
	}, nil)
	assertConflict(t, err, []uuid.UUID{feeA.ID})

	// Duplicate ids → invalid input.
	feeC := createCaseFee(t, feeRepo, childA.ID, householdA.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(9), time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageInitial,
		RunDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:  []uuid.UUID{feeC.ID, feeC.ID},
	}, nil)
	if err != service.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for duplicate ids, got %v", err)
	}

	// Concurrent reminder fee after preview → conflict with the affected base fee.
	previewedAt := time.Now().UTC().Add(-time.Hour)
	insertContactLog(t, emailLogRepo, householdA.ID, []uuid.UUID{}, "initial", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) // keep log table non-empty
	reminder := &domain.FeeExpectation{
		ID:            uuid.New(),
		ChildID:       childA.ID,
		HouseholdID:   &householdA.ID,
		FeeType:       domain.FeeTypeReminder,
		Year:          2026,
		Amount:        10,
		DueDate:       time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now().UTC(),
		ReminderForID: &feeC.ID,
	}
	if err := feeRepo.Create(ctx, reminder); err != nil {
		t.Fatalf("failed to create concurrent reminder fee: %v", err)
	}
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageFinal,
		RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{feeC.ID},
		PreviewedAt: &previewedAt,
	}, nil)
	assertConflict(t, err, []uuid.UUID{feeC.ID})
}

func assertConflict(t *testing.T, err error, expectedFeeIDs []uuid.UUID) {
	t.Helper()
	conflict, ok := err.(*service.CaseConflictError)
	if !ok {
		t.Fatalf("expected CaseConflictError, got %v", err)
	}
	if len(conflict.FeeIDs) != len(expectedFeeIDs) {
		t.Fatalf("expected conflict fees %v, got %v", expectedFeeIDs, conflict.FeeIDs)
	}
	for _, id := range expectedFeeIDs {
		found := false
		for _, got := range conflict.FeeIDs {
			if got == id {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected conflict fee %s in %v", id, conflict.FeeIDs)
		}
	}
}

func ptrInt(value int) *int {
	return &value
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && stringContains(haystack, needle)
}

func stringContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
