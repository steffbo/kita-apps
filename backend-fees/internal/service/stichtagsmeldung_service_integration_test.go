package service_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestStichtagsmeldungReport_UsesExitDateAndCareHoursHistory(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	stichtagService := service.NewStichtagsmeldungService(childRepo)

	reportDate := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	baseline, err := stichtagService.GetReport(context.Background(), reportDate)
	if err != nil {
		t.Fatalf("baseline GetReport failed: %v", err)
	}
	baselineBreakdown := toBreakdownMap(baseline.CareHoursBreakdown)

	children := []*domain.Child{
		newTestChildWithHours("SR1", time.Date(2021, time.January, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), nil, intPtr(25), intPtr(35)),
		newTestChildWithHours("SR2", time.Date(2024, time.February, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), datePtr(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)), intPtr(35), intPtr(40)),
		newTestChildWithHours("SR3", time.Date(2021, time.March, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), datePtr(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)), intPtr(45), intPtr(45)),
		newTestChildWithHours("SR4", time.Date(2024, time.April, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), nil, nil, nil),
	}

	for _, child := range children {
		if err := childRepo.Create(context.Background(), child); err != nil {
			t.Fatalf("failed to create child %s: %v", child.MemberNumber, err)
		}
	}

	report, err := stichtagService.GetReport(context.Background(), reportDate)
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}

	if report.TotalChildrenInKita-baseline.TotalChildrenInKita != 3 {
		t.Fatalf("expected total children delta to be 3, got %d", report.TotalChildrenInKita-baseline.TotalChildrenInKita)
	}

	breakdown := toBreakdownMap(report.CareHoursBreakdown)

	if breakdown["35"]-baselineBreakdown["35"] != 1 {
		t.Errorf("expected 35h bucket delta to be 1, got %d", breakdown["35"]-baselineBreakdown["35"])
	}
	if breakdown["40"]-baselineBreakdown["40"] != 1 {
		t.Errorf("expected 40h bucket delta to be 1, got %d", breakdown["40"]-baselineBreakdown["40"])
	}
	if breakdown["unknown"]-baselineBreakdown["unknown"] != 1 {
		t.Errorf("expected unknown bucket delta to be 1, got %d", breakdown["unknown"]-baselineBreakdown["unknown"])
	}

	careBreakdownByHours := toBreakdownDetailsMap(report.CareHoursBreakdown)
	if got := careBreakdownByHours["35"].Ue3Count - toBreakdownDetailsMap(baseline.CareHoursBreakdown)["35"].Ue3Count; got != 1 {
		t.Errorf("expected 35h ue3 delta to be 1, got %d", got)
	}
	if got := careBreakdownByHours["40"].U3Count - toBreakdownDetailsMap(baseline.CareHoursBreakdown)["40"].U3Count; got != 1 {
		t.Errorf("expected 40h u3 delta to be 1, got %d", got)
	}
	if got := careBreakdownByHours["unknown"].U3Count - toBreakdownDetailsMap(baseline.CareHoursBreakdown)["unknown"].U3Count; got != 1 {
		t.Errorf("expected unknown u3 delta to be 1, got %d", got)
	}

	legalBreakdown := toLegalBreakdownMap(report.LegalHoursBreakdown)
	baselineLegalBreakdown := toLegalBreakdownMap(baseline.LegalHoursBreakdown)
	if legalBreakdown["25"]-baselineLegalBreakdown["25"] != 1 {
		t.Errorf("expected 25h legal bucket delta to be 1, got %d", legalBreakdown["25"]-baselineLegalBreakdown["25"])
	}
	if legalBreakdown["35"]-baselineLegalBreakdown["35"] != 1 {
		t.Errorf("expected 35h legal bucket delta to be 1, got %d", legalBreakdown["35"]-baselineLegalBreakdown["35"])
	}
	if legalBreakdown["unknown"]-baselineLegalBreakdown["unknown"] != 1 {
		t.Errorf("expected unknown legal bucket delta to be 1, got %d", legalBreakdown["unknown"]-baselineLegalBreakdown["unknown"])
	}

	legalBreakdownByHours := toLegalBreakdownDetailsMap(report.LegalHoursBreakdown)
	if got := legalBreakdownByHours["25"].Ue3Count - toLegalBreakdownDetailsMap(baseline.LegalHoursBreakdown)["25"].Ue3Count; got != 1 {
		t.Errorf("expected 25h legal ue3 delta to be 1, got %d", got)
	}
	if got := legalBreakdownByHours["35"].U3Count - toLegalBreakdownDetailsMap(baseline.LegalHoursBreakdown)["35"].U3Count; got != 1 {
		t.Errorf("expected 35h legal u3 delta to be 1, got %d", got)
	}
	if got := legalBreakdownByHours["unknown"].U3Count - toLegalBreakdownDetailsMap(baseline.LegalHoursBreakdown)["unknown"].U3Count; got != 1 {
		t.Errorf("expected unknown legal u3 delta to be 1, got %d", got)
	}

	if report.U3ChildrenCount-baseline.U3ChildrenCount != 2 {
		t.Errorf("expected u3 children delta to be 2, got %d", report.U3ChildrenCount-baseline.U3ChildrenCount)
	}
	if report.Ue3ChildrenCount-baseline.Ue3ChildrenCount != 1 {
		t.Errorf("expected ue3 children delta to be 1, got %d", report.Ue3ChildrenCount-baseline.Ue3ChildrenCount)
	}
}

