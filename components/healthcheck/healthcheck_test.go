package healthcheck

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"portkeeper/kernel"
)

// mockLogger implements kernel.Logger for testing
type mockLogger struct {
	mu   sync.Mutex
	logs []string
}

func (ml *mockLogger) Info(msg string, args ...any)  { ml.log("INFO", msg, args) }
func (ml *mockLogger) Warn(msg string, args ...any)  { ml.log("WARN", msg, args) }
func (ml *mockLogger) Error(msg string, args ...any) { ml.log("ERROR", msg, args) }
func (ml *mockLogger) Debug(msg string, args ...any) { ml.log("DEBUG", msg, args) }

func (ml *mockLogger) log(level, msg string, args []any) {
	ml.mu.Lock()
	ml.logs = append(ml.logs, level+": "+msg)
	ml.mu.Unlock()
}

// mockEventBus for testing event emission
type mockEventBus struct {
	mu      sync.Mutex
	emitted []kernel.Event
}

func (meb *mockEventBus) Emit(event kernel.Event, ctx *kernel.Context) error {
	meb.mu.Lock()
	meb.emitted = append(meb.emitted, event)
	meb.mu.Unlock()
	return nil
}

func (meb *mockEventBus) hasEvent(name string) bool {
	meb.mu.Lock()
	defer meb.mu.Unlock()
	for _, e := range meb.emitted {
		if e.Name == name {
			return true
		}
	}
	return false
}

func TestProbeOnlineServer(t *testing.T) {
	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract port from server URL
	port := extractPort(t, server.URL)

	hc := &HealthCheck{
		results: make(map[int]HealthResult),
		config: Config{
			ProbeTimeout: 5 * time.Second,
		},
	}

	result := hc.probeHTTP(port, 5*time.Second)

	if result.Status != HealthOK {
		t.Fatalf("expected HealthOK, got %s", result.Status)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
	if result.LatencyMs <= 0 {
		t.Fatalf("expected latency > 0, got %v", result.LatencyMs)
	}
	if result.Port != port {
		t.Fatalf("expected port %d, got %d", port, result.Port)
	}
}

func TestProbeOfflinePort(t *testing.T) {
	// Use an unlikely port that should not be listening
	port := 54321

	hc := &HealthCheck{
		results: make(map[int]HealthResult),
		config: Config{
			ProbeTimeout:        5 * time.Second,
			MaxConcurrentProbes: 10,
		},
	}

	result := hc.probeHTTP(port, 1*time.Second)

	if result.Status != HealthTimeout {
		t.Fatalf("expected HealthTimeout, got %s", result.Status)
	}
	if result.StatusCode != 0 {
		t.Fatalf("expected statusCode 0, got %d", result.StatusCode)
	}
	if result.Port != port {
		t.Fatalf("expected port %d, got %d", port, result.Port)
	}
}

func TestProbeServerError(t *testing.T) {
	// Start a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	port := extractPort(t, server.URL)

	hc := &HealthCheck{
		results: make(map[int]HealthResult),
		config: Config{
			ProbeTimeout: 5 * time.Second,
		},
	}

	result := hc.probeHTTP(port, 5*time.Second)

	if result.Status != HealthError {
		t.Fatalf("expected HealthError, got %s", result.Status)
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", result.StatusCode)
	}
}

func TestProbeServerWarning(t *testing.T) {
	// Test 4xx response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	port := extractPort(t, server.URL)

	hc := &HealthCheck{
		results: make(map[int]HealthResult),
		config: Config{
			ProbeTimeout: 5 * time.Second,
		},
	}

	result := hc.probeHTTP(port, 5*time.Second)

	if result.Status != HealthWarn {
		t.Fatalf("expected HealthWarn for 4xx, got %s", result.Status)
	}
	if result.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", result.StatusCode)
	}
}

