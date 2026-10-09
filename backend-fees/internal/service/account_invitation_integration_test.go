package service_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

type fakeInvitationSender struct {
	bodies  []string
	failFor string
}

func (*fakeInvitationSender) IsEnabled() bool { return true }
func (f *fakeInvitationSender) SendTextEmail(to, _, body string) error {
	if to == f.failFor {
		return errors.New("smtp nicht erreichbar")
	}
	f.bodies = append(f.bodies, body)
	return nil
}

// invitationParent creates a parent with one child; exited children do not make a parent a candidate.
func invitationParent(t *testing.T, name string, email *string, exited ...bool) *domain.Parent {
	t.Helper()
	ctx := context.Background()
	p := &domain.Parent{ID: uuid.New(), FirstName: name, LastName: "Einladung",
		Email: email, IncomeStatus: domain.IncomeStatusUnknown}
	if err := repository.NewPostgresParentRepository(testDB).Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	gone := len(exited) > 0 && exited[0]
	child := &domain.Child{ID: uuid.New(), MemberNumber: fmt.Sprintf("TI%d", time.Now().UnixNano()%1e7),
		FirstName: "Kind" + name, LastName: "Einladung", BirthDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		EntryDate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), IsActive: !gone,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if gone {
		exit := time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC)
		child.ExitDate = &exit
	}
	children := repository.NewPostgresChildRepository(testDB)
	if err := children.Create(ctx, child); err != nil {
		t.Fatal(err)
	}
	if err := children.LinkParent(ctx, child.ID, p.ID, true); err != nil {
		t.Fatal(err)
	}
	return p
}

func tokenFromMail(t *testing.T, body string) string {
	t.Helper()
	token := regexp.MustCompile(`token=([a-f0-9]{64})`).FindStringSubmatch(body)
	if len(token) != 2 {
		t.Fatal("Einladungslink fehlt")
	}
	return token[1]
}

