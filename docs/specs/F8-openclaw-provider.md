# F8: OpenClaw Provider

> Monitor all OpenClaw agents and sessions via polling + smart diff.

## What This Feature Does

Polls `openclaw gateway call status --json` at a configurable interval, diffs snapshots to detect changes, and emits unified MonitorEvents for agent sessions, sub-agents, and cron runs.

## Package

- `internal/providers/openclaw/openclaw.go` — Provider implementation
- `internal/providers/openclaw/poller.go` — Status polling + JSON parsing
- `internal/providers/openclaw/differ.go` — Snapshot diffing → event emission
- `internal/providers/openclaw/types.go` — OpenClaw status response types

## Data Source

**Command:** `openclaw gateway call status --json`

**Response structure (relevant fields):**
```go
type StatusResponse struct {
    RuntimeVersion string         `json:"runtimeVersion"`
    Sessions       SessionsBlock  `json:"sessions"`
}

type SessionsBlock struct {
    Count   int             `json:"count"`
    ByAgent []AgentSessions `json:"byAgent"`
}

type AgentSessions struct {
    AgentID string          `json:"agentId"`
    Count   int             `json:"count"`
    Recent  []OCSession     `json:"recent"`
}

type OCSession struct {
    AgentID         string  `json:"agentId"`
    Key             string  `json:"key"`
    Kind            string  `json:"kind"`         // "direct" | "group"
    SessionID       string  `json:"sessionId"`
    UpdatedAt       int64   `json:"updatedAt"`    // unix ms
    Age             int64   `json:"age"`          // ms since last update
    AbortedLastRun  bool    `json:"abortedLastRun"`
    InputTokens     int64   `json:"inputTokens"`
    OutputTokens    int64   `json:"outputTokens"`
    CacheRead       int64   `json:"cacheRead"`
    CacheWrite      int64   `json:"cacheWrite"`
    TotalTokens     *int64  `json:"totalTokens"`  // nullable
    RemainingTokens *int64  `json:"remainingTokens"`
    PercentUsed     *int    `json:"percentUsed"`
    Model           string  `json:"model"`
    ContextTokens   int64   `json:"contextTokens"`
}
```

## Smart Diff: Snapshot Comparison

Each poll returns a snapshot. Diff two consecutive snapshots:

| Change Detected | Event Emitted | How |
|----------------|---------------|-----|
| New session key appears | `session.start` | key in current, not in previous |
| Session key disappears | `session.end` | key in previous, not in current |
| `updatedAt` changed | `status.change` → `thinking` | timestamps differ |
| `age` > threshold, no update | `status.change` → `idle` | age > idleThresholdMs (default 60000) |
| `totalTokens` increased | `status.change` (with token delta) | token diff > 0 |
| `abortedLastRun` became true | `error` | was false, now true |

```go
type Differ struct {
    previous map[string]OCSession  // keyed by session Key
    idleThresholdMs int64
}

func (d *Differ) Diff(current []OCSession) []MonitorEvent
```

## Session Key Parsing

OpenClaw session keys encode structure:

```
agent:r2d2:discord:direct:1088...     → agent session
agent:r2d2:cron:4b62933d...           → cron job
agent:r2d2:subagent:uuid              → sub-agent
agent:r2d2:cron:...:run:uuid          → cron run instance
```

Parse key to extract:
- `agentId` → `labels["agent"]`
- Session type → `labels["session_type"]` ("direct" | "group" | "cron" | "subagent" | "cron_run")
- Channel info → `labels["channel"]` ("discord", etc.)

```go
func ParseSessionKey(key string) SessionKeyInfo
```

## Event Mapping

```go
func mapToMonitorEvent(agent string, sess OCSession, eventType EventType) MonitorEvent {
    return MonitorEvent{
        ID:        uuid.New().String(),
        Source:    "openclaw",
        SessionID: sess.SessionID,
        Timestamp: time.Now().UnixMilli(),
        Event:     eventType,
        Status:    inferStatus(sess),
        Detail: &EventDetail{
            Message: fmt.Sprintf("%s [%s]", agent, sess.Key),
            TokenUsage: &TokenUsage{
                Input:     sess.InputTokens,
                Output:    sess.OutputTokens,
                CacheRead: sess.CacheRead,
            },
        },
        Labels: parseSessionKey(sess.Key).ToLabels(),
    }
}

func inferStatus(s OCSession) SessionStatus {
    if s.AbortedLastRun { return StatusError }
    if s.Age < idleThreshold { return StatusThinking }
    return StatusIdle
}
```

## Provider Lifecycle

```
Start(ctx, bus)
  ├── Initial poll → emit session.start for all sessions
  ├── Start polling goroutine (every pollIntervalMs)
  │     ├── Poll status --json
  │     ├── Diff with previous snapshot
  │     ├── Emit events for changes
  │     └── Store current as previous
  └── Return

Stop()
  └── Cancel context, wait for goroutine
```

## Config

```yaml
sources:
  - type: openclaw
    pollIntervalMs: 2000             # default 5000
    idleThresholdMs: 60000           # default: 1 minute
    # Auth: reads from OPENCLAW_GATEWAY_TOKEN env or openclaw config
```

**Auth resolution order:**
1. `token` field in holocron config (supports `${ENV_VAR}`)
2. `OPENCLAW_GATEWAY_TOKEN` env var
3. Read from `~/.openclaw/openclaw.json` gateway.auth.token

## TUI Display

OpenClaw sessions show:
```
  openclaw    r2d2          ● active    0m 30s
              discord:direct
              tokens: 111k / 1M (11%) · opus-4

  openclaw    r2d2          ◌ idle      42m
              cron:4b62933d
              tokens: 10k / 1M (1%) · opus-4

  openclaw    yoda          ● active    0m 02s
              discord:direct
              tokens: 112k / 1M (11%) · opus-4
```

## Verification

```bash
# 1. Configure openclaw source in ~/.holocron/config.yaml
# 2. Run `holo` — OpenClaw agents appear alongside Claude Code sessions
# 3. Send a message to an agent via Discord
#    → Session status changes from idle to active within poll interval
# 4. Agent finishes responding
#    → Status returns to idle after idleThresholdMs
# 5. `holo status --json | jq '.[] | select(.source=="openclaw")'`
#    → Shows OpenClaw sessions with token usage
# 6. Sub-agent spawned → appears as separate session with label session_type=subagent
# 7. Cron runs appear with label session_type=cron_run
```
