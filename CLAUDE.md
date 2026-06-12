# BigServerMonitor — Claude Code

Read CONVENTIONS.md before any git operation.

## Project
macOS menubar app for monitoring local development servers.
Native Swift 6 / SwiftUI, MenuBarExtra with LSUIElement (no dock icon).

## Commands
| Action | Command |
|--------|---------|
| Build  | `xcodebuild -project BigServerMonitor.xcodeproj -scheme BigServerMonitor build` |
| Test   | `xcodebuild test -project BigServerMonitor.xcodeproj -scheme BigServerMonitor -only-testing BigServerMonitorTests` |
| Run    | `open build/Debug/BigServerMonitor.app` or open from Xcode |
| Lint   | `swiftlint` (if installed) |
| Regen project | `xcodegen --spec project.yml` |

## Architecture
Swift 6 with actors, AsyncStream, @Observable, SwiftUI, GRDB (SQLite).
```
Sources/
  App/    AppState (@Observable @MainActor), BigServerMonitorApp (MenuBarExtra)
  Core/   ProcessMonitor, HealthChecker, ActivityStore, LogCapture, Notifier,
          SettingsStore, JSONLogger, Models, PortDiscovery, ProjectDetection
  UI/     PopoverView, ServerRowView, HealthCheckSheet, ActivityLogSheet,
          LogsSheet, SettingsSheet, Brand
```
Components communicate via AppState bridging — actors publish to @Observable,
UI reads @Observable. No direct imports between core components.

## Observability
All logs are structured JSON via JSONLogger actor. Log entries include
`time`, `level`, `msg`, and arbitrary key-value context fields.

| What | Command |
|------|---------|
| Run full test suite | `xcodebuild test -project BigServerMonitor.xcodeproj -scheme BigServerMonitor -only-testing BigServerMonitorTests` |
| View app logs | `cat ~/Library/Application\ Support/BigServerMonitor/bigservermonitor.log` |
| View errors only | `grep '"level":"error"' ~/Library/Application\ Support/BigServerMonitor/bigservermonitor.log` |
| Check config | `cat ~/Library/Application\ Support/BigServerMonitor/config.json` |
| Check DB | `ls ~/Library/Application\ Support/BigServerMonitor/activity.db` |
| Setup (idempotent) | `bash scripts/setup.sh` |
| Production build | Archive from Xcode or `xcodebuild -configuration Release build` |

## Never
- Hardcode file paths or user home directories
- Commit to main without PR
- Write between component directories
- Expose internal errors or stack traces to the UI
- Use XCTest — tests use Swift Testing (`@Suite`, `@Test`, `#expect`)

## Agent Rules
- Read specs/ before writing code
- All planning output goes to specs/
- Write minimum code that solves the stated problem
- Run tests after every change
- Regenerate xcodeproj (`xcodegen --spec project.yml`) after adding files
