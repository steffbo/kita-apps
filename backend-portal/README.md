# Kita Portal Backend

Portal API for the shared Kita portal and the first `parent_work` module.

The Phase 0 data model sketch is documented in `../PORTAL_DATA_MODEL.md`. It covers the portal identity/session tables, read-only synced master-data snapshots, parent-work tables, sync quarantine, and audit boundary.

## Local Development

```bash
cd backend-portal
go run cmd/migrate/main.go -direction up
go run cmd/server/main.go
```

Defaults:

- HTTP port: `8082`
- Database URL: `postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable&search_path=portal`
- Migration metadata table: `public.portal_schema_migrations` (kept outside the `portal` schema so `down` can drop `portal`)
- Health: `GET /health` and `GET /api/portal/v1/health`
- JWT issuer: `kita-portal`
- JWT access expiry: `15m`
- JWT refresh expiry: `168h`

Optional first-user bootstrap:

```bash
PORTAL_BOOTSTRAP_ADMIN_EMAIL=admin@example.org \
PORTAL_BOOTSTRAP_ADMIN_PASSWORD='change-this-locally' \
go run cmd/server/main.go
```

When both bootstrap variables are set, startup upserts one active `ADMIN` user with a bcrypt password hash. Leave them unset in normal operation.

## Auth API

- `POST /api/portal/v1/auth/login`
  - request: `{ "email": "...", "password": "..." }`
  - response: `{ "accessToken": "...", "refreshToken": "...", "expiresAt": "...", "user": { ... } }`
- `POST /api/portal/v1/auth/refresh`
  - request: `{ "refreshToken": "..." }`
  - atomically revokes the active persisted hashed refresh token and returns a new token pair.
- `GET /api/portal/v1/auth/me`
  - requires `Authorization: Bearer <accessToken>`.
- `POST /api/portal/v1/auth/logout`
  - requires `Authorization: Bearer <accessToken>`.
  - revokes all active refresh-token sessions for the authenticated user.

Current limitations:

- Invitation, password onboarding, password reset, sync, and parent-work endpoints are still pending.
- The portal frontend calls login and logout, but does not yet rotate refresh tokens.

## Boundary

This service owns portal identity, audit, synced read-only master data snapshots, email outbox state, and parent-work data. It must not write to the existing `fees` schema; imports from `backend-fees` land in `portal.synced_*` tables and unresolved conflicts go to `portal.sync_quarantine_items`.
