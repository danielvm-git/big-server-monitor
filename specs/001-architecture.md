# Slice 1: Architecture — "BigBase ECC for macOS"

**type:** architecture  
**status:** planning  
**verify:** `go build -o portkeeper .` succeeds and kernel registers all components

## Overview

PortKeeper follows the **BigBase Entity-Component-Construct (ECC)** pattern — the same architecture used in BigBase and Big DockLocker. A Go kernel owns lifecycle and the event bus; pluggable components own the domain logic.

The macOS native shell (system tray icon, popover window) is delivered via **Wails v2**: Go binary + React/Vite frontend, packaged as a `.app` bundle.

## Stack

| Layer | Technology | Why |
|---|---|---|
| Kernel + components | Go 1.22+ | Matches BigBase ECC architecture |
| macOS shell | Wails v2 | Go-native macOS tray + window; no Electron weight |
| Frontend UI | React 19 + Vite 8 | Same as BigBase admin console |
| Testing | Vitest 4 + Playwright 1.60 | Same as BigBase |
| Styling | BigBase Design System tokens + macOS system CSS | Visual consistency across the product family |

## ECC structure

```
Entity   = The running PortKeeper agent (the Go binary)
Component = Independent domain module (ProcessMonitor, HealthCheck, etc.)
Construct = Config that decides which components run and how
```

### Kernel responsibilities
- Component discovery and registration
- Dependency resolution
- Lifecycle management: `Init → Start → Stop`
- Event bus for hook-based communication (no direct imports between components)
- Config merge: defaults + `~/.config/portkeeper/config.json` user overrides

### Component interface (Go)

```go
type Component interface {
    Name()         string
    Version()      string
    Dependencies() []string
    ConfigSchema() json.RawMessage
    Init(ctx *Context, config json.RawMessage) error
    Start(ctx *Context) error
    Stop(ctx *Context) error
    Hooks()        []HookDef
}
```

### Component registry

| Component | File | Depends on |
|---|---|---|
| `process-monitor` | `components/processmonitor/` | — |
| `health-check` | `components/healthcheck/` | `process-monitor` |
| `activity-log` | `components/activitylog/` | `process-monitor` |
| `log-capture` | `components/logcapture/` | `process-monitor` |
| `notifications` | `components/notifications/` | `process-monitor`, `activity-log` |
| `settings` | `components/settings/` | — |

### Event bus — key events

| Event | Emitted by | Consumed by |
|---|---|---|
| `process.started` | `process-monitor` | `activity-log`, `notifications` |
| `process.stopped` | `process-monitor` | `activity-log`, `notifications`, `health-check` |
| `process.crashed` | `process-monitor` | `activity-log`, `notifications` |
| `process.unresponsive` | `health-check` | `activity-log`, `notifications` |
| `settings.changed` | `settings` | all components |

## Directory layout

```
portkeeper/
├── main.go                    # Wails app entry + kernel bootstrap
├── kernel/
│   ├── kernel.go              # Component registry, lifecycle
│   ├── context.go             # Shared context (event bus, config, logger)
│   ├── events.go              # Event bus implementation
│   └── component.go           # Component interface
├── components/
│   ├── processmonitor/        # Port scanning, process detection
│   ├── healthcheck/           # HTTP HEAD probes
│   ├── activitylog/           # Event timeline, SQLite persistence
│   ├── logcapture/            # stdout/stderr capture per process
│   ├── notifications/         # macOS UserNotifications
│   └── settings/              # Config load/save, schema
├── config/
│   └── defaults.go            # Default construct config
├── frontend/                  # React 19 + Vite 8
│   ├── src/
│   │   ├── App.tsx
│   │   ├── components/        # Popover, ServerRow, sheets, modals
│   │   ├── hooks/             # useServers, useHealthCheck, etc.
│   │   └── styles/            # BigBase tokens + macOS overrides
│   └── vite.config.ts
├── specs/                     # All planning docs (this folder)
├── go.mod
└── wails.json
```

## macOS integration points

| Feature | macOS API | Go binding |
|---|---|---|
| System tray icon + badge | `NSStatusBar` | Wails v2 tray API |
| Popover window | `NSPopover` | Wails v2 window |
| Launch at login | `SMAppService` | `wails.LoginItem()` |
| Notifications | `UNUserNotificationCenter` | Wails v2 notification API |
| App bundle | `.app` + code signing | `wails build` |

## Data persistence

- **Settings**: `~/.config/portkeeper/config.json`  
- **Activity log**: SQLite at `~/.config/portkeeper/activity.db` (30-day retention)
- **Log buffer**: in-memory ring buffer per process (last 500 lines), flushed to SQLite on crash

## Config schema (construct)

```json
{
  "scanDirectories": ["~/projects", "~/Developer", "~/opensrc"],
  "pollingIntervalSeconds": 5,
  "healthCheckIntervalSeconds": 30,
  "ignoredPorts": [80, 443, 5432, 3306, 6379, 27017],
  "notifications": {
    "crashAlerts": true,
    "showBadge": true
  },
  "launchAtLogin": false
}
```
