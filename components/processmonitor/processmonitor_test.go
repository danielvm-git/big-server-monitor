package processmonitor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"portkeeper/kernel"
)

// mockLogger for testing
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Info(msg string, args ...any)  { m.messages = append(m.messages, msg) }
func (m *mockLogger) Warn(msg string, args ...any)  { m.messages = append(m.messages, msg) }
func (m *mockLogger) Error(msg string, args ...any) { m.messages = append(m.messages, msg) }
func (m *mockLogger) Debug(msg string, args ...any) { m.messages = append(m.messages, msg) }

// mockDiscovery implements PortDiscovery for testing.
type mockDiscovery struct {
	ports []int
	info  processInfo
	err   error
}

func (m *mockDiscovery) ListeningPorts() ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.ports, nil
}

func (m *mockDiscovery) ProcessInfo(port int) (processInfo, error) {
	if m.err != nil {
		return processInfo{}, m.err
	}
	return m.info, nil
}

// TestComponentInterface verifies ProcessMonitor implements the Component interface
func TestComponentInterface(t *testing.T) {
	pm := &ProcessMonitor{}

	t.Run("Name", func(t *testing.T) {
		name := pm.Name()
		if name != "processmonitor" {
			t.Errorf("expected name 'processmonitor', got %q", name)
		}
	})

	t.Run("Version", func(t *testing.T) {
		version := pm.Version()
		if version == "" {
			t.Error("expected non-empty version")
		}
	})

	t.Run("Dependencies", func(t *testing.T) {
		deps := pm.Dependencies()
		if deps == nil {
			t.Error("expected non-nil dependencies slice")
		}
	})

	t.Run("ConfigSchema", func(t *testing.T) {
		schema := pm.ConfigSchema()
		if schema == nil {
			t.Error("expected non-nil config schema")
		}
	})

	t.Run("Hooks", func(t *testing.T) {
		hooks := pm.Hooks()
		if hooks == nil {
			t.Error("expected non-nil hooks slice")
		}
	})
}

// TestInit verifies component initialization
func TestInit(t *testing.T) {
	t.Run("InitWithoutConfig", func(t *testing.T) {
		pm := &ProcessMonitor{}
		ctx := &kernel.Context{
			Logger: &mockLogger{},
		}
		err := pm.Init(ctx, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if pm.config.PollingIntervalSec == 0 {
			t.Error("expected config to be initialized with default polling interval")
		}
	})

	t.Run("InitWithConfig", func(t *testing.T) {
		pm := &ProcessMonitor{}
		ctx := &kernel.Context{
			Logger: &mockLogger{},
		}
		config := json.RawMessage([]byte(`{
			"pollingIntervalSec": 3,
			"scanDirectories": ["/tmp"],
			"ignoredPorts": [80, 443]
		}`))
		err := pm.Init(ctx, config)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if pm.config.PollingIntervalSec != 3 {
			t.Errorf("expected polling interval 3, got %d", pm.config.PollingIntervalSec)
		}
	})
}

// TestProjectNameDetection verifies detection from common markers
func TestProjectNameDetection(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // filename -> content
		wantName string
	}{
		{
			name: "NodePackageJSON",
			files: map[string]string{
				"package.json": `{"name":"my-app"}`,
			},
			wantName: "my-app",
		},
		{
			name: "GoMod",
			files: map[string]string{
				"go.mod": `module github.com/user/myproject`,
			},
			wantName: "github.com/user/myproject",
		},
		{
			name:     "DirectoryFallback",
			files:    map[string]string{},
			wantName: "test-dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			// Create subdirectory to detect
			testDir := filepath.Join(tempDir, "test-dir")
			if err := os.Mkdir(testDir, 0755); err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}

			// Create marker files
			for filename, content := range tt.files {
				path := filepath.Join(testDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write %s: %v", filename, err)
				}
			}

			pm := &ProcessMonitor{}
			name := pm.detectProjectName(testDir)
			if name != tt.wantName {
				t.Errorf("expected %q, got %q", tt.wantName, name)
			}
		})
	}
}

// TestUptimeFormat verifies uptime string formatting
func TestUptimeFormat(t *testing.T) {
	tests := []struct {
		startTime  time.Time
		wantPrefix string
	}{
		{
			startTime:  time.Now().Add(-2 * time.Hour),
			wantPrefix: "2h",
		},
		{
			startTime:  time.Now().Add(-30 * time.Minute),
			wantPrefix: "30m",
		},
		{
			startTime:  time.Now().Add(-45 * time.Second),
			wantPrefix: "45s",
		},
	}

	for _, tt := range tests {
		uptime := formatUptime(tt.startTime)
		if !startsWith(uptime, tt.wantPrefix) {
			t.Errorf("expected uptime to start with %q, got %q", tt.wantPrefix, uptime)
		}
	}
}

// TestGetServers returns list of currently monitored servers
func TestGetServers(t *testing.T) {
	pm := &ProcessMonitor{
		mu:      &lockInfo{},
		servers: make(map[int]*Server),
	}

	// Add test servers
	pm.servers[8080] = &Server{Port: 8080, Status: StatusOnline, ProcessName: "node"}
	pm.servers[3000] = &Server{Port: 3000, Status: StatusOnline, ProcessName: "python"}

	servers := pm.GetServers()
	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}
}

