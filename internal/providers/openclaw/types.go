package openclaw

// StatusResponse is the top-level response from `openclaw gateway call status --json`.
type StatusResponse struct {
	RuntimeVersion string        `json:"runtimeVersion"`
	Sessions       SessionsBlock `json:"sessions"`
}

// SessionsBlock contains aggregated session data grouped by agent.
type SessionsBlock struct {
	Count   int             `json:"count"`
	ByAgent []AgentSessions `json:"byAgent"`
}

// AgentSessions groups sessions belonging to a single agent.
type AgentSessions struct {
	AgentID string      `json:"agentId"`
	Count   int         `json:"count"`
	Recent  []OCSession `json:"recent"`
}

// OCSession represents a single OpenClaw session.
type OCSession struct {
	AgentID         string `json:"agentId"`
	Key             string `json:"key"`
	Kind            string `json:"kind"`      // "direct" | "group"
	SessionID       string `json:"sessionId"`
	UpdatedAt       int64  `json:"updatedAt"` // unix ms
	Age             int64  `json:"age"`       // ms since last update
	AbortedLastRun  bool   `json:"abortedLastRun"`
	InputTokens     int64  `json:"inputTokens"`
	OutputTokens    int64  `json:"outputTokens"`
	CacheRead       int64  `json:"cacheRead"`
	CacheWrite      int64  `json:"cacheWrite"`
	TotalTokens     *int64 `json:"totalTokens"`
	RemainingTokens *int64 `json:"remainingTokens"`
	PercentUsed     *int   `json:"percentUsed"`
	Model           string `json:"model"`
	ContextTokens   int64  `json:"contextTokens"`
}

// SessionKeyInfo holds parsed components from an OpenClaw session key.
type SessionKeyInfo struct {
	Agent       string
	SessionType string // "direct", "group", "cron", "subagent", "cron_run"
	Channel     string // "discord", etc. (empty for cron/subagent)
}

// ToLabels converts SessionKeyInfo to a labels map for MonitorEvent.
func (i SessionKeyInfo) ToLabels() map[string]string {
	labels := map[string]string{
		"agent":        i.Agent,
		"session_type": i.SessionType,
	}
	if i.Channel != "" {
		labels["channel"] = i.Channel
	}
	return labels
}

// ParseSessionKey extracts agent, session type, and channel from an OpenClaw session key.
//
// Key formats:
//
//	agent:r2d2:discord:direct:1088...     → agent session (direct)
//	agent:r2d2:discord:group:1088...      → agent session (group)
//	agent:r2d2:cron:4b62933d...           → cron job
//	agent:r2d2:subagent:uuid              → sub-agent
//	agent:r2d2:cron:...:run:uuid          → cron run instance
func ParseSessionKey(key string) SessionKeyInfo {
	parts := splitKey(key)

	// Must start with "agent" and have at least agent ID
	if len(parts) < 3 || parts[0] != "agent" {
		return SessionKeyInfo{SessionType: "unknown"}
	}

	agent := parts[1]
	segment := parts[2]

	// Check for cron run: agent:NAME:cron:ID:run:UUID
	if segment == "cron" && len(parts) >= 6 && parts[4] == "run" {
		return SessionKeyInfo{
			Agent:       agent,
			SessionType: "cron_run",
		}
	}

	// cron: agent:NAME:cron:ID
	if segment == "cron" {
		return SessionKeyInfo{
			Agent:       agent,
			SessionType: "cron",
		}
	}

	// subagent: agent:NAME:subagent:UUID
	if segment == "subagent" {
		return SessionKeyInfo{
			Agent:       agent,
			SessionType: "subagent",
		}
	}

	// Channel-based: agent:NAME:CHANNEL:KIND:ID
	if len(parts) >= 5 {
		channel := segment
		kind := parts[3] // "direct" or "group"
		return SessionKeyInfo{
			Agent:       agent,
			SessionType: kind,
			Channel:     channel,
		}
	}

	return SessionKeyInfo{
		Agent:       agent,
		SessionType: "unknown",
	}
}

// splitKey splits a colon-separated key into parts.
func splitKey(key string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}
