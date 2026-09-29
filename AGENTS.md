# AGENTS.md

Monorepo for the Kita Knirpsenstadt fee management (Beiträge): Go backend, Vue frontend in a Bun workspace, a Bun banking-sync service, GHCR builds, homelab deployment. Elternstunden stage 1 and parent access/approval backend stage 2a live in `backend-fees`; the parent UI and reminders are planned there, without a separate portal service.

## Working Rules

- **Docs are the handoff contract.** When behavior changes, update the matching doc in the same turn. Put durable facts (ports, commands, schemas, endpoints, env vars) here; put change history and non-obvious decisions in `docs/status-*.md`. Say explicitly when something is scaffolded but not wired.
- **Keep this file short.** It is an index, not a changelog. Detail goes into `docs/`.
- **Commit locally after each coherent increment**, including the doc updates, unless the user says otherwise. Descriptive messages. Do not push unless asked.
- **DB access defaults to read-only.** No write statements against live data without an explicit request.
- Mention in the final response which docs were updated.

## Layout

| Path | What |
| --- | --- |
| `backend-fees/` | fees API (Beiträge), Dockerfile for the production image |
| `frontend/` | Bun workspace: `apps/beitraege`, Playwright e2e in `e2e/` |
| `banking-sync/` | bank CSV download/import service |
| `openapi/fees/openapi3.yaml` | generated spec (only committed artifact of `scripts/generate-api.sh`) |
| `docker/docker-compose.yml` | local dev database only |
| `scripts/` | helper scripts |
| `docs/` | deployment, status logs, backlog |

Removed in 2026-09 (Dienstplan/Zeiterfassung `backend-management` + apps, portal skeleton `backend-portal` + app, `packages/shared`, legacy Docker/Caddy/OpenAPI files, `docs/portal`, `docs/archive`, finished plan docs): last state in git tag `archive/pre-cleanup-2026-09`. Do not restore pieces from there without a decision.

In `backend-fees`, fee amounts (brackets, tables, sibling factors, food/membership fee) are versioned data in `fees.fee_schedules` (UI: Beitragsordnung), not code; pick the version valid at the relevant date. Business dates follow Europe/Berlin: use `util.Today()` (Berlin date as UTC midnight, like scanned `DATE` columns) for date comparisons and `util.Now()` for year/month and timestamps — never bare `time.Now()`.

Go services all follow: `cmd/server`, `cmd/migrate`, `internal/api` (handlers + router), `internal/service` (business logic), `internal/repository` (SQL), `migrations/`.

## Roles and Elternstunden

