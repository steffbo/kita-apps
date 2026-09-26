package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestParentWorkAccountIntegration(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewPostgresParentWorkRepository(testDB)
	svc := service.NewParentWorkService(repo)
	h1, h2, unused := uuid.New(), uuid.New(), uuid.New()
	m1, m2 := uuid.New(), uuid.New()
	user := uuid.New()
	must := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := testDB.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	must(`INSERT INTO fees.users (id,email,password_hash,role) VALUES ($1,$2,'test','ADMIN')`,
		user, user.String()+"@example.org")
	t.Cleanup(func() { must(`DELETE FROM fees.users WHERE id=$1`, user) })
	for id, name := range map[uuid.UUID]string{h1: "Familie Eins", h2: "Familie Zwei", unused: "Familie Ohne"} {
		must(`INSERT INTO fees.households (id,name) VALUES ($1,$2)`, id, name)
		t.Cleanup(func() { must(`DELETE FROM fees.households WHERE id=$1`, id) })
	}
	for _, h := range []uuid.UUID{h1, h1, h2} {
		id := uuid.New()
		must(`INSERT INTO fees.children (id,household_id,member_number,first_name,last_name,
            birth_date,entry_date) VALUES ($1,$2,$3,'Kind','Test','2020-01-01','2025-08-01')`,
			id, h, "TPW"+uuid.NewString()[:6])
		t.Cleanup(func() { must(`DELETE FROM fees.children WHERE id=$1`, id) })
	}
	must(`INSERT INTO fees.members (id,member_number,first_name,last_name,membership_start,household_id)
        VALUES ($1,$2,'Direkt','Mitglied','2025-01-01',$3)`, m1, "TPW"+uuid.NewString()[:6], h1)
	t.Cleanup(func() { must(`DELETE FROM fees.members WHERE id=$1`, m1) })
	must(`INSERT INTO fees.members (id,member_number,first_name,last_name,membership_start)
        VALUES ($1,$2,'Eltern','Mitglied','2025-01-01')`, m2, "TPW"+uuid.NewString()[:6])
	t.Cleanup(func() { must(`DELETE FROM fees.members WHERE id=$1`, m2) })
	p := uuid.New()
	must(`INSERT INTO fees.parents (id,household_id,member_id,first_name,last_name)
        VALUES ($1,$2,$3,'Eltern','Person')`, p, h2, m2)
	t.Cleanup(func() { must(`DELETE FROM fees.parents WHERE id=$1`, p) })
	for _, member := range []uuid.UUID{m1, m2} {
		term, err := svc.SaveTerm(ctx, uuid.Nil, domain.BoardTerm{MemberID: member, Office: "Vorsitz",
			StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		defer svc.DeleteTerm(ctx, term.ID)
	}
	listedTerms, err := svc.Terms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	foundLinkedFamily := false
	for _, term := range listedTerms {
		if term.MemberID == m2 && term.HouseholdID != nil && *term.HouseholdID == h2 {
			foundLinkedFamily = true
		}
	}
	if !foundLinkedFamily {
		t.Fatal("parent-linked board term has no household")
	}

	entry := func(h uuid.UUID, date string, minutes int) *domain.ParentWorkEntry {
		d, _ := time.Parse("2006-01-02", date)
		v, err := svc.SaveEntry(ctx, uuid.Nil, domain.ParentWorkEntry{HouseholdID: h, WorkDate: d,
			DurationMinutes: minutes, Occasion: "Aktion", Status: "APPROVED"}, user)
		if err != nil {
			t.Fatal(err)
		}
		deferFunc := func() { must(`DELETE FROM fees.parent_work_entries WHERE id=$1`, v.ID) }
		t.Cleanup(deferFunc)
		return v
	}
	entry(h1, "2025-09-01", 1500)
	entry(h1, "2026-09-01", 60)
	entry(h2, "2026-09-01", 120)
	voided := entry(h1, "2026-10-01", 120)
	if _, err := svc.VoidEntry(ctx, voided.ID, "Doppelt", user); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveOverride(ctx, domain.ParentWorkOverride{HouseholdID: h2, KitaYear: 2026,
		RequiredMinutes: 300, Reason: "Vereinbart"}, user); err != nil {
		t.Fatal(err)
	}
	defer svc.DeleteOverride(ctx, h2, 2026)
	overview, err := svc.Overview(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[uuid.UUID]domain.ParentWorkAccount{}
	for _, row := range overview.Households {
		rows[row.HouseholdID] = row
	}
	if _, found := rows[unused]; found {
		t.Fatal("unrelated household listed")
	}
	if rows[h1].CarryInMinutes != 180 || rows[h1].RequiredMinutes != 0 ||
		rows[h1].DoneMinutes != 60 || rows[h1].CarryOutMinutes != 60 || rows[h1].EntryCount != 2 ||
		rows[h1].ExemptReason == nil {
		t.Fatalf("direct member account: %+v", rows[h1])
	}
	if rows[h2].CalculatedMinutes != 0 || rows[h2].RequiredMinutes != 300 ||
		rows[h2].DoneMinutes != 120 || rows[h2].OpenMinutes != 180 || rows[h2].ExemptReason == nil {
		t.Fatalf("parent-linked member account: %+v", rows[h2])
	}
	blocked := uuid.New()
	must(`INSERT INTO fees.households (id,name) VALUES ($1,'Familie Mit Eintrag')`, blocked)
	t.Cleanup(func() { must(`DELETE FROM fees.households WHERE id=$1`, blocked) })
	entry(blocked, "2026-09-01", 15)
	householdSvc := service.NewHouseholdService(
		repository.NewPostgresHouseholdRepository(testDB), nil, nil)
	if err := householdSvc.Delete(ctx, blocked); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("household delete: got %v, want conflict", err)
	}

	detail, err := svc.Detail(ctx, h1, 2026)
	if err != nil || len(detail.Entries) != 2 || len(detail.BoardTerms) != 1 {
		t.Fatalf("detail: %+v, %v", detail, err)
	}
}