func TestConcurrentProbes(t *testing.T) {
	// Start multiple test servers
	servers := make([]*httptest.Server, 0, 10)
	ports := make([]int, 0, 10)

	for i := 0; i < 10; i++ {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond) // Simulate some work
			w.WriteHeader(http.StatusOK)
		}))
		servers = append(servers, server)
		ports = append(ports, extractPort(t, server.URL))
	}
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	hc := &HealthCheck{
		results: make(map[int]HealthResult),
		config: Config{
			ProbeTimeout: 5 * time.Second,
		},
	}

	start := time.Now()
	results := hc.runAllProbes(ports, 5*time.Second)
	elapsed := time.Since(start)

	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

	// All should be OK
	for _, result := range results {
		if result.Status != HealthOK {
			t.Fatalf("expected HealthOK for port %d, got %s", result.Port, result.Status)
		}
	}

	// Should complete in reasonable time (not sequentially which would be 100ms+)
	if elapsed > 2*time.Second {
		t.Fatalf("concurrent probes took too long: %v", elapsed)
	}
}

func TestComponentInterface(t *testing.T) {
	hc := New()

	if hc.Name() != "healthcheck" {
		t.Fatalf("expected name 'healthcheck', got %s", hc.Name())
	}

	if hc.Version() == "" {
		t.Fatalf("expected non-empty version")
	}

	deps := hc.Dependencies()
	if len(deps) != 0 {
		t.Fatalf("expected no dependencies, got %v", deps)
	}

	schema := hc.ConfigSchema()
	if schema == nil {
		t.Fatalf("expected non-nil config schema")
	}

	hooks := hc.Hooks()
	if len(hooks) == 0 {
		t.Fatalf("expected at least one hook")
	}
}

func TestInitAndStart(t *testing.T) {
	logger := &mockLogger{}

	hc := New()

	mockKernel := &kernel.Kernel{}
	ctx := &kernel.Context{
		Kernel:     mockKernel,
		Logger:     logger,
		Components: make(map[string]kernel.Component),
		Config:     make(map[string]json.RawMessage),
	}

	// Test Init with default config
	err := hc.Init(ctx, nil)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Start should succeed
	err = hc.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop should succeed
	err = hc.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestGetHealthResults(t *testing.T) {
	hc := New()
	logger := &mockLogger{}

	ctx := &kernel.Context{
		Kernel:     &kernel.Kernel{},
		Logger:     logger,
		Components: make(map[string]kernel.Component),
		Config:     make(map[string]json.RawMessage),
	}

	err := hc.Init(ctx, nil)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Manually add a result
	hc.mu.Lock()
	hc.results[8080] = HealthResult{
		Port:      8080,
		Status:    HealthOK,
		StatusCode: 200,
		LatencyMs: 42.5,
		Protocol:  "http",
		CheckedAt: time.Now(),
	}
	hc.mu.Unlock()

	results := hc.GetHealthResults()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Port != 8080 {
		t.Fatalf("expected port 8080, got %d", results[0].Port)
	}
	if results[0].Status != HealthOK {
		t.Fatalf("expected HealthOK, got %s", results[0].Status)
	}
	if results[0].StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", results[0].StatusCode)
	}
}

func TestProtocolDetection(t *testing.T) {
	tests := []struct {
		port     int
		expected string
	}{
		{5432, "postgres"},
		{3306, "mysql"},
		{3307, "mysql"},
		{6379, "redis"},
		{27017, "mongodb"},
		{11211, "memcached"},
		{5672, "amqp"},
		{9200, "http"},
		{8080, "http"},
		{3000, "http"},
	}

	for _, tt := range tests {
		got := getProtocol(tt.port)
		if got != tt.expected {
			t.Errorf("port %d: expected %q, got %q", tt.port, tt.expected, got)
		}
	}
}

// Helper function to extract port from httptest server URL
func extractPort(t *testing.T, url string) int {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	portStr := req.URL.Port()
	if portStr == "" {
		t.Fatalf("no port in URL: %s", url)
	}

	var port int
	for _, c := range portStr {
		if c < '0' || c > '9' {
			t.Fatalf("invalid port: %s", portStr)
		}
		port = port*10 + int(c-'0')
	}
	return port
}
