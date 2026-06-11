package healthcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"portkeeper/kernel"
)

const version = "0.1.0"

// HealthStatus represents the status of a health check result
type HealthStatus string

const (
	HealthOK      HealthStatus = "ok"      // 2xx status code
	HealthWarn    HealthStatus = "warn"    // 3xx or 4xx status code
	HealthError   HealthStatus = "error"   // 5xx status code
	HealthTimeout HealthStatus = "timeout" // no response or timeout
)

// HealthResult represents the result of a single health check probe
type HealthResult struct {
	Port       int          `json:"port"`
	Status     HealthStatus `json:"status"`
	StatusCode int          `json:"statusCode"`
	LatencyMs  float64      `json:"latencyMs"`
	Protocol   string       `json:"protocol"`
	Scheme     string       `json:"scheme,omitempty"`
	CheckedAt  time.Time    `json:"checkedAt"`
	Error      string       `json:"error,omitempty"`
}

// Config holds configuration for the HealthCheck component
type Config struct {
	ProbeTimeout        time.Duration   `json:"probeTimeout"`
	HealthCheckInterval time.Duration   `json:"healthCheckInterval"`
	MaxConcurrentProbes int             `json:"maxConcurrentProbes"`
	EnabledProtocols    map[string]bool `json:"enabledProtocols,omitempty"`
}

// HealthCheck is the main component for health checking ports
type HealthCheck struct {
	mu       sync.RWMutex
	results  map[int]HealthResult
	config   Config
	stopChan chan struct{}
	logger   kernel.Logger
	eventBus *kernel.EventBus
}

// New creates a new HealthCheck component with default configuration
func New() *HealthCheck {
	return &HealthCheck{
		results: make(map[int]HealthResult),
		config: Config{
			ProbeTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			MaxConcurrentProbes: 10,
		},
		stopChan: make(chan struct{}),
	}
}

// Name returns the component name
func (hc *HealthCheck) Name() string {
	return "healthcheck"
}

// Version returns the component version
func (hc *HealthCheck) Version() string {
	return version
}

// Dependencies returns the list of components this depends on
func (hc *HealthCheck) Dependencies() []string {
	return []string{}
}

// ConfigSchema returns the JSON schema for the config
func (hc *HealthCheck) ConfigSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"probeTimeout": map[string]any{
				"type":    "string",
				"default": "3s",
			},
			"healthCheckInterval": map[string]any{
				"type":    "string",
				"default": "30s",
			},
			"maxConcurrentProbes": map[string]any{
				"type":    "integer",
				"default": 10,
			},
		},
	}
	data, _ := json.Marshal(schema)
	return data
}

// Init initializes the component with the given configuration
func (hc *HealthCheck) Init(ctx *kernel.Context, config json.RawMessage) error {
	hc.logger = ctx.Logger
	hc.eventBus = ctx.Kernel.EventBus()

	// Parse config if provided
	if config != nil {
		var cfg Config
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		// Override defaults with provided config
		if cfg.ProbeTimeout > 0 {
			hc.config.ProbeTimeout = cfg.ProbeTimeout
		}
		if cfg.HealthCheckInterval > 0 {
			hc.config.HealthCheckInterval = cfg.HealthCheckInterval
		}
		if cfg.MaxConcurrentProbes > 0 {
			hc.config.MaxConcurrentProbes = cfg.MaxConcurrentProbes
		}
		if cfg.EnabledProtocols != nil {
			hc.config.EnabledProtocols = cfg.EnabledProtocols
		}
	}

	hc.logger.Debug("healthcheck component initialized", "config", hc.config)
	return nil
}

// Start begins the periodic health check loop
func (hc *HealthCheck) Start(ctx *kernel.Context) error {
	hc.logger.Info("healthcheck component starting")

	// Run initial health check
	hc.runAll(ctx, []int{})

	// Start background goroutine for periodic checks
	go hc.backgroundLoop(ctx)

	return nil
}

// Stop halts the health check component
func (hc *HealthCheck) Stop(ctx *kernel.Context) error {
	hc.logger.Info("healthcheck component stopping")
	select {
	case <-hc.stopChan:
		// already closed
	default:
		close(hc.stopChan)
	}
	return nil
}

// Hooks returns the list of event hooks this component registers
func (hc *HealthCheck) Hooks() []kernel.HookDef {
	return []kernel.HookDef{
		{
			Name:     "request.healthcheck",
			Priority: 0,
			Handler:  hc.handleHealthCheckRequest,
		},
	}
}

