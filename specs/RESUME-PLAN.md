# PortKeeper Resume Plan — Finishing the Build

**Generated:** 2026-06-09  
**From:** `plan-how-to-execute-kind-metcalfe.md`  
**Current status:** Workflow A complete, Workflow B partial (6 components exist, 3 have test failures), Workflow C not started

---

## Where We Are

```
Workflow A: Bootstrap & Scaffold     ✓ COMPLETE
       ↓  gate: go build ./kernel/... ✓ (8/8 tests pass)
Workflow B: Backend Components       ⚠ PARTIAL (6/6 components exist, 3 failing)
       ↓  gate: go test ./components/... ⚠ (6 failures across 3 components)
Workflow C: Integration & Ship       ✗ NOT STARTED
```

### Git State
- **Branch:** `feat/portkeeper-v1`
- **Last commits:** `chore: scaffold wails project...` + `chore: init specs`
- **Uncommitted:** `go.mod`, `go.sum` (modified), `specs/state.yaml` (modified), all of `components/` (untracked)

---

## Phase 0: Fix Failing Component Tests

Three components have test failures. Fix them in this order.

### Fix 0.1 — healthcheck: Deadlock in `TestConcurrentProbes`

**File:** `components/healthcheck/healthcheck_test.go`  
**Root cause:** Test creates `HealthCheck{}` (zero-value), so `MaxConcurrentProbes` is `0`. In `runAllProbes`, `make(chan struct{}, 0)` creates a zero-capacity semaphore — all goroutines block → deadlock → test timeout (30s).

**Fix:** In the test function, call `hc := New()` instead of `hc := &HealthCheck{}`, OR set `hc.config.MaxConcurrentProbes = 10` before calling Init. Also add a defensive minimum bound in `runAllProbes` (`components/healthcheck/healthcheck.go` line ~251): `if maxProbes < 1 { maxProbes = 1 }`.

**Additional fix — `Stop()` double-close guard:** In `healthcheck.go` line ~152, wrap the `close(hc.stopChan)` in a `sync.Once` or a `select` guard:
```go
select {
case <-hc.stopChan:
    // already closed
default:
    close(hc.stopChan)
}
```

**Verify:** `go test ./components/healthcheck/... -v -count=1 -timeout 30s` — all tests pass, no timeouts.

---

### Fix 0.2 — notifications: Nil pointer in `TestHasPermission`

**File:** `components/notifications/notifications_test.go` (line ~460-477)  
**Root cause:** `TestHasPermission` calls `n.RequestPermission(ctx)` before `n.Init(ctx, nil)`. The `logger` field is nil (set only in `Init`), causing panic at `n.logger.Info(...)` in `notifications.go` line ~239.

**Fix:** Add `_ = n.Init(ctx, nil)` before `_ = n.RequestPermission(ctx)` in the test. Optionally add a nil-guard in `RequestPermission`:
```go
if n.logger != nil {
    n.logger.Info("notification permission requested")
}
```

**Verify:** `go test ./components/notifications/... -v -count=1` — all 16 tests pass.

---

### Fix 0.3 — settings: Hardcoded paths in 4 tests

**File:** `components/settings/settings_test.go`  
**Root cause:** `SaveSettings` validates that all `ScanDirectories` exist on disk. Three test configs use paths that don't exist:

1. **Line ~75:** `ScanDirectories: []string{"/tmp/test"}` → `/tmp/test` doesn't exist → `stat` fails.  
   **Fix:** Change to `[]string{tmpDir}` (the `tmpDir` already declared from `t.TempDir()` on line ~62).

2. **Line ~186:** `testConfig := Defaults` → `Defaults` contains `~/projects`, `~/Developer`, `~/opensrc`. At least one doesn't exist on this machine.  
   **Fix:** Add `testConfig.ScanDirectories = []string{tmpDir}` after the assignment (where `tmpDir` is from `t.TempDir()` on line ~175).

3. **Same root causes in subtests** within `TestConfigPersistence` and `TestSettingsChangedEventEmitted`.

