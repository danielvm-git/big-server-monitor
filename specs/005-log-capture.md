# Slice 5: LogCapture Component — "See the Output"

**type:** epic  
**status:** planning  
**verify:** after starting a watched process, its stdout/stderr appears in the logs sheet within 2 polling cycles

## Purpose

Captures stdout and stderr from monitored processes and stores a rolling buffer (last 500 lines). Exposes formatted logs to the frontend with filtering and an "AI-ready" export format.

## Design note

macOS does not allow attaching to arbitrary process stdio after startup without code injection. PortKeeper uses a pragmatic two-tier approach:

1. **Spawned processes** (future: PortKeeper can optionally launch dev servers itself) — full pipe capture
2. **Existing processes** (v1 default) — read from macOS unified logging (`log stream --pid <PID>`) and the process's open file descriptors via `lsof`

For v1, the log capture component tails system log entries for each watched PID. This gives enough signal for the "Copy for AI" use case without requiring the user to change how they start their processes.

## Scope

- Subscribe to `process.started` to begin log capture for new PIDs
- Subscribe to `process.stopped` / `process.crashed` to flush and stop capture
- In-memory ring buffer: last 500 lines per PID
- Persist last 200 lines to SQLite on crash (for post-mortem)
- Classify lines as stdout / stderr / error / warning by heuristics
- Expose `GetLogs(port, filter)` binding
- Expose `GetLogsForAI(port)` binding — returns formatted context block

## Data model

```go
type LogLine struct {
    Seq       int64     `json:"seq"`
    Timestamp time.Time `json:"timestamp"`
    Level     LogLevel  `json:"level"` // info, warn, error
    Text      string    `json:"text"`
    Stream    string    `json:"stream"` // stdout, stderr, system
}

type LogLevel string
const (
    LogInfo  LogLevel = "info"
    LogWarn  LogLevel = "warn"
    LogError LogLevel = "error"
)

type LogFilter struct {
    Port   int
    Levels []LogLevel // empty = all
    Limit  int        // default 30
}
```

## Log classification heuristics

```go
var errorPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)\b(error|err|exception|traceback|panic|fatal|failed)\b`),
    regexp.MustCompile(`^\s*at \w+\.\w+`),  // JS/Java stack frame
    regexp.MustCompile(`(?i)exit code [^0]`),
}

var warnPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)\b(warn|warning|deprecated|caution)\b`),
}
```

## "Copy for AI" format

```
=== PortKeeper Log Export ===
Server:  bigbase-api
Process: node  (PID 12345)
Port:    :3000
Memory:  148 MB   Uptime: 2h 03m
Binary:  /usr/local/bin/node

--- stdout / stderr (30 lines) ---
[14:23:01] [node] Server started on port 3000
[14:25:44] [node] GET /api/users 200 12ms
...

--- Errors & warnings (3 lines) ---
[15:01:02] Error: Cannot read properties of undefined (reading 'id')
[15:01:02]     at UserController.getUser (/projects/bigbase/src/users.js:45)
[15:01:02]     at processTicksAndRejections (node:internal/process/task_queues:95)
```

## API surface

```go
func (lc *LogCapture) GetLogs(filter LogFilter) []LogLine
func (lc *LogCapture) GetLogsForAI(port int) string
func (lc *LogCapture) GetLogCounts(port int) map[LogLevel]int
```

## SQLite schema (crash persistence)

```sql
CREATE TABLE IF NOT EXISTS crash_logs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    port         INTEGER NOT NULL,
    process_name TEXT NOT NULL,
    project_name TEXT,
    crashed_at   DATETIME NOT NULL,
    level        TEXT NOT NULL,
    stream       TEXT NOT NULL,
    line_text    TEXT NOT NULL
);
```

## Tests

```go
// Classify error line → LogError
func TestLogClassification(t *testing.T)

// Ring buffer at 500 lines → oldest dropped when 501st arrives
func TestRingBuffer(t *testing.T)

// GetLogsForAI → string contains server name, port, PID, and log lines
func TestAIExportFormat(t *testing.T)
```
