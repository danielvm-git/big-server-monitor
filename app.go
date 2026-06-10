package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"portkeeper/components/activitylog"
	"portkeeper/components/healthcheck"
	"portkeeper/components/logcapture"
	"portkeeper/components/notifications"
	"portkeeper/components/processmonitor"
	"portkeeper/components/settings"
	"portkeeper/internal/logger"
	"portkeeper/kernel"
)

// App is the Wails application struct. It holds the ECC kernel and all six
// monitoring components, exposing their public APIs as Wails bindings for the
// React frontend.
type App struct {
	ctx context.Context

	kernel *kernel.Kernel

	processMonitor *processmonitor.ProcessMonitor
	healthCheck    *healthcheck.HealthCheck
	activityLog    *activitylog.ActivityLog
	logCapture     *logcapture.LogCapture
	notify         *notifications.Notifications
	settingsComp   *settings.Settings
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup initialises the ECC kernel and all six monitoring components.
// It is called by Wails when the application window is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Structured JSON logger — writes to stderr (captured by Wails in dev mode)
	// and to ~/.config/portkeeper/portkeeper.log for production.
	slogAdapter := logger.New("~/.config/portkeeper/portkeeper.log")
	slogAdapter.Info("portkeeper starting", "version", kernel.Version)
	a.kernel = kernel.New(slogAdapter)

	// Instantiate components.
	a.settingsComp = settings.New("")
	a.processMonitor = processmonitor.New()
	a.healthCheck = healthcheck.New()
	a.activityLog = activitylog.New()
	a.logCapture = logcapture.New()
	a.notify = notifications.New()

	// Register in dependency order (topological sort handled by kernel).
	a.kernel.Register(a.settingsComp)
	a.kernel.Register(a.processMonitor)
	a.kernel.Register(a.healthCheck)
	a.kernel.Register(a.activityLog)
	a.kernel.Register(a.logCapture)
	a.kernel.Register(a.notify)

	// Init and start all components.
	if err := a.kernel.Start(); err != nil {
		slogAdapter.Error("kernel start failed", "error", err)
	}
}

// shutdown stops the kernel and all its components.
func (a *App) shutdown(ctx context.Context) {
	if a.kernel == nil {
		return
	}
	if err := a.kernel.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "kernel stop failed: %v\n", err)
	}
}

// ---------------------------------------------------------------------------
// Wails bindings — processmonitor
// ---------------------------------------------------------------------------

// GetServers returns all currently monitored servers.
func (a *App) GetServers() []processmonitor.Server {
	if a.processMonitor == nil {
		return nil
	}
	return a.processMonitor.GetServers()
}

// GetServerByPort returns the server running on the given port, if any.
func (a *App) GetServerByPort(port int) (processmonitor.Server, error) {
	if a.processMonitor == nil {
		return processmonitor.Server{}, fmt.Errorf("process monitor not available")
	}
	return a.processMonitor.GetServerByPort(port)
}

// KillProcess sends a kill signal to the process with the given PID.
func (a *App) KillProcess(pid int) error {
	if a.processMonitor == nil {
		return fmt.Errorf("process monitor not available")
	}
	return a.processMonitor.KillProcess(pid)
}

// ---------------------------------------------------------------------------
// Wails bindings — healthcheck
// ---------------------------------------------------------------------------

// GetHealthResults returns cached health-check results for all servers.
func (a *App) GetHealthResults() []healthcheck.HealthResult {
	if a.healthCheck == nil {
		return nil
	}
	return a.healthCheck.GetHealthResults()
}

// RunHealthCheck probes the given ports (or all known ports if empty) and
// returns fresh results.
func (a *App) RunHealthCheck(ports []int) []healthcheck.HealthResult {
	if a.healthCheck == nil {
		return nil
	}
	return a.healthCheck.RunHealthCheck(ports)
}

// ---------------------------------------------------------------------------
// Wails bindings — activitylog
// ---------------------------------------------------------------------------

