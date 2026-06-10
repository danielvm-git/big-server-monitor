# Slice 6: Notifications Component — "Alert Me When Things Break"

**type:** epic  
**status:** planning  
**verify:** killing a watched process triggers a macOS notification banner within 10s

## Purpose

Delivers macOS system notifications when server crashes occur unexpectedly. Respects the user's notification preferences in Settings.

## Scope

- Subscribe to `process.crashed` events
- Request `UNUserNotificationCenter` authorization on first launch
- Deliver notification banner with title, body, and "View Details" action
- Respect `config.notifications.crashAlerts` toggle
- Rate-limit: max 1 notification per port per 60s (prevents spam on flapping servers)
- On "View Details" action: emit `ui.openDropdown` event + highlight the server row

## Notification format

```
Title:  Port :3000 went offline
Body:   node (bigbase-api) stopped after 2h 03m
Action: View Details
```

## Wails notification API

```go
import "github.com/wailsapp/wails/v2/pkg/runtime"

func (n *Notifications) deliverCrash(event ProcessCrashedEvent) {
    if !n.config.CrashAlerts { return }
    if n.isRateLimited(event.Server.Port) { return }

    runtime.Notify(n.ctx, runtime.NotifyOptions{
        Title:   fmt.Sprintf("Port :%d went offline", event.Server.Port),
        Message: fmt.Sprintf("%s (%s) stopped after %s",
            event.Server.ProcessName,
            event.Server.ProjectName,
            event.Server.UptimeStr,
        ),
        Actions: []runtime.NotifyAction{
            {Identifier: "view", Title: "View Details"},
        },
    })
    n.markRateLimited(event.Server.Port)
}
```

## Rate limiting

```go
type rateLimiter struct {
    mu      sync.Mutex
    lastAt  map[int]time.Time  // port → last notification time
    cooldown time.Duration     // default 60s
}
```

## API surface

```go
func (n *Notifications) RequestPermission() error
func (n *Notifications) HasPermission() bool
```

## Tests

```go
// Crash event when CrashAlerts=false → no notification delivered
func TestNotificationRespectsSetting(t *testing.T)

// Two crashes on same port within 60s → only one notification
func TestRateLimiting(t *testing.T)
```
