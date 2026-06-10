// Package logger provides a structured JSON logger implementing kernel.Logger
// via Go's standard library log/slog with a JSON handler.
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"portkeeper/kernel"
)

// SlogAdapter implements kernel.Logger by delegating to log/slog with a JSON
// handler. Logs are written to stderr and optionally a file.
type SlogAdapter struct {
	logger *slog.Logger
}

// New creates a SlogAdapter that writes JSON to stderr. If logPath is
// non-empty, it also appends to that file (creating it and parent directories
// as needed).
func New(logPath string) *SlogAdapter {
	writers := []io.Writer{os.Stderr}

	if logPath != "" {
		expanded := expandPath(logPath)
		dir := filepath.Dir(expanded)
		if err := os.MkdirAll(dir, 0o700); err == nil {
			f, err := os.OpenFile(expanded, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err == nil {
				writers = append(writers, f)
			}
		}
	}

	handler := slog.NewJSONHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return &SlogAdapter{
		logger: slog.New(handler),
	}
}

// Info logs an informational message with optional key-value pairs.
func (a *SlogAdapter) Info(msg string, args ...any) { a.logger.Info(msg, args...) }

// Warn logs a warning message with optional key-value pairs.
func (a *SlogAdapter) Warn(msg string, args ...any) { a.logger.Warn(msg, args...) }

// Error logs an error message with optional key-value pairs.
func (a *SlogAdapter) Error(msg string, args ...any) { a.logger.Error(msg, args...) }

// Debug logs a debug message with optional key-value pairs.
func (a *SlogAdapter) Debug(msg string, args ...any) { a.logger.Debug(msg, args...) }

// Assert SlogAdapter implements kernel.Logger at compile time.
var _ kernel.Logger = (*SlogAdapter)(nil)

// expandPath replaces a leading ~ with the user's home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
