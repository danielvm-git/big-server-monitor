package processmonitor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"portkeeper/kernel"
)

const version = "0.1.0"

// Status enum for server state
type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusUnknown Status = "unknown"
)

// MonitorStatus summarizes the health of the process monitor.
type MonitorStatus struct {
	Healthy     bool   `json:"healthy"`
	LastError   string `json:"lastError,omitempty"`
	LastPollAt  string `json:"lastPollAt,omitempty"`
	ServerCount int    `json:"serverCount"`
}

// Server represents a monitored process listening on a port
type Server struct {
	Port           int       `json:"port"`
	Status         Status    `json:"status"`
	PID            int       `json:"pid"`
	ProcessName    string    `json:"processName"`
	RuntimeVersion string    `json:"runtimeVersion"`
	BinaryPath     string    `json:"binaryPath"`
	ProjectName    string    `json:"projectName"`
	ProjectDir     string    `json:"projectDir"`
	LocalDomain    string    `json:"localDomain,omitempty"`
	TunnelURL      string    `json:"tunnelURL,omitempty"`
	EnvSnapshot    []EnvVar  `json:"envSnapshot,omitempty"`
	MemoryMB       float64   `json:"memoryMb"`
	StartedAt      time.Time `json:"startedAt"`
	UptimeStr      string    `json:"uptimeStr"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Visible bool   `json:"visible"`
}

// ProcessStartedEvent emitted when a process starts
type ProcessStartedEvent struct {
	Server Server `json:"server"`
}

// ProcessStoppedEvent emitted when a process stops cleanly
type ProcessStoppedEvent struct {
	Server   Server        `json:"server"`
	ExitCode int           `json:"exitCode"`
	Duration time.Duration `json:"duration"`
}

// ProcessCrashedEvent emitted when a process crashes unexpectedly
type ProcessCrashedEvent struct {
	Server   Server        `json:"server"`
	ExitCode int           `json:"exitCode"`
	Duration time.Duration `json:"duration"`
}

// Config for ProcessMonitor
type Config struct {
	PollingIntervalSec int    `json:"pollingIntervalSec"`
	ScanDirectories    []string `json:"scanDirectories"`
	IgnoredPorts       []int  `json:"ignoredPorts"`
}

// lockInfo wraps sync.RWMutex for convenience
type lockInfo struct {
	sync.RWMutex
}

// New creates a new ProcessMonitor component with default configuration.
func New() *ProcessMonitor {
	return &ProcessMonitor{
		mu:              &lockInfo{},
		servers:         make(map[int]*Server),
		previousServers: make(map[int]*Server),
		runtimeCache:    make(map[int]string),
		discovery:       &LsofPortDiscovery{},
		config: Config{
			PollingIntervalSec: 5,
		},
	}
}

// ProcessMonitor is the main component
type ProcessMonitor struct {
	mu              *lockInfo
	config          Config
	servers         map[int]*Server // port -> Server
	previousServers map[int]*Server // for diff detection
	runtimeCache    map[int]string   // pid -> runtime version
	status          MonitorStatus
	ctx             *kernel.Context
	ticker          *time.Ticker
	discovery       PortDiscovery
}

// Name returns the component name
func (pm *ProcessMonitor) Name() string {
	return "processmonitor"
}

// Version returns the component version
func (pm *ProcessMonitor) Version() string {
	return version
}

// Dependencies returns required dependencies
func (pm *ProcessMonitor) Dependencies() []string {
	return []string{}
}

// ConfigSchema returns JSON schema for configuration
func (pm *ProcessMonitor) ConfigSchema() json.RawMessage {
	return json.RawMessage([]byte(`{
		"type": "object",
		"properties": {
			"pollingIntervalSec": {"type": "integer", "default": 5},
			"scanDirectories": {"type": "array", "items": {"type": "string"}},
			"ignoredPorts": {"type": "array", "items": {"type": "integer"}}
		}
	}`))
}

// Init initializes the component
func (pm *ProcessMonitor) Init(ctx *kernel.Context, config json.RawMessage) error {
	pm.mu = &lockInfo{}
	pm.servers = make(map[int]*Server)
	pm.previousServers = make(map[int]*Server)
	pm.runtimeCache = make(map[int]string)
	if pm.discovery == nil {
		pm.discovery = &LsofPortDiscovery{}
	}

	// Set defaults
	pm.config = Config{
		PollingIntervalSec: 5,
		IgnoredPorts:       []int{80, 443, 5432, 3306, 6379, 27017, 2181, 9092},
	}

	// Parse provided config
	if len(config) > 0 {
		if err := json.Unmarshal(config, &pm.config); err != nil {
			ctx.Logger.Error("parse config", "error", err)
			return fmt.Errorf("parse config: %w", err)
		}
	}

	ctx.Logger.Info("initialized processmonitor", "pollingIntervalSec", pm.config.PollingIntervalSec)
	return nil
}

// Start begins the polling loop
func (pm *ProcessMonitor) Start(ctx *kernel.Context) error {
	interval := time.Duration(pm.config.PollingIntervalSec) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)

	// Store context and ticker for graceful shutdown
	pm.mu.Lock()
	pm.ctx = ctx
	pm.ticker = ticker
	pm.mu.Unlock()

	go func() {
		// Perform initial poll
		pm.poll(ctx)

		for {
			select {
			case <-ticker.C:
				pm.poll(ctx)
			}
		}
	}()

	ctx.Logger.Info("processmonitor polling started", "interval", interval.String())
	return nil
}

// Stop halts the component
func (pm *ProcessMonitor) Stop(ctx *kernel.Context) error {
	pm.mu.Lock()
	if pm.ticker != nil {
		pm.ticker.Stop()
		pm.ticker = nil
	}
	pm.mu.Unlock()
	return nil
}

// Hooks returns event handlers
func (pm *ProcessMonitor) Hooks() []kernel.HookDef {
	return []kernel.HookDef{}
}

// GetServers returns a copy of current servers
func (pm *ProcessMonitor) GetServers() []Server {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	servers := make([]Server, 0, len(pm.servers))
	for _, s := range pm.servers {
		servers = append(servers, *s)
	}
	return servers
}

// GetServerByPort retrieves a server by port number
func (pm *ProcessMonitor) GetServerByPort(port int) (Server, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	s, ok := pm.servers[port]
	if !ok {
		return Server{}, fmt.Errorf("port %d not found", port)
	}
	return *s, nil
}

// GetMonitorStatus returns the current health status of the process monitor.
func (pm *ProcessMonitor) GetMonitorStatus() MonitorStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.status
}

// KillProcess terminates a process by PID
func (pm *ProcessMonitor) KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	return process.Kill()
}

// poll checks current listening ports and emits events
func (pm *ProcessMonitor) poll(ctx *kernel.Context) {
	ports, err := pm.discovery.ListeningPorts()
	if err != nil {
		ctx.Logger.Error("list ports", "error", err)
		pm.mu.Lock()
		pm.status.Healthy = false
		pm.status.LastError = err.Error()
		pm.mu.Unlock()
		return
	}

	currentServers := make(map[int]*Server)
	var partialErrors []string

	for _, port := range ports {
		// Skip ignored ports
		if pm.isPortIgnored(port) {
			continue
		}

		// Get process info
		info, err := pm.discovery.ProcessInfo(port)
		if err != nil {
			ctx.Logger.Debug("get process info", "port", port, "error", err)
			partialErrors = append(partialErrors, fmt.Sprintf("port %d: %s", port, err.Error()))
			continue
		}

		server := &Server{
			Port:        port,
			Status:      StatusOnline,
			PID:         info.PID,
			ProcessName: info.ProcessName,
			BinaryPath:  info.BinaryPath,
			ProjectDir:  info.WorkingDir,
			ProjectName: pm.detectProjectName(info.WorkingDir),
			MemoryMB:    info.MemoryMB,
			StartedAt:   info.StartTime,
			UptimeStr:   formatUptime(info.StartTime),
		}

		// Detect runtime version (cached per PID)
		if cached, ok := pm.runtimeCache[info.PID]; ok {
			server.RuntimeVersion = cached
		} else {
			runtime := detectRuntimeVersion(info.BinaryPath)
			pm.runtimeCache[info.PID] = runtime
			server.RuntimeVersion = runtime
		}

		// Detect tunnel URL
		server.TunnelURL = pm.detectTunnelURL(port)

		// Load environment snapshot
		server.EnvSnapshot = pm.loadEnvSnapshot(info.WorkingDir)

		currentServers[port] = server
	}

	// Diff and emit events
	pm.diffAndEmit(ctx, currentServers)

	pm.mu.Lock()
	// Only update if we discovered servers or if this is the first poll.
	// Prevents a transient failure from wiping previously discovered servers.
	if len(currentServers) > 0 || len(pm.servers) == 0 {
		pm.servers = currentServers
	}

	pm.status.Healthy = true
	pm.status.LastPollAt = time.Now().Format(time.RFC3339)
	pm.status.ServerCount = len(pm.servers)
	if len(partialErrors) > 0 {
		pm.status.LastError = strings.Join(partialErrors, "; ")
	} else {
		pm.status.LastError = ""
	}
	pm.mu.Unlock()
}

// diffAndEmit detects changes and emits appropriate events
func (pm *ProcessMonitor) diffAndEmit(ctx *kernel.Context, current map[int]*Server) {
	pm.mu.RLock()
	previous := pm.previousServers
	pm.mu.RUnlock()

	// Find started servers
	for port, srv := range current {
		if _, existed := previous[port]; !existed {
			event := kernel.Event{
				Name: "process.started",
				Data: map[string]any{
					"server": srv,
				},
			}
			_ = ctx.Kernel.EventBus().Emit(event, ctx)
			ctx.Logger.Info("process started", "port", port, "pid", srv.PID, "process", srv.ProcessName)
		}
	}

	// Find stopped/crashed servers
	for port, prevSrv := range previous {
		if _, exists := current[port]; !exists {
			uptime := time.Since(prevSrv.StartedAt)
			// Assuming clean exit if no info (user stopped it)
			event := kernel.Event{
				Name: "process.stopped",
				Data: map[string]any{
					"server":   prevSrv,
					"exitCode": 0,
					"duration": uptime,
				},
			}
			_ = ctx.Kernel.EventBus().Emit(event, ctx)
			ctx.Logger.Info("process stopped", "port", port, "pid", prevSrv.PID, "process", prevSrv.ProcessName)
		}
	}

	pm.mu.Lock()
	pm.previousServers = make(map[int]*Server)
	for port, srv := range current {
		pm.previousServers[port] = srv
	}
	pm.mu.Unlock()
}

// isPortIgnored checks if port should be skipped
func (pm *ProcessMonitor) isPortIgnored(port int) bool {
	for _, p := range pm.config.IgnoredPorts {
		if p == port {
			return true
		}
	}
	return false
}

// processInfo represents retrieved process metadata
type processInfo struct {
	PID        int
	ProcessName string
	BinaryPath string
	WorkingDir string
	MemoryMB   float64
	StartTime  time.Time
}

// detectProjectName determines project name from working directory
func (pm *ProcessMonitor) detectProjectName(workDir string) string {
	if workDir == "" {
		return ""
	}

	// Expand ~ if needed
	if strings.HasPrefix(workDir, "~") {
		home, _ := os.UserHomeDir()
		workDir = filepath.Join(home, workDir[1:])
	}

	// Check for markers
	markers := []struct {
		name   string
		parser func(path string) string
	}{
		{"package.json", parsePackageJSON},
		{"go.mod", parseGoMod},
		{"Cargo.toml", parseCargoToml},
		{"pyproject.toml", parsePyprojectToml},
		{"composer.json", parseComposerJSON},
	}

	current := workDir
	for {
		for _, m := range markers {
			markerPath := filepath.Join(current, m.name)
			if _, err := os.Stat(markerPath); err == nil {
				if name := m.parser(markerPath); name != "" {
					return name
				}
			}
		}

		// Check for .git as fallback marker
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return filepath.Base(current)
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// Last resort: use directory name
	return filepath.Base(workDir)
}

// parsePackageJSON extracts name from package.json
func parsePackageJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if name, ok := m["name"].(string); ok {
		return name
	}
	return ""
}

// parseGoMod extracts module name from go.mod
func parseGoMod(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

// parseCargoToml extracts name from Cargo.toml
func parseCargoToml(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inPackage := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[package]" {
			inPackage = true
			continue
		}
		if inPackage && strings.HasPrefix(line, "name") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				name := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				return name
			}
		}
		if inPackage && strings.HasPrefix(line, "[") {
			break
		}
	}
	return ""
}

// parsePyprojectToml extracts name from pyproject.toml
func parsePyprojectToml(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inProject := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[project]" {
			inProject = true
			continue
		}
		if inProject && strings.HasPrefix(line, "name") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				name := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				return name
			}
		}
		if inProject && strings.HasPrefix(line, "[") {
			break
		}
	}
	return ""
}

// parseComposerJSON extracts name from composer.json
func parseComposerJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if name, ok := m["name"].(string); ok {
		return name
	}
	return ""
}

// detectTunnelURL checks for ngrok or frp tunnel
func (pm *ProcessMonitor) detectTunnelURL(port int) string {
	// Check ngrok
	if ngrokURL := pm.detectNgrokTunnel(port); ngrokURL != "" {
		return ngrokURL
	}

	// Check frp
	if frpURL := pm.detectFrpTunnel(port); frpURL != "" {
		return frpURL
	}

	return ""
}

// detectNgrokTunnel checks for ngrok API tunnels
func (pm *ProcessMonitor) detectNgrokTunnel(targetPort int) string {
	resp, err := http.Get("http://127.0.0.1:4040/api/tunnels")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result struct {
		Tunnels []struct {
			PublicURL string `json:"public_url"`
			Config struct {
				Addr string `json:"addr"`
			} `json:"config"`
		} `json:"tunnels"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	for _, tunnel := range result.Tunnels {
		// Parse port from addr like localhost:3000
		if strings.Contains(tunnel.Config.Addr, fmt.Sprintf(":%d", targetPort)) {
			return tunnel.PublicURL
		}
	}

	return ""
}

