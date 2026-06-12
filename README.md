# BigServerMonitor

macOS menubar app for monitoring local development servers.

Built with Swift 6 / SwiftUI, MenuBarExtra, GRDB (SQLite), and structured JSON logging.

## Features

- **Process Monitor** — polls lsof every 5s, detects started/stopped servers on local ports
- **Health Check** — HTTP probes with status classification (ok/slow/warn/error/timeout)
- **Activity Log** — persisted event timeline (SQLite) with 30-day retention
- **Log Capture** — per-port ring buffer with Copy for AI export
- **Crash Notifications** — rate-limited UserNotifications when servers stop unexpectedly
- **Settings** — configurable intervals, ignored ports, launch-at-login

## Requirements

- macOS 14.0+
- Xcode 16+ (Swift 6)

## Quick Start

```bash
# Setup (idempotent, safe to run multiple times)
bash scripts/setup.sh

# Build
xcodebuild -project BigServerMonitor.xcodeproj -scheme BigServerMonitor build

# Run
open build/Debug/BigServerMonitor.app

# Test
xcodebuild test -project BigServerMonitor.xcodeproj -scheme BigServerMonitor -only-testing BigServerMonitorTests
```

## Architecture

```
Sources/
  App/    AppState (@Observable @MainActor), BigServerMonitorApp (MenuBarExtra)
  Core/   ProcessMonitor, HealthChecker, ActivityStore, LogCapture, Notifier,
          SettingsStore, JSONLogger, Models, PortDiscovery, ProjectDetection
  UI/     PopoverView, ServerRowView, HealthCheckSheet, ActivityLogSheet,
          LogsSheet, SettingsSheet, Brand
```

Actors own async state. AppState bridges actors to SwiftUI via `@Observable` properties.
No direct imports between Core/ components.

## Observability

| What | Path |
|------|------|
| App logs | `~/Library/Application Support/BigServerMonitor/bigservermonitor.log` |
| Config | `~/Library/Application Support/BigServerMonitor/config.json` |
| Activity DB | `~/Library/Application Support/BigServerMonitor/activity.db` |
