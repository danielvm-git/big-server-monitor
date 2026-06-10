package processmonitor

import (
	"encoding/json"
	"os"
	"path/filepath"
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
			name: "DirectoryFallback",
			files: map[string]string{},
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
		startTime time.Time
		wantPrefix string
	}{
		{
			startTime: time.Now().Add(-2 * time.Hour),
			wantPrefix: "2h",
		},
		{
			startTime: time.Now().Add(-30 * time.Minute),
			wantPrefix: "30m",
		},
		{
			startTime: time.Now().Add(-45 * time.Second),
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
		key      string
		value    string
		wantVisible bool
		wantValue string
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
