package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"slices"
)

// legacyDevJWTSecret was the built-in JWT_SECRET default before 2026-09. It is
// public (in git history) and must never sign tokens.
const legacyDevJWTSecret = "dev-secret-change-in-production"

const minJWTSecretLength = 32

// Harden replaces insecure settings with safe ones and returns a warning per
// adjustment. It never fails, so local development works without any env vars:
//   - a missing, legacy-default or too short JWT_SECRET is replaced by a random
//     per-process secret (tokens then become invalid on restart);
//   - a CORS wildcard is never combined with credentials.
func (c *Config) Harden() ([]string, error) {
	var warnings []string

	if c.JWT.Secret == "" || c.JWT.Secret == legacyDevJWTSecret || len(c.JWT.Secret) < minJWTSecretLength {
		secret, err := randomSecret()
		if err != nil {
			return nil, fmt.Errorf("generate JWT secret: %w", err)
		}
		c.JWT.Secret = secret
		warnings = append(warnings, fmt.Sprintf(
			"JWT_SECRET missing, legacy default or shorter than %d chars: using a random secret, sessions end on restart",
			minJWTSecretLength))
	}

	if slices.Contains(c.Server.CORSOrigins, "*") {
		c.Server.CORSAllowCredentials = false
		warnings = append(warnings, "CORS_ORIGINS contains '*': credentials are not allowed cross-origin")
	} else {
		c.Server.CORSAllowCredentials = len(c.Server.CORSOrigins) > 0
	}

	return warnings, nil
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
