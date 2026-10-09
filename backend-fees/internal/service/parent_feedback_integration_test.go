package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

func TestParentFeedbackOtherParentAndReview(t *testing.T) {
	requireTestDatabase(t)
	ctx := context.Background()
	accountRepo := repository.NewParentAccountRepository(testDB)
	work := service.NewParentWorkService(repository.NewPostgresParentWorkRepository(testDB))
	svc := service.NewParentAccountService(accountRepo, work)
	home, foreign := uuid.New(), uuid.New()
	own, partner, stranger := uuid.New(), uuid.New(), uuid.New()
	ownUser, partnerUser, admin := uuid.New(), uuid.New(), uuid.New()
	hash, _ := auth.HashPassword("password123")
	for _, h := range []uuid.UUID{home, foreign} {
		if _, err := testDB.Exec(`INSERT INTO fees.households(id,name) VALUES ($1,'Family')`, h); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []struct{ id, household uuid.UUID }{{own, home}, {partner, home}, {stranger, foreign}} {
		if _, err := testDB.Exec(`INSERT INTO fees.parents(id,household_id,first_name,last_name,email)
            VALUES ($1,$2,'Parent','Test',$3)`, p.id, p.household,
			uuid.NewString()+"@example.org"); err != nil {
			t.Fatal(err)
		}
	}
	street := "Musterstraße"
	if _, err := testDB.Exec(`UPDATE fees.parents SET street=$2 WHERE id=$1`, partner, street); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec(`INSERT INTO fees.users(id,email,password_hash,role,parent_id)
        SELECT $1,email,$2,'PARENT',id FROM fees.parents WHERE id=$3`, ownUser, hash, own); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec(`INSERT INTO fees.users(id,email,password_hash,role,first_name,last_name)
        VALUES ($1,$2,$3,'ADMIN','Ada','Admin')`, admin, uuid.NewString()+"@example.org", hash); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testDB.Exec(`DELETE FROM fees.parent_work_entries WHERE household_id=$1`, home)
		testDB.Exec(`DELETE FROM fees.data_changes WHERE parent_id=$1`, own)
		testDB.Exec(`DELETE FROM fees.users WHERE id IN ($1,$2,$3)`, ownUser, partnerUser, admin)
		testDB.Exec(`DELETE FROM fees.parents WHERE id IN ($1,$2,$3)`, own, partner, stranger)
		testDB.Exec(`DELETE FROM fees.households WHERE id IN ($1,$2)`, home, foreign)
	})

	phone := "030 123"
	if _, err := svc.ParentContact(ctx, ownUser, stranger,
		map[string]*string{"phone": &phone}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("foreign parent: %v", err)
	}
	newEmail := uuid.NewString() + "@example.org"
	if _, err := svc.ParentContact(ctx, ownUser, partner,
		map[string]*string{"email": &newEmail}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("partner email without login: %v", err)
	}
	got, err := svc.ParentContact(ctx, ownUser, partner, map[string]*string{"phone": &phone})
	if err != nil || got.Phone == nil || *got.Phone != phone || got.Email == nil {
		t.Fatalf("partner without login: %+v %v", got, err)
	}
	if got.Street == nil || *got.Street != street {
		t.Fatalf("phone update changed address: %+v", got)
	}
	newStreet := "Andere Straße"
	if _, err = svc.ParentContact(ctx, ownUser, partner,
		map[string]*string{"street": &newStreet}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("partner address update: %v", err)
	}
	partnerEmail := *got.Email
	activity, err := svc.Activity(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range activity {
		if a.ParentID != nil && *a.ParentID == own && a.TargetName != nil && a.Field != nil &&
			*a.Field == "phone" {
			found = true
		}
	}
	if !found {
		t.Fatal("activity misses the edited partner")
	}

	if _, err = testDB.Exec(`INSERT INTO fees.users(id,email,password_hash,role,parent_id)
        VALUES ($1,$2,$3,'PARENT',$4)`, partnerUser, partnerEmail, hash, partner); err != nil {
		t.Fatal(err)
	}
	other := uuid.NewString() + "@example.org"
	if _, err = svc.ParentContact(ctx, ownUser, partner,
		map[string]*string{"email": &other}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("partner login email: %v", err)
	}
	if _, err = svc.ParentContact(ctx, ownUser, partner,
		map[string]*string{"email": &partnerEmail}); err != nil {
		t.Fatalf("partner with login, same email: %v", err)
	}

	report, err := svc.CreateReport(ctx, ownUser, service.ReportInput{Topic: "GENERAL", Message: "Fehler"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ResolveReport(ctx, report.ID, admin, "  Ist korrigiert.  "); err != nil {
		t.Fatal(err)
	}
	reports, err := svc.OwnReports(ctx, partnerUser)
	if err != nil || len(reports) != 1 || reports[0].Status != "DONE" || reports[0].Response == nil ||
		*reports[0].Response != "Ist korrigiert." || reports[0].ParentName == nil {
		t.Fatalf("partner sees answer: %+v %v", reports, err)
	}

	entry, err := svc.Submit(ctx, ownUser, domain.ParentWorkEntry{WorkDate: util.Today(),
		DurationMinutes: 60, Occasion: "Garten"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = work.ReviewEntry(ctx, entry.ID, true, "", admin); err != nil {
		t.Fatal(err)
	}
	detail, err := work.Detail(ctx, home, domain.ParentWorkKitaYear(util.Today()))
	if err != nil || len(detail.Entries) != 1 {
		t.Fatalf("detail: %+v %v", detail, err)
	}
	e := detail.Entries[0]
	if e.ReviewedBy == nil || *e.ReviewedBy != admin || e.ReviewedAt == nil || e.ReviewedByName == nil ||
		*e.ReviewedByName != "Ada Admin" {
		t.Fatalf("reviewer: %+v", e)
	}
	e.Occasion = "Garten, Beete"
	if _, err = work.SaveEntry(ctx, e.ID, e, ownUser); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.NewPostgresParentWorkRepository(testDB).GetEntry(ctx, e.ID)
	if err != nil || stored.ReviewedBy == nil || *stored.ReviewedBy != admin {
		t.Fatalf("edit keeps reviewer: %+v %v", stored, err)
	}
}
