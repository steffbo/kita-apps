# AGENTS.md

## Scope

Monorepo for Kita apps with:
- Go backends
- Vue frontends in a Bun workspace
- a separate banking sync service
- Docker/GHCR build and homelab deployment

## Repository Layout

- `backend-management/`: management API (schedule + time tracking)
- `backend-fees/`: fees API
- `frontend/`: workspace with frontend apps and shared package
- `banking-sync/`: sync/import service
- `openapi/`: generated OpenAPI specs (`management`, `fees`)
- `docker/`: compose files and Docker definitions
- `scripts/`: helper scripts

## Backend Services

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

# Start frontend apps
cd frontend
bun install
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