**Verify:** `go test ./components/settings/... -v -count=1` — all 11 tests pass.

---

### Phase 0 Gate

```bash
cd /Users/danielvm/Developer/big-server-monitor
go test ./components/... -count=1 -timeout 120s
```

**Required:** ALL tests pass. Zero failures.

---

## Phase 1: Wire Backend (C1 from original plan)

This is one sequential agent task. **main.go** and **app.go** are still Wails scaffold defaults — they need to be rewritten to boot up the ECC kernel with all 6 components.

### What to do

**File:** `main.go` + `app.go`

1. **Create App struct** that holds:
   - `kernel *kernel.Kernel`
   - Refs to all 6 components (processmonitor, healthcheck, activitylog, logcapture, notifications, settings)
   - Wails context

2. **Register components in dependency order** (see `specs/001-architecture.md` §Component registry):
   ```
   settings → processmonitor → healthcheck, activitylog, logcapture → notifications
   ```

3. **Start kernel on Wails startup** (`OnStartup` → `app.startup`):
   - Init all components with config + context
   - Start all components (topological order via kernel)

4. **Expose Wails bindings** as methods on App (as listed in original plan):
   - `GetServers() []Server` → processmonitor
   - `KillProcess(pid int) error` → processmonitor
   - `GetHealthResults() []HealthResult` → healthcheck
   - `RunHealthCheck() []HealthResult` → healthcheck
   - `GetActivityLog(filter ActivityFilter) ([]ActivityEvent, error)` → activitylog
   - `ClearHistory() error` → activitylog
   - `GetLogs(port int, filter LogFilter) []LogLine` → logcapture
   - `GetLogsForAI(port int) string` → logcapture
   - `GetSettings() Config` → settings
   - `SaveSettings(config Config) error` → settings

5. **Run `wails generate module`** to produce `frontend/wailsjs/go/main/App.ts` bindings.

**Verify:** `go build -o /dev/null .` — compiles without errors.

### Phase 1 Gate

```bash
go build -o /dev/null . && echo "main.go wired OK"
```

---

## Phase 2: Frontend Upgrade + UI Build (C2 from original plan)

### Phase 2a: Frontend Dependency Upgrade (pre-requisite)

**File:** `frontend/package.json`  
**Issue:** Current scaffold has wrong versions — React 18.2, Vite 3. The spec requires React 19 + Vite 8.

**Fix:** Update package.json to match the spec stack:
- `react` → `^19.2.7`
- `react-dom` → `^19.2.7`
- `vite` → `^8.0.16`
- `@vitejs/plugin-react` → latest compat
- Add: `lucide-react`, `react-router-dom@7.16.0`, `vitest@4.1.8`, `@playwright/test@1.60.0`
- Run `npm install`

**Verify:** `cd frontend && npm install && npx vite --version` shows Vite 8.x.

---

### Phase 2b: Frontend UI (3 parallel agents)

From the original plan's C2 section. Three independent agents, each responsible for a slice of the UI:

| Agent | Label | Focus |
|-------|-------|-------|
| 1 | `popover-core` | MenubarIcon (badge), Popover, PopoverHeader, ServerList, ServerRow, ServerRowExpanded, KillConfirmDialog, ProjectsSection |
| 2 | `sheets` | HealthCheckSheet, ActivityLogSheet, ServerLogsSheet (with Copy for AI button) |
| 3 | `settings-theme` | SettingsModal, dark mode toggle, BigBase CSS tokens, useServers/useSettings/useHealthResults hooks |

**Each agent must read:**
- `specs/008-ui-menubar.md` (full UI spec: component tree, state, design tokens, animations)
- `frontend/wailsjs/go/main/App.ts` (Wails bindings — regenerated after Phase 1)
- `frontend/src/context/ThemeContext.tsx` (copied from bigbase in Workflow A)
- `frontend/src/hooks/useToast.ts` (copied from bigbase in Workflow A)

**Each agent writes only to:** `frontend/src/` (no overlapping files between agents).

