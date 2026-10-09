package service_test

import (
	"context"
	"encoding/json"
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
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestImpersonation(t *testing.T) {
	requireTestDatabase(t)
	ctx := context.Background()
	authSvc, userSvc := newUserServices(t)
	users := repository.NewPostgresUserRepository(testDB)
	tokens := repository.NewPostgresRefreshTokenRepository(testDB)
	accountRepo := repository.NewParentAccountRepository(testDB)
	work := service.NewParentWorkService(repository.NewPostgresParentWorkRepository(testDB))

	householdID, parentID := uuid.New(), uuid.New()
	parentEmail := uuid.NewString() + "@example.org"
	if _, err := testDB.Exec(`INSERT INTO fees.households(id,name) VALUES ($1,'Imp Family')`, householdID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec(`INSERT INTO fees.parents(id,household_id,first_name,last_name,email)
		VALUES ($1,$2,'Imp','Parent',$3)`, parentID, householdID, parentEmail); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testDB.Exec(`DELETE FROM fees.data_changes WHERE parent_id=$1`, parentID)
		testDB.Exec(`DELETE FROM fees.users`)
		testDB.Exec(`DELETE FROM fees.parents WHERE id=$1`, parentID)
		testDB.Exec(`DELETE FROM fees.households WHERE id=$1`, householdID)
	})

	if _, err := authSvc.BootstrapAdmin(ctx, "admin@example.com", "env-password"); err != nil {
		t.Fatal(err)
	}
	hash, _ := auth.HashPassword("password123")
	var parentUserID, inactiveID, otherAdminID uuid.UUID
	for _, row := range []struct {
		id       *uuid.UUID
		email    string
		role     string
		active   bool
		parentID *uuid.UUID
	}{
		{&parentUserID, parentEmail, "PARENT", true, &parentID},
		{&inactiveID, "inactive@example.org", "USER", false, nil},
		{&otherAdminID, "admin2@example.org", "ADMIN", true, nil},
	} {
		*row.id = uuid.New()
		if _, err := testDB.Exec(`INSERT INTO fees.users(id,email,password_hash,role,is_active,parent_id)
			VALUES ($1,$2,$3,$4,$5,$6)`, *row.id, row.email, hash, row.role, row.active, row.parentID); err != nil {
			t.Fatal(err)
		}
	}
	adminID := service.BootstrapAdminID

	t.Run("target rules", func(t *testing.T) {
		if u, err := userSvc.ImpersonationTarget(ctx, adminID, parentUserID); err != nil || u.ID != parentUserID {
			t.Fatalf("parent target: %v %v", u, err)
		}
		for name, id := range map[string]uuid.UUID{"self": adminID, "admin": otherAdminID, "inactive": inactiveID} {
			if _, err := userSvc.ImpersonationTarget(ctx, adminID, id); !errors.Is(err, service.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
			}
		}
		if _, err := userSvc.ImpersonationTarget(ctx, adminID, uuid.New()); !errors.Is(err, service.ErrNotFound) {
			t.Errorf("unknown: err = %v, want ErrNotFound", err)
		}
	})

	jwt := auth.NewJWTService("impersonation-test", time.Minute, time.Hour, "test")
	router := api.NewRouter(&config.Config{}, &api.Handlers{
		JWTService:    jwt,
		Auth:          handler.NewAuthHandler(authSvc, jwt, auth.NewLoginLimiter(3, 10, time.Minute)),
		User:          handler.NewUserHandler(service.NewUserService(users, tokens), jwt),
		ParentAccount: handler.NewParentAccountHandler(service.NewParentAccountService(accountRepo, work)),
	})
	call := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/fees/v1"+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	adminTokens, err := jwt.GenerateTokenPair(adminID, "admin@example.com", "ADMIN")
	if err != nil {
		t.Fatal(err)
	}

	rec := call(http.MethodPost, "/users/"+parentUserID.String()+"/impersonate", adminTokens.AccessToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("impersonate: %d %s", rec.Code, rec.Body.String())
	}
	var imp struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
		t.Fatal(err)
	}
	if imp.User.ID != parentUserID.String() || imp.User.Role != "PARENT" {
		t.Fatalf("impersonated user = %+v", imp.User)
	}

	for _, id := range []uuid.UUID{adminID, otherAdminID, inactiveID} {
		if rec := call(http.MethodPost, "/users/"+id.String()+"/impersonate", adminTokens.AccessToken, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("impersonate %s: %d, want 400", id, rec.Code)
		}
	}

	t.Run("acts as the user", func(t *testing.T) {
		if rec := call(http.MethodGet, "/me", imp.AccessToken, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), parentEmail) {
			t.Fatalf("/me: %d %s", rec.Code, rec.Body.String())
		}
		if rec := call(http.MethodGet, "/auth/me", imp.AccessToken, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"PARENT"`) {
			t.Fatalf("/auth/me: %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("has no admin rights", func(t *testing.T) {
		if rec := call(http.MethodGet, "/users/", imp.AccessToken, ""); rec.Code != http.StatusForbidden {
			t.Errorf("GET /users: %d, want 403", rec.Code)
		}
		if rec := call(http.MethodPost, "/users/"+parentUserID.String()+"/impersonate", imp.AccessToken, ""); rec.Code != http.StatusForbidden {
			t.Errorf("chained impersonation: %d, want 403", rec.Code)
		}
		if rec := call(http.MethodPost, "/auth/change-password", imp.AccessToken,
			`{"currentPassword":"password123","newPassword":"another-password"}`); rec.Code != http.StatusForbidden {
			t.Errorf("change password: %d, want 403", rec.Code)
		}
	})

	t.Run("audit names the admin", func(t *testing.T) {
		if rec := call(http.MethodPut, "/me/contact", imp.AccessToken, `{"phone":"0123 456"}`); rec.Code != http.StatusOK {
			t.Fatalf("PUT /me/contact: %d %s", rec.Code, rec.Body.String())
		}
		var got struct {
			UserID         uuid.UUID  `db:"user_id"`
			ImpersonatedBy *uuid.UUID `db:"impersonated_by"`
		}
		if err := testDB.Get(&got, `SELECT user_id, impersonated_by FROM fees.data_changes
			WHERE parent_id=$1 AND field='phone'`, parentID); err != nil {
			t.Fatal(err)
		}
		if got.UserID != parentUserID || got.ImpersonatedBy == nil || *got.ImpersonatedBy != adminID {
			t.Fatalf("audit row = %+v, want user %s impersonated by %s", got, parentUserID, adminID)
		}
	})

	t.Run("normal edits carry no impersonator", func(t *testing.T) {
		own, err := jwt.GenerateTokenPair(parentUserID, parentEmail, "PARENT")
		if err != nil {
			t.Fatal(err)
		}
		if rec := call(http.MethodPut, "/me/contact", own.AccessToken, `{"phone":"0999 111"}`); rec.Code != http.StatusOK {
			t.Fatalf("PUT /me/contact: %d %s", rec.Code, rec.Body.String())
		}
		var impersonated *uuid.UUID
		if err := testDB.Get(&impersonated, `SELECT impersonated_by FROM fees.data_changes
			WHERE parent_id=$1 AND field='phone' AND new_value='0999 111'`, parentID); err != nil || impersonated != nil {
			t.Fatalf("impersonated_by = %v, err %v; want NULL", impersonated, err)
		}
	})
}
