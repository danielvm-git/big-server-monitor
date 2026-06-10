package logcapture

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"portkeeper/kernel"
)

const version = "0.1.0"
const maxBufferSize = 500
const maxCrashLogsPersisted = 200

// LogLevel represents the severity of a log line.
type LogLevel string

const (
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// LogLine represents a single line of log output.
type LogLine struct {
	Seq       int64     `json:"seq"`
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Text      string    `json:"text"`
	Stream    string    `json:"stream"` // stdout, stderr, system
}

// LogFilter is used to query logs with optional filtering.
type LogFilter struct {
	Port   int
	Levels []LogLevel
	Limit  int
}

// ProcessInfo tracks metadata about a monitored process.
type ProcessInfo struct {
	Port        int
	ProcessName string
	PID         int
	ProjectName string
	StartedAt   time.Time
	MemoryMB    int
	Binary      string
}

// LogCapture is the main component for capturing and storing logs from processes.
type LogCapture struct {
	mu      sync.RWMutex
	buffers map[int]*ringBuffer      // port -> ring buffer
	pidInfo map[int]*ProcessInfo      // port -> process info
	seq     int64                     // global sequence counter
	logger  kernel.Logger
}

// ringBuffer implements a fixed-size ring buffer for log lines.
type ringBuffer struct {
	lines    []*LogLine
	capacity int
	head     int // index where next write goes
	isFull   bool
}

// newRingBuffer creates a new ring buffer with the specified capacity.
func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		lines:    make([]*LogLine, 0, capacity),
		capacity: capacity,
	}
}

// append adds a log line to the buffer, dropping the oldest if full.
func (rb *ringBuffer) append(line *LogLine) {
	if len(rb.lines) < rb.capacity {
		rb.lines = append(rb.lines, line)
	} else {
		// Buffer is at capacity, shift oldest out
		copy(rb.lines, rb.lines[1:])
		rb.lines[rb.capacity-1] = line
	}
}

// getLines returns a copy of all lines in the buffer in order.
func (rb *ringBuffer) getLines() []*LogLine {
	result := make([]*LogLine, len(rb.lines))
	copy(result, rb.lines)
	return result
}

// New creates a new LogCapture component.
func New() *LogCapture {
	return &LogCapture{
		buffers: make(map[int]*ringBuffer),
		pidInfo: make(map[int]*ProcessInfo),
		seq:     0,
	}
}

// Name returns the component name.
func (lc *LogCapture) Name() string {
	return "logcapture"
}

// Version returns the component version.
func (lc *LogCapture) Version() string {
	return version
}

// Dependencies returns component dependencies.
func (lc *LogCapture) Dependencies() []string {
	return []string{"processmonitor"}
}

// ConfigSchema returns the JSON schema for component configuration.
func (lc *LogCapture) ConfigSchema() json.RawMessage {
	return nil
}

// Init initializes the component (placeholder for future setup).
func (lc *LogCapture) Init(ctx *kernel.Context, config json.RawMessage) error {
	lc.logger = ctx.Logger
	return nil
}

// Start starts the component and registers event hooks.
func (lc *LogCapture) Start(ctx *kernel.Context) error {
	lc.logger.Info("logcapture component started")
	return nil
}

// Stop stops the component and cleans up resources.
func (lc *LogCapture) Stop(ctx *kernel.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.logger.Info("logcapture component stopped")
	return nil
}

// Hooks returns the event hooks this component subscribes to.
func (lc *LogCapture) Hooks() []kernel.HookDef {
	return []kernel.HookDef{
		{
			Name:     "process.started",
			Priority: 10,
			Handler:  lc.onProcessStarted,
		},
		{
			Name:     "process.stopped",
			Priority: 10,
			Handler:  lc.onProcessStopped,
		},
		{
			Name:     "process.crashed",
			Priority: 10,
			Handler:  lc.onProcessCrashed,
		},
		{
			Name:     "log.line",
			Priority: 20,
			Handler:  lc.onLogLine,
		},
	}
}

