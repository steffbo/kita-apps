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
- `PORTAL_DATENSCHUTZ.md`: Phase 0 Datenschutz/provider/RBAC handoff with portal privacy notice draft, Hetzner/Resend AVV review, role matrix, and provider-account target state

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
- Phase 0 Datenschutz/provider/RBAC handoff exists in `PORTAL_DATENSCHUTZ.md` as of 2026-05-25. It documents the portal privacy notice draft, the Hetzner and Resend AVV/DPA provider review based on official sources, the role/permission matrix, and the target state that provider accounts/workspaces are assigned to the Verein/Kita context.
- Before parent rollout, the remaining Datenschutz operations are account-level: accept/export the Hetzner AVV and Resend DPA/Terms/subprocessor evidence in the correct Verein/Kita accounts, store those proofs outside Git in the Vereinsablage, finalize privacy-contact placeholders, keep Resend mail tracking disabled, and document/complete provider account handover to the Verein.
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
- Health endpoint: `GET /healthz`

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

Current state:
- Einstufung history with monthly cut-off follow-ups is implemented as of 2026-06-06.
- Migration `backend-fees/migrations/000026_einstufung_periods.up.sql` adds `valid_until`, `source_einstufung_id`, `change_date`, and `effective_from_month` to `fees.einstufungen`, removes the old one-einstufung-per-child/year uniqueness, and enforces non-overlapping child periods with a GiST exclusion constraint.
- Follow-up endpoint: `POST /api/fees/v1/einstufungen/{id}/follow-ups`. The request stores the entered `changeDate`; the API calculates `effectiveFromMonth` with the rule 1st-14th = same month and 15th+ = following month.
- Creating a follow-up closes the source Einstufung at the day before `effectiveFromMonth`, creates the new Einstufung with `annual_membership_fee = 0`, and synchronizes `CHILDCARE` fee expectations from the effective month through year-end or child exit date.
- The Beiträge frontend PDF loads the source Einstufung for follow-ups and prepends the previous month with the old monthly contribution when that month still belongs to the source period. If a follow-up shares the first Einstufung's effective month, no previous-month column is shown.
- CHILDCARE sync creates missing expectations, updates unmatched expectations, increases already matched expectations when no overpayment results, and reports decreases below matched amounts in `creditReviewRequired` without creating automatic credits. `FOOD` and `MEMBERSHIP` expectations are not changed.
- Einstufung income calculation was corrected on 2026-06-06 to treat all entered income components as fee-relevant unless modeled explicitly as deductions. `Basiselterngeld`, `Elterngeld Plus`, and `Mutterschaftsgeld` are fully fee-relevant. The income JSON also breaks out `minijobIncome`, `unemploymentBenefit`, `capitalIncome`, and `rentalIncome`; `otherIncome` remains for residual other income. Existing stored JSON without these fields reads as zero.
- Household consistency for child-parent links was corrected and guarded on 2026-06-10. Production data on `infra-dev` was cleaned for split Thränhardt/Tränhardt and Zeck households: inactive sibling records were moved into the active family household, duplicate empty households were deleted, and the global mismatch check returned `0` rows afterward. Migration `backend-fees/migrations/000027_validate_child_parent_households.up.sql` adds deferrable constraint triggers that reject `fees.child_parents` links, `fees.children.household_id` updates, and `fees.parents.household_id` updates when linked children and parents would belong to different non-null households. The same trigger SQL was applied manually to the production DB on 2026-06-10 before the code deployment, so future migrations may reapply it idempotently.
- `backend-fees/internal/service/child_service.go` now prevents silent household splits when linking parents: if the child has no household but the parent does, the child joins the existing parent household; if both already have different households, the service returns `ErrHouseholdMismatch` and the API responds with HTTP 409. Imports that link an existing parent from another household now fail that row instead of creating inconsistent data.
- New/changed OpenAPI fields and endpoint are generated in `openapi/fees/swagger.yaml` and `openapi/fees/openapi3.yaml`; `frontend/apps/beitraege/src/api/schema.d.ts` was regenerated from the OpenAPI 3 spec.
- Regression coverage: `go test ./...` in `backend-fees` includes cut-off tests for `2026-06-14` and `2026-06-15`, sync tests for paid increases and credit-review decreases, and household consistency tests for parent-link service behavior plus the DB trigger.

Household consistency audit:

```sql
SELECT c.id, c.first_name, c.last_name,
       c.household_id AS child_household_id,
       p.id AS parent_id, p.first_name, p.last_name,
       p.household_id AS parent_household_id
FROM fees.child_parents cp
JOIN fees.children c ON c.id = cp.child_id
JOIN fees.parents p ON p.id = cp.parent_id
WHERE c.household_id IS NOT NULL
  AND p.household_id IS NOT NULL
  AND c.household_id <> p.household_id;
```

### banking-sync

- Path: `banking-sync/`
- Purpose: banking/import sync service
- Built and pushed as separate GHCR image
- Main files: `server.js`, `sync.js`, `upload.js`

Current state:
- As of 2026-07-08, bank CSV download retries are implemented in `banking-sync/sync.js` and used by both `bun sync.js` and the Runner API in `banking-sync/server.js`.
- `DOWNLOAD_TIMEOUT_SECONDS` defaults to `120` and bounds the Playwright `download` event after clicking the export button.
- `DOWNLOAD_RETRY_ATTEMPTS` defaults to `3`; each retry restarts the full browser download flow with a fresh browser session. `DOWNLOAD_RETRY_DELAY_SECONDS` defaults to `30`.
- Uptime Kuma push handling now appends `status=up` for successful uploads and sends a best-effort `status=down` ping on sync failure. Runtime Compose must pass `UPTIME_KUMA_PUSH_URL` into `banking-sync-runner`; otherwise the runner logs `UPTIME_KUMA_PUSH_URL not set` once and sends no ping.
- The Docker base image is `oven/bun:1` after `infra-dev` logs on 2026-07-08 showed repeated Bun 1.1.29 segmentation faults in the long-running runner.
- Recent `infra-dev` logs also showed several daily Ofelia job failures caused by scheduler/runner coordination (`409 run already in progress`) or killed scheduler exec processes (`exit code 137`) while the underlying runner often completed the CSV download and API upload successfully. Check `kita-banking-sync-runner` logs and `/srv/homelab-data/kita/banking-sync/state/status.json` before assuming an import failed.

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

Current state:
- Mobile layout guardrails were added on 2026-06-26. `frontend/apps/beitraege/src/layouts/MainLayout.vue` prevents root horizontal overflow, uses tighter mobile content padding, and opens the mobile navigation drawer from the right to match the menu button position.
- `frontend/apps/beitraege/src/assets/main.css` constrains app-wide horizontal overflow and adds touch-friendly internal scrolling/compact cell spacing for wide table containers on small screens.
- `frontend/apps/beitraege/src/pages/ImportPage.vue` now wraps its import-history, unmatched, blacklist, and matched tables in internal horizontal scroll containers and makes the import tab bar scroll internally on mobile. If another Beitrags page still shifts the whole viewport sideways, first check for raw `<table>` markup or unbounded tab/filter rows that are missing an `overflow-x-auto` container.

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
