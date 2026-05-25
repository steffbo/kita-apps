# AGENTS.md

## Scope

Monorepo for Kita apps with:
- Go backends
- Vue frontends in a Bun workspace
- a new portal foundation with parent-work MVP scope
- a separate banking sync service
- Docker/GHCR build and homelab deployment

## Mandatory Development State Documentation

The repository documentation is the handoff contract for future sessions. Every agent must keep the current development state documented at all times.

This is mandatory:
- When implementation state changes, update `AGENTS.md` and/or the relevant project document in the same turn before finishing.
- A new session must be able to understand the latest architecture, changed files, commands, ports, migrations, known blockers, and next steps from repository documents alone, without relying on chat history.
- Document new services, frontend apps, modules, database schemas, environment variables, API endpoints, deployment steps, test commands, and temporary operational notes as soon as they become true.
- If a feature is scaffolded but not fully wired, state that explicitly in the docs.
- Do not leave docs stale after code changes. If code and docs disagree, fix the docs immediately.
- In the final response, mention which documentation was updated when development-state documentation changed.

## Mandatory Local Git Checkpoints

Development progress must be preserved in Git, even before code is ready to push.

This is mandatory:
- After a coherent development increment, stage the changed files and create a local commit before ending the turn, unless the user explicitly asks not to commit.
- A commit must include the matching documentation updates required by the development-state documentation rule above.
- Keep commits focused and use descriptive messages that state the implemented behavior or documented decision.
- If a change is intentionally left uncommitted, document why in the final response and leave `git status` easy to interpret.
- Do not push commits unless the user explicitly requests a push or deployment flow.

## Repository Layout

- `backend-management/`: management API (schedule + time tracking)
- `backend-fees/`: fees API
- `backend-portal/`: portal API foundation and parent-work module
- `frontend/`: workspace with frontend apps and shared package
- `banking-sync/`: sync/import service
- `openapi/`: generated OpenAPI specs (`management`, `fees`)
- `docker/`: compose files and Docker definitions
- `scripts/`: helper scripts
- `PORTAL_SPEC.md`: portal product, architecture, delivery, and launch scope spec
- `PORTAL_DATA_MODEL.md`: Phase 0 portal data model sketch for identity/session tables, read-only synced master-data snapshots, parent-work tables, and audit boundaries

## Backend Services

### backend-portal

- Path: `backend-portal/`
- Port: `8082`
- Purpose: shared portal API foundation, identity boundary, synced read-only master-data snapshots, audit, email outbox, and parent-work data
- Stack: Go, Chi, PostgreSQL
- Database: `kita` DB, `portal` schema
- Entrypoints:
  - server: `cmd/server/main.go`
  - migrations: `cmd/migrate/main.go`
- Health endpoints:
  - `GET /health`
  - `GET /api/portal/v1/health`

Primary areas:
- `internal/api/`
- `internal/config/`
- `internal/domain/`
- `internal/service/`
- `migrations/`

Current state:
- Phase 0 spec/architecture review is complete as of 2026-05-25. `PORTAL_SPEC.md` is marked reviewed and frozen for the August 2026 MVP: portal foundation plus `parent_work`, with legacy `fees`, `schedule`, and `time_tracking` apps outside the launch cut.
- Phase 0 data model sketch exists in `PORTAL_DATA_MODEL.md` and documents table boundaries for `portal.users`, `portal.user_roles`, `portal.refresh_tokens`, `portal.synced_*`, `portal.parent_work_*`, sync quarantine, and generic audit events.
- Initial schema exists in `migrations/000001_initial_schema.up.sql`.
- Portal migration metadata is stored in `public.portal_schema_migrations`, not in the `portal` schema, so rollback can drop `portal` without deleting the migrate bookkeeping table first.
- Schema includes portal users, roles, invitations, password resets, refresh tokens, synced households/parents/children, sync runs, sync quarantine, parent-work requirements, parent-work entries, email outbox, and audit events.
- Portal auth login foundation is implemented:
  - `POST /api/portal/v1/auth/login`
  - `POST /api/portal/v1/auth/refresh`
  - `GET /api/portal/v1/auth/me`
  - `POST /api/portal/v1/auth/logout`
- Auth uses bcrypt password hashes, JWT access/refresh tokens, persisted hashed refresh tokens in `portal.refresh_tokens`, and active `portal.users` with roles from `portal.user_roles`.
- Refresh-token rotation atomically revokes the consumed active refresh token before minting a replacement; logout revokes all active refresh-token sessions for the authenticated user.
- Optional bootstrap admin creation is available through `PORTAL_BOOTSTRAP_ADMIN_EMAIL` and `PORTAL_BOOTSTRAP_ADMIN_PASSWORD`; it upserts an active `ADMIN` user on server startup when both values are set. Do not configure a default password in committed files.
- Parent-work required-hours rule is implemented and tested in `internal/service/parent_work_rules.go`.
- Invite, onboarding, password-reset, sync, and parent-work API endpoints are not implemented yet.
- The portal backend must not write to the existing `fees` schema; fees/master data are synced read-only into `portal.synced_*`.

### backend-management

- Path: `backend-management/`
- Port: `8080`
- Purpose: schedule planning and time tracking
- Stack: Go, Chi, PostgreSQL, JWT
- Database: `kita` DB, `public` schema
- Entrypoints:
  - server: `cmd/server/main.go`
  - migrations: `cmd/migrate/main.go`
- Health endpoint: `GET /api/actuator/health`

Primary areas:
- `internal/api/`
- `internal/service/`
- `internal/repository/`
- `migrations/`

### backend-fees

- Path: `backend-fees/`
- Port: `8081`
- Purpose: fees management, import, matching
- Stack: Go, Chi, PostgreSQL, JWT
- Database: `kita` DB, `fees` schema
- Entrypoints:
  - server: `cmd/server/main.go`
  - migrations: `cmd/migrate/main.go`
