# PortKeeper — Claude Code

Read CONVENTIONS.md before any git operation.

## Project
macOS menubar app for monitoring local development servers.
ECC architecture: Go kernel + 6 components + Wails v2 + React 19 frontend.

## Commands
| Action | Command |
|--------|---------|
| Run    | wails dev |
| Build  | wails build |
| Test Go | go test ./... |
| Test TS | cd frontend && npm test |
| Lint   | golangci-lint run ./... |

## Architecture
ECC pattern: Kernel (discovery, lifecycle, event bus) + 6 pluggable components.
Components: processmonitor, healthcheck, activitylog, logcapture, notifications, settings.
Components communicate via event bus ONLY — no direct imports between components.

## Observability
All logs are structured JSON via Go's log/slog with a JSON handler. Log entries include
`time`, `level`, `msg`, and arbitrary key-value context fields.

| What | Command |
|------|---------|
| View dev logs | `wails dev` (stderr captured in terminal) |
| View prod logs | `tail -f ~/.config/portkeeper/portkeeper.log` |
| View errors only | `grep '"level":"ERROR"' ~/.config/portkeeper/portkeeper.log` |
| Run all tests | `go test ./... -count=1` |
| Health check | `go test ./... -count=1` (pass = healthy) |
| Check DB | `ls ~/.config/portkeeper/activity.db` (SQLite, created by activitylog) |
| Setup (idempotent) | `bash scripts/setup.sh` |
| Build & run | `wails dev` |
| Production build | `wails build && open build/bin/portkeeper.app` |

## Never
- Hardcode file paths or user home directories
- Commit to main without PR
- Write between component directories (each component owns only its own dir)
- Expose internal errors or stack traces to the UI

## Agent Rules
- Read specs/ before writing code
- All planning output goes to specs/
- Write minimum code that solves the stated problem
- Run tests after every change
