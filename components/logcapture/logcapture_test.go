package logcapture

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"portkeeper/kernel"
)

func TestLogClassification(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantLevel LogLevel
		stream    string
	}{
		{
			name:      "error keyword",
			text:      "ERROR: something went wrong",
			wantLevel: LogError,
			stream:    "stderr",
		},
		{
			name:      "exception keyword",
			text:      "Exception thrown in handler",
			wantLevel: LogError,
			stream:    "stderr",
		},
		{
			name:      "stack frame pattern",
			text:      "at UserController.getUser (/projects/app/src/users.js:45)",
			wantLevel: LogError,
			stream:    "stderr",
		},
		{
			name:      "warning keyword",
			text:      "WARNING: deprecated API usage",
			wantLevel: LogWarn,
			stream:    "stderr",
		},
		{
			name:      "warn keyword",
			text:      "warn: this feature will be removed",
			wantLevel: LogWarn,
			stream:    "stdout",
		},
		{
			name:      "normal info",
			text:      "Server started on port 3000",
			wantLevel: LogInfo,
			stream:    "stdout",
		},
		{
			name:      "panic keyword",
			text:      "panic: runtime error",
			wantLevel: LogError,
			stream:    "stderr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := classifyLogLine(tt.text)
			if level != tt.wantLevel {
				t.Errorf("classifyLogLine(%q) = %q, want %q", tt.text, level, tt.wantLevel)
			}
		})
	}
}

func TestRingBuffer(t *testing.T) {
	t.Run("fills buffer up to capacity", func(t *testing.T) {
		rb := newRingBuffer(10)
		for i := 0; i < 10; i++ {
			rb.append(&LogLine{
				Seq:  int64(i),
				Text: "line " + string(rune(i)),
			})
		}
		if len(rb.lines) != 10 {
			t.Errorf("expected 10 lines, got %d", len(rb.lines))
		}
	})

	t.Run("drops oldest when exceeding capacity", func(t *testing.T) {
		rb := newRingBuffer(5)
		for i := 0; i < 10; i++ {
			rb.append(&LogLine{
				Seq:  int64(i),
				Text: "line " + string(rune(i)),
			})
		}
		if len(rb.lines) != 5 {
			t.Errorf("expected 5 lines, got %d", len(rb.lines))
		}
		if rb.lines[0].Seq != 5 {
			t.Errorf("expected first line seq=5 (oldest kept), got %d", rb.lines[0].Seq)
		}
		if rb.lines[4].Seq != 9 {
			t.Errorf("expected last line seq=9 (newest), got %d", rb.lines[4].Seq)
		}
	})

	t.Run("getLines returns all lines in order", func(t *testing.T) {
		rb := newRingBuffer(5)
		for i := 0; i < 5; i++ {
			rb.append(&LogLine{
				Seq:  int64(i),
				Text: "line " + string(rune(i)),
			})
		}
		lines := rb.getLines()
		if len(lines) != 5 {
			t.Errorf("expected 5 lines, got %d", len(lines))
		}
		for i := 0; i < 5; i++ {
			if lines[i].Seq != int64(i) {
				t.Errorf("line %d: expected seq=%d, got %d", i, i, lines[i].Seq)
			}
		}
	})
}

func TestGetLogsForAIFormat(t *testing.T) {
	t.Run("exports formatted context block", func(t *testing.T) {
		lc := &LogCapture{
			buffers: make(map[int]*ringBuffer),
			pidInfo: make(map[int]*ProcessInfo),
			seq:     0,
		}

		// Add process info
		lc.pidInfo[3000] = &ProcessInfo{
			Port:        3000,
			ProcessName: "node",
			PID:         12345,
			ProjectName: "bigbase-api",
			StartedAt:   time.Now().Add(-2 * time.Hour),
			MemoryMB:    148,
		}

		// Create ring buffer with some logs
		lc.buffers[3000] = newRingBuffer(500)
		lc.buffers[3000].append(&LogLine{
			Seq:       1,
			Timestamp: time.Now(),
			Level:     LogInfo,
			Text:      "Server started on port 3000",
			Stream:    "stdout",
		})
		lc.buffers[3000].append(&LogLine{
			Seq:       2,
			Timestamp: time.Now(),
			Level:     LogError,
			Text:      "Error: Cannot read properties of undefined",
			Stream:    "stderr",
		})

		result := lc.GetLogsForAI(3000)

		if result == "" {
			t.Fatal("expected non-empty export string")
		}

		// Check for key components
		if !regexp.MustCompile(`Server:\s+bigbase-api`).MatchString(result) {
			t.Error("missing server name in export")
		}
		if !regexp.MustCompile(`Port:\s+:3000`).MatchString(result) {
			t.Error("missing port in export")
		}
		if !regexp.MustCompile(`Process:\s+node\s+\(PID 12345\)`).MatchString(result) {
			t.Error("missing process info in export")
		}
		if !regexp.MustCompile(`Server started on port 3000`).MatchString(result) {
			t.Error("missing log line in export")
		}
		if !regexp.MustCompile(`Error: Cannot read properties of undefined`).MatchString(result) {
			t.Error("missing error log in export")
		}
	})
}

