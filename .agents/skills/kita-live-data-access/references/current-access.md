# Current Access Model

## Runtime and database

The repository's current operational documentation identifies:

- runtime VM: `infra-dev`
- PostgreSQL container: `kita-db`
- database: `kita`
- fees schema: `fees`
- portal schema: `portal`
- management tables: primarily `public`

Use the current SSH host, identity file, and `docker exec ... psql` commands from the repository `AGENTS.md`; do not duplicate them here because infrastructure details can change.

Default to narrowly scoped `SELECT` statements and qualify cross-schema objects. Never place live credentials in SQL, shell history, committed files, or command output.

## Fees API authentication

Resolve the live contract from:

- `backend-fees/internal/api/router.go`
- `backend-fees/internal/api/handler/auth_handler.go`
- `backend-fees/internal/service/auth_service.go`
- `backend-fees/internal/config/config.go`
- `backend-fees/migrations/000036_users.up.sql`
- `backend-fees/internal/service/auth_service.go`, `user_service.go`
- `openapi/fees/openapi3.yaml`

Current behavior:

- login is `POST /api/fees/v1/auth/login`
- protected routes use a JWT bearer access token
- accounts live in `fees.users` (bcrypt hash, role `ADMIN`/`USER`, `is_active`); never read `password_hash`
- `USER_NAME` / `USER_PASSWORD` only bootstrap the admin account on first start; its password may since have been changed in the app, so the env value is not guaranteed to work
- role and email in tokens are taken from the DB on login and refresh; inactive accounts cannot log in or refresh

Prefer an already authenticated browser, an injected short-lived token, or a future dedicated service account. If current runtime credentials must be used, keep acquisition, login, and API invocation inside a non-echoing process and never expose the values.

## Dedicated agent identity target

A separate agent identity would improve revocation, rotation, and auditability, and is now possible: an admin can create a `USER` account for it on the "Benutzer" page. That is an explicit, user-authorized step; do not create it yourself.

Prefer these properties when that work is authorized:

- distinct non-human identity
- least-privilege API permissions
- no interactive UI requirement
- independently rotatable credential or client secret
- short-lived access tokens
- attributable audit events
- explicit enablement for production writes