func TestChildService_AddCareHoursHistory_SplitsAndUpdatesExistingStart(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	childService := service.NewChildService(childRepo, nil, nil)

	child := newTestChildWithDates("SR5", time.Date(2022, time.May, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), nil, intPtr(35))
	if err := childRepo.Create(context.Background(), child); err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	if err := childService.AddCareHoursHistory(context.Background(), child.ID, service.AddCareHoursHistoryInput{
		CareHours: intPtr(40),
		ValidFrom: "2026-02-01",
	}); err != nil {
		t.Fatalf("AddCareHoursHistory failed: %v", err)
	}

	if err := childService.AddCareHoursHistory(context.Background(), child.ID, service.AddCareHoursHistoryInput{
		CareHours: intPtr(45),
		ValidFrom: "2026-02-01",
	}); err != nil {
		t.Fatalf("AddCareHoursHistory update failed: %v", err)
	}

	history, err := childService.ListCareHoursHistory(context.Background(), child.ID)
	if err != nil {
		t.Fatalf("ListCareHoursHistory failed: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}

	latest := history[0]
	if latest.CareHours == nil || *latest.CareHours != 45 {
		t.Fatalf("expected latest care hours to be 45, got %v", latest.CareHours)
	}
	if latest.EffectiveFrom.Format("2006-01-02") != "2026-02-01" {
		t.Fatalf("expected latest history to start on 2026-02-01, got %s", latest.EffectiveFrom.Format("2006-01-02"))
	}
	if latest.EffectiveUntil != nil {
		t.Fatalf("expected latest history to stay open-ended, got %v", latest.EffectiveUntil)
	}

	previous := history[1]
	if previous.EffectiveUntil == nil || previous.EffectiveUntil.Format("2006-01-02") != "2026-01-31" {
		t.Fatalf("expected previous history to end on 2026-01-31, got %v", previous.EffectiveUntil)
	}
}

func newTestChildWithDates(suffix string, birthDate, entryDate time.Time, exitDate *time.Time, careHours *int) *domain.Child {
	return newTestChildWithHours(suffix, birthDate, entryDate, exitDate, nil, careHours)
}

func newTestChildWithHours(suffix string, birthDate, entryDate time.Time, exitDate *time.Time, legalHours *int, careHours *int) *domain.Child {
	memberNum := "T" + suffix
	if len(memberNum) > 10 {
		memberNum = memberNum[:10]
	}

	return &domain.Child{
		ID:           uuid.New(),
		MemberNumber: memberNum,
		FirstName:    "Test",
		LastName:     "Kind",
		BirthDate:    birthDate,
		EntryDate:    entryDate,
		ExitDate:     exitDate,
		LegalHours:   legalHours,
		CareHours:    careHours,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func datePtr(value time.Time) *time.Time {
	return &value
}

func toBreakdownMap(rows []domain.CareHoursBreakdown) map[string]int {
	breakdown := map[string]int{}
	for _, row := range rows {
		key := "unknown"
		if row.CareHours != nil {
			key = strconv.Itoa(*row.CareHours)
		}
		breakdown[key] = row.Count
	}
	return breakdown
}

func toLegalBreakdownMap(rows []domain.LegalHoursBreakdown) map[string]int {
	breakdown := map[string]int{}
	for _, row := range rows {
		key := "unknown"
		if row.LegalHours != nil {
			key = strconv.Itoa(*row.LegalHours)
		}
		breakdown[key] = row.Count
	}
	return breakdown
}

func toBreakdownDetailsMap(rows []domain.CareHoursBreakdown) map[string]domain.CareHoursBreakdown {
	breakdown := map[string]domain.CareHoursBreakdown{}
	for _, row := range rows {
		key := "unknown"
		if row.CareHours != nil {
			key = strconv.Itoa(*row.CareHours)
		}
		breakdown[key] = row
	}
	return breakdown
}

func toLegalBreakdownDetailsMap(rows []domain.LegalHoursBreakdown) map[string]domain.LegalHoursBreakdown {
	breakdown := map[string]domain.LegalHoursBreakdown{}
	for _, row := range rows {
		key := "unknown"
		if row.LegalHours != nil {
			key = strconv.Itoa(*row.LegalHours)
		}
		breakdown[key] = row
	}
	return breakdown
}
