#!/bin/bash

set -e

echo "🏦 Banking Sync Service"
echo "======================"
echo ""

echo "⏰ Running once"
bun sync.js "$@"
