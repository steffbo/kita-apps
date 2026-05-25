# Kita Portal Backend

Portal API for the shared Kita portal and the first `parent_work` module.

## Local Development

```bash
cd backend-portal
go run cmd/migrate/main.go -direction up
go run cmd/server/main.go
```

Defaults:

- HTTP port: `8082`
- Database URL: `postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable&search_path=portal`
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
  - rotates the persisted hashed refresh token.
- `GET /api/portal/v1/auth/me`
  - requires `Authorization: Bearer <accessToken>`.
- `POST /api/portal/v1/auth/logout`
  - request: `{ "refreshToken": "..." }`
  - revokes the matching persisted refresh token when present.

Current limitations:

- Invitation, password onboarding, password reset, sync, and parent-work endpoints are still pending.
- The portal frontend calls login, but does not yet call refresh/logout endpoints.

## Boundary

This service owns portal identity, audit, synced read-only master data snapshots, email outbox state, and parent-work data. It must not write to the existing `fees` schema; imports from `backend-fees` land in `portal.synced_*` tables and unresolved conflicts go to `portal.sync_quarantine_items`.
