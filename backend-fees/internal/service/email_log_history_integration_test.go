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

func createHistoryHousehold(t *testing.T, householdRepo repository.HouseholdRepository, name string) *domain.Household {
	t.Helper()
	household := &domain.Household{
		ID:               uuid.New(),
		Name:             name,
		IncomeStatus:     domain.IncomeStatusUnknown,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := householdRepo.Create(context.Background(), household); err != nil {
		t.Fatalf("failed to create household: %v", err)
	}
	return household
}

func createHistoryFee(t *testing.T, feeRepo repository.FeeRepository, childID, householdID uuid.UUID, feeType domain.FeeType, amount float64, year int, month int, dueDate time.Time) *domain.FeeExpectation {
	t.Helper()
	fee := &domain.FeeExpectation{
		ID:          uuid.New(),
		ChildID:     childID,
		HouseholdID: &householdID,
		FeeType:     feeType,
		Year:        year,
		Month:       &month,
		Amount:      amount,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
	}
	if err := feeRepo.Create(context.Background(), fee); err != nil {
		t.Fatalf("failed to create fee: %v", err)
	}
	return fee
}

func createEmailLog(t *testing.T, emailLogRepo repository.EmailLogRepository, emailType domain.EmailLogType, sentAt time.Time, payload map[string]any, householdID *uuid.UUID) *domain.EmailLog {
	t.Helper()
	var raw *json.RawMessage
	if payload != nil {
		bytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		msg := json.RawMessage(bytes)
		raw = &msg
	}
	entry := &domain.EmailLog{
		ID:          uuid.New(),
		SentAt:      sentAt,
		ToEmail:     "parents@example.test",
		Subject:     "Kita Test",
		EmailType:   emailType,
		Payload:     raw,
		HouseholdID: householdID,
	}
	if err := emailLogRepo.Create(context.Background(), entry); err != nil {
		t.Fatalf("failed to create email log: %v", err)
	}
	return entry
}

func TestEmailLogBackfill_MapsSingleHousehold(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)

	household := createHistoryHousehold(t, householdRepo, "TEST Backfill Single")
	child, err := createTestChild(childRepo, "BS")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	feeA := createHistoryFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeFood, 45, 2026, 8, time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	feeB := createHistoryFee(t, feeRepo, child.ID, household.ID, domain.FeeTypeChildcare, 120, 2026, 8, time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	createEmailLog(t, emailLogRepo, domain.EmailLogTypeReminderInitial, time.Now().Add(-48*time.Hour), map[string]any{
		"stage":   "initial",
		"runDate": "2026-09-01",
		"feeIds":  []uuid.UUID{feeA.ID, feeB.ID},
	}, nil)

	updated, err := emailLogRepo.BackfillHouseholdIDs(ctx)
	if err != nil {
		t.Fatalf("BackfillHouseholdIDs failed: %v", err)
	}
	if updated < 1 {
		t.Fatalf("expected at least one updated log, got %d", updated)
	}

	logs, err := emailLogRepo.ListByHouseholdAndTypes(ctx, household.ID, []domain.EmailLogType{domain.EmailLogTypeReminderInitial})
	if err != nil {
		t.Fatalf("ListByHouseholdAndTypes failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log for household, got %d", len(logs))
	}
}

func TestEmailLogBackfill_SkipsAmbiguousAndUnresolvable(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)

	householdA := createHistoryHousehold(t, householdRepo, "TEST Backfill A")
	householdB := createHistoryHousehold(t, householdRepo, "TEST Backfill B")
	child, err := createTestChild(childRepo, "BA")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	feeA := createHistoryFee(t, feeRepo, child.ID, householdA.ID, domain.FeeTypeFood, 45, 2026, 8, time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	feeB := createHistoryFee(t, feeRepo, child.ID, householdB.ID, domain.FeeTypeFood, 45, 2026, 9, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))

	// Ambiguous: fees of two households.
	createEmailLog(t, emailLogRepo, domain.EmailLogTypeReminderInitial, time.Now().Add(-48*time.Hour), map[string]any{
		"stage":   "initial",
		"runDate": "2026-09-01",
		"feeIds":  []uuid.UUID{feeA.ID, feeB.ID},
	}, nil)
	// Unresolvable: unknown fee id.
	createEmailLog(t, emailLogRepo, domain.EmailLogTypeReminderFinal, time.Now().Add(-24*time.Hour), map[string]any{
		"stage":   "final",
		"runDate": "2026-09-02",
		"feeIds":  []uuid.UUID{uuid.New()},
	}, nil)
	// No payload at all.
	createEmailLog(t, emailLogRepo, domain.EmailLogTypePasswordReset, time.Now().Add(-1*time.Hour), nil, nil)

	updated, err := emailLogRepo.BackfillHouseholdIDs(ctx)
	if err != nil {
		t.Fatalf("BackfillHouseholdIDs failed: %v", err)
	}
	if updated != 0 {
		t.Fatalf("expected 0 updated logs, got %d", updated)
	}

	logsA, err := emailLogRepo.ListByHouseholdAndTypes(ctx, householdA.ID, []domain.EmailLogType{domain.EmailLogTypeReminderInitial, domain.EmailLogTypeReminderFinal})
	if err != nil {
		t.Fatalf("ListByHouseholdAndTypes failed: %v", err)
	}
	if len(logsA) != 0 {
		t.Fatalf("expected no mapped logs for household A, got %d", len(logsA))
	}
}