func TestAccountInvitations(t *testing.T) {
	ctx := context.Background()
	authSvc, userSvc := newUserServices(t)
	testDB.Exec(`DELETE FROM fees.email_logs`)
	t.Cleanup(func() {
		testDB.Exec(`DELETE FROM fees.email_logs`)
		testDB.Exec(`DELETE FROM fees.child_parents WHERE child_id IN
			(SELECT id FROM fees.children WHERE last_name = 'Einladung')`)
		testDB.Exec(`DELETE FROM fees.children WHERE last_name = 'Einladung'`)
		testDB.Exec(`DELETE FROM fees.parents WHERE last_name = 'Einladung'`)
	})
	emailA, emailB := "invite-a@example.test", "invite-b@example.test"
	same := "double@example.test"
	sameUpper := "DOUBLE@example.test"
	unlinkedEmail := "staff-email@example.test"
	existing := "existing@example.test"
	a := invitationParent(t, "Anna", &emailA)
	b := invitationParent(t, "Bert", &emailB)
	invitationParent(t, "Ohne", nil)
	invitationParent(t, "Doppel1", &same)
	invitationParent(t, "Doppel2", &sameUpper)
	invitationParent(t, "Staffadresse", &unlinkedEmail)
	old := invitationParent(t, "Vorhanden", &existing)
	formerEmail := "former@example.test"
	former := invitationParent(t, "Ehemalig", &formerEmail, true)
	existingUser, err := userSvc.Create(ctx, service.UserInput{Email: existing,
		Role: domain.UserRolePARENT, IsActive: true}, "password1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := userSvc.Create(ctx, service.UserInput{Email: unlinkedEmail,
		Role: domain.UserRoleUser, IsActive: true}, "password1"); err != nil {
		t.Fatal(err)
	}

	sender := &fakeInvitationSender{}
	repo := repository.NewAccountInvitationRepository(testDB)
	svc := service.NewAccountInvitationService(repository.NewPostgresUserRepository(testDB),
		repo, repository.NewPostgresEmailLogRepository(testDB), sender,
		"http://localhost:5175/beitraege")
	candidates, err := svc.Candidates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Temporarily the own linked PARENT account is a candidate too (UserID set).
	if len(candidates) != 4 {
		t.Fatalf("Kandidaten: %d, erwartet 4: %+v", len(candidates), candidates)
	}
	var duplicateID uuid.UUID
	for _, c := range candidates {
		if c.ParentID == old.ID && (c.UserID == nil || *c.UserID != existingUser.ID) {
			t.Errorf("bestehendes Konto nicht markiert: %+v", c)
		}
		if c.ParentID != old.ID && c.UserID != nil {
			t.Errorf("Konto bei %s gesetzt", c.FirstName)
		}
		if c.ParentID == former.ID {
			t.Error("Elternteil ohne aktives Kind als Kandidat")
		}
		if c.ParentID == a.ID && (len(c.Children) != 1 || c.Children[0] != "KindAnna Einladung") {
			t.Errorf("Kinder von Anna: %v", c.Children)
		}
		if c.Email == same {
			if !c.Ambiguous {
				t.Error("doppelte E-Mail nicht markiert")
			}
			duplicateID = c.ParentID
		}
	}
	results, err := svc.Invite(ctx, []uuid.UUID{a.ID, b.ID, duplicateID}, existingUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || !results[0].Success || !results[1].Success || results[2].Success {
		t.Fatalf("Sammelanlage: %+v", results)
	}
	if len(sender.bodies) != 2 {
		t.Fatalf("Mails: %d", len(sender.bodies))
	}
	account, err := repository.NewPostgresUserRepository(testDB).GetByEmail(ctx, emailA)
	if err != nil || account.ParentID == nil || *account.ParentID != a.ID || !account.InvitationPending {
		t.Fatalf("Elternverknüpfung/Status: %+v, %v", account, err)
	}
	if _, err := authSvc.Authenticate(ctx, emailA, "password1"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("Login vor Passwort: %v", err)
	}
	token := tokenFromMail(t, sender.bodies[0])
	if err := svc.SetPassword(ctx, token, "new-password"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetPassword(ctx, token, "new-password"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("Token mehrfach verwendbar: %v", err)
	}
	if _, err := authSvc.Authenticate(ctx, emailA, "new-password"); err != nil {
		t.Errorf("Login nach Passwort: %v", err)
	}
	var body string
	if err := testDB.Get(&body, `SELECT body FROM fees.email_logs
        WHERE email_type = 'ACCOUNT_INVITATION' AND to_email = $1`, emailA); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, token) || !strings.Contains(body, "[ausgeblendet]") {
		t.Error("E-Mail-Log enthält Token oder keine Maskierung")
	}

	oldToken := tokenFromMail(t, sender.bodies[1])
	bUser, err := repository.NewPostgresUserRepository(testDB).GetByEmail(ctx, emailB)
	if err != nil || bUser.ParentID == nil || *bUser.ParentID != b.ID {
		t.Fatal("zweites Konto fehlt")
	}
	if err := svc.Resend(ctx, bUser.ID, existingUser.ID); err != nil {
		t.Fatal(err)
	}
	newToken := tokenFromMail(t, sender.bodies[2])
	if err := svc.SetPassword(ctx, oldToken, "new-password"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("alter Token nach erneutem Senden: %v", err)
	}
	if _, err := testDB.Exec(`UPDATE fees.account_invitations SET expires_at = $1
        WHERE user_id = $2`, time.Now().Add(-time.Hour), bUser.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetPassword(ctx, newToken, "new-password"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("abgelaufener Token: %v", err)
	}

	// Existing account: new set-password link, old password stops working once pending.
	results, err = svc.Invite(ctx, []uuid.UUID{old.ID}, existingUser.ID)
	if err != nil || len(results) != 1 || !results[0].Success {
		t.Fatalf("Einladung bestehendes Konto: %+v, %v", results, err)
	}
	if _, err := authSvc.Authenticate(ctx, existing, "password1"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("Login mit altem Passwort: %v", err)
	}
	if err := svc.SetPassword(ctx, tokenFromMail(t, sender.bodies[3]), "renewed-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := authSvc.Authenticate(ctx, existing, "renewed-pass"); err != nil {
		t.Errorf("Login nach neuem Passwort: %v", err)
	}
}

func TestAccountInvitations_PartialMailFailure(t *testing.T) {
	ctx := context.Background()
	authSvc, _ := newUserServices(t)
	if _, err := authSvc.BootstrapAdmin(ctx, "invite-admin@example.test", "password1"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testDB.Exec(`DELETE FROM fees.parents WHERE last_name = 'Einladung'`) })
	emailA, emailB := "ok@example.test", "fail@example.test"
	a := invitationParent(t, "OK", &emailA)
	b := invitationParent(t, "Fehler", &emailB)
	sender := &fakeInvitationSender{failFor: emailB}
	users := repository.NewPostgresUserRepository(testDB)
	svc := service.NewAccountInvitationService(users, repository.NewAccountInvitationRepository(testDB),
		repository.NewPostgresEmailLogRepository(testDB), sender, "http://localhost:5175/beitraege")
	results, err := svc.Invite(ctx, []uuid.UUID{b.ID, a.ID}, service.BootstrapAdminID)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Success || !results[1].Success {
		t.Fatalf("Teilerfolg: %+v", results)
	}
	failed, err := users.GetByEmail(ctx, emailB)
	if err != nil || !failed.InvitationPending {
		t.Fatalf("fehlgeschlagenes Konto: %+v, %v", failed, err)
	}
}
