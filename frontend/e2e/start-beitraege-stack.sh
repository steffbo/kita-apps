#!/usr/bin/env bash
# Starts a throwaway, production-like Beiträge stack for the Playwright suite:
#   1. PostgreSQL in a disposable Docker container (fresh DB every run)
#   2. Beiträge frontend built with Vite and embedded into backend-fees
#      (-tags embed_frontend, same origin and cookie paths as production)
#   3. migrations + backend-fees on $E2E_PORT, admin bootstrapped from
#      $E2E_ADMIN_EMAIL / $E2E_ADMIN_PASSWORD
# Playwright starts this as `webServer` and stops it after the run; the
# database container is removed on exit. Never points at a real database.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
E2E_PORT="${E2E_PORT:-18081}"
E2E_DB_PORT="${E2E_DB_PORT:-55432}"
DB_CONTAINER="kita-e2e-db-$E2E_PORT"
BUILD_DIR="$(mktemp -d)"
SERVER_PID=""

cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  docker rm -f "$DB_CONTAINER" >/dev/null 2>&1 || true
  rm -rf "$BUILD_DIR"
}
trap cleanup EXIT
trap 'exit 143' TERM INT

echo "[e2e] starting PostgreSQL ($DB_CONTAINER on :$E2E_DB_PORT)"
docker rm -f "$DB_CONTAINER" >/dev/null 2>&1 || true
docker run -d --rm --name "$DB_CONTAINER" \
  -e POSTGRES_USER=kita -e POSTGRES_PASSWORD=kita -e POSTGRES_DB=kita_e2e \
  -p "127.0.0.1:$E2E_DB_PORT:5432" postgres:16-alpine >/dev/null

echo "[e2e] building frontend"
(cd "$ROOT_DIR/frontend/apps/beitraege" && bun run vite build --logLevel warn)
EMBED_DIR="$ROOT_DIR/backend-fees/internal/frontend/beitraege"
find "$EMBED_DIR" -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
cp -r "$ROOT_DIR/frontend/apps/beitraege/dist/." "$EMBED_DIR/"

echo "[e2e] building backend"
cd "$ROOT_DIR/backend-fees"
go build -tags embed_frontend -o "$BUILD_DIR/server" ./cmd/server
go build -o "$BUILD_DIR/migrate" ./cmd/migrate

for _ in $(seq 1 60); do
  docker exec "$DB_CONTAINER" pg_isready -U kita -d kita_e2e >/dev/null 2>&1 && break
  sleep 0.5
done

export DATABASE_URL="postgres://kita:kita@127.0.0.1:$E2E_DB_PORT/kita_e2e?sslmode=disable&search_path=fees"
echo "[e2e] migrating"
"$BUILD_DIR/migrate" -direction up

LOG_FILE="${E2E_BACKEND_LOG:-$ROOT_DIR/frontend/test-results/e2e-backend.log}"
mkdir -p "$(dirname "$LOG_FILE")"
echo "[e2e] backend on http://127.0.0.1:$E2E_PORT (log: $LOG_FILE)"
PORT="$E2E_PORT" \
JWT_SECRET="e2e-secret-e2e-secret-e2e-secret-e2e" \
USER_NAME="${E2E_ADMIN_EMAIL:-admin@e2e.test}" \
USER_PASSWORD="${E2E_ADMIN_PASSWORD:-e2e-admin-password}" \
  "$BUILD_DIR/server" >"$LOG_FILE" 2>&1 &
SERVER_PID=$!
wait "$SERVER_PID"
