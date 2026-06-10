package activitylog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"portkeeper/kernel"
)

const version = "0.1.0"

// EventType represents the type of activity event
type EventType string

const (
	EventStarted      EventType = "started"
	EventStopped      EventType = "stopped"
	EventCrashed      EventType = "crashed"
	EventUnresponsive EventType = "unresponsive"
)

// ActivityEvent represents a single activity log entry
type ActivityEvent struct {
	ID          int64      `json:"id"`
	Type        EventType  `json:"type"`
	Port        int        `json:"port"`
	ProcessName string     `json:"processName"`
	ProjectName string     `json:"projectName"`
	ProjectDir  string     `json:"projectDir"`
	Timestamp   time.Time  `json:"timestamp"`
	Duration    *string    `json:"duration,omitempty"`
	ExitCode    *int       `json:"exitCode,omitempty"`
	Message     string     `json:"message"`
}

// ActivityFilter specifies how to query the activity log
type ActivityFilter struct {
	ProjectName string
	Port        int
	EventTypes  []EventType
	Since       time.Time
	Limit       int
	Offset      int
}

// Config represents the component configuration
type Config struct {
	DBPath       string `json:"db_path"`
	RetentionDays int   `json:"retention_days"`
}

// ActivityLog is the main component
type ActivityLog struct {
	mu              sync.RWMutex
	db              *sql.DB
	dbPath          string
	retentionDays   int
	lastCleanupTime time.Time
}

// New creates a new ActivityLog component
func New() *ActivityLog {
	return &ActivityLog{
		retentionDays: 30,
	}
}

// Name implements kernel.Component
func (al *ActivityLog) Name() string {
	return "activitylog"
}

// Version implements kernel.Component
func (al *ActivityLog) Version() string {
	return version
}

// Dependencies implements kernel.Component
func (al *ActivityLog) Dependencies() []string {
	return []string{}
}

// ConfigSchema implements kernel.Component
func (al *ActivityLog) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"db_path": {"type": "string", "description": "Path to SQLite database"},
			"retention_days": {"type": "integer", "description": "Days to retain activity logs", "default": 30}
		}
	}`)
}

// Init implements kernel.Component
func (al *ActivityLog) Init(ctx *kernel.Context, config json.RawMessage) error {
	var cfg Config
	if config != nil {
		if err := json.Unmarshal(config, &cfg); err != nil {
			ctx.Logger.Error("failed to unmarshal config", "error", err)
			return fmt.Errorf("unmarshal config: %w", err)
		}
	}

	// Set defaults
	if cfg.DBPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}
		cfg.DBPath = filepath.Join(home, ".config", "portkeeper", "activity.db")
	}

	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}

	al.dbPath = cfg.DBPath
	al.retentionDays = cfg.RetentionDays

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(al.dbPath), 0755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	return nil
}

// Start implements kernel.Component
func (al *ActivityLog) Start(ctx *kernel.Context) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	var err error
	al.db, err = sql.Open("sqlite", al.dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := al.migrate(); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	al.lastCleanupTime = time.Now()

	// Run cleanup on startup
	if err := al.cleanup(); err != nil {
		ctx.Logger.Error("initial cleanup failed", "error", err)
	}

	// Subscribe to process events
	for _, hookDef := range al.Hooks() {
		ctx.Kernel.EventBus().Subscribe(hookDef)
	}

	return nil
}

// Stop implements kernel.Component
func (al *ActivityLog) Stop(ctx *kernel.Context) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.db != nil {
		return al.db.Close()
	}
	return nil
}

// Hooks implements kernel.Component
func (al *ActivityLog) Hooks() []kernel.HookDef {
	return []kernel.HookDef{
		{
			Name:     "process.started",
			Priority: 0,
			Handler:  al.handleProcessStarted,
		},
		{
			Name:     "process.stopped",
			Priority: 0,
			Handler:  al.handleProcessStopped,
		},
		{
			Name:     "process.crashed",
			Priority: 0,
			Handler:  al.handleProcessCrashed,
		},
		{
			Name:     "process.unresponsive",
			Priority: 0,
			Handler:  al.handleProcessUnresponsive,
		},
	}
}

// GetActivityLog retrieves activity log entries based on filter
func (al *ActivityLog) GetActivityLog(filter ActivityFilter) ([]ActivityEvent, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if al.db == nil {
		return []ActivityEvent{}, nil
	}

	query := "SELECT id, type, port, process_name, project_name, project_dir, timestamp, duration_s, exit_code, message FROM activity_events WHERE 1=1"
	args := []any{}

	if filter.ProjectName != "" {
		query += " AND project_name = ?"
		args = append(args, filter.ProjectName)
	}

	if filter.Port > 0 {
		query += " AND port = ?"
		args = append(args, filter.Port)
	}

	if len(filter.EventTypes) > 0 {
		placeholders := ""
		for i, et := range filter.EventTypes {
			if i > 0 {
				placeholders += ", "
			}
			placeholders += "?"
			args = append(args, string(et))
		}
		query += " AND type IN (" + placeholders + ")"
	}

	if !filter.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	query += " LIMIT ?"
	args = append(args, filter.Limit)

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := al.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query activity log: %w", err)
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var event ActivityEvent
		var durationSec sql.NullInt64
		var exitCode sql.NullInt64

		if err := rows.Scan(&event.ID, &event.Type, &event.Port, &event.ProcessName,
			&event.ProjectName, &event.ProjectDir, &event.Timestamp, &durationSec,
			&exitCode, &event.Message); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if durationSec.Valid {
			dur := formatDuration(time.Duration(durationSec.Int64) * time.Second)
			event.Duration = &dur
		}

		if exitCode.Valid {
			code := int(exitCode.Int64)
			event.ExitCode = &code
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// ClearHistory deletes all activity log entries
func (al *ActivityLog) ClearHistory() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.db == nil {
		return nil
	}

	_, err := al.db.Exec("DELETE FROM activity_events")
	if err != nil {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

// GetEventCounts returns a count of events by type
func (al *ActivityLog) GetEventCounts() map[EventType]int {
	al.mu.RLock()
	defer al.mu.RUnlock()

	counts := make(map[EventType]int)

	if al.db == nil {
		return counts
	}

	rows, err := al.db.Query("SELECT type, COUNT(*) FROM activity_events GROUP BY type")
	if err != nil {
		return counts
	}
	defer rows.Close()

	for rows.Next() {
		var typeStr string
		var count int
		if err := rows.Scan(&typeStr, &count); err != nil {
			continue
		}
		counts[EventType(typeStr)] = count
	}

	return counts
}

// Private methods

func (al *ActivityLog) migrate() error {
	schema := `
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
	CREATE INDEX IF NOT EXISTS idx_activity_project ON activity_events(project_name);
	`

	_, err := al.db.Exec(schema)
	return err
}

func (al *ActivityLog) insertEvent(event ActivityEvent) error {
	if al.db == nil {
		return nil
	}

	var durationSec *int64
	if event.Duration != nil {
		ds := int64(0) // placeholder, should be calculated from actual duration
		durationSec = &ds
	}

	_, err := al.db.Exec(
		`INSERT INTO activity_events (type, port, process_name, project_name, project_dir, timestamp, duration_s, exit_code, message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(event.Type), event.Port, event.ProcessName, event.ProjectName, event.ProjectDir,
		event.Timestamp, durationSec, event.ExitCode, event.Message,
	)
	return err
}