func TestComponentInterface(t *testing.T) {
	t.Run("implements kernel.Component", func(t *testing.T) {
		lc := New()
		var _ kernel.Component = lc

		if lc.Name() != "logcapture" {
			t.Errorf("expected name 'logcapture', got %q", lc.Name())
		}

		if lc.Version() == "" {
			t.Error("expected non-empty version")
		}

		deps := lc.Dependencies()
		if len(deps) > 0 {
			// logcapture might depend on other components like monitor
			t.Logf("dependencies: %v", deps)
		}

		schema := lc.ConfigSchema()
		if schema != nil && len(schema) > 0 {
			// Should be valid JSON
			var m map[string]any
			if err := json.Unmarshal(schema, &m); err != nil {
				t.Errorf("ConfigSchema is not valid JSON: %v", err)
			}
		}

		hooks := lc.Hooks()
		if len(hooks) == 0 {
			t.Error("expected at least one hook subscription")
		}

		// Check for process.started hook
		hasProcessStarted := false
		for _, hook := range hooks {
			if hook.Name == "process.started" {
				hasProcessStarted = true
				break
			}
		}
		if !hasProcessStarted {
			t.Error("expected process.started hook")
		}
	})
}

func TestGetLogs(t *testing.T) {
	t.Run("filters by level", func(t *testing.T) {
		lc := &LogCapture{
			buffers: make(map[int]*ringBuffer),
			pidInfo: make(map[int]*ProcessInfo),
		}

		lc.buffers[3000] = newRingBuffer(500)
		lc.buffers[3000].append(&LogLine{
			Level: LogInfo,
			Text:  "info line",
		})
		lc.buffers[3000].append(&LogLine{
			Level: LogWarn,
			Text:  "warn line",
		})
		lc.buffers[3000].append(&LogLine{
			Level: LogError,
			Text:  "error line",
		})

		// Get all
		all := lc.GetLogs(LogFilter{Port: 3000, Limit: 100})
		if len(all) != 3 {
			t.Errorf("expected 3 lines, got %d", len(all))
		}

		// Get only errors
		errorsOnly := lc.GetLogs(LogFilter{
			Port:   3000,
			Levels: []LogLevel{LogError},
			Limit:  100,
		})
		if len(errorsOnly) != 1 {
			t.Errorf("expected 1 error line, got %d", len(errorsOnly))
		}
		if errorsOnly[0].Level != LogError {
			t.Errorf("expected LogError, got %v", errorsOnly[0].Level)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		lc := &LogCapture{
			buffers: make(map[int]*ringBuffer),
			pidInfo: make(map[int]*ProcessInfo),
		}

		lc.buffers[3000] = newRingBuffer(500)
		for i := 0; i < 100; i++ {
			lc.buffers[3000].append(&LogLine{
				Seq:   int64(i),
				Level: LogInfo,
				Text:  "line",
			})
		}

		result := lc.GetLogs(LogFilter{Port: 3000, Limit: 10})
		if len(result) != 10 {
			t.Errorf("expected 10 lines, got %d", len(result))
		}
	})
}

type mockLogger struct{}

func (mockLogger) Info(msg string, args ...any)  {}
func (mockLogger) Warn(msg string, args ...any)  {}
func (mockLogger) Error(msg string, args ...any) {}
func (mockLogger) Debug(msg string, args ...any) {}

func newTestContext() *kernel.Context {
	return &kernel.Context{
		Kernel:     &kernel.Kernel{},
		Logger:     mockLogger{},
		Components: make(map[string]kernel.Component),
		Config:     make(map[string]json.RawMessage),
	}
}

func findHook(hooks []kernel.HookDef, name string) kernel.HookDef {
	for _, h := range hooks {
		if h.Name == name {
			return h
		}
	}
	return kernel.HookDef{}
}

func TestInit(t *testing.T) {
	lc := New()
	ctx := newTestContext()

	err := lc.Init(ctx, nil)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	if lc.logger == nil {
		t.Error("expected logger to be set after Init")
	}
}

func TestOnProcessStarted(t *testing.T) {
	lc := New()
	ctx := newTestContext()
	if err := lc.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	hook := findHook(lc.Hooks(), "process.started")
	if hook.Handler == nil {
		t.Fatal("process.started hook not found or has nil handler")
	}

	err := hook.Handler(ctx, kernel.Event{
		Name: "process.started",
		Data: map[string]any{
			"port": 3000,
			"pid":  12345,
		},
	})
	if err != nil {
		t.Fatalf("process.started handler returned error: %v", err)
	}

	// GetLogs should not panic; buffer should exist
	logs := lc.GetLogs(LogFilter{Port: 3000})
	if logs == nil {
		t.Error("expected non-nil result from GetLogs")
	}
}

func TestOnLogLine(t *testing.T) {
	lc := New()
	ctx := newTestContext()
	if err := lc.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Emit process.started first
	startedHook := findHook(lc.Hooks(), "process.started")
	if err := startedHook.Handler(ctx, kernel.Event{
		Name: "process.started",
		Data: map[string]any{
			"port": 3000,
			"pid":  12345,
		},
	}); err != nil {
		t.Fatalf("process.started handler returned error: %v", err)
	}

	// Emit log.line
	logHook := findHook(lc.Hooks(), "log.line")
	if err := logHook.Handler(ctx, kernel.Event{
		Name: "log.line",
		Data: map[string]any{
			"port": 3000,
			"text": "Error: connection refused",
		},
	}); err != nil {
		t.Fatalf("log.line handler returned error: %v", err)
	}

	logs := lc.GetLogs(LogFilter{Port: 3000})
	if len(logs) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logs))
	}
	if logs[0].Level != LogError {
		t.Errorf("expected level=error, got %s", logs[0].Level)
	}
	if logs[0].Text != "Error: connection refused" {
		t.Errorf("expected text='Error: connection refused', got %q", logs[0].Text)
	}
}

