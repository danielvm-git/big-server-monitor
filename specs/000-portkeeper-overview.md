# PortKeeper — Product Overview

**type:** prd  
**status:** planning  
**verify:** app appears in macOS menubar and lists running processes on `:*` ports

## What it is

PortKeeper is a macOS menubar app that monitors locally running development servers. It lives in the system menu bar — always visible, zero friction — and gives the developer instant visibility into what is running, what crashed, and how to debug it.

It belongs to the same product family as **Big DockLocker**: same company, same BigBase Design System (indigo accent, Inter type, Lucide icons), same architectural DNA (BigBase ECC pattern with a Go kernel + pluggable components).

## Why it exists

Solo devs running AI agents and multi-service local stacks lose time switching to a terminal to check which process is alive, which port is in use, and what error killed a server. PortKeeper collapses that context-switch to a single click.

Primary persona: a developer running 2–5 local servers simultaneously, using AI coding agents, who needs to paste logs quickly into a chat to fix errors.

## Core screens

| Screen | Access | Purpose |
|---|---|---|
| Menubar icon + badge | always visible | Shows live count of active servers |
| Main dropdown (popover) | click icon | List of all monitored servers |
| Server detail expand | click row | PID, memory, binary path |
| Server logs sheet | click logs icon | Last 30 lines stdout/stderr + "Copy for AI" |
| Health Check sheet | button in dropdown | HTTP HEAD results per port |
| Activity Log sheet | button in dropdown | Timeline of start/crash/unresponsive events |
| Settings modal | button in dropdown | Directories, polling, toggles |
| Kill confirm dialog | click ✕ on server row | "Kill node on :3000?" [Cancel] [Kill] |
| macOS notification | system | Server crash banner with "View Details" |

## Feature requirements

### Menubar icon
- Small SVG icon (⚡ lightning bolt, BigBase indigo tint on active state)
- Red badge showing count of active servers
- Click opens/closes popover

### Main dropdown
- Header: "PortKeeper" title + "N active" subtitle + refresh button
- Server rows: status dot (green/red/grey) + port + process name + project name + uptime + kill button
- Hover on row: expand shows PID, binary path, memory usage
- Section "Projects": groups servers by parent directory
- Footer buttons: Health Check / Activity Log / Settings

### Health Check sheet
- Per-port: HTTP HEAD result (status code + latency ms) + color indicator
- Auto-refresh every 30s (configurable)
- "Test All Now" button

### Activity Log sheet
- Vertical timeline: started / crashed / unresponsive events
- Filter by project, port, event type
- "Clear History" button

### Server Logs sheet
- Last 30 lines of stdout/stderr per process
- Filter: All / Errors / Warnings (with count)
- "Copy" — copies visible lines
- "Copy for AI" — formats full context block (server name, port, process, PID, memory, binary, logs) ready to paste into an AI agent

### Settings modal
- Scan directories (editable list, default: `~/projects`, `~/opensrc`, `~/Developer`)
- Polling interval (default: 5s)
- Health check interval (default: 30s)
- Ignore ports list (default: 80, 443, 5432, 3306, 6379)
- Toggle: crash notifications
- Toggle: launch at login (macOS login item)
- Toggle: show badge count on icon

### macOS notifications
- Trigger: server process disappears unexpectedly
- Title: "Port :3000 went offline"
- Body: "node (bigbase-api) stopped after 2h 03m"
- Action: "View Details" → opens dropdown + highlights server row

## Visual style

Design system: **BigBase Design System** (shared with BigBase Admin Console and Big DockLocker).

| Token role | macOS adaptation |
|---|---|
| `--blue` (#007AFF) | system accent, interactive elements |
| `--green` (#30D158) | online status dot |
| `--red` (#FF3B30) | offline dot, kill button, badge |
| `--orange` (#FF9F0A) | warning / unresponsive state |
| Font UI | `-apple-system, BlinkMacSystemFont` (SF Pro) |
| Font mono | `'SF Mono', Menlo` |
| Vibrancy | `backdrop-filter: blur(40px) saturate(200%)` on menubar and popover |
| Popover radius | 12px |
| Card bg | `rgba(0,0,0,0.04)` light / `rgba(255,255,255,0.06)` dark |

Light and dark mode: full support via `[data-theme="dark"]` CSS attribute.

## ServBay-inspired additions (observatory-only)

These features were identified by comparing PortKeeper to ServBay. All are read-only — PortKeeper observes, never manages.

| Feature | How PortKeeper adapts it |
|---|---|
| **Runtime version tracking** | Executes `node --version` / `python --version` / etc. once per PID; shows version chip next to process name |
| **Local domain resolution** | Reads `/etc/hosts` at startup; surfaces `myapp.local` aliases in the expanded server row as clickable links |
| **Ngrok/FRP tunnel awareness** | Polls ngrok's local API (`localhost:4040/api/tunnels`) when ngrok is detected; shows public URL in the server row |
| **Protocol-aware health checks** | Uses TCP handshakes (not HTTP HEAD) for DB ports — Postgres, MySQL, Redis, MongoDB, Memcached get correct liveness checks |
| **Env var snapshot** | Reads `.env` / `.env.local` from the project directory; shows safe keys (NODE_ENV, PORT, LOG_LEVEL) with values, redacts secrets (`*_KEY`, `*_TOKEN`, `*_PASSWORD`, etc.) |

## Non-goals (v1)

- Remote server monitoring (localhost only)
- Docker container tracking
- Custom alert rules / webhooks
- iOS / iPad companion app
- Windows / Linux support