// onProcessStarted handles process.started events to initialize log capture.
func (lc *LogCapture) onProcessStarted(ctx *kernel.Context, event kernel.Event) error {
	port, ok := event.Data["port"].(int)
	if !ok {
		return fmt.Errorf("process.started: invalid or missing port")
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Initialize ring buffer if not present
	if _, exists := lc.buffers[port]; !exists {
		lc.buffers[port] = newRingBuffer(maxBufferSize)
	}

	// Store process info
	lc.pidInfo[port] = &ProcessInfo{
		Port:        port,
		ProcessName: getStringValue(event.Data, "process_name"),
		PID:         getIntValue(event.Data, "pid"),
		ProjectName: getStringValue(event.Data, "project_name"),
		StartedAt:   time.Now(),
		Binary:      getStringValue(event.Data, "binary"),
	}

	lc.logger.Info("logcapture: initialized for port", "port", port)
	return nil
}

// onProcessStopped handles process.stopped events to flush logs.
func (lc *LogCapture) onProcessStopped(ctx *kernel.Context, event kernel.Event) error {
	port, ok := event.Data["port"].(int)
	if !ok {
		return fmt.Errorf("process.stopped: invalid or missing port")
	}

	lc.mu.RLock()
	defer lc.mu.RUnlock()

	lc.logger.Info("logcapture: process stopped", "port", port)
	// Logs remain in buffer for retrieval after stop
	return nil
}

// onProcessCrashed handles process.crashed events to persist logs.
func (lc *LogCapture) onProcessCrashed(ctx *kernel.Context, event kernel.Event) error {
	port, ok := event.Data["port"].(int)
	if !ok {
		return fmt.Errorf("process.crashed: invalid or missing port")
	}

	lc.mu.RLock()
	defer lc.mu.RUnlock()

	lc.logger.Warn("logcapture: process crashed", "port", port)
	// In future: persist last N lines to SQLite via ctx.Kernel().DB()
	return nil
}

// onLogLine handles log.line events to capture new log output.
func (lc *LogCapture) onLogLine(ctx *kernel.Context, event kernel.Event) error {
	port, ok := event.Data["port"].(int)
	if !ok {
		return nil // silently ignore if port not specified
	}

	text, ok := event.Data["text"].(string)
	if !ok {
		return nil // silently ignore if text not specified
	}

	stream := getStringValue(event.Data, "stream")
	if stream == "" {
		stream = "stdout"
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Ensure buffer exists
	if _, exists := lc.buffers[port]; !exists {
		lc.buffers[port] = newRingBuffer(maxBufferSize)
	}

	// Increment sequence counter
	lc.seq++

	// Classify the log line
	level := classifyLogLine(text)

	// Create and append log line
	logLine := &LogLine{
		Seq:       lc.seq,
		Timestamp: time.Now(),
		Level:     level,
		Text:      text,
		Stream:    stream,
	}

	lc.buffers[port].append(logLine)
	return nil
}

// GetLogs retrieves logs for a specific port with optional filtering.
func (lc *LogCapture) GetLogs(filter LogFilter) []*LogLine {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	buffer, exists := lc.buffers[filter.Port]
	if !exists {
		return []*LogLine{}
	}

	lines := buffer.getLines()

	// Filter by level if specified
	if len(filter.Levels) > 0 {
		levelSet := make(map[LogLevel]bool)
		for _, l := range filter.Levels {
			levelSet[l] = true
		}

		filtered := make([]*LogLine, 0, len(lines))
		for _, line := range lines {
			if levelSet[line.Level] {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}

	// Apply limit (default 30)
	limit := filter.Limit
	if limit <= 0 {
		limit = 30
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	return lines
}

// GetLogsForAI returns logs formatted for AI consumption with metadata.
func (lc *LogCapture) GetLogsForAI(port int) string {
	lc.mu.RLock()
	pidInfo := lc.pidInfo[port]
	buffer := lc.buffers[port]
	lc.mu.RUnlock()

	if pidInfo == nil || buffer == nil {
		return ""
	}

	lines := buffer.getLines()

	// Calculate uptime
	uptime := time.Since(pidInfo.StartedAt)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60

	result := "=== PortKeeper Log Export ===\n"
	result += fmt.Sprintf("Server:  %s\n", pidInfo.ProjectName)
	result += fmt.Sprintf("Process: %s  (PID %d)\n", pidInfo.ProcessName, pidInfo.PID)
	result += fmt.Sprintf("Port:    :%d\n", port)
	result += fmt.Sprintf("Memory:  %d MB   Uptime: %dh %02dm\n", pidInfo.MemoryMB, hours, minutes)
	result += fmt.Sprintf("Binary:  %s\n", pidInfo.Binary)
	result += "\n"

	// Separate stdout/stderr from errors/warnings
	normalLines := make([]*LogLine, 0, len(lines))
	errorWarnLines := make([]*LogLine, 0, len(lines))

	for _, line := range lines {
		if line.Level == LogError || line.Level == LogWarn {
			errorWarnLines = append(errorWarnLines, line)
		} else {
			normalLines = append(normalLines, line)
		}
	}

	// Format stdout/stderr section (most recent, up to 30)
	result += fmt.Sprintf("--- stdout / stderr (%d lines) ---\n", len(normalLines))
	start := 0
	if len(normalLines) > 30 {
		start = len(normalLines) - 30
	}
	for _, line := range normalLines[start:] {
		result += fmt.Sprintf("[%s] [%s] %s\n",
			line.Timestamp.Format("15:04:05"),
			pidInfo.ProcessName,
			line.Text)
	}
	result += "\n"

	// Format errors & warnings section
	result += fmt.Sprintf("--- Errors & warnings (%d lines) ---\n", len(errorWarnLines))
	for _, line := range errorWarnLines {
		result += fmt.Sprintf("[%s] %s\n",
			line.Timestamp.Format("15:04:05"),
			line.Text)
	}

	return result
}

// GetLogCounts returns counts of log lines by level for a port.
func (lc *LogCapture) GetLogCounts(port int) map[LogLevel]int {
	lc.mu.RLock()
	buffer := lc.buffers[port]
	lc.mu.RUnlock()

	counts := make(map[LogLevel]int)
	if buffer == nil {
		return counts
	}

	lines := buffer.getLines()
	for _, line := range lines {
		counts[line.Level]++
	}
	return counts
}

// LogCount returns the current log count for a port by level.
func (lc *LogCapture) LogCount(port int, level LogLevel) int {
	counts := lc.GetLogCounts(port)
	return counts[level]
}

// classifyLogLine examines a log line and returns its likely severity level.
func classifyLogLine(text string) LogLevel {
	// Check error patterns first (higher priority)
	for _, pattern := range errorPatterns {
		if pattern.MatchString(text) {
			return LogError
		}
	}

	// Check warning patterns
	for _, pattern := range warnPatterns {
		if pattern.MatchString(text) {
			return LogWarn
		}
	}

	// Default to info
	return LogInfo
}

var (
	// Error patterns for log classification
	errorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(error|err|exception|traceback|panic|fatal|failed)\b`),
		regexp.MustCompile(`^\s*at \w+\.\w+`), // JS/Java stack frame
		regexp.MustCompile(`(?i)exit code [^0]`),
	}

	// Warning patterns for log classification
	warnPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(warn|warning|deprecated|caution)\b`),
	}
)

// Helper functions for event data extraction
func getStringValue(data map[string]any, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getIntValue(data map[string]any, key string) int {
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	if val, ok := data[key].(int); ok {
		return val
	}
	return 0
}

// Verify interface compliance
var _ kernel.Component = (*LogCapture)(nil)