**Tech rules:**
- React 19 + TypeScript + Vite 8
- Lucide React icons
- BigBase CSS tokens from `specs/008-ui-menubar.md` §Design tokens
- State: React `useState`/`useEffect` only
- All backend calls via Wails bindings (`import from wailsjs/go/main/App`)
- Dark mode: toggle `[data-theme="dark"]` on `<html>`, persist to `localStorage`

**Verify (per agent):** `cd frontend && npm run build` compiles without errors.

### Phase 2 Gate

```bash
cd frontend && npm run build && echo "frontend OK"
```

---

## Phase 3: Final Audit & Ship (C3 from original plan)

One sequential agent.

### Steps

1. **Run `audit-code` skill** — verify:
   - CONVENTIONS.md compliance (camelCase, PascalCase, ALLCAPS for acronyms, error wrapping, table-driven tests)
   - No hardcoded file paths or user home directories
   - Test coverage is adequate
   - No direct imports between components

2. **Full test suite:**
   ```bash
   go test ./... -count=1 -timeout 120s   # all Go tests
   cd frontend && npm test                 # all TS tests
   ```

3. **Build the app:**
   ```bash
   wails build
   ```

4. **Verify the bundle:**
   ```bash
   ls ./build/bin/portkeeper.app
   codesign --verify --deep --strict ./build/bin/portkeeper.app
   ```

5. **Run `commit-message` skill** → generate Conventional Commits message.

6. **Git commit and push:** Commit all changes with the generated message.

7. **Run `release-branch` skill** → merge `feat/portkeeper-v1` → `main`.

### Phase 3 Gate

```bash
ls ./build/bin/portkeeper.app && echo "SHIPPED"
```

---

## Verify Gates Summary

| Phase | Gate Command | Expected |
|-------|-------------|----------|
| 0 (Fix tests) | `go test ./components/... -count=1` | All pass, 0 failures |
| 1 (Wire backend) | `go build -o /dev/null .` | Success |
| 2 (Frontend) | `cd frontend && npm run build` | Success |
| 3 (Ship) | `ls ./build/bin/portkeeper.app` | File exists |

---

## Known Risks (from exploration)

| Risk | Severity | Mitigation |
|------|----------|------------|
| healthcheck `Stop()` double-close panic | Medium | Wrap in `sync.Once` or `select` guard (included in Fix 0.1) |
| activitylog hardcoded DB path `~/.config/portkeeper/` | Low | Violates "never hardcode paths" rule — document as tech debt or fix during audit |
| activitylog `durationSec` always = 0 | Low | Duration from events not persisted; document as limitation |
| logcapture `ConfigSchema()` returns nil | Low | Return `{}` instead; cosmetic |
| logcapture dependency on "monitor" | Low | Soft dep — kernel topo sort will handle if component name matches |
| processmonitor Linux `/proc` paths dead code on macOS | Low | Doesn't fail, just dead code; cleanup later |
| frontend package.json has wrong versions | High | **Phase 2a fixes this before UI build** |
| Wails bindings mismatch | Medium | **Phase 1 regenerates bindings before Phase 2** |
| Haiku context limits on large specs | Medium | Agents receive file paths, not full inline content |

---

## Time Estimate

| Phase | Est. Time | Notes |
|-------|-----------|-------|
| 0 (Fix 3 components) | 5 min | Three small test fixes |
| 1 (Wire backend) | 10 min | One agent, sequential |
| 2a (Upgrade deps) | 2 min | npm install |
| 2b (Frontend UI) | 10-15 min | 3 parallel agents |
| 3 (Ship) | 10 min | Audit + build + release |
| **Total** | **~35-40 min** | |

---

## Execution Order

```
Phase 0 (fix tests) → gate ✓
    ↓
Phase 1 (wire backend) → gate ✓
    ↓
Phase 2a (upgrade frontend deps) → gate ✓
    ↓
Phase 2b (frontend UI, 3 parallel) → gate ✓
    ↓
Phase 3 (ship) → DONE
```
