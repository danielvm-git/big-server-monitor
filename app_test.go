package main

import (
	"context"
	"testing"

	"portkeeper/components/activitylog"
	"portkeeper/components/logcapture"
	"portkeeper/components/settings"
)

func TestNewApp(t *testing.T) {
	a := NewApp()
	if a == nil {
		t.Fatal("expected non-nil App")
	}
}

func TestStartupRegistersComponents(t *testing.T) {
	a := NewApp()
	a.startup(context.Background())

	if a.kernel == nil {
		t.Fatal("expected non-nil kernel after startup")
	}

	components := a.kernel.ListComponents()
	if len(components) != 6 {
		t.Fatalf("expected 6 components, got %d", len(components))
	}

	names := make(map[string]bool)
	for _, c := range components {
		names[c.Name] = true
	}

	expected := []string{"settings", "processmonitor", "healthcheck", "activitylog", "logcapture", "notifications"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected component %q to be registered", name)
		}
	}
}

func TestAllBindingsNilGuards(t *testing.T) {
	a := NewApp()

	// processmonitor bindings
	if s := a.GetServers(); s != nil {
		t.Error("GetServers: expected nil")
	}
	if _, err := a.GetServerByPort(3000); err == nil {
		t.Error("GetServerByPort: expected error")
	}
	if err := a.KillProcess(1); err == nil {
		t.Error("KillProcess: expected error")
	}

	// healthcheck bindings
	if r := a.GetHealthResults(); r != nil {
		t.Error("GetHealthResults: expected nil")
	}
	if r := a.RunHealthCheck(nil); r != nil {
		t.Error("RunHealthCheck: expected nil")
	}

	// activitylog bindings
	if _, err := a.GetActivityLog(activitylog.ActivityFilter{}); err == nil {
		t.Error("GetActivityLog: expected error")
	}
	if err := a.ClearHistory(); err == nil {
		t.Error("ClearHistory: expected error")
	}
	if c := a.GetEventCounts(); c != nil {
		t.Error("GetEventCounts: expected nil")
	}

	// logcapture bindings
	if l := a.GetLogs(logcapture.LogFilter{}); l != nil {
		t.Error("GetLogs: expected nil")
	}
	if s := a.GetLogsForAI(3000); s != "" {
		t.Error("GetLogsForAI: expected empty string")
	}
	if c := a.GetLogCounts(3000); c != nil {
		t.Error("GetLogCounts: expected nil")
	}

	// notifications bindings
	if err := a.RequestNotificationPermission(); err == nil {
		t.Error("RequestNotificationPermission: expected error")
	}
	if a.HasNotificationPermission() {
		t.Error("HasNotificationPermission: expected false")
	}

	// settings bindings
	_ = a.GetSettings() // should return defaults, not panic
	if err := a.SaveSettings(settings.Config{}); err == nil {
		t.Error("SaveSettings: expected error")
	}
	if err := a.ResetSettings(); err == nil {
		t.Error("ResetSettings: expected error")
	}
	if err := a.AddScanDirectory("x"); err == nil {
		t.Error("AddScanDirectory: expected error")
	}
	if err := a.RemoveScanDirectory("x"); err == nil {
		t.Error("RemoveScanDirectory: expected error")
	}
}

func TestGetServersNilGuard(t *testing.T) {
	a := NewApp()
	s := a.GetServers()
	if s != nil {
		t.Error("expected nil from uninitialized App")
	}
}

func TestSettingsBindings(t *testing.T) {
	a := NewApp()
	cfg := a.GetSettings()
	if cfg.PollingIntervalSeconds == 0 {
		t.Error("expected default PollingIntervalSeconds, got 0")
	}
}

func TestHealthCheckBindingsNilSafe(t *testing.T) {
	a := NewApp()
	r := a.GetHealthResults()
	if r != nil {
		t.Error("expected nil health results")
	}
	r2 := a.RunHealthCheck(nil)
	if r2 != nil {
		t.Error("expected nil from RunHealthCheck")
	}
}

func TestGetMonitorStatusBinding(t *testing.T) {
	a := NewApp()
	status := a.GetMonitorStatus()
	// Should not panic, return "not started"
	if status.Healthy {
		t.Error("expected unhealthy status for uninitialized app")
	}
}

