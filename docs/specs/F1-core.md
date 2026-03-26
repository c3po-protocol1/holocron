# F1: Core Types + EventBus + SQLite Store

> The foundation. Nothing works without this.

## What This Feature Does

Defines the unified event schema, provides in-process event routing (EventBus), and persists events to SQLite. No UI, no providers — just the data backbone.

## Packages

- `internal/collector/types.go` — MonitorEvent, SessionState, enums
- `internal/collector/bus.go` — EventBus (channel-based pub/sub)
- `internal/store/store.go` — Store interface
- `internal/store/sqlite/` — SQLite implementation

## Event Schema

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
    Input     int64 `json:"input"`
    Output    int64 `json:"output"`
    CacheRead int64 `json:"cacheRead,omitempty"`
}
```

## Session State (aggregated from events)

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

## EventBus

```go
type EventBus interface {
    Publish(event MonitorEvent)
    Subscribe() <-chan MonitorEvent
    Unsubscribe(ch <-chan MonitorEvent)
}
```

- Multiple subscribers allowed (Store + TUI)
- Non-blocking publish (drop if subscriber is slow, log warning)
- Channel buffer size: 256

## Store Interface

```go
type Store interface {
    Save(event MonitorEvent) error
    ListSessions() ([]SessionState, error)
    GetSession(source, sessionID string) (*SessionState, error)
    GetEvents(source, sessionID string, since int64, limit int) ([]MonitorEvent, error)
    Close() error
}
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

`sessions` table is updated on every `Save()` via upsert — a materialized view of latest state per session.

## Verification

```bash
# Unit tests pass
go test ./internal/collector/... ./internal/store/...

# Specific checks:
# 1. Publish event → subscriber receives it
# 2. Publish event → Store.Save persists to SQLite
# 3. ListSessions returns aggregated state
# 4. GetEvents returns filtered, ordered events
# 5. Multiple subscribers each get every event
# 6. Unsubscribe stops delivery
```
