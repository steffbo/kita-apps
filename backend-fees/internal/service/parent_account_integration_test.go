package service_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestParentAccountLinking(t *testing.T) {
	ctx := context.Background()
	users := repository.NewPostgresUserRepository(testDB)
	tokens := repository.NewPostgresRefreshTokenRepository(testDB)
	svc := service.NewUserService(users, tokens)
	email := uuid.NewString() + "@example.org"
	input := service.UserInput{Email: email, Role: domain.UserRolePARENT, IsActive: true}
	_, err := svc.Create(ctx, input, "password123")
	if !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("zero matches: %v", err)
	}
	p1, p2 := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{p1, p2} {
		if _, err := testDB.Exec(`INSERT INTO fees.parents (id,first_name,last_name,email)
            VALUES ($1,'Parent','Test',$2)`, id, email); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { testDB.Exec(`DELETE FROM fees.parents WHERE id=$1`, id) })
	}
	_, err = svc.Create(ctx, input, "password123")
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("two matches: %v", err)
	}
	if _, err := testDB.Exec(`UPDATE fees.parents SET email=NULL WHERE id=$1`, p2); err != nil {
		t.Fatal(err)
	}
	user, err := svc.Create(ctx, input, "password123")
	if err != nil || user.ParentID == nil || *user.ParentID != p1 || user.ParentName == nil {
		t.Fatalf("one match: %+v, %v", user, err)
	}
	t.Cleanup(func() { testDB.Exec(`DELETE FROM fees.users WHERE id=$1`, user.ID) })
	otherEmail := uuid.NewString() + "@example.org"
	if _, err := testDB.Exec(`UPDATE fees.parents SET email=$2 WHERE id=$1`, p1, otherEmail); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(ctx, service.UserInput{Email: otherEmail, Role: domain.UserRolePARENT,
		IsActive: true}, "password123")
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("already linked: %v", err)
	}
	moved, err := svc.Update(ctx, uuid.New(), user.ID, service.UserInput{Email: email,
		Role: domain.UserRoleUser, IsActive: true})
	if err != nil || moved.ParentID != nil {
		t.Fatalf("role switch: %+v %v", moved, err)
	}
	moved, err = svc.Update(ctx, uuid.New(), user.ID, service.UserInput{Email: otherEmail,
		Role: domain.UserRolePARENT, IsActive: true})
	if err != nil || moved.ParentID == nil || *moved.ParentID != p1 {
		t.Fatalf("relink: %+v %v", moved, err)
	}
}