func (al *ActivityLog) cleanup() error {
	if al.db == nil {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -al.retentionDays)
	_, err := al.db.Exec("DELETE FROM activity_events WHERE timestamp < ?", cutoff)
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}
	al.lastCleanupTime = time.Now()
	return nil
}

// Hook handlers

func (al *ActivityLog) handleProcessStarted(ctx *kernel.Context, event kernel.Event) error {
	port := extractInt(event.Data, "port")
	processName, _ := event.Data["processName"].(string)
	projectName, _ := event.Data["projectName"].(string)
	projectDir, _ := event.Data["projectDir"].(string)

	message := fmt.Sprintf("%s started on port %d", processName, port)

	actEvent := ActivityEvent{
		Type:        EventStarted,
		Port:        port,
		ProcessName: processName,
		ProjectName: projectName,
		ProjectDir:  projectDir,
		Timestamp:   time.Now(),
		Message:     message,
	}

	al.mu.Lock()
	err := al.insertEvent(actEvent)
	al.mu.Unlock()

	return err
}

func (al *ActivityLog) handleProcessStopped(ctx *kernel.Context, event kernel.Event) error {
	port := extractInt(event.Data, "port")
	processName, _ := event.Data["processName"].(string)
	projectName, _ := event.Data["projectName"].(string)
	projectDir, _ := event.Data["projectDir"].(string)
	durationSec := extractInt(event.Data, "duration")
	exitCode := extractInt(event.Data, "exitCode")

	var duration *string
	var exitCodePtr *int
	if durationSec > 0 {
		durStr := formatDuration(time.Duration(durationSec) * time.Second)
		duration = &durStr
	}
	if exitCode > 0 {
		exitCodePtr = &exitCode
	}

	message := fmt.Sprintf("%s stopped (exit code: %d)", processName, exitCode)

	actEvent := ActivityEvent{
		Type:        EventStopped,
		Port:        port,
		ProcessName: processName,
		ProjectName: projectName,
		ProjectDir:  projectDir,
		Timestamp:   time.Now(),
		Duration:    duration,
		ExitCode:    exitCodePtr,
		Message:     message,
	}

	al.mu.Lock()
	err := al.insertEvent(actEvent)
	al.mu.Unlock()

	return err
}

func (al *ActivityLog) handleProcessCrashed(ctx *kernel.Context, event kernel.Event) error {
	port := extractInt(event.Data, "port")
	processName, _ := event.Data["processName"].(string)
	projectName, _ := event.Data["projectName"].(string)
	projectDir, _ := event.Data["projectDir"].(string)

	message := fmt.Sprintf("%s crashed on port %d", processName, port)

	actEvent := ActivityEvent{
		Type:        EventCrashed,
		Port:        port,
		ProcessName: processName,
		ProjectName: projectName,
		ProjectDir:  projectDir,
		Timestamp:   time.Now(),
		Message:     message,
	}

	al.mu.Lock()
	err := al.insertEvent(actEvent)
	al.mu.Unlock()

	return err
}

func (al *ActivityLog) handleProcessUnresponsive(ctx *kernel.Context, event kernel.Event) error {
	port := extractInt(event.Data, "port")
	processName, _ := event.Data["processName"].(string)
	projectName, _ := event.Data["projectName"].(string)
	projectDir, _ := event.Data["projectDir"].(string)

	message := fmt.Sprintf("%s became unresponsive on port %d", processName, port)

	actEvent := ActivityEvent{
		Type:        EventUnresponsive,
		Port:        port,
		ProcessName: processName,
		ProjectName: projectName,
		ProjectDir:  projectDir,
		Timestamp:   time.Now(),
		Message:     message,
	}

	al.mu.Lock()
	err := al.insertEvent(actEvent)
	al.mu.Unlock()

	return err
}

// Utility functions

func extractInt(data map[string]any, key string) int {
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	if val, ok := data[key].(int); ok {
		return val
	}
	return 0
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// Ensure ActivityLog implements kernel.Component
var _ kernel.Component = (*ActivityLog)(nil)
