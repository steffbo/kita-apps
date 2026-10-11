#!/usr/bin/env bash
# Local Beiträge stack on a temporary copy of the live database, for testing
# changes with real data before a deployment.
#
#   scripts/live-copy.sh up        copy live DB, build current checkout, start
#   scripts/live-copy.sh restart   rebuild + migrate + restart, keep copied data
#   scripts/live-copy.sh refresh   fresh copy of the live DB, then restart
#   scripts/live-copy.sh down      stop server and delete the copy
#   scripts/live-copy.sh status    show what is running
#
# The live database is only read (pg_dump over SSH). The copy lives in a
# throwaway container without a volume; `down` removes it. Migrations of the
# current checkout run against the copy. SMTP and banking-sync are not
# configured, so the copy never sends mail or talks to the bank.
# Log in with the normal live account. The JWT secret is kept in the state
# directory, so sessions survive `restart`.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${LIVE_COPY_PORT:-18090}"
DB_PORT="${LIVE_COPY_DB_PORT:-55490}"
DB_CONTAINER="kita-live-copy-db"
SSH_HOST="${LIVE_COPY_SSH_HOST:-vm-infra-dev}"
STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/kita-live-copy"
PID_FILE="$STATE_DIR/server.pid"
LOG_FILE="$STATE_DIR/server.log"
DATABASE_URL="postgres://kita:kita@127.0.0.1:$DB_PORT/kita?sslmode=disable&search_path=fees"

mkdir -p "$STATE_DIR"
chmod 700 "$STATE_DIR"

log() { echo "[live-copy] $*"; }

stop_server() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    kill "$(cat "$PID_FILE")"
    for _ in $(seq 1 50); do kill -0 "$(cat "$PID_FILE")" 2>/dev/null || break; sleep 0.2; done
    log "server stopped"
  fi
  rm -f "$PID_FILE"
}

start_db() {
  if docker ps --format '{{.Names}}' | grep -qx "$DB_CONTAINER"; then
    return
  fi
  log "starting PostgreSQL ($DB_CONTAINER on 127.0.0.1:$DB_PORT)"
  docker run -d --rm --name "$DB_CONTAINER" \
    -e POSTGRES_USER=kita -e POSTGRES_PASSWORD=kita -e POSTGRES_DB=kita \
    -p "127.0.0.1:$DB_PORT:5432" postgres:16-alpine >/dev/null
  for _ in $(seq 1 60); do
    docker exec "$DB_CONTAINER" pg_isready -U kita -d kita >/dev/null 2>&1 && return
    sleep 0.5
  done
  log "database did not become ready"; exit 1
}

copy_live_data() {
  log "copying live database from $SSH_HOST (read-only pg_dump)"
  docker exec "$DB_CONTAINER" psql -U kita -d postgres -qc 'DROP DATABASE IF EXISTS kita WITH (FORCE)' \
    -c 'CREATE DATABASE kita' >/dev/null
  ssh "$SSH_HOST" "sudo docker exec kita-db pg_dump -U kita -d kita -Fc" \
    | docker exec -i "$DB_CONTAINER" pg_restore -U kita -d kita --no-owner --exit-on-error
  log "copied: $(docker exec "$DB_CONTAINER" psql -U kita -d kita -Atc \
    "SELECT count(*) || ' children, ' || (SELECT count(*) FROM fees.einstufungen) || ' Einstufungen' FROM fees.children")"
}

build_and_start() {
  stop_server
  log "building frontend"
  (cd "$ROOT_DIR/frontend/apps/beitraege" && bun run vite build --logLevel warn)
  local embed_dir="$ROOT_DIR/backend-fees/internal/frontend/beitraege"
  find "$embed_dir" -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
  cp -r "$ROOT_DIR/frontend/apps/beitraege/dist/." "$embed_dir/"

  log "building backend"
  (cd "$ROOT_DIR/backend-fees" &&
    go build -tags embed_frontend -o "$STATE_DIR/server" ./cmd/server &&
    go build -o "$STATE_DIR/migrate" ./cmd/migrate)

  log "migrating the copy"
  (cd "$ROOT_DIR/backend-fees" && DATABASE_URL="$DATABASE_URL" "$STATE_DIR/migrate" -direction up)

  if [ ! -s "$STATE_DIR/jwt-secret" ]; then
    (umask 077 && head -c 48 /dev/urandom | base64 >"$STATE_DIR/jwt-secret")
  fi
  # env -i: no SMTP_*, BANKING_SYNC_* or USER_* from the calling shell.
  setsid env -i PATH="$PATH" HOME="$HOME" \
    PORT="$PORT" DATABASE_URL="$DATABASE_URL" JWT_SECRET="$(cat "$STATE_DIR/jwt-secret")" \
    APP_BASE_URL="http://$(lan_ip):$PORT/beitraege" \
    "$STATE_DIR/server" >"$LOG_FILE" 2>&1 </dev/null &
  echo $! >"$PID_FILE"
  for _ in $(seq 1 50); do
    curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && break
    sleep 0.2
  done
  curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null || { log "server did not start, see $LOG_FILE"; exit 1; }
  log "running: http://$(lan_ip):$PORT/beitraege/ (log: $LOG_FILE)"
}

lan_ip() {
  ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i = 1; i < NF; i++) if ($i == "src") print $(i + 1)}'
}

case "${1:-}" in
  up)
    start_db
    copy_live_data
    build_and_start
    ;;
  restart)
    start_db
    build_and_start
    ;;
  refresh)
    stop_server
    start_db
    copy_live_data
    build_and_start
    ;;
  down)
    stop_server
    docker rm -f "$DB_CONTAINER" >/dev/null 2>&1 && log "copy deleted" || true
    ;;
  status)
    if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
      log "server running: http://$(lan_ip):$PORT/beitraege/"
    else
      log "server not running"
    fi
    docker ps --filter "name=$DB_CONTAINER" --format '[live-copy] database: {{.Names}} {{.Status}}'
    ;;
  *)
    sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
    exit 1
    ;;
esac