func TestOnLogLineCaptureMultipleLevels(t *testing.T) {
	lc := New()
	ctx := newTestContext()
	if err := lc.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Start process
	startedHook := findHook(lc.Hooks(), "process.started")
	if err := startedHook.Handler(ctx, kernel.Event{
		Name: "process.started",
		Data: map[string]any{"port": 3000, "pid": 12345},
	}); err != nil {
		t.Fatalf("process.started handler returned error: %v", err)
	}

	logHook := findHook(lc.Hooks(), "log.line")

	// Emit info
	logHook.Handler(ctx, kernel.Event{
		Name: "log.line",
		Data: map[string]any{"port": 3000, "text": "Server listening"},
	})
	// Emit warn
	logHook.Handler(ctx, kernel.Event{
		Name: "log.line",
		Data: map[string]any{"port": 3000, "text": "WARNING: deprecated API"},
	})
	// Emit error
	logHook.Handler(ctx, kernel.Event{
		Name: "log.line",
		Data: map[string]any{"port": 3000, "text": "Error: something broke"},
	})

	counts := lc.GetLogCounts(3000)
	if counts[LogInfo] != 1 {
		t.Errorf("expected 1 info, got %d", counts[LogInfo])
	}
	if counts[LogWarn] != 1 {
		t.Errorf("expected 1 warn, got %d", counts[LogWarn])
	}
	if counts[LogError] != 1 {
		t.Errorf("expected 1 error, got %d", counts[LogError])
	}
}

func TestProcessStoppedClearsBuffer(t *testing.T) {
	lc := New()
	ctx := newTestContext()
	if err := lc.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Start process and add log lines
	startedHook := findHook(lc.Hooks(), "process.started")
	startedHook.Handler(ctx, kernel.Event{
		Name: "process.started",
		Data: map[string]any{"port": 3000, "pid": 12345},
	})

	logHook := findHook(lc.Hooks(), "log.line")
	logHook.Handler(ctx, kernel.Event{
		Name: "log.line",
		Data: map[string]any{"port": 3000, "text": "some log line"},
	})

	// Verify line exists
	if len(lc.GetLogs(LogFilter{Port: 3000})) != 1 {
		t.Fatal("expected 1 log line before stop")
	}

	// Emit process.stopped
	stoppedHook := findHook(lc.Hooks(), "process.stopped")
	if err := stoppedHook.Handler(ctx, kernel.Event{
		Name: "process.stopped",
		Data: map[string]any{"port": 3000},
	}); err != nil {
		t.Fatalf("process.stopped handler returned error: %v", err)
	}

	// Assert buffer is cleared
	logs := lc.GetLogs(LogFilter{Port: 3000})
	if len(logs) != 0 {
		t.Errorf("expected empty buffer after stop, got %d lines", len(logs))
	}
}

func TestStartStopLifecycle(t *testing.T) {
	lc := New()
	ctx := newTestContext()

	if err := lc.Init(ctx, nil); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if err := lc.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestGetLogCounts(t *testing.T) {
	t.Run("counts by level", func(t *testing.T) {
		lc := &LogCapture{
			buffers: make(map[int]*ringBuffer),
			pidInfo: make(map[int]*ProcessInfo),
		}

		lc.buffers[3000] = newRingBuffer(500)
		lc.buffers[3000].append(&LogLine{Level: LogInfo, Text: "info1"})
		lc.buffers[3000].append(&LogLine{Level: LogInfo, Text: "info2"})
		lc.buffers[3000].append(&LogLine{Level: LogWarn, Text: "warn1"})
		lc.buffers[3000].append(&LogLine{Level: LogError, Text: "error1"})

		counts := lc.GetLogCounts(3000)

		if counts[LogInfo] != 2 {
			t.Errorf("expected 2 info logs, got %d", counts[LogInfo])
		}
		if counts[LogWarn] != 1 {
			t.Errorf("expected 1 warn log, got %d", counts[LogWarn])
		}
		if counts[LogError] != 1 {
			t.Errorf("expected 1 error log, got %d", counts[LogError])
		}
	})
}
