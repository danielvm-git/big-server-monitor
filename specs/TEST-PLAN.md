# PortKeeper TDD Test Plan

**Status:** plan  
**Root cause of "no servers":** ProcessMonitor depends entirely on `lsof` for port discovery. If `lsof` fails (permissions, sandboxing, race), the app shows empty — no error propagates to the UI. 42.9% overall coverage, 0% on the critical path.

---

## Root Cause Analysis

The `processmonitor.poll()` → `listeningPorts()` → `listeningPortsViaPS()` chain has **zero test coverage**. When `lsof` fails:

```
lsof fails → listeningPortsViaPS returns error → poll() logs Error at level ERROR →
poll() returns WITHOUT updating pm.servers → GetServers() returns []Server{} →
frontend shows "no servers" forever
```

No error propagates to the UI. No graceful fallback. The user sees nothing.

### Fix Before Testing (prerequisite)

Before the TDD suite can verify behavior, the backend needs a fallback port discovery method:

1. Add `listeningPortsViaNetstat()` — parse `netstat -an -p tcp | grep LISTEN` on macOS
2. Chain: try `lsof` first, fall back to `netstat` on error
3. Propagate errors to a new `GetStatus()` Wails binding so the UI can show an error banner

---

## Phase 1 — Integration Smoke Tests (app_test.go)

**Goal:** Verify the Wails bridge works — kernel boots, components register, bindings return data. Catches F6 (kernel.Start failure), F17 (nil config), and F7 (frontend race).

### Test 1.1: NewApp returns non-nil App
```go
func TestNewApp(t *testing.T)
```
- `a := NewApp()` shouldn't panic
- All component fields are nil before startup
- **verify:** `go test -run TestNewApp -count=1`

### Test 1.2: Startup registers all 6 components
```go
func TestStartupRegistersComponents(t *testing.T)
```
- Create mock Logger
- Call `a.startup(ctx)` and verify `a.kernel.ListComponents()` returns 6 entries
- Verify specific component names: "settings", "processmonitor", "healthcheck", "activitylog", "logcapture", "notifications"
- **verify:** `go test -run TestStartupRegistersComponents -count=1`

### Test 1.3: GetServers returns data after server started
```go
func TestGetServersAfterStart(t *testing.T)
```
- Start a real HTTP server on localhost:0 (random port)
- Call `a.startup(ctx)`, wait for first poll (use mock processmonitor)
- Call `a.GetServers()` and verify non-empty
- **verify:** `go test -run TestGetServersAfterStart -count=1`

### Test 1.4: GetServers nil-guard when processMonitor is nil
```go
func TestGetServersNilGuard(t *testing.T)
```
- Don't call startup
- Call `a.GetServers()` → must return nil (not panic)
- **verify:** `go test -run TestGetServersNilGuard -count=1`

### Test 1.5: All 19 bindings have nil guards
```go
func TestAllBindingsNilGuards(t *testing.T)
```
- Create uninitialized App (no startup)
- Call every binding method
- All must return zero values or errors, never panic
- **verify:** `go test -run TestAllBindingsNilGuards -count=1`

### Test 1.6: Shutdown stops kernel cleanly
```go
func TestShutdown(t *testing.T)
```
- Call `a.startup(ctx)`, verify kernel started
- Call `a.shutdown(ctx)`, verify no error
- Call `a.shutdown(ctx)` again, verify no double-close panic
- **verify:** `go test -run TestShutdown -count=1`

---

## Phase 2 — ProcessMonitor Core Path (RED → GREEN → REFACTOR)

**Goal:** 100% coverage of the polling loop, port discovery, and diff+emit. This fixes the "no servers" problem at the source.

### Step 2.1: Port discovery with mockable interface (REFACTOR first)

**Problem:** `listeningPorts()` and `getProcessInfo()` call `exec.Command` directly — untestable without real processes.

**Refactor:** Extract a `PortDiscovery` interface:

```go
type PortDiscovery interface {
    ListeningPorts() ([]int, error)
    ProcessInfo(port int) (processInfo, error)
}
```

Create two implementations:
- `LsofPortDiscovery` — existing code, production implementation
- `MockPortDiscovery` — test implementation returning canned data

Inject via `ProcessMonitor` struct or `New()` constructor.

### Step 2.2: RED — Test polling loop with mock

```go
func TestPollDiscoversNewPorts(t *testing.T)
```
- Mock `PortDiscovery.ListeningPorts()` returns `[3000, 8080]`
- Mock `PortDiscovery.ProcessInfo()` returns fake process info
- Call `pm.poll(ctx)`
- Assert `pm.GetServers()` returns 2 servers
- Assert servers[0].Port == 3000, servers[1].Port == 8080
- **verify:** `go test -run TestPollDiscoversNewPorts -count=1` → RED (fails)