// handleHealthCheckRequest is the hook handler for on-demand health checks
func (hc *HealthCheck) handleHealthCheckRequest(ctx *kernel.Context, event kernel.Event) error {
	ports := []int{}
	if data, ok := event.Data["ports"]; ok {
		if portList, ok := data.([]int); ok {
			ports = portList
		}
	}
	hc.runAll(ctx, ports)
	return nil
}

// GetHealthResults returns a copy of all current health check results
func (hc *HealthCheck) GetHealthResults() []HealthResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make([]HealthResult, 0, len(hc.results))
	for _, result := range hc.results {
		results = append(results, result)
	}
	return results
}

// RunHealthCheck performs an on-demand health check for specified ports
func (hc *HealthCheck) RunHealthCheck(ports []int) []HealthResult {
	ctx := &kernel.Context{
		Logger: hc.logger,
	}
	hc.runAll(ctx, ports)
	return hc.GetHealthResults()
}

// backgroundLoop runs periodic health checks
func (hc *HealthCheck) backgroundLoop(ctx *kernel.Context) {
	ticker := time.NewTicker(hc.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.runAll(ctx, []int{})
		case <-hc.stopChan:
			return
		}
	}
}

// runAll runs health checks for all online servers (or specified ports)
func (hc *HealthCheck) runAll(ctx *kernel.Context, ports []int) {
	if len(ports) == 0 {
		// For now, no servers are registered. This will be called with specific ports
		// from the listener component once that's integrated
		return
	}

	results := hc.runAllProbes(ports, hc.config.ProbeTimeout)

	// Store results and emit events for any unresponsive servers
	hc.mu.Lock()
	for _, result := range results {
		hc.results[result.Port] = result

		// Emit event if server became unresponsive
		if result.Status == HealthTimeout || result.Status == HealthError {
			event := kernel.Event{
				Name: "process.unresponsive",
				Data: map[string]any{
					"port":   result.Port,
					"status": result.Status,
					"error":  result.Error,
				},
			}
			_ = hc.eventBus.Emit(event, ctx)
		}
	}
	hc.mu.Unlock()
}

// runAllProbes probes multiple ports concurrently using a bounded goroutine pool
func (hc *HealthCheck) runAllProbes(ports []int, timeout time.Duration) []HealthResult {
	results := make([]HealthResult, len(ports))
	maxProbes := hc.config.MaxConcurrentProbes
	if maxProbes < 1 {
		maxProbes = 1 // defensive floor: prevent zero-capacity channel deadlock
	}
	semaphore := make(chan struct{}, maxProbes)
	var wg sync.WaitGroup

	for i, port := range ports {
		wg.Add(1)
		go func(idx int, p int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			proto := getProtocol(p)
			var result HealthResult

			switch proto {
			case "postgres":
				result = hc.probePostgres(p, timeout)
			case "mysql":
				result = hc.probeMySQL(p, timeout)
			case "redis":
				result = hc.probeRedis(p, timeout)
			case "mongodb":
				result = hc.probeMongoDB(p, timeout)
			case "memcached":
				result = hc.probeMemcached(p, timeout)
			default:
				result = hc.probeHTTP(p, timeout)
			}

			results[idx] = result
		}(i, port)
	}

	wg.Wait()
	return results
}

// probeHTTP probes an HTTP/HTTPS endpoint
func (hc *HealthCheck) probeHTTP(port int, timeout time.Duration) HealthResult {
	result := HealthResult{
		Port:      port,
		Protocol:  "http",
		CheckedAt: time.Now(),
	}

	// Try HTTP first, then HTTPS
	schemes := []string{"http", "https"}
	for _, scheme := range schemes {
		url := fmt.Sprintf("%s://localhost:%d/", scheme, port)
		client := &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		start := time.Now()
		resp, err := client.Head(url)
		latency := time.Since(start).Seconds() * 1000

		if err == nil && resp != nil {
			result.StatusCode = resp.StatusCode
			result.LatencyMs = latency
			result.Scheme = scheme

			// Discard body if any
			if resp.Body != nil {
				_, _ = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
			}

			// Determine status based on status code
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				result.Status = HealthOK
			} else if resp.StatusCode >= 300 && resp.StatusCode < 500 {
				result.Status = HealthWarn
			} else if resp.StatusCode >= 500 {
				result.Status = HealthError
			}

			return result
		}

		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}

	// Both schemes failed
	result.Status = HealthTimeout
	result.Error = "connection refused or timeout"
	return result
}

