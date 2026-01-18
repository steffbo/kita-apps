#!/bin/bash
# Generate API code from OpenAPI spec

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🔧 Generating API code from OpenAPI spec..."

# Generate TypeScript types for frontend
echo "📝 Generating TypeScript types..."
cd "$ROOT_DIR/frontend"
bun run --filter @kita/shared generate:api

# Generate Spring Boot interfaces (done automatically during Maven build)
echo "☕ Backend interfaces will be generated during Maven build"
echo "   Run: cd backend && mvn compile"

echo "✅ API generation complete!"