### Step 2.3: GREEN — Implement mockable poll

Wire the mock into `poll()`. Make it pass.

### Step 2.4: Test poll ignores configured ports

```go
func TestPollIgnoresPorts(t *testing.T)
```
- Set `config.IgnoredPorts = []int{3000}`
- Mock returns ports [3000, 8080]
- Assert only port 8080 appears in GetServers()
- **verify:** `go test -run TestPollIgnoresPorts -count=1`

### Step 2.5: Test poll handles discovery errors

```go
func TestPollHandlesDiscoveryError(t *testing.T)
```
- Mock `ListeningPorts()` returns error
- Call `pm.poll(ctx)`
- Assert `GetServers()` returns previous data (does NOT wipe existing servers)
- **verify:** `go test -run TestPollHandlesDiscoveryError -count=1`

### Step 2.6: Test diffAndEmit emits process.started

```go
func TestDiffAndEmitProcessStarted(t *testing.T)
```
- Mock `EventBus` with subscribe
- First poll: no servers → servers stays empty
- Second poll: port 3000 appears → diff detects NEW
- Assert `process.started` event emitted with port=3000
- **verify:** `go test -run TestDiffAndEmitProcessStarted -count=1`

### Step 2.7: Test diffAndEmit emits process.stopped

```go
func TestDiffAndEmitProcessStopped(t *testing.T)
```
- First poll: port 3000 present
- Second poll: port 3000 gone → diff detects REMOVED
- Assert `process.stopped` event emitted with port=3000
- **verify:** `go test -run TestDiffAndEmitProcessStopped -count=1`

### Step 2.8: Test KillProcess

```go
func TestKillProcessSuccess(t *testing.T)
```
- Start real `sleep` process
- Call `pm.KillProcess(pid)` → no error
- Verify process no longer exists
- **verify:** `go test -run TestKillProcess -count=1`

### Step 2.9: Test KillProcess nonexistent PID

```go
func TestKillProcessNonexistent(t *testing.T)
```
- Call `pm.KillProcess(999999)` → returns error
- **verify:** `go test -run TestKillProcessNonexistent -count=1`

### Step 2.10: Test GetServerByPort

```go
func TestGetServerByPortFound(t *testing.T)
func TestGetServerByPortNotFound(t *testing.T)
```
- Populate servers map via poll
- Query with existing/nonexistent port
- **verify:** `go test -run TestGetServerByPort -count=1`

### Step 2.11: Test real lsof integration (integration test)

```go
func TestLsofIntegration(t *testing.T)
```
- Start a real HTTP server on a random port
- Use the REAL `LsofPortDiscovery` implementation
- Assert the port is discovered
- Skip test if lsof is not available (build tag: `//go:build !ci`)
- **verify:** `go test -run TestLsofIntegration -count=1 -tags=integration`

---

## Phase 3 — HealthCheck DB Protocol Probes

**Goal:** Cover all 5 database protocol probes (PostgreSQL, MySQL, Redis, MongoDB, Memcached) which are currently at 0%. These are a core differentiator per spec 003.

### Step 3.1: RED — Test PostgreSQL probe

```go
func TestProbePostgres(t *testing.T)
```
- Mock TCP server that responds with PostgreSQL startup packet
- Call `hc.probePostgres(port, timeout)`
- Assert HealthOK, protocol="postgres"
- **verify:** `go test -run TestProbePostgres -count=1` → RED

### Step 3.2: GREEN — Make PostgreSQL probe pass
Then run the test again.

### Step 3.3: Test Redis probe
```go
func TestProbeRedis(t *testing.T)
```
- Mock TCP server responding with `+PONG\r\n`
- Assert HealthOK, protocol="redis"
- **verify:** `go test -run TestProbeRedis -count=1`

### Step 3.4: Test MySQL probe
```go
func TestProbeMySQL(t *testing.T)
```
- Mock TCP server with MySQL greeting packet
- Assert HealthOK, protocol="mysql"
- **verify:** `go test -run TestProbeMySQL -count=1`

### Step 3.5: Test MongoDB probe
```go
func TestProbeMongoDB(t *testing.T)
```
- Mock TCP server with isMaster response
- Assert HealthOK, protocol="mongodb"
- **verify:** `go test -run TestProbeMongoDB -count=1`

### Step 3.6: Test Memcached probe
```go
func TestProbeMemcached(t *testing.T)
```
- Mock TCP server with version response
- Assert HealthOK, protocol="memcached"
- **verify:** `go test -run TestProbeMemcached -count=1`

### Step 3.7: Test RunHealthCheck binding
```go
func TestRunHealthCheckBinding(t *testing.T)
```
- Create App, start a test HTTP server
- Call `a.RunHealthCheck([]int{port})`
- Assert result has status, latency, protocol
- **verify:** `go test -run TestRunHealthCheckBinding -count=1`

