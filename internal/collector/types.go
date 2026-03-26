package collector

// EventType identifies the kind of monitor event.
type EventType string

const (
	EventSessionStart EventType = "session.start"
	EventSessionEnd   EventType = "session.end"
	EventStatusChange EventType = "status.change"
	EventToolStart    EventType = "tool.start"
	EventToolEnd      EventType = "tool.end"
	EventMessage      EventType = "message"
	EventError        EventType = "error"
)

// SessionStatus represents the current state of a session.
type SessionStatus string

const (
	StatusIdle        SessionStatus = "idle"
	StatusThinking    SessionStatus = "thinking"
	StatusToolRunning SessionStatus = "tool_running"
	StatusWaiting     SessionStatus = "waiting"
	StatusDone        SessionStatus = "done"
	StatusError       SessionStatus = "error"
)

// TokenUsage tracks token consumption for a session or event.
type TokenUsage struct {
	Input     int64 `json:"input"`
	Output    int64 `json:"output"`
	CacheRead int64 `json:"cacheRead,omitempty"`
}

// EventDetail contains optional details about an event.
type EventDetail struct {
	Tool       string      `json:"tool,omitempty"`
	Target     string      `json:"target,omitempty"`
	Message    string      `json:"message,omitempty"`
	ElapsedMs  int64       `json:"elapsedMs,omitempty"`
	TokenUsage *TokenUsage `json:"tokenUsage,omitempty"`
}

// MonitorEvent is the unified event schema emitted by all providers.
type MonitorEvent struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	SessionID string           `json:"sessionId"`
	Workspace string           `json:"workspace,omitempty"`
	Timestamp int64            `json:"timestamp"`
	Event     EventType        `json:"event"`
	Status    SessionStatus    `json:"status"`
	Detail    *EventDetail     `json:"detail,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// SessionState is the aggregated state of a session, derived from events.
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