// GetActivityLog returns activity events matching the supplied filter.
func (a *App) GetActivityLog(filter activitylog.ActivityFilter) ([]activitylog.ActivityEvent, error) {
	if a.activityLog == nil {
		return nil, fmt.Errorf("activity log not available")
	}
	return a.activityLog.GetActivityLog(filter)
}

// ClearHistory removes all persisted activity events.
func (a *App) ClearHistory() error {
	if a.activityLog == nil {
		return fmt.Errorf("activity log not available")
	}
	return a.activityLog.ClearHistory()
}

// GetEventCounts returns a breakdown of event counts by type.
func (a *App) GetEventCounts() map[activitylog.EventType]int {
	if a.activityLog == nil {
		return nil
	}
	return a.activityLog.GetEventCounts()
}

// ---------------------------------------------------------------------------
// Wails bindings — logcapture
// ---------------------------------------------------------------------------

// GetLogs returns captured log lines for the given filter.
func (a *App) GetLogs(filter logcapture.LogFilter) []*logcapture.LogLine {
	if a.logCapture == nil {
		return nil
	}
	return a.logCapture.GetLogs(filter)
}

// GetLogsForAI returns the captured logs for the given port formatted as
// a plain-text block suitable for pasting into an AI chat.
func (a *App) GetLogsForAI(port int) string {
	if a.logCapture == nil {
		return ""
	}
	return a.logCapture.GetLogsForAI(port)
}

// GetLogCounts returns a breakdown of log counts by level for the given port.
func (a *App) GetLogCounts(port int) map[logcapture.LogLevel]int {
	if a.logCapture == nil {
		return nil
	}
	return a.logCapture.GetLogCounts(port)
}

// ---------------------------------------------------------------------------
// Wails bindings — notifications
// ---------------------------------------------------------------------------

// RequestNotificationPermission marks that notification permission was
// requested (real macOS prompt will be implemented via Wails runtime).
func (a *App) RequestNotificationPermission() error {
	if a.notify == nil {
		return fmt.Errorf("notifications not available")
	}
	return a.notify.RequestPermission(nil)
}

// HasNotificationPermission returns whether the user has granted notification
// permission.
func (a *App) HasNotificationPermission() bool {
	if a.notify == nil {
		return false
	}
	return a.notify.HasPermission()
}

// ---------------------------------------------------------------------------
// Wails bindings — settings
// ---------------------------------------------------------------------------

// GetSettings returns the current application configuration.
func (a *App) GetSettings() settings.Config {
	if a.settingsComp == nil {
		return settings.Defaults
	}
	return a.settingsComp.GetSettings()
}

// SaveSettings persists the supplied configuration and emits a
// settings.changed event.
func (a *App) SaveSettings(config settings.Config) error {
	if a.settingsComp == nil {
		return fmt.Errorf("settings not available")
	}
	return a.settingsComp.SaveSettings(config)
}

// ResetSettings restores the default configuration.
func (a *App) ResetSettings() error {
	if a.settingsComp == nil {
		return fmt.Errorf("settings not available")
	}
	return a.settingsComp.ResetToDefaults()
}

// AddScanDirectory adds a directory to the monitored scan list.
func (a *App) AddScanDirectory(dir string) error {
	if a.settingsComp == nil {
		return fmt.Errorf("settings not available")
	}
	return a.settingsComp.AddScanDirectory(dir)
}

// RemoveScanDirectory removes a directory from the monitored scan list.
func (a *App) RemoveScanDirectory(dir string) error {
	if a.settingsComp == nil {
		return fmt.Errorf("settings not available")
	}
	return a.settingsComp.RemoveScanDirectory(dir)
}

// ---------------------------------------------------------------------------
// Lifecycle helpers
// ---------------------------------------------------------------------------

// marshalJSON is a convenience helper for returning JSON snippets to the UI.
func marshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return string(b)
}
