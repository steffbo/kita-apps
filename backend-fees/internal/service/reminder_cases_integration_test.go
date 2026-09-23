package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
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
	fail    bool
}

func (f *fakeReminderSender) SendTextEmailMulti(to []string, subject, body string) error {
	f.sent++
	if f.fail {
		return errors.New("smtp unavailable")
	}
	return nil
}

func (f *fakeReminderSender) SendTextAndHTMLEmailMulti(to []string, subject, textBody, htmlBody string, inlineImageCID string, inlineImagePNG []byte) error {
	f.sent++
	if f.fail {
		return errors.New("smtp unavailable")
	}
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
	setReliableCutoff(t, "2026-01-01")

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
	setReliableCutoff(t, "2026-01-01")

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
	setReliableCutoff(t, "2026-01-01")

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
	setReliableCutoff(t, "2026-01-01")

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

func TestReminderCase_Preview_InitialStagePlansNoFees(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()
	setReliableCutoff(t, "2026-01-01")

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{})

	household := createCaseHousehold(t, householdRepo, "TEST Initial No Fees")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Anna")
	child, err := createTestChild(childRepo, "INF")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	foodFee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	preview, err := reminderService.PreviewReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageInitial,
		RunDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:  []uuid.UUID{foodFee.ID},
	})
	if err != nil {
		t.Fatalf("PreviewReminderCase failed: %v", err)
	}
	if len(preview.PlannedReminderFees) != 0 {
		t.Fatalf("initial stage must not plan reminder fees, got %d", len(preview.PlannedReminderFees))
	}
	if preview.TotalAmount != 45.40 {
		t.Fatalf("expected total 45.40 without planned fees, got %v", preview.TotalAmount)
	}
}

func TestReminderCase_Send_CreatesFeesLogsAndPreventsDuplicates(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()
	setReliableCutoff(t, "2026-01-01")

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
	previewedAt := time.Now().UTC().Add(-time.Hour)
	result, err := reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageFinal,
		RunDate:     runDate,
		FeeIDs:      []uuid.UUID{foodFee.ID, membershipFee.ID},
		PreviewedAt: &previewedAt,
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

	// Second send (repeat dunning): a fresh preview happened after the first
	// send created its fees, so the reminder fees no longer count as
	// "created after the preview". Still sends, but no duplicate fees.
	previewedAt = time.Now().UTC()
	repeat, err := reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageFinal,
		RunDate:     time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{foodFee.ID, membershipFee.ID},
		PreviewedAt: &previewedAt,
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
	setReliableCutoff(t, "2026-01-01")

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

	// Missing previewedAt → invalid input (concurrency guard is mandatory).
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:   service.ReminderStageInitial,
		RunDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:  []uuid.UUID{feeA.ID},
	}, nil)
	if err != service.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for missing previewedAt, got %v", err)
	}

	// Foreign fee id → conflict.
	previewedAtA := time.Now().UTC().Add(-time.Hour)
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageInitial,
		RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{feeB.ID},
		PreviewedAt: &previewedAtA,
	}, nil)
	assertConflict(t, err, []uuid.UUID{feeB.ID})

	// Paid fee → conflict.
	matchFeeFully(t, feeA)
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageInitial,
		RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{feeA.ID},
		PreviewedAt: &previewedAtA,
	}, nil)
	assertConflict(t, err, []uuid.UUID{feeA.ID})

	// Duplicate ids → invalid input.
	feeC := createCaseFee(t, feeRepo, childA.ID, householdA.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(9), time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	_, err = reminderService.SendReminderCase(ctx, householdA.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageInitial,
		RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{feeC.ID, feeC.ID},
		PreviewedAt: &previewedAtA,
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

// setReliableCutoff pins the reminder history cutoff to a past date so fees
// created "now" in tests count as reliably contactable, and restores the
// original value afterwards.
func setReliableCutoff(t *testing.T, date string) {
	t.Helper()
	var original string
	if err := testDB.Get(&original, `SELECT value FROM fees.app_settings WHERE key = 'reminder_history_reliable_from'`); err != nil {
		t.Fatalf("failed to read cutoff: %v", err)
	}
	if _, err := testDB.Exec(`UPDATE fees.app_settings SET value = $1, updated_at = NOW() WHERE key = 'reminder_history_reliable_from'`, date); err != nil {
		t.Fatalf("failed to set cutoff: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testDB.Exec(`UPDATE fees.app_settings SET value = $1, updated_at = NOW() WHERE key = 'reminder_history_reliable_from'`, original); err != nil {
			t.Fatalf("failed to reset cutoff: %v", err)
		}
	})
}

func countReminderFeesForBase(t *testing.T, baseFeeID uuid.UUID) int {
	t.Helper()
	var count int
	if err := testDB.Get(&count, `SELECT COUNT(*) FROM fees.fee_expectations WHERE fee_type = 'REMINDER' AND reminder_for_id = $1`, baseFeeID); err != nil {
		t.Fatalf("failed to count reminder fees: %v", err)
	}
	return count
}

// Two concurrent final sends of the same base fee must produce exactly one
// reminder fee; the loser surfaces as a conflict instead of a duplicate fee.
func TestReminderCase_Send_ConcurrentSendsCreateSingleReminderFee(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()
	setReliableCutoff(t, "2026-01-01")

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{enabled: true})

	household := createCaseHousehold(t, householdRepo, "TEST Concurrent Send")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Conny")
	child, err := createTestChild(childRepo, "CCS")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	fee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	previewedAt := time.Now().UTC().Add(-time.Hour)
	send := func() error {
		_, err := reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
			Stage:       service.ReminderStageFinal,
			RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
			FeeIDs:      []uuid.UUID{fee.ID},
			PreviewedAt: &previewedAt,
		}, nil)
		return err
	}

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = send()
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
			continue
		}
		if _, ok := err.(*service.CaseConflictError); !ok {
			t.Fatalf("expected success or CaseConflictError, got %v", err)
		}
	}
	if successes == 0 {
		t.Fatalf("expected at least one successful send, got errors: %v / %v", results[0], results[1])
	}
	if got := countReminderFeesForBase(t, fee.ID); got != 1 {
		t.Fatalf("expected exactly 1 reminder fee after concurrent sends, got %d", got)
	}
}