Router groups by role: fee routes `ADMIN`/`USER`, `/users` `ADMIN` (incl. `POST /users/{id}/impersonate`, see `docs/status-fees.md`), `/parent-work/**` `ADMIN`/`PARENT_WORK`
(rules write `ADMIN`), `/me/**` `PARENT`, `/activity`, `/parent-reports/**`, `/parents/{id}/changes` and `/children/{id}/changes` `ADMIN`.
`PARENT` is linked by login email, uses `/familie/**`, and sees its own family (may edit the other parent's contact data, not their login email); linked contact and login emails stay synchronized. `PARENT_WORK` sees only `/beitraege/elternstunden/**`. Parent edits to contact and child data are audited in `fees.data_changes`; error reports live in `fees.parent_reports`.
Parent work is booked per household
(`fees.households`), stored in minutes (multiples of 15); rules (hours per child, rate per missing hour, carry-over cap)
are versioned data in `fees.parent_work_rules`, board terms in `fees.board_terms`. Calculation:
`domain.CalculateParentWork`. Business rules and history: `docs/status-fees.md`.

## Services

| Service | Port | DB schema | Health | Status |
| --- | --- | --- | --- | --- |
| `backend-fees` | 8081 | `kita.fees` | `GET /health` | in production, actively developed |
| `banking-sync` | — | — | — | in production (GHCR image, Ofelia-scheduled) |

| Frontend app | Package | Dev port | Status |
| --- | --- | --- | --- |
| `frontend/apps/beitraege` | `@kita/beitraege` | 5175 | in production at `kita.remer.cc/beitraege/`, embedded into the `backend-fees` image |

`backend-fees` is the source of truth for children, parents and households. The `kita` database has no other application schema; `public` is empty.

## Commands

```bash
# DB
cd docker && docker compose up db -d

# Backend
cd backend-fees && go run cmd/migrate/main.go && go run cmd/server/main.go
cd backend-fees && go test ./...    # integration tests use testcontainers (Docker)

# Frontend
cd frontend && bun install
bun run dev:beitraege
bun run --filter @kita/beitraege typecheck
bun run test:e2e         # Playwright, Beiträge; starts its own stack (Docker, Go, Bun) — docs/e2e-beitraege.md

# OpenAPI spec + frontend types (swag → swagger2openapi → schema.d.ts).
# Go annotations are the source of truth; rerun after API changes and commit
# both outputs — CI job `openapi-drift` fails on stale files.
scripts/generate-api.sh
```

## Deployment

Target: homelab VM `infra-dev`. Flow: commit + push → GitHub Actions runs CI (`.github/workflows/ci.yml`: gofmt/vet/test for `backend-fees` incl. testcontainers integration tests, typecheck for `beitraege`, OpenAPI drift check, Playwright e2e for `beitraege`; also on PRs) and only then builds GHCR images (`.github/workflows/build-images.yml`, on `main`) → deploy from `../homelab`. A red CI means no new image.

```bash
expected_sha=$(git rev-parse HEAD)
run_id=$(gh run list -R steffbo/kita-apps --branch main --limit 20 --json databaseId,headSha \
  --jq ".[] | select(.headSha == \"$expected_sha\") | .databaseId" | head -n1)
test -n "$run_id"
gh run watch "$run_id" -R steffbo/kita-apps --exit-status

cd /home/stefan/workspace/homelab/ansible
ansible-playbook playbooks/deploy-app.yml -e "app=kita" \
  -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"
```

Do not deploy after a fixed sleep or merely watch the newest workflow run: pin the run to the pushed commit and require success. After deployment, verify the container OCI revision, `https://kita.remer.cc/health`, and the affected public page. An interrupted deploy has unknown state until inspected; verify before retrying.

Routing lives in the homelab repo (`stacks/edge/caddy/Caddyfile`): `/beitraege/*`, `/api/fees/*`, `/health` → `backend-fees`; everything else redirects to `/beitraege/`. Details: `docs/deployment-ghcr.md`, `docs/deployment-homelab.md`.

## Live Data on infra-dev

```bash
ssh vm-infra-dev \
  "sudo docker exec kita-db psql -U kita -d kita -c 'SELECT COUNT(*) FROM fees.children;'"
```

Qualify schemas explicitly (`fees.*`). `backend-fees` users live in `fees.users` (migration `000036`, bcrypt hashes, roles `ADMIN`/`USER`/`PARENT_WORK`/`PARENT`, managed on the "Benutzer" page). `USER_NAME` / `USER_PASSWORD` only bootstrap the admin when no account with that email exists (same ID as the former static admin, `a0eebc99-…`); they never overwrite it. There is no agent service account yet.

Note on the kita stack `.env` (`/srv/homelab/stacks/infra-dev/apps/kita/.env` on infra-dev): values with special characters (e.g. `USER_PASSWORD`) are wrapped in single quotes. Docker Compose strips the quotes when passing them into containers, so login works — but when reading the file manually (scripts, shell parsing), strip surrounding `'` yourself or authentication will fail.

## Repository Skills (`.agents/skills/`)

- `kita-fees-einstufung`: case-neutral Einstufung workflow (evidence → annual income → contribution → explanation → authorized follow-ups). Uses current code + OpenAPI as the rules source.
- `kita-live-data-access`: API-first live-data workflow with read-only DB verification and secret-handling guardrails.

Both are family-data-neutral; re-check their code maps against the implementation before each calculation or mutation.

## Document Map

| Doc | Content |
| --- | --- |
| `README.md` | user-facing project overview and quickstart |
| `docs/status-fees.md` | Beiträge backend + frontend change log and decisions |
| `docs/todo-fees-improvements.md` | ordered Beiträge improvement backlog (worked top to bottom) |
| `docs/status-banking-sync.md` | banking-sync behavior and operational notes |
| `docs/deployment-ghcr.md` | generic GHCR/Compose deployment guide |
| `docs/deployment-homelab.md` | kita.remer.cc / infra-dev deployment workflow |
| `docs/e2e-beitraege.md` | Playwright e2e suite for Beiträge (disposable stack, test rules) |
