package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"portkeeper/kernel"
)

const version = "0.1.0"

// Config represents the application configuration schema
type Config struct {
	ScanDirectories            []string `json:"scanDirectories"`
	PollingIntervalSeconds     int      `json:"pollingIntervalSeconds"`
	HealthCheckIntervalSeconds int      `json:"healthCheckIntervalSeconds"`
	IgnoredPorts               []int    `json:"ignoredPorts"`
	LogRetentionDays           int      `json:"logRetentionDays"`
	Notifications              struct {
		CrashAlerts bool `json:"crashAlerts"`
		ShowBadge   bool `json:"showBadge"`
	} `json:"notifications"`
	LaunchAtLogin bool `json:"launchAtLogin"`
}

// Defaults provides the default configuration
var Defaults = Config{
	ScanDirectories:            []string{"~/projects", "~/Developer", "~/opensrc"},
	PollingIntervalSeconds:     5,
	HealthCheckIntervalSeconds: 30,
	IgnoredPorts:               []int{80, 443, 5432, 3306, 6379, 27017},
	LogRetentionDays:           30,
	Notifications: struct {
		CrashAlerts bool `json:"crashAlerts"`
		ShowBadge   bool `json:"showBadge"`
	}{CrashAlerts: true, ShowBadge: true},
	LaunchAtLogin: false,
}

// mapMutex wraps sync.RWMutex to make it compatible with the Settings struct
type mapMutex struct {
	mu sync.RWMutex
}

// EventBusInterface for dependency injection
type EventBusInterface interface {
	Emit(event kernel.Event, ctx *kernel.Context) error
}

// Settings is the settings component
type Settings struct {
	configPath string
	current    Config
	mu         *mapMutex
	eventBus   EventBusInterface
}

// New creates a new Settings component
func New(configPath string) *Settings {
	return &Settings{
		configPath: configPath,
		current:    Defaults,
		mu:         &mapMutex{},
	}
}

// Name returns the component name
func (s *Settings) Name() string {
	return "settings"
}

// Version returns the component version
func (s *Settings) Version() string {
	return version
}

// Dependencies returns the list of dependencies
func (s *Settings) Dependencies() []string {
	return []string{}
}

// ConfigSchema returns the JSON schema for configuration
func (s *Settings) ConfigSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scanDirectories": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
			"pollingIntervalSeconds": map[string]any{
				"type": "integer",
			},
			"healthCheckIntervalSeconds": map[string]any{
				"type": "integer",
			},
			"ignoredPorts": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "integer",
				},
			},
			"logRetentionDays": map[string]any{
				"type": "integer",
			},
			"notifications": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"crashAlerts": map[string]any{
						"type": "boolean",
					},
					"showBadge": map[string]any{
						"type": "boolean",
					},
				},
			},
			"launchAtLogin": map[string]any{
				"type": "boolean",
			},
		},
	}
	data, _ := json.Marshal(schema)
	return data
}

// Hooks returns the list of hooks
func (s *Settings) Hooks() []kernel.HookDef {
	return []kernel.HookDef{}
}

// Init initializes the settings component
func (s *Settings) Init(ctx *kernel.Context, config json.RawMessage) error {
	// Initialize config path if not already set
	if s.configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		s.configPath = filepath.Join(home, ".config/portkeeper/config.json")
	}

	s.eventBus = ctx.Kernel.EventBus()

	// Try to load existing config
	err := s.loadConfig()
	if err != nil && !os.IsNotExist(err) {
		ctx.Logger.Warn("failed to load config, using defaults", "error", err)
		s.current = Defaults
	} else if os.IsNotExist(err) {
		ctx.Logger.Info("config file not found, using defaults")
		s.current = Defaults
	}

	ctx.Logger.Info("settings initialized", "path", s.configPath)
	return nil
}

// Start starts the settings component
func (s *Settings) Start(ctx *kernel.Context) error {
	ctx.Logger.Info("settings started")
	return nil
}

// Stop stops the settings component and persists config
func (s *Settings) Stop(ctx *kernel.Context) error {
	if err := s.persistConfig(); err != nil {
		ctx.Logger.Error("failed to persist config on stop", "error", err)
		return err
	}
	ctx.Logger.Info("settings stopped")
	return nil
}

// GetSettings returns the current settings (thread-safe)
func (s *Settings) GetSettings() Config {
	s.mu.mu.RLock()
	defer s.mu.mu.RUnlock()
	return s.current
}

// SaveSettings validates and saves the configuration
func (s *Settings) SaveSettings(config Config) error {
	// Validate that all scan directories exist
	for _, dir := range config.ScanDirectories {
		expanded := s.expandPath(dir)
		if _, err := os.Stat(expanded); err != nil {
			return fmt.Errorf("invalid directory: %w", err)
		}
	}

	// Update current config
	s.mu.mu.Lock()
	s.current = config
	s.mu.mu.Unlock()

	// Persist to disk
	if err := s.persistConfig(); err != nil {
		return fmt.Errorf("persist config: %w", err)
	}

	// Emit settings.changed event
	event := kernel.Event{
		Name: "settings.changed",
		Data: map[string]any{
			"config": config,
		},
	}

	// We need a minimal context for EventBus.Emit since it requires a context
	ctx := &kernel.Context{
		Logger: &noopLogger{},
	}

	if s.eventBus != nil {
		if err := s.eventBus.Emit(event, ctx); err != nil {
			return fmt.Errorf("emit event: %w", err)
		}
	}

	return nil
}

// ResetToDefaults resets configuration to defaults
func (s *Settings) ResetToDefaults() error {
	return s.SaveSettings(Defaults)
}

// AddScanDirectory adds a directory to scan list
func (s *Settings) AddScanDirectory(dir string) error {
	expanded := s.expandPath(dir)
	if _, err := os.Stat(expanded); err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	s.mu.mu.Lock()
	s.current.ScanDirectories = append(s.current.ScanDirectories, dir)
	s.mu.mu.Unlock()

	return s.persistConfig()
}

// RemoveScanDirectory removes a directory from scan list
func (s *Settings) RemoveScanDirectory(dir string) error {
	s.mu.mu.Lock()
	filtered := []string{}
	for _, d := range s.current.ScanDirectories {
		if d != dir {
			filtered = append(filtered, d)
		}
	}
	s.current.ScanDirectories = filtered
	s.mu.mu.Unlock()

	return s.persistConfig()
}

// loadConfig loads configuration from disk
func (s *Settings) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	s.mu.mu.Lock()
	defer s.mu.mu.Unlock()
	s.current = config

	return nil
}

// persistConfig persists configuration to disk
func (s *Settings) persistConfig() error {
	s.mu.mu.RLock()
	defer s.mu.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.configPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// expandPath expands ~ to home directory
func (s *Settings) expandPath(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if len(path) > 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// noopLogger is a no-op logger for internal use
type noopLogger struct{}

func (n *noopLogger) Info(msg string, args ...any)  {}
func (n *noopLogger) Warn(msg string, args ...any)  {}
func (n *noopLogger) Error(msg string, args ...any) {}
func (n *noopLogger) Debug(msg string, args ...any) {}

// Verify Settings implements kernel.Component
var _ kernel.Component = (*Settings)(nil)