// TestStatusConstants verifies status enum values
func TestStatusConstants(t *testing.T) {
	if StatusOnline != "online" {
		t.Errorf("expected StatusOnline='online', got %q", StatusOnline)
	}
	if StatusOffline != "offline" {
		t.Errorf("expected StatusOffline='offline', got %q", StatusOffline)
	}
	if StatusUnknown != "unknown" {
		t.Errorf("expected StatusUnknown='unknown', got %q", StatusUnknown)
	}
}

// TestEnvVarRedaction verifies safe keys are shown and others redacted
func TestEnvVarRedaction(t *testing.T) {
	tests := []struct {
		key         string
		value       string
		wantVisible bool
		wantValue   string
	}{
		{"NODE_ENV", "production", true, "production"},
		{"PORT", "3000", true, "3000"},
		{"DATABASE_PASSWORD", "secret123", false, "***"},
		{"AWS_SECRET_ACCESS_KEY", "key", false, "***"},
		{"CUSTOM_VAR", "value", false, "***"},
	}

	for _, tt := range tests {
		visible, value := redactEnvVar(tt.key, tt.value)
		if visible != tt.wantVisible {
			t.Errorf("%s: expected visible=%v, got %v", tt.key, tt.wantVisible, visible)
		}
		if value != tt.wantValue {
			t.Errorf("%s: expected value=%q, got %q", tt.key, tt.wantValue, value)
		}
	}
}

// Helper function
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func mockContext() *kernel.Context {
	return &kernel.Context{
		Kernel: kernel.New(&mockLogger{}),
		Logger: &mockLogger{},
	}
}

func TestPollDiscoversNewPorts(t *testing.T) {
	md := &mockDiscovery{
		ports: []int{3000, 8080},
		info: processInfo{
			PID:         1234,
			ProcessName: "node",
			BinaryPath:  "/usr/local/bin/node",
			WorkingDir:  "/app",
			MemoryMB:    50.0,
			StartTime:   time.Now(),
		},
	}

	pm := &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         make(map[int]*Server),
		previousServers: make(map[int]*Server),
		runtimeCache:    make(map[int]string),
		discovery:       md,
		config: Config{
			PollingIntervalSec: 5,
		},
	}

	ctx := mockContext()
	pm.poll(ctx)

	servers := pm.GetServers()
	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}
}

func TestPollIgnoresPorts(t *testing.T) {
	md := &mockDiscovery{
		ports: []int{443, 3000},
		info: processInfo{
			PID:         1234,
			ProcessName: "test",
			StartTime:   time.Now(),
		},
	}

	pm := &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         make(map[int]*Server),
		previousServers: make(map[int]*Server),
		runtimeCache:    make(map[int]string),
		discovery:       md,
		config: Config{
			PollingIntervalSec: 5,
			IgnoredPorts:       []int{443},
		},
	}

	ctx := mockContext()
	pm.poll(ctx)

	servers := pm.GetServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 server (443 ignored), got %d", len(servers))
	}
	if len(servers) > 0 && servers[0].Port != 3000 {
		t.Errorf("expected port 3000, got %d", servers[0].Port)
	}
}

func TestPollHandlesDiscoveryError(t *testing.T) {
	md := &mockDiscovery{
		err: fmt.Errorf("lsof unavailable"),
	}

	prevServer := &Server{Port: 8080, Status: StatusOnline, ProcessName: "node"}
	pm := &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         map[int]*Server{8080: prevServer},
		previousServers: map[int]*Server{8080: prevServer},
		runtimeCache:    make(map[int]string),
		discovery:       md,
		config: Config{
			PollingIntervalSec: 5,
		},
	}

	ctx := mockContext()
	pm.poll(ctx)

	// servers should be unchanged
	servers := pm.GetServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 server preserved from error, got %d", len(servers))
	}
}

func TestPollPreservesServersOnEmptyResult(t *testing.T) {
	md := &mockDiscovery{
		ports: []int{},
	}

	prevServer := &Server{Port: 3000, Status: StatusOnline, ProcessName: "python"}
	pm := &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         map[int]*Server{3000: prevServer},
		previousServers: map[int]*Server{3000: prevServer},
		runtimeCache:    make(map[int]string),
		discovery:       md,
		config: Config{
			PollingIntervalSec: 5,
		},
	}

	ctx := mockContext()
	pm.poll(ctx)

	servers := pm.GetServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 server preserved from empty result, got %d", len(servers))
	}
}

