package notifications

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"portkeeper/kernel"
)

// MockLogger for testing
type MockLogger struct {
	mu   sync.Mutex
	logs []string
}

func (ml *MockLogger) Info(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, "INFO: "+msg)
}

func (ml *MockLogger) Warn(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, "WARN: "+msg)
}

func (ml *MockLogger) Error(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, "ERROR: "+msg)
}

func (ml *MockLogger) Debug(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, "DEBUG: "+msg)
}

// MockNotifier for testing
type MockNotifier struct {
	mu           sync.Mutex
	notifications []struct {
		title   string
		message string
	}
}

func (mn *MockNotifier) Notify(ctx context.Context, title, message string) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()
	mn.notifications = append(mn.notifications, struct {
		title   string
		message string
	}{title, message})
	return nil
}

// MockEventBus for testing
type MockEventBus struct {
	mu          sync.Mutex
	emittedEvents []kernel.Event
}

func (meb *MockEventBus) Subscribe(hook kernel.HookDef) func() {
	return func() {}
}

func (meb *MockEventBus) Emit(event kernel.Event, ctx *kernel.Context) error {
	meb.mu.Lock()
	defer meb.mu.Unlock()
	meb.emittedEvents = append(meb.emittedEvents, event)
	return nil
}

// MockKernel for testing
type MockKernel struct {
	eventBus *MockEventBus
}

func (mk *MockKernel) EventBus() *MockEventBus {
	return mk.eventBus
}

// TestComponentInterface verifies the Component interface is implemented
func TestComponentInterface(t *testing.T) {
	n := New()
	var _ kernel.Component = n
}

func TestName(t *testing.T) {
	n := New()
	if got := n.Name(); got != "notifications" {
		t.Errorf("Name() = %q, want %q", got, "notifications")
	}
}

func TestVersion(t *testing.T) {
	n := New()
	if got := n.Version(); got != "0.1.0" {
		t.Errorf("Version() = %q, want %q", got, "0.1.0")
	}
}

func TestDependencies(t *testing.T) {
	n := New()
	deps := n.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Dependencies() = %v, want []", deps)
	}
}

func TestConfigSchema(t *testing.T) {
	n := New()
	schema := n.ConfigSchema()
	if schema == nil {
		t.Error("ConfigSchema() returned nil, want non-nil json.RawMessage")
	}
}

func TestInit(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		n := New()
		ctx := &kernel.Context{
			Logger: &MockLogger{},
		}
		cfg := json.RawMessage(`{"crashAlerts": true}`)
		err := n.Init(ctx, cfg)
		if err != nil {
			t.Fatalf("Init() failed: %v", err)
		}
		if !n.config.CrashAlerts {
			t.Errorf("CrashAlerts not set from config")
		}
	})

	t.Run("defaults when no config", func(t *testing.T) {
		n := New()
		ctx := &kernel.Context{
			Logger: &MockLogger{},
		}
		err := n.Init(ctx, nil)
		if err != nil {
			t.Fatalf("Init() failed: %v", err)
		}
		if !n.config.CrashAlerts {
			t.Errorf("CrashAlerts should default to true")
		}
	})
}

func TestStart(t *testing.T) {
	n := New()
	mockLogger := &MockLogger{}
	ctx := &kernel.Context{
		Logger: mockLogger,
	}

	// Init must be called before Start
	err := n.Init(ctx, nil)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	err = n.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
}

func TestStop(t *testing.T) {
	n := New()
	mockLogger := &MockLogger{}
	ctx := &kernel.Context{
		Logger: mockLogger,
	}

	// Init must be called before Stop
	err := n.Init(ctx, nil)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	err = n.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
}

func TestHooks(t *testing.T) {
	n := New()
	hooks := n.Hooks()
	if len(hooks) == 0 {
		t.Error("Hooks() returned empty, want hooks for process.crashed event")
	}

	// Verify process.crashed hook is registered
	hasCrashHook := false
	for _, h := range hooks {
		if h.Name == "process.crashed" {
			hasCrashHook = true
			break
		}
	}
	if !hasCrashHook {
		t.Error("process.crashed hook not registered")
	}
}

func TestNotificationRespectsSetting(t *testing.T) {
	n := New()

	mockLogger := &MockLogger{}
	mockNotifier := &MockNotifier{}
	n.notifier = mockNotifier

	mockKernel := &kernel.Kernel{}
	ctx := &kernel.Context{
		Logger: mockLogger,
		Kernel: mockKernel,
	}

	// Init to set up the logger with crashAlerts=false
	if err := n.Init(ctx, json.RawMessage(`{"crashAlerts": false}`)); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Find the crash handler
	hooks := n.Hooks()
	var crashHandler kernel.HookFunc
	for _, h := range hooks {
		if h.Name == "process.crashed" {
			crashHandler = h.Handler
			break
		}
	}
	if crashHandler == nil {
		t.Fatal("crash handler not found")
	}

	// Emit a crash event
	event := kernel.Event{
		Name: "process.crashed",
		Data: map[string]any{
			"port": 3000,
			"processName": "node",
			"projectName": "bigbase-api",
			"uptimeStr": "2h 03m",
		},
	}

	err := crashHandler(ctx, event)
	if err != nil {
		t.Fatalf("crash handler failed: %v", err)
	}

	// Verify no notification was delivered (crashAlerts=false)
	mockNotifier.mu.Lock()
	notificationCount := len(mockNotifier.notifications)
	mockNotifier.mu.Unlock()

	if notificationCount > 0 {
		t.Errorf("Expected no notifications when crashAlerts=false, got %d", notificationCount)
	}
}

