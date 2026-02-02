#!/bin/bash

set -e

echo "🏦 Banking Sync Service"
echo "======================"
echo ""

# Check if running via cron or direct
if [ -z "$SYNC_SCHEDULE" ]; then
  # No schedule = run once
  echo "⏰ Running once (no schedule set)"
  node sync.js
  exit $?
fi

# Setup cron
echo "⏰ Setting up cron schedule: $SYNC_SCHEDULE"
echo "$SYNC_SCHEDULE cd /app && node sync.js >> /var/log/banking-sync.log 2>&1" | crontab -

# Create log file
touch /var/log/banking-sync.log

echo "✅ Cron scheduled. Logs: /var/log/banking-sync.log"
echo "🚀 Starting cron daemon..."

# Run cron in foreground
exec cron -f
