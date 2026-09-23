package config

import "testing"

func TestHarden_ReplacesInsecureJWTSecret(t *testing.T) {
	for _, secret := range []string{"", legacyDevJWTSecret, "too-short"} {
		c := &Config{JWT: JWTConfig{Secret: secret}}
		warnings, err := c.Harden()
		if err != nil {
			t.Fatal(err)
		}
		if c.JWT.Secret == secret || len(c.JWT.Secret) < minJWTSecretLength {
			t.Errorf("secret %q was not replaced (got len %d)", secret, len(c.JWT.Secret))
		}
		if len(warnings) != 1 {
			t.Errorf("secret %q: want 1 warning, got %v", secret, warnings)
		}
	}
}

func TestHarden_KeepsStrongJWTSecret(t *testing.T) {
	strong := "0123456789abcdef0123456789abcdef0123"
	c := &Config{JWT: JWTConfig{Secret: strong}}
	warnings, err := c.Harden()
	if err != nil {
		t.Fatal(err)
	}
	if c.JWT.Secret != strong || len(warnings) != 0 {
		t.Fatalf("strong secret changed or warned: %q %v", c.JWT.Secret, warnings)
	}
}

func TestHarden_CORS(t *testing.T) {
	strong := "0123456789abcdef0123456789abcdef0123"
	tests := []struct {
		origins   []string
		wantCreds bool
	}{
		{origins: nil, wantCreds: false},
		{origins: []string{"https://kita.remer.cc"}, wantCreds: true},
		{origins: []string{"*"}, wantCreds: false},
		{origins: []string{"https://a.example", "*"}, wantCreds: false},
	}
	for _, tt := range tests {
		c := &Config{JWT: JWTConfig{Secret: strong}, Server: ServerConfig{CORSOrigins: tt.origins}}
		if _, err := c.Harden(); err != nil {
			t.Fatal(err)
		}
		if c.Server.CORSAllowCredentials != tt.wantCreds {
			t.Errorf("origins %v: credentials = %v, want %v", tt.origins, c.Server.CORSAllowCredentials, tt.wantCreds)
		}
	}
}
