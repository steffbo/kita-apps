#!/bin/bash
# Regenerate the fees OpenAPI spec and the frontend TypeScript types from it.
#
# Pipeline (see backend-fees/README.md):
#   1. swag init            → openapi/fees/swagger.yaml      (Swagger 2.0 from Go annotations)
#   2. swagger2openapi      → openapi/fees/openapi3.yaml     (OpenAPI 3)
#   3. openapi-typescript   → frontend/apps/beitraege/src/api/schema.d.ts
#
# Requirements:
#   - swag CLI:  go install github.com/swaggo/swag/cmd/swag@latest
#     (expected at ~/go/bin/swag, override with SWAG=...)
#   - swagger2openapi: npx/bunx reachable, or a local binary via S2O=...
#     NOTE: the Artifactory npm mirror blocks this package — fetch it from
#     registry.npmjs.org directly, e.g.:
#       bun add swagger2openapi --registry https://registry.npmjs.org/
#
# Usage: scripts/generate-api.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SWAG="${SWAG:-$HOME/go/bin/swag}"

command -v "$SWAG" >/dev/null 2>&1 || {
  echo "error: swag CLI not found at '$SWAG'" >&2
  echo "       install with: go install github.com/swaggo/swag/cmd/swag@latest" >&2
  exit 1
}

run_s2o() {
  if [ -n "${S2O:-}" ]; then "$S2O" "$@"; return; fi
  if command -v swagger2openapi >/dev/null 2>&1; then swagger2openapi "$@"; return; fi
  if command -v bunx >/dev/null 2>&1; then bunx --registry https://registry.npmjs.org/ swagger2openapi "$@"; return; fi
  npx --registry=https://registry.npmjs.org/ swagger2openapi "$@"
}

echo "1/3  swag init (backend-fees annotations → swagger.yaml)"
(cd "$ROOT_DIR/backend-fees" && "$SWAG" init -g cmd/server/main.go -o ../openapi/fees --outputTypes yaml)

echo "2/3  swagger2openapi (swagger.yaml → openapi3.yaml)"
run_s2o "$ROOT_DIR/openapi/fees/swagger.yaml" -o "$ROOT_DIR/openapi/fees/openapi3.yaml"

echo "3/3  openapi-typescript (openapi3.yaml → schema.d.ts)"
(cd "$ROOT_DIR/frontend/apps/beitraege" && bun run generate:api)

echo "done."
