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
- `backend-fees/migrations/000020_drop_users.up.sql`
- `openapi/fees/openapi3.yaml`

Current behavior:

- login is `POST /api/fees/v1/auth/login`
- protected routes use a JWT bearer access token
- credentials come from `USER_NAME` and `USER_PASSWORD`
- the authenticated identity is a static admin returned by `AuthService`
- `fees.users` was dropped; do not query it to discover login users
- refresh-token persistence does not imply that multiple application users exist

Prefer an already authenticated browser, an injected short-lived token, or a future dedicated service account. If current runtime credentials must be used, keep acquisition, login, and API invocation inside a non-echoing process and never expose the values.

## Dedicated agent identity target

A separate agent identity would improve revocation, rotation, and auditability, but it is not supported by the current single-user fees auth implementation. Implementing it requires an application change, not merely adding another environment variable.

Prefer these properties when that work is authorized:

- distinct non-human identity
- least-privilege API permissions
- no interactive UI requirement
- independently rotatable credential or client secret
- short-lived access tokens
- attributable audit events
- explicit enablement for production writes
