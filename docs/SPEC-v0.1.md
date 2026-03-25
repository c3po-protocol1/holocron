# Holocron — Specification v0.1

> "If a record is not in the Holocron, it does not exist."

## Vision

A standalone developer tool that provides **real-time visibility** into all AI coding sessions and orchestrator sessions running on your machine — regardless of which tool or orchestrator is used.

**Not** an orchestration dashboard. **Not** tied to any single platform.
Think `htop` for AI-assisted development.

**Name:** Holocron (`holo` CLI alias)
**Language:** Go
**TUI:** Bubbletea (Charm ecosystem)
**Storage:** SQLite

## Core Principles

1. **Loose coupling** — Each source (Claude Code, Codex, OpenClaw, etc.) is an independent stream. No source knows about another.
2. **Config over magic** — You tell it where to look (source types). It discovers sessions within those sources automatically.
3. **View-layer linking** — Associating "R2 is using this Claude Code session" is a UI concern, not a data concern.
4. **Renderer-agnostic** — TUI first, Web later. Both consume the same unified event stream.
5. **Works today** — No fantasy protocols. Uses what tools actually expose right now.
6. **Separation of concerns** — Collector, Providers, Store, and Renderer are independent packages. Process boundaries can be drawn later without code changes.

## Architecture

```
holocron/
├── internal/
│   ├── collector/          ← Core: EventBus, orchestration
│   │   ├── collector.go
│   │   ├── bus.go          ← EventBus (pub/sub)
│   │   └── types.go        ← MonitorEvent, SessionState
│   │
│   ├── store/              ← Persistence
│   │   ├── store.go        ← Store interface
│   │   └── sqlite/
│   │       └── sqlite.go   ← SQLite implementation
│   │
│   ├── provider/           ← Provider interface + registry
│   │   └── provider.go
│   │
│   ├── providers/          ← Source adapters
│   │   ├── claudecode/     ← Claude Code (file scan + ps + JSONL tail)
│   │   ├── openclaw/       ← OpenClaw (Gateway WS/HTTP)
│   │   └── codex/          ← Codex CLI
│   │
│   └── tui/                ← Bubbletea TUI
│       ├── app.go
│       ├── session_list.go
│       └── detail.go
│
├── cmd/
│   └── holocron/
│       └── main.go         ← Entry point
│
├── config.yaml             ← Example config
├── go.mod
└── README.md
```

### Data Flow

```
Phase 1 (on-demand, single process):

  [Claude Code Provider] ──┐
  [OpenClaw Provider]   ───┤──▶ [EventBus] ──▶ [SQLite Store]
  [Codex Provider]      ───┘        │
                                    ▼
                               [TUI Renderer]
                               (subscribes to chan MonitorEvent)

Future (daemon + client):

  [Collector Daemon]                [TUI Client]
  ┌──────────────────────┐         ┌──────────────┐
  │ Providers → EventBus │──WS──▶  │ Subscribe()  │
  │         → SQLite     │         │ Render()     │
  └──────────────────────┘         └──────────────┘
                                   [Web Client]
                                   ┌──────────────┐
                               ──▶ │ Browser UI   │
                                   └──────────────┘
```

### Key Interface: TUI sees only `<-chan MonitorEvent`

```go
// TUI doesn't know or care where events come from
type Renderer interface {
    Run(events <-chan MonitorEvent) error
}
```

Same channel whether events come from in-process Collector or remote WebSocket.

## Unified Event Schema

```go
type MonitorEvent struct {
    ID        string            `json:"id"`
    Source    string            `json:"source"`     // "claude-code" | "openclaw" | "codex"
    SessionID string           `json:"sessionId"`
    Workspace string           `json:"workspace,omitempty"`
    Timestamp int64            `json:"timestamp"`   // unix ms

    Event     EventType        `json:"event"`
    Status    SessionStatus    `json:"status"`

    Detail    *EventDetail     `json:"detail,omitempty"`
    Labels    map[string]string `json:"labels,omitempty"`
}

type EventType string
const (
    EventSessionStart  EventType = "session.start"
    EventSessionEnd    EventType = "session.end"
    EventStatusChange  EventType = "status.change"
    EventToolStart     EventType = "tool.start"
    EventToolEnd       EventType = "tool.end"
    EventMessage       EventType = "message"
    EventError         EventType = "error"
)

type SessionStatus string
const (
    StatusIdle        SessionStatus = "idle"
    StatusThinking    SessionStatus = "thinking"
    StatusToolRunning SessionStatus = "tool_running"
    StatusWaiting     SessionStatus = "waiting"
    StatusDone        SessionStatus = "done"
    StatusError       SessionStatus = "error"
)

type EventDetail struct {
    Tool       string     `json:"tool,omitempty"`
    Target     string     `json:"target,omitempty"`
    Message    string     `json:"message,omitempty"`
    ElapsedMs  int64      `json:"elapsedMs,omitempty"`
    TokenUsage *TokenUsage `json:"tokenUsage,omitempty"`
}

type TokenUsage struct {
    Input      int64 `json:"input"`
    Output     int64 `json:"output"`
    CacheRead  int64 `json:"cacheRead,omitempty"`
}
```

