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

func newUserServices(t *testing.T) (*service.AuthService, *service.UserService) {
	t.Helper()
	cleanupUsers := func() {
		testDB.Exec(`DELETE FROM fees.refresh_tokens`)
		testDB.Exec(`DELETE FROM fees.users`)
	}
	cleanupUsers()
	t.Cleanup(cleanupUsers)
	users := repository.NewPostgresUserRepository(testDB)
	tokens := repository.NewPostgresRefreshTokenRepository(testDB)
	return service.NewAuthService(users, time.Hour, tokens), service.NewUserService(users, tokens)
}

// The env admin keeps the former static ID and password; restarts never overwrite it.
func TestBootstrapAdmin_KeepsStaticIDAndNeverOverwrites(t *testing.T) {
	ctx := context.Background()
	authSvc, _ := newUserServices(t)

	created, err := authSvc.BootstrapAdmin(ctx, "admin@example.com", "env-password")
	if err != nil || !created {
		t.Fatalf("bootstrap: created=%v err=%v", created, err)
	}
	user, err := authSvc.Authenticate(ctx, "Admin@Example.com", "env-password")
	if err != nil {
		t.Fatalf("login with env password: %v", err)
	}
	if user.ID != service.BootstrapAdminID || user.Role != domain.UserRoleAdmin {
		t.Fatalf("admin = %s/%s, want %s/ADMIN", user.ID, user.Role, service.BootstrapAdminID)
	}

	if err := authSvc.ChangePassword(ctx, user.ID, "env-password", "changed-in-app"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	created, err = authSvc.BootstrapAdmin(ctx, "admin@example.com", "env-password")
	if err != nil || created {
		t.Fatalf("second bootstrap: created=%v err=%v, want no-op", created, err)
	}
	if _, err := authSvc.Authenticate(ctx, "admin@example.com", "env-password"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("old password: err = %v, want ErrUnauthorized", err)
	}
	if _, err := authSvc.Authenticate(ctx, "admin@example.com", "changed-in-app"); err != nil {
		t.Errorf("new password: %v", err)
	}

	// A different env email gets a fresh ID because the static one is taken.
	created, err = authSvc.BootstrapAdmin(ctx, "other@example.com", "env-password")
	if err != nil || !created {
		t.Fatalf("bootstrap other: created=%v err=%v", created, err)
	}
	other, err := authSvc.Authenticate(ctx, "other@example.com", "env-password")
	if err != nil || other.ID == service.BootstrapAdminID {
		t.Fatalf("other admin = %v, err %v; want new ID", other, err)
	}
}

func TestChangePassword_RevokesSessions(t *testing.T) {
	ctx := context.Background()
	authSvc, _ := newUserServices(t)
	if _, err := authSvc.BootstrapAdmin(ctx, "admin@example.com", "env-password"); err != nil {
		t.Fatal(err)
	}
	if err := authSvc.StoreRefreshToken(ctx, service.BootstrapAdminID, "token"); err != nil {
		t.Fatal(err)
	}

	if err := authSvc.ChangePassword(ctx, service.BootstrapAdminID, "wrong", "new-password"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("wrong current password: err = %v, want ErrUnauthorized", err)
	}
	if err := authSvc.ChangePassword(ctx, service.BootstrapAdminID, "env-password", "short"); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("short password: err = %v, want ErrInvalidInput", err)
	}
	if err := authSvc.ChangePassword(ctx, service.BootstrapAdminID, "env-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	valid, _ := authSvc.ValidateRefreshToken(ctx, service.BootstrapAdminID, "token")
	if valid {
		t.Error("refresh token still valid after password change")
	}
}

func TestUserService_ManageAccounts(t *testing.T) {
	ctx := context.Background()
	authSvc, userSvc := newUserServices(t)
	if _, err := authSvc.BootstrapAdmin(ctx, "admin@example.com", "env-password"); err != nil {
		t.Fatal(err)
	}
	adminID := service.BootstrapAdminID
	name := "  Erika "

	if _, err := userSvc.Create(ctx, service.UserInput{Email: "no-at", Role: domain.UserRoleUser, IsActive: true}, "password1"); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("invalid email: err = %v", err)
	}
	if _, err := userSvc.Create(ctx, service.UserInput{Email: "x@example.com", Role: "BOSS", IsActive: true}, "password1"); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("invalid role: err = %v", err)
	}
	if _, err := userSvc.Create(ctx, service.UserInput{Email: "x@example.com", Role: domain.UserRoleUser, IsActive: true}, "short"); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("short password: err = %v", err)
	}
	if _, err := userSvc.Create(ctx, service.UserInput{Email: "ADMIN@example.com", Role: domain.UserRoleUser, IsActive: true}, "password1"); !errors.Is(err, service.ErrConflict) {
		t.Errorf("duplicate email (case-insensitive): err = %v, want ErrConflict", err)
	}

	erika, err := userSvc.Create(ctx, service.UserInput{Email: " erika@example.com ", FirstName: &name, Role: domain.UserRoleUser, IsActive: true}, "password1")
	if err != nil {
		t.Fatal(err)
	}
	if erika.Email != "erika@example.com" || erika.FirstName == nil || *erika.FirstName != "Erika" {
		t.Errorf("not normalized: %q %v", erika.Email, erika.FirstName)
	}
	if _, err := authSvc.Authenticate(ctx, "erika@example.com", "password1"); err != nil {
		t.Errorf("login new user: %v", err)
	}

	// Self-protection: no self-deactivation or self-demotion.
	selfInput := service.UserInput{Email: "admin@example.com", Role: domain.UserRoleUser, IsActive: true}
	if _, err := userSvc.Update(ctx, adminID, adminID, selfInput); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("self-demotion: err = %v, want ErrInvalidInput", err)
	}
	selfInput = service.UserInput{Email: "admin@example.com", Role: domain.UserRoleAdmin, IsActive: false}
	if _, err := userSvc.Update(ctx, adminID, adminID, selfInput); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("self-deactivation: err = %v, want ErrInvalidInput", err)
	}
	if _, err := userSvc.Update(ctx, adminID, uuid.New(), service.UserInput{Email: "a@b.de", Role: domain.UserRoleUser}); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("unknown user: err = %v, want ErrNotFound", err)
	}

	// Deactivation blocks login and ends sessions.
	if err := authSvc.StoreRefreshToken(ctx, erika.ID, "erika-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := userSvc.Update(ctx, adminID, erika.ID, service.UserInput{Email: "erika@example.com", Role: domain.UserRoleUser, IsActive: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := authSvc.Authenticate(ctx, "erika@example.com", "password1"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("inactive login: err = %v, want ErrUnauthorized", err)
	}
	if valid, _ := authSvc.ValidateRefreshToken(ctx, erika.ID, "erika-token"); valid {
		t.Error("refresh token still valid after deactivation")
	}

	// Admin password reset.
	if err := userSvc.SetPassword(ctx, erika.ID, "short"); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("short reset: err = %v", err)
	}
	if err := userSvc.SetPassword(ctx, uuid.New(), "password2"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("reset unknown: err = %v, want ErrNotFound", err)
	}
	if _, err := userSvc.Update(ctx, adminID, erika.ID, service.UserInput{Email: "erika@example.com", Role: domain.UserRoleAdmin, IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if err := userSvc.SetPassword(ctx, erika.ID, "password2"); err != nil {
		t.Fatal(err)
	}
	user, err := authSvc.Authenticate(ctx, "erika@example.com", "password2")
	if err != nil || user.Role != domain.UserRoleAdmin {
		t.Errorf("login after reset: %v, role %v", err, user)
	}

	list, err := userSvc.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %d users, err %v", len(list), err)
	}
}
