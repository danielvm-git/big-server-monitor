# Slice 7: Settings Component — "Configure It"

**type:** epic  
**status:** planning  
**verify:** changing polling interval in settings persists to disk and takes effect on next poll cycle without restart

## Purpose

Manages user configuration: persists settings to `~/.config/portkeeper/config.json`, validates schema, and emits `settings.changed` so all components can react without restart.

## Scope

- Load config on startup (create defaults if missing)
- Validate config against schema
- `SaveSettings(config)` binding — persists and emits `settings.changed`
- `ResetToDefaults()` binding
- `GetSettings()` binding
- Handle `launchAtLogin` toggle by calling macOS `SMAppService`
- Expand `~` in directory paths on read

## Config schema (full)

```go
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
```

## Launch at login (macOS 13+)

```go
import "github.com/progrium/macdriver/macos/servicemanagement"

func setLaunchAtLogin(enabled bool) error {
    svc := servicemanagement.SMAppService_MainAppService()
    if enabled {
        return svc.RegisterAndReturnError(nil)
    }
    return svc.UnregisterAndReturnError(nil)
}
```

## Events emitted

```go
// settings.changed — all components reload their slice of the config
type SettingsChangedEvent struct {
    Config Config
    // ChangedKeys []string  // future: targeted reloads
}
```

## API surface

```go
func (s *Settings) GetSettings() Config
func (s *Settings) SaveSettings(config Config) error
func (s *Settings) ResetToDefaults() error
func (s *Settings) AddScanDirectory(dir string) error
func (s *Settings) RemoveScanDirectory(dir string) error
```

## Tests

```go
// Save config → file exists at expected path with correct JSON
func TestConfigPersistence(t *testing.T)

// Load non-existent config → returns Defaults
func TestDefaultsOnMissingFile(t *testing.T)

// SaveSettings with invalid directory → returns validation error
func TestInvalidDirectoryValidation(t *testing.T)

// SaveSettings → settings.changed event emitted
func TestSettingsChangedEventEmitted(t *testing.T)
```
