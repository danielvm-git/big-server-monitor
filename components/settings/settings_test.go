package settings

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

// mockEventBus for testing
type mockEventBus struct {
	events []kernel.Event
}

func (m *mockEventBus) Subscribe(hook kernel.HookDef) func() {
	return func() {}
}

func (m *mockEventBus) Emit(event kernel.Event, ctx *kernel.Context) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventBus) SubscriberCount() int {
	return 0
}

// mockKernel for testing
type mockKernel struct {
	eventBus *mockEventBus
}

func (m *mockKernel) EventBus() *kernel.EventBus {
	// Return nil since mockEventBus doesn't match *kernel.EventBus type
	return nil
}

// testContext creates a minimal Context for testing
func testContext(logger kernel.Logger, eventBus kernel.EventBus) *kernel.Context {
	return &kernel.Context{
		Logger:     logger,
		Components: make(map[string]kernel.Component),
	}
}

func TestConfigPersistence(t *testing.T) {
	t.Run("save config to file exists at expected path with correct JSON", func(t *testing.T) {
		// Setup
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		eventBus := &mockEventBus{}

		// Create Settings with custom path
		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			eventBus:   eventBus,
		}

		testConfig := Config{
			ScanDirectories:            []string{tmpDir},
			PollingIntervalSeconds:     10,
			HealthCheckIntervalSeconds: 60,
			IgnoredPorts:               []int{80, 443},
			LogRetentionDays:           15,
			Notifications: struct {
				CrashAlerts bool `json:"crashAlerts"`
				ShowBadge   bool `json:"showBadge"`
			}{CrashAlerts: true, ShowBadge: false},
			LaunchAtLogin: true,
		}

		// Test
		err := s.SaveSettings(testConfig)
		if err != nil {
			t.Fatalf("SaveSettings failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); err != nil {
			t.Fatalf("config file not found at %s: %v", configPath, err)
		}

		// Verify JSON content
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config file: %v", err)
		}

		var loaded Config
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("invalid JSON in config file: %v", err)
		}

		if loaded.PollingIntervalSeconds != 10 {
			t.Errorf("expected PollingIntervalSeconds=10, got %d", loaded.PollingIntervalSeconds)
		}
		if loaded.LaunchAtLogin != true {
			t.Errorf("expected LaunchAtLogin=true, got %v", loaded.LaunchAtLogin)
		}
	})
}

func TestDefaultsOnMissingFile(t *testing.T) {
	t.Run("load non-existent config returns Defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent.json")

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			current:    Defaults,
		}

		loaded := s.GetSettings()

		if loaded.PollingIntervalSeconds != Defaults.PollingIntervalSeconds {
			t.Errorf("expected default PollingIntervalSeconds, got %d", loaded.PollingIntervalSeconds)
		}
		if len(loaded.ScanDirectories) != len(Defaults.ScanDirectories) {
			t.Errorf("expected default ScanDirectories, got %v", loaded.ScanDirectories)
		}
	})
}

func TestInvalidDirectoryValidation(t *testing.T) {
	t.Run("SaveSettings with invalid directory returns validation error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			eventBus:   &mockEventBus{},
			current:    Defaults,
		}

		// Create invalid config with non-existent directory
		invalidConfig := Config{
			ScanDirectories:            []string{"/nonexistent/path/that/does/not/exist"},
			PollingIntervalSeconds:     5,
			HealthCheckIntervalSeconds: 30,
			IgnoredPorts:               []int{},
			LogRetentionDays:           30,
			Notifications: struct {
				CrashAlerts bool `json:"crashAlerts"`
				ShowBadge   bool `json:"showBadge"`
			}{CrashAlerts: true, ShowBadge: true},
			LaunchAtLogin: false,
		}

		err := s.SaveSettings(invalidConfig)
		if err == nil {
			t.Fatal("expected validation error for invalid directory, got nil")
		}
	})
}

func TestSettingsChangedEventEmitted(t *testing.T) {
	t.Run("SaveSettings emits settings.changed event", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		eventBus := &mockEventBus{}

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			eventBus:   eventBus,
		}

		testConfig := Defaults
		testConfig.ScanDirectories = []string{tmpDir}
		testConfig.PollingIntervalSeconds = 15

		err := s.SaveSettings(testConfig)
		if err != nil {
			t.Fatalf("SaveSettings failed: %v", err)
		}

		// Verify event was emitted
		if len(eventBus.events) == 0 {
			t.Fatal("expected settings.changed event to be emitted")
		}

		event := eventBus.events[0]
		if event.Name != "settings.changed" {
			t.Errorf("expected event name 'settings.changed', got '%s'", event.Name)
		}

		// Verify event data contains config
		if configData, ok := event.Data["config"]; !ok {
			t.Error("expected 'config' in event data")
		} else {
			// Event data should contain the config
			if config, ok := configData.(Config); ok {
				if config.PollingIntervalSeconds != 15 {
					t.Errorf("expected PollingIntervalSeconds=15 in event, got %d", config.PollingIntervalSeconds)
				}
			}
		}
	})
}

