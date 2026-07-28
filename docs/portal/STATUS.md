# Portal: Implementation Status

Stand: 2026-07-28 (verified against code, not chat history)

Related docs: `SPEC.md` (product/architecture spec), `DATA_MODEL.md` (Phase 0 data model), `DATENSCHUTZ.md` (privacy/provider/RBAC).

## TL;DR

The portal is an **abandoned Phase-1 skeleton**, not a partially working product.

- All portal code was written in a **single day, 2026-05-25**, across 4 commits, then never touched again.
- What works: portal DB schema, JWT login/refresh/me/logout, a Vue shell with role-based navigation and route guards.
- What does not exist: every business feature. No invitations, no onboarding, no password reset, no master-data sync, no parent-work API, no email, no audit writes, no deployment.
- Size: ~1.6k lines Go, ~0.9k lines Vue/TS. The parent-work UI pages are static mockups with no backend behind them.
- The spec's August-2026 launch scope is not achievable from this base without essentially building Phases 1–4 from scratch.

## Commit history (complete)

| Commit | Date | Content |
| --- | --- | --- |
| `379ee72` | 2026-05-25 | Portal foundation + auth checkpoint (backend skeleton, schema, frontend shell) |
| `1fe2763` | 2026-05-25 | `PORTAL_DATA_MODEL.md` (docs only) |
| `b2b1e5f` | 2026-05-25 | Fix refresh-token rotation + migration metadata table |
| `b1144ed` | 2026-05-25 | Route guards + logout session revocation |

Plus docs-only commits `a3b2b03` (scope freeze) and `1c2b96e` (Datenschutz). No portal commit after 2026-05-25.

## Backend (`backend-portal/`, port 8082)

Builds and tests pass (`go build ./...`, `go test ./...`).

Implemented:

- Migration `000001_initial_schema` creates the full `portal` schema: `users`, `user_roles`, `invitations`, `password_reset_tokens`, `refresh_tokens`, `synced_households`, `synced_parents`, `synced_children`, `synced_child_parents`, `user_households`, `sync_runs`, `sync_quarantine_items`, `parent_work_requirements`, `parent_work_entries`, `email_outbox`, `audit_events`.
- Migration bookkeeping lives in `public.portal_schema_migrations` so `down` can drop the `portal` schema.
- Auth endpoints: `POST /api/portal/v1/auth/login`, `POST .../refresh`, `GET .../me`, `POST .../logout`, plus `GET /health` and `GET /api/portal/v1/health`.
- bcrypt hashes, JWT access (15m) / refresh (168h), hashed refresh tokens persisted in `portal.refresh_tokens`, atomic rotation on refresh, full session revoke on logout.
- Optional first-user bootstrap via `PORTAL_BOOTSTRAP_ADMIN_EMAIL` / `PORTAL_BOOTSTRAP_ADMIN_PASSWORD` (never commit a default password).
- Pure rule function `RequiredParentWorkHours` in `internal/service/parent_work_rules.go` (9 h/year, 3 h per started Kita-Tertial, override + exemption), unit-tested. **Not reachable via any API.**

Missing (of ~16 tables, only 3 are used by code):

- Invitation create/revoke/redeem, password onboarding, password reset (tables exist, no code).
- Master-data sync from `backend-fees` into `portal.synced_*`, sync runs, quarantine handling.
- Parent-work CRUD, submission, approval/rejection, requirements, year overview.
- Email outbox / Resend integration, reminders.
- Audit event writing (table exists, nothing writes to it).
- Rate limiting, account lockout, password breach check (spec requires them).
- No OpenAPI spec, no generated frontend types (unlike `fees`/`management`).

## Frontend (`frontend/apps/portal/`, dev port 5176)

- Vite base `/portal/`, dev proxy `/api/portal` → `localhost:8082`.
- Real backend integration: login (`stores/auth.ts` → `POST /auth/login`) and logout. Tokens + user are stored in `localStorage`.
- Refresh-token rotation is **not** used by the frontend, so sessions die when the 15m access token expires.
- `PasswordResetPage.vue` only flips a local `isSubmitted` flag — no API call.
- `ParentWorkPage.vue` (entry form, stats `0,0 h`), `ReviewQueuePage.vue`, `AdminPage.vue` (invite form) are static mockups; none submit anywhere.
- `DashboardPage.vue` + `lib/modules.ts` do role-filtered module tiles; router guards check `meta.roles`.
- Modules marked `ready` (`parent_work`, `review`, `admin`) are ready as *UI shells only*; `master_data`, `schedule`, `time_tracking`, `fees` are `planned` placeholders pointing at the dashboard.

## Ops / deployment

- No Dockerfile, no GHCR workflow entry, no Compose service, no Caddy route for portal backend or frontend.
- `frontend/package.json` has `dev:portal` and `build:portal` scripts; that is the only wiring outside the app folder.
- Nothing portal-related is deployed on `infra-dev` or anywhere else.

## Spec vs. reality

| Spec phase | Claimed status in spec | Actual |
| --- | --- | --- |
| Phase 0 Spec/Architecture freeze | done | done (docs only) |
| Phase 1 Portal foundation | — | ~40%: login/session yes; invite, onboarding, reset, Resend, admin UI missing |
| Phase 2 Parent work MVP | — | ~5%: schema + one rule function; no API, no UI logic |
| Phase 3 Homelab staging / Vorstand demo | — | 0% |
| Phase 4 VPS production | — | 0% |

The spec's `Status` table still says "Ziel-Launch August 2026" and "Phase 0 eingefroren". Treat those as historical intent, not as a plan in progress.

## If work resumes: decision points first

1. **Keep or drop the spec.** It was produced by a "grill me relentlessly" session and is long, German/English mixed, and over-specified for what is a single-developer side project. Decide whether the August-2026 framing and the Hetzner-VPS/Verein-handover track are still wanted.
2. **Separate service or module?** A 4th Go backend plus a 4th frontend app increases operational surface. Consider folding parent-work into `backend-fees` (which already owns children/parents/households) instead of maintaining a read-only sync layer, quarantine tables and duplicate identity.
3. **If keeping `backend-portal`,** the shortest path to something usable is: invitation + onboarding + password reset (with Resend), then master-data sync, then parent-work API, then wire the three mockup pages. Deployment (Dockerfile/GHCR/Caddy/Compose) can piggyback on the existing homelab stack before considering a separate VPS.
4. **Refresh-token handling in the frontend** must be fixed before anyone tests the shell seriously; currently a session silently breaks after 15 minutes.