func TestEmailLogHistory_NewLogsCarryHousehold(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	emailLogRepo := repository.NewPostgresEmailLogRepository(testDB)

	household := createHistoryHousehold(t, householdRepo, "TEST New Logs")
	householdID := household.ID

	entry := createEmailLog(t, emailLogRepo, domain.EmailLogTypeMembershipReminderInitial, time.Now(), map[string]any{
		"stage":   "initial",
		"runDate": "2026-09-10",
		"feeIds":  []uuid.UUID{},
	}, &householdID)

	loaded, err := emailLogRepo.ListByHousehold(ctx, household.ID, 10)
	if err != nil {
		t.Fatalf("ListByHousehold failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != entry.ID {
		t.Fatalf("expected log listed for household, got %d", len(loaded))
	}
	if loaded[0].HouseholdID == nil || *loaded[0].HouseholdID != household.ID {
		t.Fatalf("expected household_id to be persisted")
	}
}

func TestResolveFeeContacts_NewestLogWins(t *testing.T) {
	feeID := uuid.New()
	older := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)

	mustPayload := func(t *testing.T, values map[string]any) *json.RawMessage {
		t.Helper()
		bytes, err := json.Marshal(values)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		raw := json.RawMessage(bytes)
		return &raw
	}

	logs := []domain.EmailLog{
		{
			ID:      uuid.New(),
			SentAt:  newer,
			Subject: "Mahnung",
			Payload: mustPayload(t, map[string]any{"stage": "final", "runDate": "2026-09-08", "feeIds": []uuid.UUID{feeID}}),
		},
		{
			ID:      uuid.New(),
			SentAt:  older,
			Subject: "Erinnerung",
			Payload: mustPayload(t, map[string]any{"stage": "initial", "runDate": "2026-09-01", "feeIds": []uuid.UUID{feeID}}),
		},
		{
			ID:      uuid.New(),
			SentAt:  newer.Add(time.Hour),
			Subject: "Ohne Bezug",
			Payload: mustPayload(t, map[string]any{"stage": "final", "runDate": "2026-09-08", "feeIds": []uuid.UUID{uuid.New()}}),
		},
	}

	contacts := service.ResolveFeeContacts(logs)
	contact, ok := contacts[feeID]
	if !ok {
		t.Fatalf("expected contact for fee")
	}
	if contact.Stage != "final" {
		t.Fatalf("expected newest log stage final, got %s", contact.Stage)
	}
	if !contact.RunDate.Equal(time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected runDate: %v", contact.RunDate)
	}
	if !contact.LastContactAt.Equal(newer) {
		t.Fatalf("unexpected lastContactAt: %v", contact.LastContactAt)
	}
}

func TestGetHistoryReliableFrom_SetByMigration(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	settingsRepo := repository.NewPostgresSettingsRepository(testDB)
	reminderService := service.NewReminderService(
		repository.NewPostgresFeeRepository(testDB),
		repository.NewPostgresChildRepository(testDB),
		repository.NewPostgresHouseholdRepository(testDB),
		settingsRepo,
		repository.NewPostgresEmailLogRepository(testDB),
		nil,
	)

	cutoff, err := reminderService.GetHistoryReliableFrom(ctx)
	if err != nil {
		t.Fatalf("GetHistoryReliableFrom failed: %v", err)
	}
	if cutoff.IsZero() {
		t.Fatalf("expected migration to set reminder_history_reliable_from")
	}
	if cutoff.After(time.Now().UTC()) {
		t.Fatalf("cutoff should not be in the future: %v", cutoff)
	}
}
