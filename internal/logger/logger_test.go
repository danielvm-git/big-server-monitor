package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlogAdapterWritesJSON(t *testing.T) {
	var buf bytes.Buffer

	// Create a logger writing to our buffer instead of stderr.
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	lg := slog.New(handler)
	adapter := &SlogAdapter{logger: lg}

	adapter.Info("server started", "port", 3000, "name", "api")
	adapter.Warn("high memory", "mb", 512)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %q", len(lines), buf.String())
	}

	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\n%s", i, err, line)
		}
		if entry["level"] == nil {
			t.Errorf("line %d missing level field", i)
		}
		if entry["time"] == nil {
			t.Errorf("line %d missing time field", i)
		}
		if entry["msg"] == nil {
			t.Errorf("line %d missing msg field", i)
		}
	}

	// Verify content of first line
	var first map[string]any
	json.Unmarshal([]byte(lines[0]), &first)
	if first["msg"] != "server started" {
		t.Errorf("expected msg 'server started', got %v", first["msg"])
	}
	if first["port"] != float64(3000) { // JSON numbers are float64
		t.Errorf("expected port 3000, got %v", first["port"])
	}
}

func TestNewCreatesFileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	adapter := New(logPath)

	adapter.Info("should appear in file")

	// Check the file was created and has content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("log file is not valid JSON: %v\n%s", err, string(data))
	}

	if entry["msg"] != "should appear in file" {
		t.Errorf("unexpected log message: %v", entry["msg"])
	}
}

func TestNewWithEmptyPathSucceeds(t *testing.T) {
	adapter := New("")
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	// Should not panic
	adapter.Debug("test")
}

func TestSlogAdapterImplementsInterface(t *testing.T) {
	// Compile-time check already done via var _ line, but also verify at test time.
	var logger interface{ Info(string, ...any) } = (*SlogAdapter)(nil)
	_ = logger
}

func TestExpandPath(t *testing.T) {
	t.Run("tilde prefix expands to home", func(t *testing.T) {
		result := expandPath("~/test")
		if strings.HasPrefix(result, "~") {
			t.Errorf("~ was not expanded: %s", result)
		}
		home, _ := os.UserHomeDir()
		if !strings.HasPrefix(result, home) {
			t.Errorf("expected path under home dir %s, got %s", home, result)
		}
	})

	t.Run("absolute path unchanged", func(t *testing.T) {
		result := expandPath("/var/log/test")
		if result != "/var/log/test" {
			t.Errorf("absolute path changed: %s", result)
		}
	})

	t.Run("relative path unchanged", func(t *testing.T) {
		result := expandPath("relative/path")
		if result != "relative/path" {
			t.Errorf("relative path changed: %s", result)
		}
	})
}
