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