---

## Phase 4 — LogCapture Integration & Lifecycle

**Goal:** Cover Init/Start/Stop lifecycle and process event hooks (0% currently). Verify crash persistence to SQLite.

### Step 4.1: Test Init creates buffer for each port
```go
func TestLogCaptureInit(t *testing.T)
```
- Create LogCapture, call `Init(ctx, nil)`
- Assert no error, config defaults
- **verify:** `go test -run TestLogCaptureInit -count=1`

### Step 4.2: Test process.started hook creates capture slot
```go
func TestOnProcessStarted(t *testing.T)
```
- Emit `process.started` event with port=3000, pid=12345
- Assert internal buffer is created for port 3000
- **verify:** `go test -run TestOnProcessStarted -count=1`

### Step 4.3: Test log.line hook appends to buffer
```go
func TestOnLogLine(t *testing.T)
```
- After process.started for port 3000
- Emit `log.line` event with text="Error: connection refused"
- Assert `GetLogs({Port: 3000})` returns 1 line
- Assert line.level == "error"
- **verify:** `go test -run TestOnLogLine -count=1`

### Step 4.4: Test process.crashed hook persists to SQLite
```go
func TestCrashPersistence(t *testing.T)
```
- Start process at port 3000
- Add 3 log lines to buffer
- Emit `process.crashed` event for port 3000
- Assert logs are saved to SQLite crash_logs table
- **verify:** `go test -run TestCrashPersistence -count=1`

### Step 4.5: Test process.stopped hook flushes buffer
```go
func TestProcessStoppedClearsBuffer(t *testing.T)
```
- After process.started + log lines
- Emit `process.stopped` event
- Assert `GetLogs()` returns empty for that port
- **verify:** `go test -run TestProcessStoppedClearsBuffer -count=1`

---

## Phase 5 — Frontend Component Tests (Vitest + React Testing Library)

**Goal:** Cover all 5 spec-listed UI test cases + 12 React components + 6 hooks. Currently 0 frontend tests.

### Prerequisites

```bash
cd frontend
npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom
```

Create `frontend/vitest.config.ts`:
```ts
import { defineConfig } from 'vitest/config'
export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
  },
})
```

### Step 5.1: RED — ServerRow renders server data
```tsx
test('ServerRow renders port, process name, and uptime', () => {
  render(<ServerRow server={mockServer} onKill={vi.fn()} />)
  expect(screen.getByText(':3000')).toBeInTheDocument()
  expect(screen.getByText('node')).toBeInTheDocument()
  expect(screen.getByText('2h 15m')).toBeInTheDocument()
})
```
- **verify:** `cd frontend && npx vitest run -t "ServerRow renders"`

### Step 5.2: ServerRow expands on click
```tsx
test('ServerRow expands on click showing PID and memory', () => {
  const { container } = render(<ServerRow server={mockServer} onKill={vi.fn()} />)
  fireEvent.click(screen.getByText(':3000'))
  expect(screen.getByText(/PID: 12345/)).toBeInTheDocument()
  expect(screen.getByText(/128 MB/)).toBeInTheDocument()
})
```
- **verify:** `cd frontend && npx vitest run -t "ServerRow expands"`

### Step 5.3: KillConfirmDialog renders
```tsx
test('KillConfirmDialog shows port and process name', () => {
  render(<KillConfirmDialog server={mockServer} onCancel={vi.fn()} onConfirm={vi.fn()} />)
  expect(screen.getByText(/Kill node on :3000/)).toBeInTheDocument()
  expect(screen.getByText('Cancel')).toBeInTheDocument()
  expect(screen.getByText('Kill')).toBeInTheDocument()
})
```
- **verify:** `cd frontend && npx vitest run -t "KillConfirmDialog"`

### Step 5.4: KillConfirmDialog confirm calls onConfirm
```tsx
test('KillConfirmDialog confirm button calls onConfirm', () => {
  const onConfirm = vi.fn()
  render(<KillConfirmDialog server={mockServer} onCancel={vi.fn()} onConfirm={onConfirm} />)
  fireEvent.click(screen.getByText('Kill'))
  expect(onConfirm).toHaveBeenCalledOnce()
})
```
- **verify:** `cd frontend && npx vitest run -t "confirm button"`

### Step 5.5: Dark mode toggle
```tsx
test('Dark mode toggle flips data-theme attribute', async () => {
  render(<App />)  // App wraps ThemeProvider
  const button = screen.getByTitle('Toggle theme')
  expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  fireEvent.click(button)
  expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
})
```
- **verify:** `cd frontend && npx vitest run -t "Dark mode"`

