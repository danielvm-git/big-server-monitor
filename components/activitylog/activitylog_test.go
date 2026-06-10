package activitylog

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"portkeeper/kernel"
)

type mockLogger struct{}

func (mockLogger) Info(msg string, args ...any)  {}
func (mockLogger) Warn(msg string, args ...any)  {}
func (mockLogger) Error(msg string, args ...any) {}
func (mockLogger) Debug(msg string, args ...any) {}

func newMockKernel() *kernel.Kernel {
	return kernel.New(mockLogger{})
}

func newMockContext(t *testing.T) (*kernel.Context, string, func()) {
	tmpDir := t.TempDir()
	k := newMockKernel()
	return &kernel.Context{
		Kernel:     k,
		Logger:     mockLogger{},
		Components: make(map[string]kernel.Component),
		Config:     make(map[string]json.RawMessage),
	}, tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestEventPersistence(t *testing.T) {
	ctx, tmpDir, cleanup := newMockContext(t)
	defer cleanup()

	cfg := json.RawMessage(`{"db_path":"` + tmpDir + `/activity.db"}`)

	al := New()
	if err := al.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := al.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer al.Stop(ctx)

	// Emit a started event
	event := kernel.Event{
		Name: "process.started",
		Data: map[string]any{
			"port":        8080,
			"processName": "web-server",
			"projectName": "myproject",
			"projectDir":  "/home/user/myproject",
		},
	}

	if err := ctx.Kernel.EventBus().Emit(event, ctx); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	// Give the hook time to process
	time.Sleep(100 * time.Millisecond)

	// Query the activity log
	filter := ActivityFilter{
		Limit: 100,
	}
	logs, err := al.GetActivityLog(filter)
	if err != nil {
		t.Fatalf("GetActivityLog failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.Type != EventStarted {
		t.Errorf("expected EventStarted, got %v", entry.Type)
	}
	if entry.Port != 8080 {
		t.Errorf("expected port 8080, got %d", entry.Port)
	}
	if entry.ProcessName != "web-server" {
		t.Errorf("expected processName 'web-server', got %q", entry.ProcessName)
	}
	if entry.ProjectName != "myproject" {
		t.Errorf("expected projectName 'myproject', got %q", entry.ProjectName)
	}
}

func TestFilterByProject(t *testing.T) {
	ctx, tmpDir, cleanup := newMockContext(t)
	defer cleanup()

	cfg := json.RawMessage(`{"db_path":"` + tmpDir + `/activity.db"}`)

	al := New()
	if err := al.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := al.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer al.Stop(ctx)

	// Emit events for different projects
	events := []kernel.Event{
		{
			Name: "process.started",
			Data: map[string]any{
				"port":        8080,
				"processName": "server1",
				"projectName": "project-a",
				"projectDir":  "/home/user/project-a",
			},
		},
		{
			Name: "process.started",
			Data: map[string]any{
				"port":        8081,
				"processName": "server2",
				"projectName": "project-b",
				"projectDir":  "/home/user/project-b",
			},
		},
	}

	for _, event := range events {
		if err := ctx.Kernel.EventBus().Emit(event, ctx); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	// Filter by project
	filter := ActivityFilter{
		ProjectName: "project-a",
		Limit:       100,
	}
	logs, err := al.GetActivityLog(filter)
	if err != nil {
		t.Fatalf("GetActivityLog failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry for project-a, got %d", len(logs))
	}

	if logs[0].ProjectName != "project-a" {
		t.Errorf("expected projectName 'project-a', got %q", logs[0].ProjectName)
	}
}

func TestClearHistory(t *testing.T) {
	ctx, tmpDir, cleanup := newMockContext(t)
	defer cleanup()

	cfg := json.RawMessage(`{"db_path":"` + tmpDir + `/activity.db"}`)

	al := New()
	if err := al.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := al.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer al.Stop(ctx)

	// Insert an event
	event := kernel.Event{
		Name: "process.started",
		Data: map[string]any{
			"port":        8080,
			"processName": "server",
			"projectName": "test",
			"projectDir":  "/test",
		},
	}

	if err := ctx.Kernel.EventBus().Emit(event, ctx); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify event is there
	logs, err := al.GetActivityLog(ActivityFilter{Limit: 100})
	if err != nil {
		t.Fatalf("GetActivityLog failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry before clear, got %d", len(logs))
	}

	// Clear history
	if err := al.ClearHistory(); err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}

	// Verify table is empty
	logs, err = al.GetActivityLog(ActivityFilter{Limit: 100})
	if err != nil {
		t.Fatalf("GetActivityLog after clear failed: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected 0 log entries after clear, got %d", len(logs))
	}
}

func TestRetentionCleanup(t *testing.T) {
	ctx, tmpDir, cleanup := newMockContext(t)
	defer cleanup()

	cfg := json.RawMessage(`{"db_path":"` + tmpDir + `/activity.db","retention_days":1}`)

	al := New()
	if err := al.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := al.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer al.Stop(ctx)

	// Manually insert an old event
	oldTime := time.Now().AddDate(0, 0, -40)
	if err := al.insertEvent(ActivityEvent{
		Type:        EventStarted,
		Port:        8080,
		ProcessName: "old-server",
		ProjectName: "test",
		ProjectDir:  "/test",
		Timestamp:   oldTime,
		Message:     "old event",
	}); err != nil {
		t.Fatalf("insertEvent failed: %v", err)
	}

	// Insert a recent event
	recentEvent := kernel.Event{
		Name: "process.started",
		Data: map[string]any{
			"port":        8081,
			"processName": "new-server",
			"projectName": "test",
			"projectDir":  "/test",
		},
	}

	if err := ctx.Kernel.EventBus().Emit(recentEvent, ctx); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Run cleanup
	if err := al.cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Verify old event is gone, recent is still there
	logs, err := al.GetActivityLog(ActivityFilter{Limit: 100})
	if err != nil {
		t.Fatalf("GetActivityLog failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry after retention cleanup, got %d", len(logs))
	}

	if logs[0].ProcessName != "new-server" {
		t.Errorf("expected 'new-server', got %q", logs[0].ProcessName)
	}
}

func TestEventCounts(t *testing.T) {
	ctx, tmpDir, cleanup := newMockContext(t)
	defer cleanup()

	cfg := json.RawMessage(`{"db_path":"` + tmpDir + `/activity.db"}`)

	al := New()
	if err := al.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := al.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer al.Stop(ctx)

	// Emit multiple event types
	events := []kernel.Event{
		{
			Name: "process.started",
			Data: map[string]any{
				"port":        8080,
				"processName": "s1",
				"projectName": "p1",
				"projectDir":  "/p1",
			},
		},
		{
			Name: "process.started",
			Data: map[string]any{
				"port":        8081,
				"processName": "s2",
				"projectName": "p2",
				"projectDir":  "/p2",
			},
		},
		{
			Name: "process.crashed",
			Data: map[string]any{
				"port":        8080,
				"processName": "s1",
				"projectName": "p1",
				"projectDir":  "/p1",
			},
		},
	}

	for _, event := range events {
		if err := ctx.Kernel.EventBus().Emit(event, ctx); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	counts := al.GetEventCounts()
	if counts[EventStarted] != 2 {
		t.Errorf("expected 2 started events, got %d", counts[EventStarted])
	}
	if counts[EventCrashed] != 1 {
		t.Errorf("expected 1 crashed event, got %d", counts[EventCrashed])
	}
}

func TestComponentInterface(t *testing.T) {
	al := New()

	if al.Name() != "activitylog" {
		t.Errorf("expected name 'activitylog', got %q", al.Name())
	}

	if al.Version() == "" {
		t.Errorf("expected non-empty version")
	}

	deps := al.Dependencies()
	if len(deps) > 0 {
		t.Errorf("expected no dependencies, got %v", deps)
	}

	hooks := al.Hooks()
	// Should have hooks for: process.started, process.stopped, process.crashed, process.unresponsive
	if len(hooks) < 4 {
		t.Errorf("expected at least 4 hooks, got %d", len(hooks))
	}
}

func TestPaginationAndOffset(t *testing.T) {
	ctx, tmpDir, cleanup := newMockContext(t)
	defer cleanup()

	cfg := json.RawMessage(`{"db_path":"` + tmpDir + `/activity.db"}`)

	al := New()
	if err := al.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := al.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer al.Stop(ctx)

	// Insert multiple events
	for i := 0; i < 10; i++ {
		event := kernel.Event{
			Name: "process.started",
			Data: map[string]any{
				"port":        8080 + i,
				"processName": "server",
				"projectName": "test",
				"projectDir":  "/test",
			},
		}
		if err := ctx.Kernel.EventBus().Emit(event, ctx); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	// Test pagination
	page1 := ActivityFilter{Limit: 5, Offset: 0}
	logs1, err := al.GetActivityLog(page1)
	if err != nil {
		t.Fatalf("GetActivityLog failed: %v", err)
	}
	if len(logs1) != 5 {
		t.Fatalf("expected 5 entries on page 1, got %d", len(logs1))
	}

	page2 := ActivityFilter{Limit: 5, Offset: 5}
	logs2, err := al.GetActivityLog(page2)
	if err != nil {
		t.Fatalf("GetActivityLog failed: %v", err)
	}
	if len(logs2) != 5 {
		t.Fatalf("expected 5 entries on page 2, got %d", len(logs2))
	}

	// Ensure pages don't overlap
	if logs1[0].ID == logs2[0].ID {
		t.Errorf("pages should not have overlapping entries")
	}
}
