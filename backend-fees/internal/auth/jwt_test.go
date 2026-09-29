package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestImpersonationToken(t *testing.T) {
	svc := NewJWTService("secret", time.Minute, time.Hour, "test")
	admin, user := uuid.New(), uuid.New()

	token, expiresAt, err := svc.GenerateImpersonationToken(admin, user, "user@example.org", "PARENT")
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expiresAt = %v, want in the future", expiresAt)
	}
	claims, err := svc.ValidateToken(token, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != user || claims.Role != "PARENT" || claims.ImpersonatorID == nil || *claims.ImpersonatorID != admin {
		t.Fatalf("claims = %+v", claims)
	}
	if _, err := svc.ValidateToken(token, TokenTypeRefresh); err == nil {
		t.Error("impersonation token must not validate as refresh token")
	}

	pair, err := svc.GenerateTokenPair(user, "user@example.org", "PARENT")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := svc.ValidateToken(pair.AccessToken, TokenTypeAccess)
	if err != nil || plain.ImpersonatorID != nil {
		t.Fatalf("plain token impersonator = %v, err %v", plain.ImpersonatorID, err)
	}
}