### Step 5.6: Copy for AI button invokes GetLogsForAI
```tsx
// Mock Wails binding
vi.mock('../../wailsjs/go/main/App', () => ({
  GetLogsForAI: vi.fn().mockResolvedValue('LOGS FOR AI'),
}))
Object.assign(navigator, { clipboard: { writeText: vi.fn() } })

test('Copy for AI button calls GetLogsForAI and writes to clipboard', async () => {
  render(<ServerLogsSheet port={3000} processName="node" logs={mockLogs} onClose={vi.fn()} />)
  fireEvent.click(screen.getByText('Copy for AI'))
  await waitFor(() => {
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('LOGS FOR AI')
  })
})
```
- **verify:** `cd frontend && npx vitest run -t "Copy for AI"`

### Step 5.7-12: Test remaining components
- HealthCheckSheet renders results with status badges
- ActivityLogSheet renders events with filter chips
- SettingsModal renders all form fields
- SettingsModal save calls SaveSettings binding
- PopoverHeader shows server count
- ProjectsSection groups servers by project

---

## Phase 6 — Wails Integration E2E (Playwright)

**Goal:** End-to-end test that the .app launches and the frontend communicates with the Go backend.

### Step 6.1: RED — App launches
```ts
test('app launches and shows PopoverHeader', async ({ page }) => {
  await page.goto('http://localhost:5174') // wails dev URL
  await expect(page.locator('.pk-popover-header')).toBeVisible()
})
```
- **verify:** Start `wails dev`, then `cd frontend && npx playwright test`

### Step 6.2: Server list appears
```ts
test('popover shows ServerList after Go backend discovers servers', async ({ page }) => {
  // Start a test HTTP server on :9876 before running
  await page.waitForSelector('.pk-server-row', { timeout: 10000 })
  await expect(page.locator('.pk-server-row')).toHaveCount(1)
})
```
- **verify:** `cd frontend && npx playwright test -t "ServerList"`

### Step 6.3: Kill dialog flow
```ts
test('clicking kill opens dialog, cancel closes it', async ({ page }) => {
  await page.locator('.pk-server-row .pk-kill-btn').click()
  await expect(page.locator('.pk-kill-confirm')).toBeVisible()
  await page.locator('.pk-kill-cancel').click()
  await expect(page.locator('.pk-kill-confirm')).not.toBeVisible()
})
```

---

## Phase 7 — Error Propagation (New Feature)

**Goal:** When the backend can't discover servers, the frontend shows an error, not silent emptiness.

### Step 7.1: Add GetStatus binding to processmonitor

```go
type MonitorStatus struct {
    Healthy    bool   `json:"healthy"`
    LastError  string `json:"lastError,omitempty"`
    LastPollAt string `json:"lastPollAt,omitempty"`
    ServerCount int   `json:"serverCount"`
}

func (a *App) GetMonitorStatus() MonitorStatus
```

### Step 7.2: RED — GetMonitorStatus returns error when lsof fails
### Step 7.3: GREEN — Implement error tracking in poll()
### Step 7.4: Frontend error banner using useMonitorStatus hook

---

## Execution Order

```
Phase 1 (integration) → Phase 2 (processmonitor core) → Phase 3 (healthcheck DB probes)
→ Phase 4 (logcapture lifecycle) → Phase 5 (frontend components)
→ Phase 6 (Playwright e2e) → Phase 7 (error propagation)
```

### Verify Gates

| After | Command | Expected |
|-------|---------|----------|
| Phase 1 | `go test -run "TestNewApp|TestStartup|TestGetServers|TestNilGuard|TestBinding|TestShutdown" -count=1` | All pass |
| Phase 2 | `go test ./components/processmonitor/... -cover -count=1` | Coverage ≥ 70% |
| Phase 3 | `go test ./components/healthcheck/... -cover -count=1` | Coverage ≥ 80% |
| Phase 4 | `go test ./components/logcapture/... -cover -count=1` | Coverage ≥ 75% |
| Phase 5 | `cd frontend && npx vitest run` | ≥ 15 tests, all pass |
| Phase 6 | `cd frontend && npx playwright test` | All 3 pass |
| Phase 7 | `go test -run TestGetMonitorStatus -count=1` | Pass |
| Final | `go test ./... -coverprofile=cover.out && go tool cover -func=cover.out` | Overall ≥ 65% |

---

## Summary

| Metric | Current | Target |
|--------|---------|--------|
| Go test count | 110 | ~150+ |
| Overall coverage | 42.9% | ≥ 65% |
| ProcessMonitor coverage | ~35% | ≥ 70% |
| app.go coverage | 0% | ≥ 60% |
| Frontend tests | 0 | ≥ 15 |
| E2E tests | 0 | ≥ 3 |
| HealthCheck DB probes | 0% | ≥ 80% |
| LogCapture lifecycle | 0% | ≥ 75% |