## Session State (aggregated)

```go
type SessionState struct {
    Source        string            `json:"source"`
    SessionID     string            `json:"sessionId"`
    Workspace     string            `json:"workspace,omitempty"`
    Status        SessionStatus     `json:"status"`
    StartedAt     int64             `json:"startedAt"`
    LastEventAt   int64             `json:"lastEventAt"`
    ElapsedMs     int64             `json:"elapsedMs"`
    CurrentTool   string            `json:"currentTool,omitempty"`
    CurrentTarget string            `json:"currentTarget,omitempty"`
    EventCount    int               `json:"eventCount"`
    Labels        map[string]string `json:"labels,omitempty"`
    TokenUsage    *TokenUsage       `json:"tokenUsage,omitempty"`
}
```

## Core Interfaces

```go
// Provider — each source implements this
type Provider interface {
    Name() string
    Start(ctx context.Context, bus EventBus) error
    Stop() error
}

// EventBus — pub/sub for events
type EventBus interface {
    Publish(event MonitorEvent)
    Subscribe() <-chan MonitorEvent
    Unsubscribe(ch <-chan MonitorEvent)
}

// Store — persistence
type Store interface {
    Save(event MonitorEvent) error
    ListSessions() ([]SessionState, error)
    GetSession(source, sessionID string) (*SessionState, error)
    GetEvents(source, sessionID string, since int64, limit int) ([]MonitorEvent, error)
}
```

## Source Providers

### Provider: Claude Code

**Discovery methods:**
| Method | What it gives |
|--------|--------------|
| `~/.claude/projects/` scan | Session history (JSONL), workspace mapping |
| `ps aux \| grep claude` | Currently running sessions |
| JSONL file tailing (`fsnotify`) | Near-real-time events from active sessions |

**Session JSONL format** (observed):
- Each line: JSON with `type`, `sessionId`, `timestamp`
- Types: `user`, `assistant`, `tool_use`, `tool_result`, `queue-operation`
- Contains: `cwd`, `gitBranch`, `version`

**Config:**
```yaml
sources:
  - type: claude-code
    discover: auto
    sessionDir: ~/.claude/projects/    # override if non-standard
    watchProcesses: true
    tailActive: true
    pollIntervalMs: 2000               # process detection interval
```

### Provider: OpenClaw

**Discovery methods:**
| Method | What it gives |
|--------|--------------|
| `gateway call status --json` | All agents, sessions, tokens, timestamps |
| Gateway WebSocket | Real-time events |

**Config:**
```yaml
sources:
  - type: openclaw
    gateway: ws://127.0.0.1:18789
    token: ${OPENCLAW_GATEWAY_TOKEN}
    pollIntervalMs: 5000
```

### Provider: Codex CLI

**Config:**
```yaml
sources:
  - type: codex
    discover: auto
    watchProcesses: true
```

### Provider: Generic (file-watch)

For unsupported tools — drop JSONL in a watched directory:

```yaml
sources:
  - type: file-watch
    path: /tmp/holocron/*.jsonl
    format: monitor-event
```

## Configuration

```yaml
# ~/.holocron/config.yaml

sources:
  - type: claude-code
    discover: auto
    watchProcesses: true
    tailActive: true

  - type: openclaw
    gateway: ws://127.0.0.1:18789
    token: ${OPENCLAW_GATEWAY_TOKEN}

store:
  type: sqlite
  path: ~/.holocron/holocron.db
  retentionDays: 30           # auto-cleanup old events

view:
  refreshMs: 1000
  showIdle: true
  groupBy: source              # source | workspace | label

labels:
  rules:
    - match:
        source: openclaw
        sessionKey: "agent:r2d2:*"
      set:
        agent: r2d2
    - match:
        source: claude-code
        workspace: "/Users/*/Projects/*"
      set:
        context: project
```

## TUI Layout (Bubbletea)

