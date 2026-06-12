#!/usr/bin/env bash
# BigServerMonitor — idempotent setup script.
# Run this before first launch or after a fresh clone.
# Safe to run multiple times — every step checks before acting.
set -euo pipefail

APP_SUPPORT="$HOME/Library/Application Support/BigServerMonitor"
CONFIG_FILE="$APP_SUPPORT/config.json"
LOG_FILE="$APP_SUPPORT/bigservermonitor.log"

echo "==> BigServerMonitor setup"
echo "    App Support: $APP_SUPPORT"
echo "    Config:      $CONFIG_FILE"
echo "    Log:         $LOG_FILE"
echo ""

# 1. Create Application Support directory (idempotent)
if [ ! -d "$APP_SUPPORT" ]; then
    mkdir -p "$APP_SUPPORT"
    echo "✓ Created $APP_SUPPORT"
else
    echo "• $APP_SUPPORT already exists, skipping"
fi

# 2. Generate Xcode project (idempotent)
if command -v xcodegen &> /dev/null; then
    xcodegen --spec project.yml --quiet 2>/dev/null || xcodegen --spec project.yml
    echo "✓ Xcode project generated"
else
    echo "⚠ xcodegen not found — install with: brew install xcodegen"
fi

# 3. Build (idempotent — Xcode handles incremental builds)
xcodebuild -project BigServerMonitor.xcodeproj -scheme BigServerMonitor build 2>&1 | tail -1
echo "✓ Build complete"

# 4. DB and config are created by the app on first run — no action needed.
echo "• App will create $CONFIG_FILE and $LOG_FILE on first launch"

echo ""
echo "==> Setup complete. Run: open build/Debug/BigServerMonitor.app"