func TestParentAccountIsolationAndAudit(t *testing.T) {
	ctx := context.Background()
	accountRepo := repository.NewParentAccountRepository(testDB)
	work := service.NewParentWorkService(repository.NewPostgresParentWorkRepository(testDB))
	households := []uuid.UUID{uuid.New(), uuid.New()}
	parents := []uuid.UUID{uuid.New(), uuid.New()}
	users := []uuid.UUID{uuid.New(), uuid.New()}
	children := []uuid.UUID{uuid.New(), uuid.New()}
	for i := 0; i < 2; i++ {
		if _, err := testDB.Exec(`INSERT INTO fees.households(id,name) VALUES ($1,$2)`, households[i],
			"Family "+string(rune('A'+i))); err != nil {
			t.Fatal(err)
		}
		email := uuid.NewString() + "@example.org"
		hash, _ := auth.HashPassword("password123")
		if _, err := testDB.Exec(`INSERT INTO fees.parents(id,household_id,first_name,last_name,email)
            VALUES ($1,$2,'Parent','Test',$3)`, parents[i], households[i], email); err != nil {
			t.Fatal(err)
		}
		if _, err := testDB.Exec(`INSERT INTO fees.users(id,email,password_hash,role,parent_id)
            VALUES ($1,$2,$4,'PARENT',$3)`, users[i], email, parents[i], hash); err != nil {
			t.Fatal(err)
		}
		if _,
			err := testDB.Exec(`INSERT INTO fees.children(id,household_id,member_number,first_name,last_name,
            birth_date,entry_date) VALUES ($1,$2,$3,'Child','Test','2020-01-01','2025-08-01')`,
			children[i], households[i], uuid.NewString()[:8]); err != nil {
			t.Fatal(err)
		}
		if _,
			err := testDB.Exec(`INSERT INTO
			fees.fee_expectations(child_id,fee_type,year,month,amount,due_date)
            VALUES ($1,'FOOD',2026,9,45,'2026-09-01')`, children[i]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for i := 0; i < 2; i++ {
			testDB.Exec(`DELETE FROM fees.children WHERE id=$1`, children[i])
			testDB.Exec(`DELETE FROM fees.parent_work_entries WHERE household_id=$1`, households[i])
			testDB.Exec(`DELETE FROM fees.data_changes WHERE parent_id=$1 OR (entity_type='PARENT' AND
	entity_id=$1)
	`, parents[i])
			testDB.Exec(`DELETE FROM fees.users WHERE id=$1`, users[i])
			testDB.Exec(`DELETE FROM fees.parents WHERE id=$1`, parents[i])
			testDB.Exec(`DELETE FROM fees.households WHERE id=$1`, households[i])
		}
	})
	a, err := accountRepo.Account(ctx, users[0])
	if err != nil || a.Parent.ID != parents[0] {
		t.Fatal(err)
	}
	gotChildren, err := accountRepo.Children(ctx, a.Parent)
	if err != nil || len(gotChildren) != 1 || gotChildren[0].ID != children[0] {
		t.Fatalf("children: %+v %v", gotChildren, err)
	}
	fees, err := accountRepo.Fees(ctx, a.Parent, 2026)
	if err != nil || len(fees) != 1 || fees[0].ChildID != children[0] || fees[0].PaidAt != nil {
		t.Fatalf("fees: %+v %v", fees, err)
	}
	// A paid fee carries the booking date; a Mahngebühr from 2027 belongs to its 2026 base fee.
	paidFee, txID := uuid.New(), uuid.New()
	if _, err = testDB.Exec(`INSERT INTO fees.fee_expectations(id,child_id,fee_type,year,month,amount,due_date)
        VALUES ($1,$2,'FOOD',2026,8,45,'2026-08-01')`, paidFee, children[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = testDB.Exec(`INSERT INTO fees.bank_transactions(id,booking_date,value_date,amount)
        VALUES ($1,'2026-08-03','2026-08-03',45)`, txID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testDB.Exec(`DELETE FROM fees.bank_transactions WHERE id=$1`, txID) })
	if _, err = testDB.Exec(`INSERT INTO fees.payment_matches(transaction_id,expectation_id,match_type,amount)
        VALUES ($1,$2,'MANUAL',45)`, txID, paidFee); err != nil {
		t.Fatal(err)
	}
	if _, err = testDB.Exec(`INSERT INTO fees.fee_expectations(child_id,fee_type,year,amount,due_date,
        reminder_for_id) VALUES ($1,'REMINDER',2027,10,'2027-01-10',$2)`, children[0], fees[0].ID); err != nil {
		t.Fatal(err)
	}
	fees, err = accountRepo.Fees(ctx, a.Parent, 2026)
	if err != nil || len(fees) != 3 {
		t.Fatalf("fees with reminder: %+v %v", fees, err)
	}
	for _, f := range fees {
		switch {
		case f.ID == paidFee:
			if f.Status != "PAID" || f.PaidAt == nil || f.PaidAt.Format("2006-01-02") != "2026-08-03" {
				t.Fatalf("paid fee: %+v", f)
			}
		case f.FeeType == "REMINDER":
			if f.ReminderForID == nil || *f.BaseFeeType != "FOOD" || *f.BaseYear != 2026 || *f.BaseMonth != 9 {
				t.Fatalf("reminder: %+v", f)
			}
		}
	}
	date := domain.ParentWorkYearStart(2026)
	entry, err := work.SubmitParentEntry(ctx, domain.ParentWorkEntry{HouseholdID: households[1],
		WorkDate: date, DurationMinutes: 60, Occasion: "Garden"}, users[1])
	if err != nil {
		t.Fatal(err)
	}
	if _, err = work.WithdrawParentEntry(ctx, entry.ID, households[0], users[0]); !errors.Is(err,
		service.ErrNotFound) {
		t.Fatalf("foreign withdrawal: %v", err)
	}
	own, err := work.SubmitParentEntry(ctx, domain.ParentWorkEntry{HouseholdID: households[0],
		WorkDate: date, DurationMinutes: 60, Occasion: "Garden"}, users[0])
	if err != nil {
		t.Fatal(err)
	}
	detail, err := work.Detail(ctx, households[0], 2026)
	if err != nil || detail.DoneMinutes != 0 || len(detail.Entries) != 1 {
		t.Fatalf("submitted: %+v %v", detail, err)
	}
	if _, err = work.ReviewEntry(ctx, own.ID, true, "", users[1]); err != nil {
		t.Fatal(err)
	}
	detail, err = work.Detail(ctx, households[0], 2026)
	if err != nil || detail.DoneMinutes != 60 {
		t.Fatalf("approved: %+v %v", detail, err)
	}
	rejected, err := work.ReviewEntry(ctx, entry.ID, false, "Unterschrift fehlt", users[0])
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != domain.ParentWorkStatusRejected || rejected.RejectReason == nil ||
		*rejected.RejectReason != "Unterschrift fehlt" {
		t.Fatalf("rejection: %+v", rejected)
	}
	if err = accountRepo.SaveContact(ctx, parents[0], users[0], map[string]*string{
		"email": nil, "phone": nil, "street": nil, "street_no": nil, "postal_code": nil,
		"city": nil}); !errors.Is(err, repository.ErrEmailRequired) {
		t.Fatalf("linked email must be required: %v", err)
	}
	email := *a.Parent.Email
	phone := "123"
	city := "Berlin"
	if err = accountRepo.SaveContact(ctx, parents[0], users[0], map[string]*string{
		"email": &email, "phone": &phone, "street": nil, "street_no": nil, "postal_code": nil,
		"city": &city}); err != nil {
		t.Fatal(err)
	}
	changes, err := accountRepo.Changes(ctx, "PARENT", parents[0])
	if err != nil || len(changes) != 2 {
		t.Fatalf("changes: %+v %v", changes, err)
	}
	activity, err := accountRepo.Activity(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	contact, report := false, false
	for _, v := range activity {
		contact = contact || v.Type == "CONTACT_CHANGED"
		report = report || v.Type == "PARENT_WORK_SUBMITTED"
	}
	if !contact || !report {
		t.Fatalf("activity missing types: %+v", activity)
	}

	accountSvc := service.NewParentAccountService(accountRepo, work)
	street := "Musterstraße"
	if _, err := testDB.Exec(`UPDATE fees.parents SET street=$2 WHERE id=$1`, parents[0], street); err != nil {
		t.Fatal(err)
	}
	newStreet := "Andere Straße"
	if _, err := accountSvc.Contact(ctx, users[0],
		map[string]*string{"street": &newStreet}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("address update: %v", err)
	}
	phone = "030 12345"
	updatedParent, err := accountSvc.Contact(ctx, users[0], map[string]*string{"phone": &phone})
	if err != nil || updatedParent.Street == nil || *updatedParent.Street != street {
		t.Fatalf("phone update changed address: %+v %v", updatedParent, err)
	}
	if _, err := accountSvc.CreateReport(ctx, users[0], service.ReportInput{Topic: "CHILD",
		ReferenceID: &children[1], Message: "Wrong"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("foreign report: %v", err)
	}
	if _, err := accountSvc.CreateReport(ctx, users[0], service.ReportInput{Topic: "CHILD",
		ReferenceID: &children[0], Message: "Please check"}); err != nil {
		t.Fatal(err)
	}
	activity, err = accountRepo.Activity(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, v := range activity {
		seen[v.Type] = true
	}
	for _, kind := range []string{"CONTACT_CHANGED", "PARENT_WORK_SUBMITTED",
		"REPORT_CREATED"} {
		if !seen[kind] {
			t.Fatalf("missing activity %s", kind)
		}
	}

	userRepo := repository.NewPostgresUserRepository(testDB)
	current, err := userRepo.GetByID(ctx, users[0])
	if err != nil {
		t.Fatal(err)
	}
	oldEmail := current.Email
	taken, err := userRepo.GetByID(ctx, users[1])
	if err != nil {
		t.Fatal(err)
	}
	if _, err = accountSvc.Contact(ctx, users[0],
		map[string]*string{"email": &taken.Email}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("email conflict: %v", err)
	}
	current, err = userRepo.GetByID(ctx, users[0])
	if err != nil || current.Email != oldEmail {
		t.Fatalf("conflict changed user: %+v %v", current, err)
	}
	a, err = accountRepo.Account(ctx, users[0])
	if err != nil || *a.Parent.Email != oldEmail {
		t.Fatalf("conflict changed parent: %+v %v", a, err)
	}
	newEmail := uuid.NewString() + "@example.org"
	if _, err = accountSvc.Contact(ctx, users[0], map[string]*string{"email": &newEmail}); err != nil {
		t.Fatal(err)
	}
	authSvc := service.NewAuthService(userRepo, time.Hour,
		repository.NewPostgresRefreshTokenRepository(testDB))
	if _, err = authSvc.Authenticate(ctx, newEmail, "password123"); err != nil {
		t.Fatalf("new login: %v", err)
	}
	if _, err = authSvc.Authenticate(ctx, oldEmail, "password123"); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("old login: %v", err)
	}
	staff := service.NewParentService(repository.NewPostgresParentRepository(testDB), nil, nil, nil)
	staff.SetAccountChanges(accountRepo)
	if _, err = staff.Update(ctx, parents[0], service.UpdateParentInput{Email: &taken.Email,
		ActorID: users[1]}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("staff conflict: %v", err)
	}
	a, err = accountRepo.Account(ctx, users[0])
	if err != nil || *a.Parent.Email != newEmail {
		t.Fatalf("staff conflict changed parent: %+v %v", a, err)
	}
	current, err = userRepo.GetByID(ctx, users[0])
	if err != nil || current.Email != newEmail {
		t.Fatalf("staff conflict changed user: %+v %v", current, err)
	}
	staffEmail := uuid.NewString() + "@example.org"
	if _, err = staff.Update(ctx, parents[0], service.UpdateParentInput{Email: &staffEmail,
		ActorID: users[1]}); err != nil {
		t.Fatalf("staff email: %v", err)
	}
	if _, err = authSvc.Authenticate(ctx, staffEmail, "password123"); err != nil {
		t.Fatalf("staff login: %v", err)
	}
	userSvc := service.NewUserService(userRepo, repository.NewPostgresRefreshTokenRepository(testDB))
	if _, err = userSvc.Update(ctx, users[1], users[0], service.UserInput{Email: taken.Email,
		Role: domain.UserRolePARENT, IsActive: true}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("admin conflict: %v", err)
	}
	a, err = accountRepo.Account(ctx, users[0])
	if err != nil || *a.Parent.Email != staffEmail {
		t.Fatalf("admin conflict changed parent: %+v %v", a, err)
	}
	adminEmail := uuid.NewString() + "@example.org"
	if _, err = userSvc.Update(ctx, users[1], users[0], service.UserInput{Email: adminEmail,
		Role: domain.UserRolePARENT, IsActive: true}); err != nil {
		t.Fatalf("admin login email: %v", err)
	}
	a, err = accountRepo.Account(ctx, users[0])
	if err != nil || *a.Parent.Email != adminEmail {
		t.Fatalf("admin email sync: %+v %v", a, err)
	}

	jwt := auth.NewJWTService("parent-account-test", time.Minute, time.Hour, "test")
	router := api.NewRouter(&config.Config{}, &api.Handlers{JWTService: jwt,
		ParentAccount: handler.NewParentAccountHandler(accountRepo, work)})
	tokens, err := jwt.GenerateTokenPair(users[0], "parent@example.org", "PARENT")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/me", "/me/fees?year=2026", "/me/parent-work?year=2026"} {
		req := httptest.NewRequest(http.MethodGet, "/api/fees/v1"+path, nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != 200 || strings.Contains(rec.Body.String(), households[1].String()) ||
			strings.Contains(rec.Body.String(), children[1].String()) {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/fees/v1/me/parent-work/entries/"+
		entry.ID.String()+"/withdraw", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("foreign withdrawal: %d %s", rec.Code, rec.Body.String())
	}
}