```
┌─ Holocron 🔭 ────────────────────────────────────┐
│                                                    │
│  SOURCE         SESSION       STATUS    ELAPSED    │
│  ────────────────────────────────────────────────  │
│▶ claude-code    a8837a23..   ● editing  2m 13s    │
│                 ~/Projects/my-app                   │
│                 Edit → src/index.ts                  │
│                                                    │
│  claude-code    323ac29b..   ◌ idle     15m 02s   │
│                 ~/Projects/agent-monitor             │
│                                                    │
│  openclaw       r2d2         ● running  5m 30s    │
│                 cron:4b62933d                        │
│                 tokens: 12.5k / 1M (1%)             │
│                                                    │
│  openclaw       yoda         ● active   0m 45s    │
│                 discord:direct                       │
│                 tokens: 16.7k / 1M (2%)             │
│                                                    │
├────────────────────────────────────────────────────┤
│ [q]uit  [l]abels  [g]roup  [f]ilter  [d]etail     │
│ 4 sessions │ 2 active │ SQLite: 1,247 events       │
└────────────────────────────────────────────────────┘
```

**Key bindings:**
- `↑/↓` or `j/k`: navigate
- `Enter`: detail view (event log)
- `l`: assign/edit labels
- `g`: cycle groupBy
- `f`: filter by source/status/label
- `d`: toggle side detail panel
- `r`: refresh now
- `q`: quit
- `?`: help

## CLI Commands

```bash
holo                    # Launch TUI (default)
holo tui                # Same as above
holo status             # One-shot: print current sessions
holo sources            # List configured sources
holo tail <sessionId>   # Stream events for a session
holo history            # Query stored events
holo config             # Show/edit config
holo version            # Version info
```

## SQLite Schema

```sql
CREATE TABLE events (
    id          TEXT PRIMARY KEY,
    source      TEXT NOT NULL,
    session_id  TEXT NOT NULL,
    workspace   TEXT,
    timestamp   INTEGER NOT NULL,
    event       TEXT NOT NULL,
    status      TEXT NOT NULL,
    detail_json TEXT,
    labels_json TEXT,
    created_at  INTEGER DEFAULT (strftime('%s','now') * 1000)
);

CREATE INDEX idx_events_session ON events(source, session_id);
CREATE INDEX idx_events_timestamp ON events(timestamp);

CREATE TABLE sessions (
    source        TEXT NOT NULL,
    session_id    TEXT NOT NULL,
    workspace     TEXT,
    status        TEXT NOT NULL,
    started_at    INTEGER,
    last_event_at INTEGER,
    event_count   INTEGER DEFAULT 0,
    labels_json   TEXT,
    token_json    TEXT,
    PRIMARY KEY (source, session_id)
);
```

## Implementation Phases

### Phase 1: Core + Claude Code + TUI
- [ ] Project scaffold (Go modules, directory structure)
- [ ] Unified event types
- [ ] EventBus (in-memory pub/sub)
- [ ] SQLite store
- [ ] Claude Code provider (session scan + process detect + JSONL tail)
- [ ] Bubbletea TUI (session list + status)
- [ ] Config file loading
- [ ] `holo` and `holo status` commands

### Phase 2: OpenClaw Provider
- [ ] Gateway HTTP polling (`status --json`)
- [ ] Gateway WebSocket (if API available)
- [ ] Agent + session mapping
- [ ] Token usage display

### Phase 3: Labels & Linking
- [ ] Label system (manual + rule-based)
- [ ] GroupBy / Filter in TUI
- [ ] Session detail view (event log)

### Phase 4: Daemon + Web
- [ ] Split into daemon + client
- [ ] WebSocket API for external clients
- [ ] Web UI (Vite + framework TBD)
- [ ] Embed web UI in Go binary (`embed.FS`)

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go |
| TUI | Bubbletea + Lipgloss + Bubbles (Charm) |
| Storage | SQLite (via `modernc.org/sqlite` — pure Go, no CGO) |
| Config | YAML (`gopkg.in/yaml.v3`) |
| File watching | `fsnotify/fsnotify` |
| WebSocket | `gorilla/websocket` or `nhooyr.io/websocket` |
| CLI | `spf13/cobra` |
| Logging | `log/slog` (stdlib) |

## Open Questions

1. ~~Language~~ → **Go** ✅
2. ~~Daemon vs on-demand~~ → **On-demand with separated architecture, SQLite from day 1** ✅
3. ~~Naming~~ → **Holocron** ✅
4. **Distribution** — Homebrew tap? `go install`? Both?
5. **Testing strategy** — provider mocks? integration tests with real Claude Code sessions?

---

*Spec authored during brainstorming session, 2026-03-25*
*Yoda + Soo*