// probePostgres probes a PostgreSQL server
func (hc *HealthCheck) probePostgres(port int, timeout time.Duration) HealthResult {
	result := HealthResult{
		Port:      port,
		Protocol:  "postgres",
		CheckedAt: time.Now(),
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		result.Status = HealthTimeout
		result.Error = "connection failed"
		return result
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		result.Status = HealthTimeout
		result.Error = "set deadline failed"
		return result
	}
	buf := make([]byte, 5)
	_, err = conn.Read(buf)

	result.LatencyMs = latency
	if err == nil {
		result.Status = HealthOK
	} else {
		result.Status = HealthTimeout
		result.Error = "read failed"
	}

	return result
}

// probeMySQL probes a MySQL server
func (hc *HealthCheck) probeMySQL(port int, timeout time.Duration) HealthResult {
	result := HealthResult{
		Port:      port,
		Protocol:  "mysql",
		CheckedAt: time.Now(),
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		result.Status = HealthTimeout
		result.Error = "connection failed"
		return result
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		result.Status = HealthTimeout
		result.Error = "set deadline failed"
		return result
	}
	buf := make([]byte, 1)
	_, err = conn.Read(buf)

	result.LatencyMs = latency
	if err == nil && buf[0] == 0x0a {
		result.Status = HealthOK
	} else {
		result.Status = HealthTimeout
		result.Error = "invalid server greeting"
	}

	return result
}

// probeRedis probes a Redis server
func (hc *HealthCheck) probeRedis(port int, timeout time.Duration) HealthResult {
	result := HealthResult{
		Port:      port,
		Protocol:  "redis",
		CheckedAt: time.Now(),
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		result.Status = HealthTimeout
		result.Error = "connection failed"
		return result
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		result.Status = HealthTimeout
		result.Error = "set deadline failed"
		return result
	}

	_, err = fmt.Fprintf(conn, "PING\r\n")
	if err != nil {
		result.Status = HealthTimeout
		result.Error = "write failed"
		return result
	}

	buf := make([]byte, 7)
	_, err = conn.Read(buf)

	result.LatencyMs = latency
	if err == nil && string(buf) == "+PONG\r\n" {
		result.Status = HealthOK
	} else {
		result.Status = HealthError
		result.Error = "unexpected response"
	}

	return result
}

// probeMongoDB probes a MongoDB server
func (hc *HealthCheck) probeMongoDB(port int, timeout time.Duration) HealthResult {
	result := HealthResult{
		Port:      port,
		Protocol:  "mongodb",
		CheckedAt: time.Now(),
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		result.Status = HealthTimeout
		result.Error = "connection failed"
		return result
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		result.Status = HealthTimeout
		result.Error = "set deadline failed"
		return result
	}
	buf := make([]byte, 1)
	_, err = conn.Read(buf)

	result.LatencyMs = latency
	if err == nil {
		result.Status = HealthOK
	} else {
		result.Status = HealthTimeout
		result.Error = "read failed"
	}

	return result
}

// probeMemcached probes a Memcached server
func (hc *HealthCheck) probeMemcached(port int, timeout time.Duration) HealthResult {
	result := HealthResult{
		Port:      port,
		Protocol:  "memcached",
		CheckedAt: time.Now(),
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		result.Status = HealthTimeout
		result.Error = "connection failed"
		return result
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		result.Status = HealthTimeout
		result.Error = "set deadline failed"
		return result
	}

	_, err = fmt.Fprintf(conn, "version\r\n")
	if err != nil {
		result.Status = HealthTimeout
		result.Error = "write failed"
		return result
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)

	result.LatencyMs = latency
	if err == nil && n > 0 && len(buf) >= 8 && string(buf[:8]) == "VERSION " {
		result.Status = HealthOK
	} else {
		result.Status = HealthError
		result.Error = "unexpected response"
	}

	return result
}

// getProtocol maps a port number to its expected protocol
func getProtocol(port int) string {
	knownProtocols := map[int]string{
		5432:  "postgres",
		3306:  "mysql",
		3307:  "mysql",
		6379:  "redis",
		27017: "mongodb",
		11211: "memcached",
		5672:  "amqp",
		9200:  "http",
	}

	if proto, ok := knownProtocols[port]; ok {
		return proto
	}
	return "http"
}

// Ensure HealthCheck implements the Component interface
var _ kernel.Component = (*HealthCheck)(nil)