- Health endpoint: `GET /health`

Primary areas:
- `internal/api/`
- `internal/service/`
- `internal/repository/`
- `migrations/`

### banking-sync

- Path: `banking-sync/`
- Purpose: banking/import sync service
- Built and pushed as separate GHCR image
- Main files: `server.js`, `sync.js`, `upload.js`

## Frontend Apps

All frontend apps are under `frontend/apps/*`.

### portal

- Path: `frontend/apps/portal/`
- Package: `@kita/portal`
- Dev port: `5176`
- Backend target: `backend-portal` on port `8082`
- Purpose: shared portal shell, login/reset screens, role-based module navigation, first parent-work UI surface

Current state:
- Login and password-reset screens exist.
- Auth store is wired to the implemented `/api/portal/v1/auth/login` backend endpoint and stores access token, refresh token, and user profile in local storage.
- Frontend logout calls `POST /api/portal/v1/auth/logout` before clearing local session state.
- Frontend refresh-token rotation and password-reset API integration are not implemented yet.
- Authenticated shell, dashboard, parent-work entry/history placeholders, review queue placeholder, and admin invitation placeholder exist.
- Module visibility is role-based in `src/lib/modules.ts`, and ready module routes are guarded with route-level role metadata.
- Ready UI modules: `parent_work`, `review`, `admin`.
- Planned UI modules shown as placeholders: `master_data`, `schedule`, `time_tracking`, `fees`.

### dienstplan

- Path: `frontend/apps/dienstplan/`
- Package: `@kita/dienstplan`
- Dev port: `5173`
- Status: development currently paused

### zeiterfassung

- Path: `frontend/apps/zeiterfassung/`
- Package: `@kita/zeiterfassung`
- Dev port: `5174`
- Status: development currently paused

### beitraege

- Path: `frontend/apps/beitraege/`
- Package: `@kita/beitraege`
- Dev port: `5175`
- Uses its own OpenAPI type generation from `openapi/fees/openapi3.yaml`

### shared package

- Path: `frontend/packages/shared/`
- Package: `@kita/shared`
- Shared components, composables, API client/types

## Local Development Commands

```bash
# Start local DB
cd docker && docker compose up db -d

# Start management backend
cd backend-management
go run cmd/migrate/main.go up
go run cmd/server/main.go

# Start fees backend
cd backend-fees
go run cmd/migrate/main.go up
go run cmd/server/main.go

# Start portal backend
cd backend-portal
go run cmd/migrate/main.go -direction up
go run cmd/server/main.go

# Start frontend apps
cd frontend
bun install
bun run dev:portal
bun run dev:plan
bun run dev:zeit
bun run dev:beitraege
```

## OpenAPI and Type Generation

```bash
# Management shared types
cd frontend/packages/shared
bun run generate:api

# Fees frontend types
cd frontend/apps/beitraege
bun run generate:api
```

Sources:
- `openapi/management/openapi3.yaml`
- `openapi/fees/openapi3.yaml`

## Deployment (Homelab)

Target environment: homelab VM `infra-dev`.

Use the homelab skill for deployment, logs, and runtime debugging.

### Required release flow

1. Commit and push changes.
2. Watch GitHub Actions until GHCR images are built.
3. Deploy from `../homelab` via Ansible playbook `deploy-app`.

```bash
# 1) Commit and push
cd /Users/stefan.remer/workspace/kita-apps
git add <files>
git commit -m "<message>"
git push

# 2) Watch build
gh run list -R steffbo/kita-apps --branch main --limit 5
gh run watch <run-id> -R steffbo/kita-apps

# Optional: watch latest run directly
gh run watch $(gh run list -R steffbo/kita-apps --branch main --limit 1 --json databaseId --jq '.[0].databaseId')

# 3) Deploy
cd /Users/stefan.remer/workspace/homelab/ansible
ansible-playbook playbooks/deploy-app.yml -e "app=kita"
```

Notes:
- Workflow file: `.github/workflows/build-images.yml`
- Trigger: pushes to `main`
- Portal backend/frontend Docker images, GHCR workflow entries, Caddy routing, and Docker Compose services are intentionally still pending. Add them when local portal development is complete and the portal is ready for VPS deployment.

## Direct Database Reads on infra-dev

Database is on `infra-dev` (`kita` DB, including `fees` schema).

### SSH + psql in container

```bash
ssh -i ~/.ssh/PVE_id_ed25519 stefan@192.168.188.207

# interactive psql
sudo docker exec -it kita-db psql -U kita -d kita

# one-off query (fees schema)
sudo docker exec -it kita-db psql -U kita -d kita -c "SET search_path TO fees; SELECT COUNT(*) FROM children;"
```

### Direct one-liners

```bash
# public schema
ssh -i ~/.ssh/PVE_id_ed25519 stefan@192.168.188.207 \
  "sudo docker exec kita-db psql -U kita -d kita -c 'SELECT COUNT(*) FROM employees;'"

# fees schema
ssh -i ~/.ssh/PVE_id_ed25519 stefan@192.168.188.207 \
  "sudo docker exec kita-db psql -U kita -d kita -c 'SELECT COUNT(*) FROM fees.children;'"
```

Safety:
- default to `SELECT`
- no write statements without explicit request
- qualify schema names in cross-schema analysis (`public.*`, `fees.*`)

## Code Navigation Shortcuts

- Business logic: `internal/service/*`
- API behavior: `internal/api/handler/*` + router setup
- Data model and SQL: `internal/repository/*` + `migrations/*`
- Frontend data flow: app-local `src/*` and `frontend/packages/shared/src/api/*`
