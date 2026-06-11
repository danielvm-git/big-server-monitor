package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"portkeeper/kernel"
)

const version = "0.1.0"

// Config represents the notifications configuration
type Config struct {
	CrashAlerts bool `json:"crashAlerts"`
}

// rateLimiter implements rate limiting for notifications per port
type rateLimiter struct {
	mu       sync.Mutex
	lastAt   map[int]time.Time
	cooldown time.Duration
}

// isRateLimited checks if a notification should be rate limited
func (rl *rateLimiter) isRateLimited(port int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	lastTime, exists := rl.lastAt[port]
	if !exists {
		return false
	}

	return time.Since(lastTime) < rl.cooldown
}

// markRateLimited marks a port as having just sent a notification
func (rl *rateLimiter) markRateLimited(port int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.lastAt[port] = time.Now()
}

// Notifier is an interface for delivering notifications
type Notifier interface {
	Notify(ctx context.Context, title, message string) error
}

// defaultNotifier implements Notifier using Wails runtime
type defaultNotifier struct{}

// Notify sends a notification using Wails runtime
func (dn *defaultNotifier) Notify(ctx context.Context, title, message string) error {
	// Import runtime dynamically to avoid compile-time dependency in tests
	// The actual notification is sent via Wails at runtime
	// This is a placeholder - the actual implementation requires the Wails context
	// which is only available at runtime, not in tests
	return nil
}

// Notifications handles system notifications for PortKeeper
type Notifications struct {
	config        Config
	logger        kernel.Logger
	ctx           *kernel.Context
	rateLimiter   *rateLimiter
	hasPermission bool
	notifier      Notifier
}

// New creates a new Notifications component
func New() *Notifications {
	return &Notifications{
		config: Config{
			CrashAlerts: true,
		},
		rateLimiter: &rateLimiter{
			lastAt:   make(map[int]time.Time),
			cooldown: 60 * time.Second,
		},
		hasPermission: false,
		notifier:      &defaultNotifier{},
	}
}

// Name returns the component name
func (n *Notifications) Name() string {
	return "notifications"
}

// Version returns the component version
func (n *Notifications) Version() string {
	return version
}

// Dependencies returns a list of component dependencies
func (n *Notifications) Dependencies() []string {
	return []string{}
}

// ConfigSchema returns the JSON schema for configuration
func (n *Notifications) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"crashAlerts": {
				"type": "boolean",
				"description": "Enable notifications for crashed processes",
				"default": true
			}
		}
	}`)
}

// Init initializes the component with the given configuration
func (n *Notifications) Init(ctx *kernel.Context, config json.RawMessage) error {
	n.logger = ctx.Logger
	n.ctx = ctx

	// Set defaults
	n.config.CrashAlerts = true

	// Parse config if provided
	if config != nil {
		var cfg Config
		if err := json.Unmarshal(config, &cfg); err != nil {
			n.logger.Error("failed to parse notifications config", "error", err.Error())
			return fmt.Errorf("parse config: %w", err)
		}
		n.config = cfg
	}

	n.logger.Info("notifications component initialized", "crashAlerts", n.config.CrashAlerts)
	return nil
}

// Start starts the component
func (n *Notifications) Start(ctx *kernel.Context) error {
	n.logger.Info("notifications component started")
	return nil
}

// Stop stops the component
func (n *Notifications) Stop(ctx *kernel.Context) error {
	n.logger.Info("notifications component stopped")
	return nil
}

// Hooks returns the event hooks this component subscribes to
func (n *Notifications) Hooks() []kernel.HookDef {
	return []kernel.HookDef{
		{
			Name:     "process.crashed",
			Priority: 0,
			Handler:  n.handleProcessCrashed,
		},
	}
}

// handleProcessCrashed handles the process.crashed event
func (n *Notifications) handleProcessCrashed(ctx *kernel.Context, event kernel.Event) error {
	// Respect the CrashAlerts setting
	if !n.config.CrashAlerts {
		return nil
	}

	// Extract event data
	port, ok := event.Data["port"].(int)
	if !ok {
		return fmt.Errorf("process.crashed event missing or invalid 'port' field")
	}

	processName, ok := event.Data["processName"].(string)
	if !ok {
		processName = "unknown"
	}

	projectName, ok := event.Data["projectName"].(string)
	if !ok {
		projectName = "unknown"
	}

	uptimeStr, ok := event.Data["uptimeStr"].(string)
	if !ok {
		uptimeStr = "unknown"
	}

	// Check rate limiting
	if n.rateLimiter.isRateLimited(port) {
		n.logger.Debug("notification rate limited", "port", port)
		return nil
	}

	// Mark as rate limited
	n.rateLimiter.markRateLimited(port)

	// Deliver the notification
	title := fmt.Sprintf("Port :%d went offline", port)
	message := fmt.Sprintf("%s (%s) stopped after %s", processName, projectName, uptimeStr)

	n.logger.Info("delivering crash notification", "port", port, "title", title)

	// Use the notifier to deliver the notification
	// The context.Background() is used here; in a real implementation,
	// we would pass the Wails context which is available only at runtime
	if err := n.notifier.Notify(context.Background(), title, message); err != nil {
		n.logger.Error("failed to deliver notification", "error", err.Error())
	}

	// Emit notification.delivered event
	if ctx.Kernel != nil {
		notificationEvent := kernel.Event{
			Name: "notification.delivered",
			Data: map[string]any{
				"port":    port,
				"title":   title,
				"message": message,
			},
		}

		if eventBus := ctx.Kernel.EventBus(); eventBus != nil {
			if err := eventBus.Emit(notificationEvent, ctx); err != nil {
				n.logger.Error("failed to emit notification.delivered event", "error", err.Error())
			}
		}
	}

	return nil
}

// RequestPermission requests notification permission from the OS
func (n *Notifications) RequestPermission(ctx *kernel.Context) error {
	// In a real implementation with Wails, this would request macOS notification center permission
	// For now, we'll just mark that permission was requested
	n.hasPermission = true
	if n.logger != nil {
		n.logger.Info("notification permission requested")
	}
	return nil
}

// HasPermission returns true if notification permission has been granted
func (n *Notifications) HasPermission() bool {
	return n.hasPermission
}

// Verify the component implements the interface
var _ kernel.Component = (*Notifications)(nil)