func TestDiffAndEmitProcessStarted(t *testing.T) {
	k := kernel.New(&mockLogger{})
	receivedEvents := make([]kernel.Event, 0)
	var mu sync.Mutex

	k.EventBus().Subscribe(kernel.HookDef{
		Name:     "process.started",
		Priority: 0,
		Handler: func(ctx *kernel.Context, event kernel.Event) error {
			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()
			return nil
		},
	})

	ctx := &kernel.Context{
		Kernel: k,
		Logger: &mockLogger{},
	}

	md := &mockDiscovery{
		ports: []int{3000},
		info: processInfo{
			PID:         9999,
			ProcessName: "test-server",
			StartTime:   time.Now(),
		},
	}

	pm := &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         make(map[int]*Server),
		previousServers: make(map[int]*Server),
		runtimeCache:    make(map[int]string),
		discovery:       md,
		config: Config{
			PollingIntervalSec: 5,
		},
	}

	pm.poll(ctx)

	mu.Lock()
	count := len(receivedEvents)
	mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 process.started event, got %d", count)
	}
}

func TestDiffAndEmitProcessStopped(t *testing.T) {
	k := kernel.New(&mockLogger{})
	receivedEvents := make([]kernel.Event, 0)
	var mu sync.Mutex

	k.EventBus().Subscribe(kernel.HookDef{
		Name:     "process.stopped",
		Priority: 0,
		Handler: func(ctx *kernel.Context, event kernel.Event) error {
			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()
			return nil
		},
	})

	ctx := &kernel.Context{
		Kernel: k,
		Logger: &mockLogger{},
	}

	md := &mockDiscovery{
		ports: []int{}, // no ports discovered
	}

	prevServer := &Server{Port: 8080, Status: StatusOnline, ProcessName: "old-node", StartedAt: time.Now().Add(-1 * time.Hour)}
	pm := &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         map[int]*Server{8080: prevServer},
		previousServers: map[int]*Server{8080: prevServer},
		runtimeCache:    make(map[int]string),
		discovery:       md,
		config: Config{
			PollingIntervalSec: 5,
		},
	}

	pm.poll(ctx)

	mu.Lock()
	count := len(receivedEvents)
	mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 process.stopped event, got %d", count)
	}
}

func TestKillProcessSuccess(t *testing.T) {
	// Start a sleep process we can kill
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("could not start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer func() { _ = cmd.Process.Kill() }()

	pm := &ProcessMonitor{}
	err := pm.KillProcess(pid)
	if err != nil {
		t.Errorf("expected no error killing PID %d, got %v", pid, err)
	}
}

func TestKillProcessNonexistent(t *testing.T) {
	pm := &ProcessMonitor{}
	err := pm.KillProcess(99999)
	if err == nil {
		t.Error("expected error for nonexistent PID")
	}
}

func TestGetServerByPort(t *testing.T) {
	pm := &ProcessMonitor{
		mu:      &lockInfo{},
		servers: make(map[int]*Server),
	}

	srv := &Server{Port: 3000, Status: StatusOnline, ProcessName: "node"}
	pm.servers[3000] = srv

	t.Run("found", func(t *testing.T) {
		got, err := pm.GetServerByPort(3000)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.Port != 3000 {
			t.Errorf("expected port 3000, got %d", got.Port)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := pm.GetServerByPort(9999)
		if err == nil {
			t.Error("expected error for unmatched port")
		}
	})
}

func TestGetServersDeepCopy(t *testing.T) {
	pm := &ProcessMonitor{
		mu:      &lockInfo{},
		servers: make(map[int]*Server),
	}
	pm.servers[8080] = &Server{Port: 8080, Status: StatusOnline, ProcessName: "node"}

	servers := pm.GetServers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	// Mutate the returned slice
	servers[0].Port = 9999

	// Internal state should be unchanged
	internal, err := pm.GetServerByPort(8080)
	if err != nil {
		t.Fatalf("internal server should still exist: %v", err)
	}
	if internal.Port != 8080 {
		t.Errorf("expected internal port 8080, got %d — deep copy was not made", internal.Port)
	}
}

func TestGetMonitorStatus(t *testing.T) {
	pm := New()

	// Before any poll, status should be default (healthy=false)
	status := pm.GetMonitorStatus()
	if status.Healthy {
		t.Error("expected Healthy=false before first poll")
	}

	// After successful poll, should be healthy
	mock := &mockDiscovery{
		ports: []int{3000},
		info:  processInfo{PID: 12345, ProcessName: "node"},
	}
	pm.discovery = mock
	ctx := mockContext()
	if err := pm.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	pm.poll(ctx)

	status = pm.GetMonitorStatus()
	if !status.Healthy {
		t.Error("expected Healthy=true after successful poll")
	}
	if status.ServerCount != 1 {
		t.Errorf("expected ServerCount=1, got %d", status.ServerCount)
	}
	if status.LastPollAt == "" {
		t.Error("expected LastPollAt to be set")
	}
}

func TestGetMonitorStatusWithError(t *testing.T) {
	pm := New()
	ctx := mockContext()
	if err := pm.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Poll with error
	mock := &mockDiscovery{err: fmt.Errorf("lsof not available")}
	pm.discovery = mock
	pm.poll(ctx)

	status := pm.GetMonitorStatus()
	if status.Healthy {
		t.Error("expected Healthy=false after error")
	}
	if status.LastError == "" {
		t.Error("expected LastError to be set")
	}
}
