# Slice 2: ProcessMonitor Component — "See What's Running"

**type:** epic  
**status:** planning  
**verify:** component lists all TCP listening ports with PID, process name, and uptime within one polling cycle

## Purpose

Core component. Polls the OS every N seconds to discover which TCP ports are listening, maps each port to its process (PID, binary name, binary path), detects project names from the process working directory, and emits events when processes start or stop.

## Scope

- Enumerate all listening TCP ports (IPv4 + IPv6)
- Resolve PID → process name, binary path, working directory, memory usage, start time
- Detect project name from working directory (reads `package.json`, `go.mod`, `Cargo.toml`, `pyproject.toml`, etc.)
- Compute uptime per process
- Poll on configurable interval (default 5s)
- Emit `process.started`, `process.stopped`, `process.crashed` events
- Expose current server list via `GetServers() []Server`

## Data model

```go
type Server struct {
    Port           int       `json:"port"`
    Status         Status    `json:"status"`    // online, offline, unknown
    PID            int       `json:"pid"`
    ProcessName    string    `json:"processName"`
    RuntimeVersion string    `json:"runtimeVersion"` // e.g. "node v20.11.0"
    BinaryPath     string    `json:"binaryPath"`
    ProjectName    string    `json:"projectName"`
    ProjectDir     string    `json:"projectDir"`
    LocalDomain    string    `json:"localDomain,omitempty"` // e.g. "myapp.local"
    TunnelURL      string    `json:"tunnelURL,omitempty"`   // public ngrok/frp URL
    EnvSnapshot    []EnvVar  `json:"envSnapshot,omitempty"` // key vars from .env
    MemoryMB       float64   `json:"memoryMb"`
    StartedAt      time.Time `json:"startedAt"`
    UptimeStr      string    `json:"uptimeStr"`  // "2h 03m"
}

type EnvVar struct {
    Key     string `json:"key"`
    Value   string `json:"value"`   // redacted to "***" unless in allowlist
    Visible bool   `json:"visible"` // true only for safe keys (NODE_ENV, PORT, etc.)
}

type Status string
const (
    StatusOnline  Status = "online"
    StatusOffline Status = "offline"
    StatusUnknown Status = "unknown"
)
```

## macOS syscall approach

Port enumeration uses `syscall.SysctlRaw("net.inet.tcp.pcblist64")` to get the kernel TCP control block list — zero external dependencies.

PID → process info uses `syscall.Sysctl("kern.proc.pid", pid)` or `/proc`-equivalent via `ps` fallback.

```go
func listeningPorts() ([]portEntry, error) {
    // net.inet.tcp.pcblist64 → parse xtcpcb64 structs
    // filter: st_state == TCPS_LISTEN
    // return {localPort, foreignPort, pid}
}

func processInfo(pid int) (ProcessInfo, error) {
    // kern.proc.pid → name, path
    // proc_pidinfo(pid, PROC_PIDTBSDINFO) → start time, uid
    // task_info via mach_task_self → memory (resident set size)
}
```

## Project name detection

Walk up from process working directory until a project marker is found:

```go
var projectMarkers = []string{
    "package.json",    // Node.js (read "name" field)
    "go.mod",          // Go (read module name)
    "Cargo.toml",      // Rust (read [package].name)
    "pyproject.toml",  // Python (read [project].name)
    "composer.json",   // PHP
    ".git",            // fallback: use directory name
}
```

Only scan directories listed in `config.scanDirectories`. Unknown working directories → project name = directory basename.

## Polling loop

```go
func (pm *ProcessMonitor) Start(ctx *Context) error {
    ticker := time.NewTicker(pm.config.PollingInterval)
    go func() {
        for {
            select {
            case <-ticker.C:
                pm.poll(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
    return nil
}
```

Diff previous snapshot vs current snapshot → emit events for new and disappeared ports.

## Events emitted

```go
// process.started
type ProcessStartedEvent struct {
    Server Server
}

// process.stopped  — clean exit (process exited 0 or was killed by user)
type ProcessStoppedEvent struct {
    Server   Server
    ExitCode int
    Duration time.Duration
}

// process.crashed  — process disappeared without user action (exit != 0 or unknown)
type ProcessCrashedEvent struct {
    Server   Server
    ExitCode int
    Duration time.Duration
}
```

## Runtime version detection