// A failing SMTP send must not leave orphaned reminder fees behind; a retry
// afterwards plans them again.
func TestReminderCase_Send_SMTPFailureCompensatesReminderFees(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()
	setReliableCutoff(t, "2026-01-01")

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	sender := &fakeReminderSender{enabled: true, fail: true}
	reminderService := newReminderCaseService(sender)

	household := createCaseHousehold(t, householdRepo, "TEST SMTP Failure")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Sven")
	child, err := createTestChild(childRepo, "CSF")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	fee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	previewedAt := time.Now().UTC().Add(-time.Hour)
	_, err = reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageFinal,
		RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{fee.ID},
		PreviewedAt: &previewedAt,
	}, nil)
	if err == nil {
		t.Fatalf("expected SMTP error, got none")
	}
	if got := countReminderFeesForBase(t, fee.ID); got != 0 {
		t.Fatalf("expected no reminder fees after failed send, got %d", got)
	}

	// Retry with a working sender plans the reminder fee again.
	sender.fail = false
	result, err := reminderService.SendReminderCase(ctx, household.ID, &service.ReminderCaseRequest{
		Stage:       service.ReminderStageFinal,
		RunDate:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		FeeIDs:      []uuid.UUID{fee.ID},
		PreviewedAt: &previewedAt,
	}, nil)
	if err != nil {
		t.Fatalf("retry SendReminderCase failed: %v", err)
	}
	if len(result.CreatedReminderFees) != 1 {
		t.Fatalf("expected 1 reminder fee on retry, got %d", len(result.CreatedReminderFees))
	}
	if got := countReminderFeesForBase(t, fee.ID); got != 1 {
		t.Fatalf("expected 1 reminder fee after retry, got %d", got)
	}
}

// A not-yet-due fee stays never_contacted even when it was contacted early;
// actionability is defined by the due date alone.
func TestReminderCase_NotDueFeeWithContactStaysNeverContacted(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{enabled: false})

	household := createCaseHousehold(t, householdRepo, "TEST Early Contact")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Ida")
	child, err := createTestChild(childRepo, "CEC")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	// Not due as of 2026-09-15, contacted on 2026-09-01 (7-day deadline long past).
	fee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(10), time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	insertContactLog(t, emailLogRepo, household.ID, []uuid.UUID{fee.ID}, "initial", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))

	result, err := reminderService.ListReminderCases(ctx, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), service.ReminderCasesScopeAll)
	if err != nil {
		t.Fatalf("ListReminderCases failed: %v", err)
	}
	var target *service.ReminderCase
	for i := range result.Cases {
		if result.Cases[i].HouseholdID == household.ID {
			target = &result.Cases[i]
		}
	}
	if target == nil {
		t.Fatalf("expected household in scope=all result")
	}
	for _, listed := range target.Fees {
		if listed.FeeID == fee.ID && listed.Status != service.FeeStatusNeverContacted {
			t.Fatalf("expected never_contacted for not-due fee, got %s", listed.Status)
		}
	}
}

// Fees created on the reliability cutoff day itself have no assignable
// history (calendar-day comparison, not exact timestamps).
func TestReminderCase_FeeCreatedOnCutoffDayIsHistoryUnknown(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	reminderService := newReminderCaseService(&fakeReminderSender{enabled: false})

	var originalCutoff string
	if err := testDB.Get(&originalCutoff, `SELECT value FROM fees.app_settings WHERE key = 'reminder_history_reliable_from'`); err != nil {
		t.Fatalf("failed to read cutoff: %v", err)
	}
	if _, err := testDB.Exec(`UPDATE fees.app_settings SET value = $1, updated_at = NOW() WHERE key = 'reminder_history_reliable_from'`, time.Now().UTC().Format("2006-01-02")); err != nil {
		t.Fatalf("failed to set cutoff: %v", err)
	}
	defer func() {
		if _, err := testDB.Exec(`UPDATE fees.app_settings SET value = $1, updated_at = NOW() WHERE key = 'reminder_history_reliable_from'`, originalCutoff); err != nil {
			t.Fatalf("failed to reset cutoff: %v", err)
		}
	}()

	household := createCaseHousehold(t, householdRepo, "TEST Cutoff Day")
	createCaseParent(t, repository.NewPostgresParentRepository(testDB), household.ID, "Kai")
	child, err := createTestChild(childRepo, "CCD")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	// Overdue fee created today (after the cutoff timestamp at midnight UTC,
	// but on the cutoff calendar day).
	fee := createCaseFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45.40, 2026, ptrInt(8), time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	result, err := reminderService.ListReminderCases(ctx, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), service.ReminderCasesScopeAll)
	if err != nil {
		t.Fatalf("ListReminderCases failed: %v", err)
	}
	for _, entry := range result.Cases {
		if entry.HouseholdID != household.ID {
			continue
		}
		for _, listed := range entry.Fees {
			if listed.FeeID == fee.ID && listed.Status != service.FeeStatusHistoryUnknown {
				t.Fatalf("expected history_unknown for fee created on cutoff day, got %s", listed.Status)
			}
		}
	}
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
