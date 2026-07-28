# AGENTS.md

Monorepo for the Kita Knirpsenstadt apps: Go backends, Vue frontends in a Bun workspace, a Bun banking-sync service, Docker/GHCR builds, homelab deployment.

## Working Rules

- **Docs are the handoff contract.** When behavior changes, update the matching doc in the same turn. Put durable facts (ports, commands, schemas, endpoints, env vars) here; put change history and non-obvious decisions in `docs/status-*.md`. Say explicitly when something is scaffolded but not wired.
- **Keep this file short.** It is an index, not a changelog. Detail goes into `docs/`.
- **Commit locally after each coherent increment**, including the doc updates, unless the user says otherwise. Descriptive messages. Do not push unless asked.
- **DB access defaults to read-only.** No write statements against live data without an explicit request.
- Mention in the final response which docs were updated.

## Layout

| Path | What |
| --- | --- |
| `backend-management/` | schedule + time tracking API |
| `backend-fees/` | fees API (Beiträge) |
| `backend-portal/` | portal API skeleton — see `docs/portal/STATUS.md` |
| `frontend/` | Bun workspace: `apps/*` + `packages/shared` |
| `banking-sync/` | bank CSV download/import service |
| `openapi/` | generated specs (`management`, `fees`) |
| `docker/` | compose files, Caddyfile, Dockerfiles |
| `scripts/` | helper scripts |
| `docs/` | deployment, status logs, portal spec set, archive |

Go services all follow: `cmd/server`, `cmd/migrate`, `internal/api` (handlers + router), `internal/service` (business logic), `internal/repository` (SQL), `migrations/`.

## Services

| Service | Port | DB schema | Health | Status |
| --- | --- | --- | --- | --- |
| `backend-management` | 8080 | `kita.public` | `GET /healthz` | in production, paused development |
| `backend-fees` | 8081 | `kita.fees` | `GET /health` | in production, actively developed |
| `backend-portal` | 8082 | `kita.portal` | `GET /health`, `GET /api/portal/v1/health` | skeleton, not deployed |
| `banking-sync` | — | — | — | in production (GHCR image, Ofelia-scheduled) |

| Frontend app | Package | Dev port | Status |
| --- | --- | --- | --- |
| `frontend/apps/dienstplan` | `@kita/dienstplan` | 5173 | paused |
| `frontend/apps/zeiterfassung` | `@kita/zeiterfassung` | 5174 | paused |
| `frontend/apps/beitraege` | `@kita/beitraege` | 5175 | actively developed |
| `frontend/apps/portal` | `@kita/portal` | 5176 | shell only, not deployed |

Boundaries: `backend-fees` is the source of truth for children/parents/households. `backend-portal` must not write to the `fees` schema; it reads snapshots into `portal.synced_*` (sync not implemented yet).

## Commands

```bash
# DB
cd docker && docker compose up db -d

# Backends (per service)
cd backend-management && go run cmd/migrate/main.go up && go run cmd/server/main.go
cd backend-fees       && go run cmd/migrate/main.go up && go run cmd/server/main.go
cd backend-portal     && go run cmd/migrate/main.go -direction up && go run cmd/server/main.go

# Tests
go test ./...            # inside a backend dir

# Frontend
cd frontend && bun install
bun run dev:plan | dev:zeit | dev:beitraege | dev:portal

# OpenAPI types
cd frontend/packages/shared && bun run generate:api   # from openapi/management/openapi3.yaml
cd frontend/apps/beitraege  && bun run generate:api   # from openapi/fees/openapi3.yaml
```

## Deployment

Target: homelab VM `infra-dev`. Flow: commit + push → GitHub Actions builds GHCR images (`.github/workflows/build-images.yml`, on `main`) → deploy from `../homelab`.

```bash
gh run watch $(gh run list -R steffbo/kita-apps --branch main --limit 1 --json databaseId --jq '.[0].databaseId')
cd /Users/stefan.remer/workspace/homelab/ansible && ansible-playbook playbooks/deploy-app.yml -e "app=kita"
```

Portal backend/frontend are intentionally not in Docker/GHCR/Caddy/Compose yet. Details: `docs/deployment-ghcr.md`, `docs/deployment-homelab.md`.

## Live Data on infra-dev

```bash
ssh -i ~/.ssh/PVE_id_ed25519 stefan@192.168.188.207 \
  "sudo docker exec kita-db psql -U kita -d kita -c 'SELECT COUNT(*) FROM fees.children;'"
```

Qualify schemas explicitly (`public.*`, `fees.*`, `portal.*`). `backend-fees` auth is still the single static admin identity from `USER_NAME` / `USER_PASSWORD`; there is no agent service account (`fees.users` was dropped in migration `000020`).

## Repository Skills (`.agents/skills/`)

- `kita-fees-einstufung`: case-neutral Einstufung workflow (evidence → annual income → contribution → explanation → authorized follow-ups). Uses current code + OpenAPI as the rules source.
- `kita-live-data-access`: API-first live-data workflow with read-only DB verification and secret-handling guardrails.

Both are family-data-neutral; re-check their code maps against the implementation before each calculation or mutation.

## Document Map

| Doc | Content |
| --- | --- |
| `README.md` | user-facing project overview and quickstart |
| `docs/status-fees.md` | Beiträge backend + frontend change log and decisions |
| `docs/status-banking-sync.md` | banking-sync behavior and operational notes |
| `docs/portal/STATUS.md` | **what actually exists in the portal** (verified against code) |
| `docs/portal/SPEC.md` | portal product/architecture spec (aspirational, largely unimplemented) |
| `docs/portal/DATA_MODEL.md` | portal Phase 0 data model sketch |
| `docs/portal/DATENSCHUTZ.md` | portal privacy notice draft, AVV review, role matrix |
| `docs/deployment-ghcr.md` | generic GHCR/Compose deployment guide |
| `docs/deployment-homelab.md` | kita.remer.cc / infra-dev deployment workflow |
| `docs/test-seeding.md` | e2e test DB and seeding setup |
| `docs/archive/` | superseded plans, kept for history only — do not treat as current |
