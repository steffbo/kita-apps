# Status: banking-sync

Bank CSV download/import service (`banking-sync/`, Bun + Playwright), built and pushed as a separate GHCR image.
Main files: `server.js` (Runner API), `sync.js` (download/upload flow), `upload.js`.

## Download retries (2026-07-08)

- Retries live in `sync.js` and are used by both `bun sync.js` and the Runner API in `server.js`.
- `DOWNLOAD_TIMEOUT_SECONDS` (default `120`) bounds the Playwright `download` event after clicking the export button.
- `DOWNLOAD_RETRY_ATTEMPTS` (default `3`) restarts the full browser flow with a fresh session per attempt; `DOWNLOAD_RETRY_DELAY_SECONDS` defaults to `30`.

## Monitoring

- Uptime Kuma pings append `status=up` on successful upload and send a best-effort `status=down` on sync failure.
- Runtime Compose must pass `UPTIME_KUMA_PUSH_URL` into `banking-sync-runner`; otherwise the runner logs `UPTIME_KUMA_PUSH_URL not set` once and sends nothing.

## Operational notes

- Docker base image is `oven/bun:1` after repeated Bun 1.1.29 segmentation faults in the long-running runner on `infra-dev` (2026-07-08).
- `infra-dev` logs showed Ofelia job failures from scheduler/runner coordination (`409 run already in progress`) or killed scheduler exec processes (`exit code 137`) while the runner itself completed download and upload. Check `kita-banking-sync-runner` logs and `/srv/homelab-data/kita/banking-sync/state/status.json` before assuming an import failed.
