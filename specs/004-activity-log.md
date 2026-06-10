# Slice 4: ActivityLog Component — "What Happened?"

**type:** epic  
**status:** planning  
**verify:** after killing a test process, activity log shows a crash event with correct timestamp and project name

## Purpose

Persists a timeline of server lifecycle events (started, stopped, crashed, unresponsive) to SQLite. Exposes paginated, filterable history to the frontend.

## Scope

- Subscribe to `process.started`, `process.stopped`, `process.crashed`, `process.unresponsive` events
- Persist each event to SQLite (`~/.config/portkeeper/activity.db`)
- Expose `GetActivityLog(filter)` with pagination
- 30-day rolling retention (configurable)
- `ClearHistory()` binding
- Count by event type for filter badges

## Data model

```go
type ActivityEvent struct {
    ID          int64      `json:"id"`
    Type        EventType  `json:"type"`   // started, stopped, crashed, unresponsive
    Port        int        `json:"port"`
    ProcessName string     `json:"processName"`
    ProjectName string     `json:"projectName"`
    ProjectDir  string     `json:"projectDir"`
    Timestamp   time.Time  `json:"timestamp"`
    Duration    *string    `json:"duration,omitempty"` // "2h 03m" for stopped/crashed
    ExitCode    *int       `json:"exitCode,omitempty"`
    Message     string     `json:"message"`   // human-readable summary line
}

type EventType string
const (
    EventStarted      EventType = "started"
    EventStopped      EventType = "stopped"
    EventCrashed      EventType = "crashed"
    EventUnresponsive EventType = "unresponsive"
)
```

## SQLite schema

```sql
CREATE TABLE IF NOT EXISTS activity_events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    type         TEXT NOT NULL,
    port         INTEGER NOT NULL,
    process_name TEXT NOT NULL,
    project_name TEXT,
    project_dir  TEXT,
    timestamp    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration_s   INTEGER,
    exit_code    INTEGER,
    message      TEXT
);

CREATE INDEX IF NOT EXISTS idx_activity_timestamp ON activity_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_activity_port ON activity_events(port);
```

## Filter model

```go
type ActivityFilter struct {
    ProjectName string    // empty = all
    Port        int       // 0 = all
    EventTypes  []EventType // empty = all
    Since       time.Time
    Limit       int       // default 100
    Offset      int
}
```

## Retention

Cron job inside the component runs daily at midnight (or on startup if last run > 24h ago) to delete rows older than `config.RetentionDays` (default 30).

## API surface

```go
func (al *ActivityLog) GetActivityLog(filter ActivityFilter) ([]ActivityEvent, error)
func (al *ActivityLog) ClearHistory() error
func (al *ActivityLog) GetEventCounts() map[EventType]int
```

## Tests

```go
// Emit started event → row appears in DB with correct fields
func TestEventPersistence(t *testing.T)

// Filter by project → only matching rows returned
func TestFilterByProject(t *testing.T)

// ClearHistory → table empty
func TestClearHistory(t *testing.T)

// Retention: insert row with old timestamp → cleanup deletes it
func TestRetentionCleanup(t *testing.T)
```