func TestRateLimiting(t *testing.T) {
	n := New()
	n.config.CrashAlerts = true

	mockLogger := &MockLogger{}
	mockNotifier := &MockNotifier{}
	n.notifier = mockNotifier

	mockKernel := &kernel.Kernel{}
	ctx := &kernel.Context{
		Logger: mockLogger,
		Kernel: mockKernel,
	}

	// Init to set up the logger
	if err := n.Init(ctx, nil); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Find the crash handler
	hooks := n.Hooks()
	var crashHandler kernel.HookFunc
	for _, h := range hooks {
		if h.Name == "process.crashed" {
			crashHandler = h.Handler
			break
		}
	}
	if crashHandler == nil {
		t.Fatal("crash handler not found")
	}

	port := 3000

	// First crash on port 3000
	event1 := kernel.Event{
		Name: "process.crashed",
		Data: map[string]any{
			"port": port,
			"processName": "node",
			"projectName": "bigbase-api",
			"uptimeStr": "2h 03m",
		},
	}

	err := crashHandler(ctx, event1)
	if err != nil {
		t.Fatalf("crash handler failed: %v", err)
	}

	mockNotifier.mu.Lock()
	notificationCount1 := len(mockNotifier.notifications)
	mockNotifier.mu.Unlock()

	if notificationCount1 != 1 {
		t.Errorf("Expected 1 notification after first crash, got %d", notificationCount1)
	}

	// Second crash on same port within 60s
	event2 := kernel.Event{
		Name: "process.crashed",
		Data: map[string]any{
			"port": port,
			"processName": "node",
			"projectName": "bigbase-api",
			"uptimeStr": "10m",
		},
	}

	err = crashHandler(ctx, event2)
	if err != nil {
		t.Fatalf("crash handler failed: %v", err)
	}

	// Rate limited - no additional notification should be delivered
	mockNotifier.mu.Lock()
	notificationCount2 := len(mockNotifier.notifications)
	mockNotifier.mu.Unlock()

	if notificationCount2 != 1 {
		t.Errorf("Expected 1 notification after rate-limited crash, got %d", notificationCount2)
	}
}

func TestRateLimitingExpiry(t *testing.T) {
	n := New()
	n.config.CrashAlerts = true
	n.rateLimiter.cooldown = 100 * time.Millisecond // Short cooldown for testing

	mockLogger := &MockLogger{}
	mockNotifier := &MockNotifier{}
	n.notifier = mockNotifier

	mockKernel := &kernel.Kernel{}
	ctx := &kernel.Context{
		Logger: mockLogger,
		Kernel: mockKernel,
	}

	// Init to set up the logger
	if err := n.Init(ctx, nil); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Find the crash handler
	hooks := n.Hooks()
	var crashHandler kernel.HookFunc
	for _, h := range hooks {
		if h.Name == "process.crashed" {
			crashHandler = h.Handler
			break
		}
	}
	if crashHandler == nil {
		t.Fatal("crash handler not found")
	}

	port := 3000

	// First crash
	event1 := kernel.Event{
		Name: "process.crashed",
		Data: map[string]any{
			"port": port,
			"processName": "node",
			"projectName": "bigbase-api",
			"uptimeStr": "2h 03m",
		},
	}

	err := crashHandler(ctx, event1)
	if err != nil {
		t.Fatalf("crash handler failed: %v", err)
	}

	mockNotifier.mu.Lock()
	notificationCount1 := len(mockNotifier.notifications)
	mockNotifier.mu.Unlock()

	if notificationCount1 != 1 {
		t.Errorf("Expected 1 notification after first crash, got %d", notificationCount1)
	}

	// Wait for rate limit to expire
	time.Sleep(150 * time.Millisecond)

	// Second crash after cooldown expired
	event2 := kernel.Event{
		Name: "process.crashed",
		Data: map[string]any{
			"port": port,
			"processName": "node",
			"projectName": "bigbase-api",
			"uptimeStr": "15m",
		},
	}

	err = crashHandler(ctx, event2)
	if err != nil {
		t.Fatalf("crash handler failed: %v", err)
	}

	// Should allow notification after cooldown expires
	mockNotifier.mu.Lock()
	notificationCount2 := len(mockNotifier.notifications)
	mockNotifier.mu.Unlock()

	if notificationCount2 != 2 {
		t.Errorf("Expected 2 notifications after cooldown expires, got %d", notificationCount2)
	}
}

func TestRequestPermission(t *testing.T) {
	n := New()
	mockLogger := &MockLogger{}
	ctx := &kernel.Context{
		Logger: mockLogger,
	}

	// Init to set up the logger
	if err := n.Init(ctx, nil); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	err := n.RequestPermission(ctx)
	if err != nil {
		t.Fatalf("RequestPermission() failed: %v", err)
	}
}

func TestHasPermission(t *testing.T) {
	n := New()
	// Should return false initially
	if n.HasPermission() {
		t.Error("HasPermission() should return false initially")
	}

	// After requesting permission, it should be true
	mockLogger := &MockLogger{}
	ctx := &kernel.Context{
		Logger: mockLogger,
	}
	// Init must be called first to set up the logger
	_ = n.Init(ctx, nil)
	_ = n.RequestPermission(ctx)

	if !n.HasPermission() {
		t.Error("HasPermission() should return true after RequestPermission()")
	}
}