// detectFrpTunnel checks for FRP tunnel configuration
func (pm *ProcessMonitor) detectFrpTunnel(targetPort int) string {
	// Would need to find frpc process and read its config
	// Simplified for now
	return ""
}

// loadEnvSnapshot reads environment variables from .env files
func (pm *ProcessMonitor) loadEnvSnapshot(workDir string) []EnvVar {
	if workDir == "" {
		return nil
	}

	vars := make([]EnvVar, 0)
	envFiles := []string{".env", ".env.local", ".env.development"}

	for _, filename := range envFiles {
		path := filepath.Join(workDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			visible, redactedValue := redactEnvVar(key, value)
			vars = append(vars, EnvVar{
				Key:     key,
				Value:   redactedValue,
				Visible: visible,
			})
		}
	}

	return vars
}

// safeEnvKeys are environment variables safe to display
var safeEnvKeys = map[string]bool{
	"NODE_ENV":      true,
	"APP_ENV":       true,
	"GO_ENV":        true,
	"PORT":          true,
	"HOST":          true,
	"DATABASE_URL":  true,
	"REDIS_URL":     true,
	"LOG_LEVEL":     true,
	"DEBUG":         true,
}

// redactEnvPatterns mask sensitive env vars
var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(key|secret|token|password|pass|pwd|auth)`),
	regexp.MustCompile(`(?i)^(AWS_|GITHUB_|STRIPE_|TWILIO_)`),
}

// redactEnvVar determines if a variable should be visible
func redactEnvVar(key, value string) (visible bool, redacted string) {
	if safeEnvKeys[key] {
		return true, value
	}

	for _, pattern := range redactPatterns {
		if pattern.MatchString(key) {
			return false, "***"
		}
	}

	return false, "***"
}

// formatUptime formats a start time as human-readable uptime
func formatUptime(startTime time.Time) string {
	elapsed := time.Since(startTime)

	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// detectRuntimeVersion detects the runtime version of a binary
func detectRuntimeVersion(binaryPath string) string {
	if binaryPath == "" {
		return ""
	}

	name := filepath.Base(binaryPath)

	versionFlags := map[string][]string{
		"node":    {"--version"},
		"python3": {"--version"},
		"python":  {"--version"},
		"go":      {"version"},
		"ruby":    {"--version"},
		"java":    {"-version"},
		"php":     {"--version"},
		"cargo":   {"--version"},
	}

	flags, ok := versionFlags[name]
	if !ok {
		return ""
	}

	cmd := exec.Command(binaryPath, flags...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// Parse version from output
	return parseVersionToken(string(output))
}

// parseVersionToken extracts version string from command output
func parseVersionToken(output string) string {
	// Simple extraction of version-like strings
	re := regexp.MustCompile(`v?\d+\.\d+\.\d+`)
	match := re.FindString(output)
	if match != "" {
		return match
	}

	// Try to get first line if no version pattern found
	lines := strings.Split(output, "\n")
	if len(lines) > 0 && lines[0] != "" {
		return strings.TrimSpace(lines[0])
	}

	return ""
}

// Verify component implements Component interface
var _ kernel.Component = (*ProcessMonitor)(nil)
