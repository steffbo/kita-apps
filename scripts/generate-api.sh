#!/bin/bash
# Regenerate the fees OpenAPI spec and the frontend TypeScript types from it.
#
# Pipeline (see backend-fees/README.md):
#   1. swag init            → swagger.yaml in a temp dir     (Swagger 2.0 from Go annotations, not committed)
#   2. swagger2openapi      → openapi/fees/openapi3.yaml     (OpenAPI 3, committed)
#   3. openapi-typescript   → frontend/apps/beitraege/src/api/schema.d.ts
#
# Requirements (versions pinned below as SWAG_VERSION / S2O_VERSION, so CI job
# `openapi-drift` in ci.yml and local runs produce byte-identical output;
# bump them here and in ci.yml together):
#   - swag CLI:  go install github.com/swaggo/swag/cmd/swag@v1.16.6
#     (expected at ~/go/bin/swag, override with SWAG=...)
#   - swagger2openapi: fetched via bunx/npx, or a local binary via S2O=...
#     NOTE: the Artifactory npm mirror blocks this package, so it is fetched
#     from registry.npmjs.org directly.
#   - openapi-typescript: pinned in frontend/apps/beitraege/package.json
#     (needs `bun install` in frontend/).
#
# Usage: scripts/generate-api.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SWAG="${SWAG:-$HOME/go/bin/swag}"
SWAG_VERSION="v1.16.6"
S2O_VERSION="7.0.8"

command -v "$SWAG" >/dev/null 2>&1 || {
  echo "error: swag CLI not found at '$SWAG'" >&2
  echo "       install with: go install github.com/swaggo/swag/cmd/swag@$SWAG_VERSION" >&2
  exit 1
}
# `swag --version` is unreliable (v1.16.6 still reports v1.16.4), so read the
# module version from the binary's build info.
swag_mod_version="$(go version -m "$SWAG" 2>/dev/null | awk '$1 == "mod" { print $3 }' || true)"
[ "$swag_mod_version" = "$SWAG_VERSION" ] || {
  echo "error: expected swag $SWAG_VERSION, got: ${swag_mod_version:-unknown}" >&2
  echo "       install with: go install github.com/swaggo/swag/cmd/swag@$SWAG_VERSION" >&2
  exit 1
}

run_s2o() {
  if [ -n "${S2O:-}" ]; then "$S2O" "$@"; return; fi
  if command -v bunx >/dev/null 2>&1; then BUN_CONFIG_REGISTRY=https://registry.npmjs.org/ bunx "swagger2openapi@$S2O_VERSION" "$@"; return; fi
  npx --registry=https://registry.npmjs.org/ "swagger2openapi@$S2O_VERSION" "$@"
}

SWAGGER_DIR="$(mktemp -d)"
trap 'rm -rf "$SWAGGER_DIR"' EXIT

echo "1/3  swag init (backend-fees annotations → swagger.yaml)"
(cd "$ROOT_DIR/backend-fees" && "$SWAG" init -g cmd/server/main.go -o "$SWAGGER_DIR" --outputTypes yaml --requiredByDefault)

echo "2/3  swagger2openapi (swagger.yaml → openapi3.yaml)"
run_s2o "$SWAGGER_DIR/swagger.yaml" -o "$ROOT_DIR/openapi/fees/openapi3.yaml"

echo "3/3  openapi-typescript (openapi3.yaml → schema.d.ts)"
(cd "$ROOT_DIR/frontend/apps/beitraege" && bun run generate:api)

echo "done."