Read the runtime version from the binary at `server.BinaryPath` by executing it with a version flag. Done once per PID at first detection; result cached for the lifetime of the process.

```go
var versionFlags = map[string][]string{
    "node":    {"--version"},           // "v20.11.0"
    "python3": {"--version"},           // "Python 3.11.4"
    "python":  {"--version"},
    "go":      {"version"},             // "go version go1.22.0 darwin/amd64"
    "ruby":    {"--version"},           // "ruby 3.3.0 ..."
    "java":    {"-version"},            // stderr: "java version \"21.0.1\""
    "php":     {"--version"},
    "cargo":   {"--version"},
}

func detectRuntimeVersion(binaryPath string) string {
    name := filepath.Base(binaryPath)
    flags, ok := versionFlags[name]
    if !ok { return "" }
    out, err := exec.Command(binaryPath, flags...).CombinedOutput()
    if err != nil { return "" }
    return parseFirstVersionToken(string(out))  // extract "v20.11.0" etc.
}
```

## Local domain resolution

Read `/etc/hosts` once on startup and on `settings.changed`. Build an index of `127.0.0.1` / `::1` → hostname entries. Match against each server's port by checking if any mapped hostname has a path suffix that looks like a port (rare), or store all localhost aliases and surface the best match based on project name similarity.

```go
func loadLocalhostAliases() map[string]string {
    // parse /etc/hosts, return map[ip][]hostname for 127.x.x.x and ::1
    // → invert to map[hostname]bool for quick lookup
}
```

Shown in the expanded server row as: `myapp.local` → links to `http://myapp.local` in the default browser.

## Ngrok / FRP tunnel detection

Poll the ngrok local API at `http://127.0.0.1:4040/api/tunnels` if port 4040 is detected as listening. Map each tunnel's `config.addr` (e.g. `http://localhost:3000`) to a server by port number, then attach the public URL.

```go
type NgrokTunnel struct {
    PublicURL string `json:"public_url"`
    Config    struct {
        Addr string `json:"addr"`
    } `json:"config"`
}

func fetchNgrokTunnels() ([]NgrokTunnel, error) {
    resp, err := http.Get("http://127.0.0.1:4040/api/tunnels")
    // parse and return tunnels
}
```

Also check FRP: if `frpc` process is detected, read its config file from the binary's working directory to extract the server addr.

Tunnel URL displayed in the server row under the port, as a clickable link.

## Env var snapshot

When a project directory is detected, look for `.env`, `.env.local`, `.env.development` files. Parse key=value pairs. Show only keys in the **safe allowlist**; redact all others.

```go
var safeEnvKeys = map[string]bool{
    "NODE_ENV": true, "APP_ENV": true, "GO_ENV": true,
    "PORT": true, "HOST": true,
    "DATABASE_URL": true,  // shown as "***" — key visible, value hidden
    "REDIS_URL": true,
    "LOG_LEVEL": true, "DEBUG": true,
}

// Never show: *_KEY, *_SECRET, *_TOKEN, *_PASSWORD, *_PASS, *_PWD, AWS_*, GITHUB_*
var redactPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)(key|secret|token|password|pass|pwd|auth)`),
    regexp.MustCompile(`(?i)^(AWS_|GITHUB_|STRIPE_|TWILIO_)`),
}
```

Env vars shown in the expanded server row as a read-only key/value list. Value cells show the actual value for safe keys, `***` for redacted ones. No write access.

## Ports to ignore

Always skip `config.ignoredPorts`. Default: `[80, 443, 5432, 3306, 6379, 27017, 2181, 9092]`.

Also skip ports owned by non-user UIDs (system services) unless the process working directory is under a `scanDirectory`.

## API surface (called by frontend via Wails bindings)

```go
func (pm *ProcessMonitor) GetServers() []Server
func (pm *ProcessMonitor) KillProcess(pid int) error
func (pm *ProcessMonitor) GetServerByPort(port int) (Server, error)
```

## Tests

```go
// Table-driven: given a mock pcblist, assert correct Server slice
func TestListeningPortParsing(t *testing.T)

// Given a PID from a live process, assert name and path are non-empty  
func TestProcessInfoResolution(t *testing.T)

// Given a temp dir with package.json, assert project name extracted correctly
func TestProjectNameDetection(t *testing.T)

// Diff: server appears → started event; server disappears → stopped/crashed event
func TestPollDiffEmitsCorrectEvents(t *testing.T)
```