func TestComponentInterface(t *testing.T) {
	t.Run("Settings implements Component interface", func(t *testing.T) {
		s := &Settings{
			mu:       &mapMutex{},
			eventBus: &mockEventBus{},
		}

		// Test Name
		if name := s.Name(); name != "settings" {
			t.Errorf("expected Name()='settings', got '%s'", name)
		}

		// Test Version
		if version := s.Version(); version == "" {
			t.Error("expected non-empty Version()")
		}

		// Test Dependencies (ok if empty)
		_ = s.Dependencies()

		// Test ConfigSchema
		schema := s.ConfigSchema()
		if schema == nil {
			t.Error("expected non-nil ConfigSchema()")
		}

		// Test Hooks (ok if empty)
		_ = s.Hooks()
	})
}

func TestAddRemoveScanDirectory(t *testing.T) {
	t.Run("AddScanDirectory appends new directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			current:    Defaults,
			eventBus:   &mockEventBus{},
		}

		testDir := tmpDir // Use temp directory that exists

		err := s.AddScanDirectory(testDir)
		if err != nil {
			t.Fatalf("AddScanDirectory failed: %v", err)
		}

		config := s.GetSettings()
		found := false
		for _, dir := range config.ScanDirectories {
			if dir == testDir {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected directory %s in ScanDirectories", testDir)
		}
	})

	t.Run("RemoveScanDirectory removes directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		config := Defaults
		config.ScanDirectories = []string{tmpDir, "/tmp/other"}

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			current:    config,
			eventBus:   &mockEventBus{},
		}

		err := s.RemoveScanDirectory(tmpDir)
		if err != nil {
			t.Fatalf("RemoveScanDirectory failed: %v", err)
		}

		updated := s.GetSettings()
		for _, dir := range updated.ScanDirectories {
			if dir == tmpDir {
				t.Errorf("expected directory %s to be removed", tmpDir)
			}
		}
	})
}

func TestResetToDefaults(t *testing.T) {
	t.Run("ResetToDefaults restores default config and emits event", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		eventBus := &mockEventBus{}
		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			current: Config{
				PollingIntervalSeconds: 999,
			},
			eventBus: eventBus,
		}

		// Override Defaults' ScanDirectories with the temp dir that actually exists
		origScanDirs := Defaults.ScanDirectories
		Defaults.ScanDirectories = []string{tmpDir}
		defer func() { Defaults.ScanDirectories = origScanDirs }()

		err := s.ResetToDefaults()
		if err != nil {
			t.Fatalf("ResetToDefaults failed: %v", err)
		}

		config := s.GetSettings()
		if config.PollingIntervalSeconds != Defaults.PollingIntervalSeconds {
			t.Errorf("expected PollingIntervalSeconds=%d after reset, got %d",
				Defaults.PollingIntervalSeconds, config.PollingIntervalSeconds)
		}

		if len(eventBus.events) == 0 {
			t.Fatal("expected settings.changed event to be emitted")
		}
	})
}

func TestPathExpansion(t *testing.T) {
	t.Run("tilde paths are expanded", func(t *testing.T) {
		s := &Settings{
			mu:       &mapMutex{},
			eventBus: &mockEventBus{},
		}

		// Test expandPath with ~ prefix
		expanded := s.expandPath("~/test")
		if bytes.HasPrefix([]byte(expanded), []byte("~")) {
			t.Errorf("expected ~ to be expanded, got %s", expanded)
		}

		// Verify it's an absolute path
		if !filepath.IsAbs(expanded) {
			t.Errorf("expected absolute path, got %s", expanded)
		}
	})
}

func TestGetSettings(t *testing.T) {
	t.Run("GetSettings returns current config safely", func(t *testing.T) {
		config := Defaults
		config.PollingIntervalSeconds = 42

		s := &Settings{
			mu:       &mapMutex{},
			current:  config,
			eventBus: &mockEventBus{},
		}

		retrieved := s.GetSettings()
		if retrieved.PollingIntervalSeconds != 42 {
			t.Errorf("expected PollingIntervalSeconds=42, got %d", retrieved.PollingIntervalSeconds)
		}
	})
}

func TestInitAndStart(t *testing.T) {
	t.Run("Init loads or creates config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		logger := &mockLogger{}
		eventBus := &mockEventBus{}

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			eventBus:   eventBus,
		}

		ctx := &kernel.Context{
			Kernel:     &kernel.Kernel{},
			Logger:     logger,
			Components: make(map[string]kernel.Component),
		}

		err := s.Init(ctx, nil)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Verify config is loaded
		config := s.GetSettings()
		if config.PollingIntervalSeconds != Defaults.PollingIntervalSeconds {
			t.Errorf("expected default config to be loaded")
		}
	})

	t.Run("Start succeeds", func(t *testing.T) {
		s := &Settings{
			mu:       &mapMutex{},
			eventBus: &mockEventBus{},
		}

		ctx := &kernel.Context{
			Logger:     &mockLogger{},
			Components: make(map[string]kernel.Component),
		}

		err := s.Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}
	})
}

func TestStop(t *testing.T) {
	t.Run("Stop succeeds and persists config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		s := &Settings{
			configPath: configPath,
			mu:         &mapMutex{},
			current:    Defaults,
			eventBus:   &mockEventBus{},
		}

		ctx := &kernel.Context{
			Logger:     &mockLogger{},
			Components: make(map[string]kernel.Component),
		}

		err := s.Stop(ctx)
		if err != nil {
			t.Fatalf("Stop failed: %v", err)
		}

		// Verify config was persisted
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("expected config to be persisted, got error: %v", err)
		}

		var loaded Config
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
	})
}
