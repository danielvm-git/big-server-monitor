# Slice 3: HealthCheck Component — "Are They Responding?"

**type:** epic  
**status:** planning  
**verify:** `GET /health` returns HTTP status + latency for each online server within 5s

## Purpose

Probes each monitored port with an HTTP HEAD request, records the response status and latency, and marks servers as "unresponsive" if they don't reply within the timeout. Runs on a configurable interval and on-demand.

## Scope

- HTTP HEAD probe per online server
- Record: status code, latency ms, timestamp, content-type, server header
- Timeout: 3s per probe (configurable)
- Concurrency: all probes in parallel, bounded goroutine pool (max 10)
- Auto-refresh on configurable interval (default 30s)
- On-demand trigger: `RunHealthCheck()` binding
- Emit `process.unresponsive` when a listening port returns no HTTP response
- Expose results via `GetHealthResults() []HealthResult`

## Data model

```go
type HealthResult struct {
    Port       int           `json:"port"`
    Status     HealthStatus  `json:"status"`     // ok, warn, error, timeout
    StatusCode int           `json:"statusCode"`  // 0 if timeout or non-HTTP
    LatencyMs  float64       `json:"latencyMs"`
    Protocol   string        `json:"protocol"`   // "http", "https", "postgres", "redis", etc.
    Scheme     string        `json:"scheme"`      // http or https (HTTP only)
    CheckedAt  time.Time     `json:"checkedAt"`
    Error      string        `json:"error,omitempty"`
}

type HealthStatus string
const (
    HealthOK      HealthStatus = "ok"       // 2xx
    HealthWarn    HealthStatus = "warn"      // 3xx or 4xx
    HealthError   HealthStatus = "error"     // 5xx
    HealthTimeout HealthStatus = "timeout"   // no response
)
```

## Protocol-aware probe dispatch

Different port types require different probe strategies. ServBay-inspired: HTTP HEAD is wrong for databases — use a protocol-specific handshake instead.

```go
// Known DB/service ports → protocol type
var knownProtocols = map[int]string{
    5432:  "postgres",
    3306:  "mysql",
    3307:  "mysql",
    6379:  "redis",
    27017: "mongodb",
    11211: "memcached",
    5672:  "amqp",    // RabbitMQ
    9200:  "http",    // Elasticsearch (HTTP API)
}

func probe(server Server, timeout time.Duration) HealthResult {
    proto := knownProtocols[server.Port]
    if proto == "" {
        proto = "http"  // default: try HTTP
    }
    switch proto {
    case "postgres":  return probePostgres(server.Port, timeout)
    case "mysql":     return probeMySQL(server.Port, timeout)
    case "redis":     return probeRedis(server.Port, timeout)
    case "mongodb":   return probeMongoDB(server.Port, timeout)
    case "memcached": return probeMemcached(server.Port, timeout)
    default:          return probeHTTP(server.Port, timeout)
    }
}
```

### HTTP probe (default)

```go
func probeHTTP(port int, timeout time.Duration) HealthResult {
    // try http first, then https on failure
    schemes := []string{"http", "https"}
    for _, scheme := range schemes {
        url := fmt.Sprintf("%s://localhost:%d/", scheme, port)
        client := &http.Client{Timeout: timeout, CheckRedirect: noFollow}
        start := time.Now()
        resp, err := client.Head(url)
        latency := time.Since(start).Seconds() * 1000
        if err == nil {
            return HealthResult{StatusCode: resp.StatusCode, LatencyMs: latency, ...}
        }
    }
    return HealthResult{Status: HealthTimeout, ...}
}
```

### Database TCP probes

```go
func probePostgres(port int, timeout time.Duration) HealthResult {
    // Open TCP connection → read startup message (first 5 bytes)
    // Postgres sends: 'R' (auth request) or 'E' (error) — either means it's alive
    conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
    if err != nil { return HealthResult{Status: HealthTimeout} }
    defer conn.Close()
    conn.SetReadDeadline(time.Now().Add(timeout))
    buf := make([]byte, 5)
    conn.Read(buf)
    return HealthResult{Status: HealthOK, Protocol: "postgres", ...}
}

func probeRedis(port int, timeout time.Duration) HealthResult {
    // Send PING\r\n → expect +PONG\r\n
    conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
    if err != nil { return HealthResult{Status: HealthTimeout} }
    defer conn.Close()
    fmt.Fprintf(conn, "PING\r\n")
    buf := make([]byte, 7)
    conn.Read(buf)
    if string(buf) == "+PONG\r\n" {
        return HealthResult{Status: HealthOK, Protocol: "redis", ...}
    }
    return HealthResult{Status: HealthError, ...}
}

// MySQL: read server greeting (first byte 0x0a = protocol v10)
// MongoDB: send isMaster command over wire protocol
// Memcached: send "version\r\n" → expect "VERSION x.y.z\r\n"
```

## Auto-refresh loop

```go
func (hc *HealthCheck) Start(ctx *Context) error {
    hc.runAll(ctx) // immediate first run
    ticker := time.NewTicker(hc.config.HealthCheckInterval)
    go func() {
        for {
            select {
            case <-ticker.C:  hc.runAll(ctx)
            case <-ctx.Done(): return
            }
        }
    }()
    return nil
}
```

## API surface

```go
func (hc *HealthCheck) GetHealthResults() []HealthResult
func (hc *HealthCheck) RunHealthCheck() []HealthResult   // on-demand, blocks until done
```

## Tests

```go
// Mock HTTP server on a random port → assert HealthOK + latency > 0
func TestProbeOnlineServer(t *testing.T)

// No server on port → assert HealthTimeout
func TestProbeOfflinePort(t *testing.T)

// 5xx response → assert HealthError
func TestProbeServerError(t *testing.T)

// All probes run concurrently in < timeout+1s for 10 servers
func TestConcurrentProbes(t *testing.T)
```
