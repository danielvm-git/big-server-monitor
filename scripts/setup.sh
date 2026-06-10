#!/usr/bin/env bash
# PortKeeper — idempotent setup script.
# Run this before first launch or after a fresh clone.
# Safe to run multiple times — every step checks before acting.
set -euo pipefail

CONFIG_DIR="$HOME/.config/portkeeper"
CONFIG_FILE="$CONFIG_DIR/config.json"
LOG_FILE="$CONFIG_DIR/portkeeper.log"
DB_FILE="$CONFIG_DIR/activity.db"

echo "==> PortKeeper setup"
echo "    Config dir:  $CONFIG_DIR"
echo "    Config file: $CONFIG_FILE"
echo "    Log file:    $LOG_FILE"
echo "    DB file:     $DB_FILE"
echo ""

# 1. Create config directory (idempotent)
if [ ! -d "$CONFIG_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    echo "✓ Created $CONFIG_DIR"
else
    echo "• $CONFIG_DIR already exists, skipping"
fi

# 2. Create default config file if missing (idempotent)
if [ ! -f "$CONFIG_FILE" ]; then
    cat > "$CONFIG_FILE" <<'JSON'
{
  "scanDirectories": ["~/projects", "~/Developer", "~/opensrc"],
  "pollingIntervalSeconds": 5,
  "healthCheckIntervalSeconds": 30,
  "ignoredPorts": [80, 443, 5432, 3306, 6379, 27017],
  "logRetentionDays": 30,
  "notifications": {
    "crashAlerts": true,
    "showBadge": true
  },
  "launchAtLogin": false
}
JSON
    echo "✓ Created default $CONFIG_FILE"
else
    echo "• $CONFIG_FILE already exists, skipping"
fi

# 3. Create log file if missing (idempotent)
if [ ! -f "$LOG_FILE" ]; then
    touch "$LOG_FILE"
    echo "✓ Created $LOG_FILE"
else
    echo "• $LOG_FILE already exists, skipping"
fi

# 4. DB is created by activitylog component on first run — no action needed.
echo "• $DB_FILE will be created by the activitylog component on first run"

echo ""
echo "==> Setup complete. Run 'wails dev' to start."
