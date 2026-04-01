# F11: Rich Event Log

> See what the agent thinks, says, and does — not just timestamps and labels.

## What This Feature Does

Transforms the detail view event log from a sparse activity timeline into a **conversation-grade log** that shows actual message content, tool call details, and results. A verbose toggle (`v` key) switches between compact one-liners and full content view.

## Problem

Current event log shows:
```
22:45:01  ● tool.start      Read → src/main.go
22:44:58  ○ message          Analyzing the...
22:44:55  ● tool.end         Read → src/main.go
```

What users want to see (compact mode):
```
22:44:48  👤 user            Fix the auth bug in login.go
22:44:50  🤖 assistant       Looking at the auth handler...
22:44:52  🔧 tool.start      Read → src/login.go
22:44:55  ✅ tool.end        Read → src/login.go (247 lines)
22:44:58  🤖 assistant       Found the issue. The token...
22:45:01  🔧 tool.start      Edit → src/login.go
22:45:03  ✅ tool.end        Edit → src/login.go (ok)
```

What users want to see (verbose mode, press `v`):
```
22:44:48  👤 USER ─────────────────────────────────────
  Fix the auth bug in login.go. The JWT validation
  is failing for tokens with custom claims.

22:44:50  🤖 ASSISTANT ────────────────────────────────
  Looking at the auth handler to understand the JWT
  validation flow...

22:44:52  🔧 TOOL: Read ──────────────────────────────
  Target: src/login.go

22:44:55  ✅ RESULT: Read ─────────────────────────────
  [247 lines returned]
  func validateToken(tokenStr string) (*Claims, error) {
      token, err := jwt.Parse(tokenStr, func(t *jwt.Token) ...
      ...
  }

22:44:58  🤖 ASSISTANT ────────────────────────────────
  Found the issue. The token parser doesn't handle
  custom claims because the Claims struct only has
  the standard fields. I'll add a CustomClaims map.

22:45:01  🔧 TOOL: Edit ──────────────────────────────
  Target: src/login.go
  Operation: replace lines 45-52
  ┌─ old ──────────────────────────
  │ type Claims struct {
  │     jwt.StandardClaims
  │ }
  ├─ new ──────────────────────────
  │ type Claims struct {
  │     jwt.StandardClaims
  │     Custom map[string]interface{} `json:"custom"`
  │ }
  └────────────────────────────────

22:45:03  ✅ RESULT: Edit ─────────────────────────────
  ok
```

## Data Model Changes

### Extended EventDetail

```go
// EventDetail contains optional details about an event.
type EventDetail struct {
    // Existing fields
    Tool       string      `json:"tool,omitempty"`
    Target     string      `json:"target,omitempty"`
    Message    string      `json:"message,omitempty"`       // short summary (compact mode)
    ElapsedMs  int64       `json:"elapsedMs,omitempty"`
    TokenUsage *TokenUsage `json:"tokenUsage,omitempty"`

    // New fields for rich content
    Content    string      `json:"content,omitempty"`       // full message text (verbose mode)
    ToolInput  string      `json:"toolInput,omitempty"`     // tool call arguments/input
    ToolOutput string      `json:"toolOutput,omitempty"`    // tool call result/output
    Role       string      `json:"role,omitempty"`          // "user" | "assistant" | "system" | "tool"
}
```

### New Event Types

```go
const (
    // Existing
    EventSessionStart EventType = "session.start"
    EventSessionEnd   EventType = "session.end"
    EventStatusChange EventType = "status.change"
    EventToolStart    EventType = "tool.start"
    EventToolEnd      EventType = "tool.end"
    EventMessage      EventType = "message"
    EventError        EventType = "error"

    // New: distinguish message roles
    EventUserMessage      EventType = "user.message"
    EventAssistantMessage EventType = "assistant.message"
    EventToolResult       EventType = "tool.result"       // distinct from tool.end
)
```

> **Backward compat:** `EventMessage` remains valid. Providers that can't distinguish roles keep using it. The TUI handles both.

## Provider Changes

### Claude Code Provider — `tailer.go`

The JSONL files already contain everything. Current parser discards it. Changes:

```go
func ParseJSONLLine(line []byte, sessionID, workspace string) *collector.MonitorEvent {
    var entry RawJSONLEntry
    if err := json.Unmarshal(line, &entry); err != nil {
        return nil
    }

    switch entry.Type {
    case "user":
        msg := extractTextContent(entry.Message)
        return &collector.MonitorEvent{
            Event:  collector.EventUserMessage,
            Status: collector.StatusThinking,
            Detail: &collector.EventDetail{
                Role:    "user",
                Message: truncate(msg, 80),    // compact summary
                Content: msg,                   // full content
            },
        }

    case "assistant":
        msg := extractAssistantText(entry.Message)
        return &collector.MonitorEvent{
            Event:  collector.EventAssistantMessage,
            Status: collector.StatusIdle,
            Detail: &collector.EventDetail{
                Role:    "assistant",
                Message: truncate(msg, 80),
                Content: msg,
                TokenUsage: extractTokenUsage(entry.Message),
            },
        }

    case "tool_use":
        input := extractToolInput(entry.Input)
        target := extractToolTarget(entry.Name, entry.Input)
        return &collector.MonitorEvent{
            Event:  collector.EventToolStart,
            Status: collector.StatusToolRunning,
            Detail: &collector.EventDetail{
                Tool:      entry.Name,
                Target:    target,
                ToolInput: input,    // full tool arguments
                Role:      "tool",
            },
        }

    case "tool_result":
        content := extractToolResultContent(entry.Content)
        return &collector.MonitorEvent{
            Event:  collector.EventToolResult,
            Status: collector.StatusThinking,
            Detail: &collector.EventDetail{
                Message:    truncate(content, 80),
                ToolOutput: content,
                Role:       "tool",
            },
        }
    }
    return nil
}
```

**Helper functions needed:**

```go
// extractTextContent pulls text from a message JSON (handles both string
// and content-block array formats).
func extractTextContent(raw json.RawMessage) string

// extractAssistantText extracts text blocks from assistant message,
// skipping thinking blocks.
func extractAssistantText(raw json.RawMessage) string

// extractTokenUsage pulls usage from assistant message metadata.
func extractTokenUsage(raw json.RawMessage) *collector.TokenUsage

// extractToolInput formats tool input as a readable string.
// For Read/Edit: show filename. For Bash: show command. For others: JSON.
func extractToolInput(raw json.RawMessage) string

// extractToolTarget returns a short target identifier
// (filename for Read/Edit, command preview for Bash).
func extractToolTarget(toolName string, input json.RawMessage) string

// extractToolResultContent pulls text from tool_result content blocks.
func extractToolResultContent(raw json.RawMessage) string
```

**Content format of Claude Code JSONL entries (observed):**

```jsonc
// user message
{
  "type": "user",
  "message": {
    "role": "user",
    "content": "Fix the auth bug..."   // can be string or content blocks
  }
}

// assistant message (final, has usage)
{
  "type": "assistant",
  "message": {
    "role": "assistant",
    "content": [
      {"type": "thinking", "thinking": "..."},  // skip this
      {"type": "text", "text": "Here's what I found..."}
    ],
    "usage": {
      "input_tokens": 1234,
      "output_tokens": 567,
      "cache_read_input_tokens": 8900
    }
  }
}

// tool_use
{
  "type": "tool_use",
  "name": "Read",
  "input": {"file_path": "src/login.go"}
}

// tool_result
{
  "type": "tool_result",
  "content": "func validateToken(tokenStr string)..."  // or content blocks
}
```

### OpenClaw Provider — Future Enhancement

The OpenClaw provider currently polls `openclaw gateway call status --json` which returns only session-level metadata (tokens, model, status) — **not** conversation content.

**Two paths to add rich data:**

1. **Read session logs directly** — If OpenClaw stores conversation history in a file/DB accessible to Holocron, tail it like Claude Code
2. **New API endpoint** — `openclaw gateway call session-events --session-id <id> --since <ts>` returning recent events with content

> **For this spec (F11), focus on Claude Code first.** OpenClaw rich events are deferred to F11b or a future spec once the API is available.
>
> OpenClaw sessions will continue showing status-change events in the detail view. Compact and verbose mode still apply — verbose just shows more token/label detail.

## Storage Changes

### SQLite Schema

The `events` table needs a larger content column:

```sql
-- New columns (added via migration)
ALTER TABLE events ADD COLUMN content TEXT DEFAULT '';
ALTER TABLE events ADD COLUMN tool_input TEXT DEFAULT '';
ALTER TABLE events ADD COLUMN tool_output TEXT DEFAULT '';
ALTER TABLE events ADD COLUMN role TEXT DEFAULT '';
```

**Content trimming policy:**
- Content fields stored up to **32 KB** each (larger values truncated with `[...truncated]` marker)
- Events older than `retention_days` (default: 7) have content fields cleared to empty (metadata kept)
- Trimming runs at startup and once per hour

```go
// store/sqlite
func (s *SQLiteStore) TrimOldContent(olderThanMs int64) error
```

### Store Interface

```go
type Store interface {
    // Existing
    Save(event MonitorEvent) error
    GetEvents(source, sessionID string, since int64, limit int) ([]MonitorEvent, error)

    // New
    TrimOldContent(olderThanMs int64) error
}
```

## TUI Changes

### Verbose Toggle

```go
type DetailModel struct {
    // Existing fields...
    verbose bool   // NEW: toggle between compact and verbose rendering
}
```

**Key binding:** `v` toggles verbose mode.

### Compact Mode Rendering (default)

One line per event. Same height as current, but richer content:

```go
func FormatEventRowCompact(ev collector.MonitorEvent) string {
    ts := formatTime(ev.Timestamp)
    icon := eventIcon(ev.Event)        // 👤🤖🔧✅◌▶■✕
    label := eventLabel(ev.Event)      // "user", "assistant", "Read", etc.
    summary := compactSummary(ev)      // one-liner from Message field

    return fmt.Sprintf("  %s  %s %-16s %s", ts, icon, label, summary)
}
```

**Icons by event type:**

| Event | Icon | Color |
|-------|------|-------|
| `user.message` | `👤` | white/bright |
| `assistant.message` | `🤖` | cyan |
| `tool.start` | `🔧` | yellow |
| `tool.result` / `tool.end` | `✅` | green |
| `message` (generic) | `○` | dim |
| `status.change` | `◌` | dim |
| `session.start` | `▶` | green |
| `session.end` | `■` | dim |
| `error` | `✕` | red |

### Verbose Mode Rendering

Multi-line blocks with separators. Each event gets a header line + indented content:

```go
func FormatEventRowVerbose(ev collector.MonitorEvent, width int) string {
    ts := formatTime(ev.Timestamp)
    icon := eventIcon(ev.Event)
    label := strings.ToUpper(eventLabel(ev.Event))
    separator := strings.Repeat("─", max(0, width-30))

    var b strings.Builder

    // Header line
    b.WriteString(fmt.Sprintf("  %s  %s %s %s\n", ts, icon, label, separator))

    // Content (word-wrapped to width - 4 for indent)
    content := verboseContent(ev)
    if content != "" {
        wrapped := wordWrap(content, width-4)
        for _, line := range strings.Split(wrapped, "\n") {
            b.WriteString("  " + line + "\n")
        }
    }

    return b.String()
}

func verboseContent(ev collector.MonitorEvent) string {
    if ev.Detail == nil {
        return ""
    }
    d := ev.Detail

    switch ev.Event {
    case collector.EventUserMessage, collector.EventAssistantMessage:
        if d.Content != "" {
            return d.Content
        }
        return d.Message

    case collector.EventToolStart:
        var parts []string
        if d.Target != "" {
            parts = append(parts, "Target: "+d.Target)
        }
        if d.ToolInput != "" {
            parts = append(parts, d.ToolInput)
        }
        return strings.Join(parts, "\n")

    case collector.EventToolResult:
        if d.ToolOutput != "" {
            return d.ToolOutput
        }
        return d.Message

    default:
        return d.Message
    }
}
```

### Scrolling Adjustment

In verbose mode, events take multiple lines. The scroll model must change from "scroll by event index" to "scroll by rendered line":

```go
type DetailModel struct {
    // ...
    verbose       bool
    renderedLines []renderedLine   // cached rendered output
}

type renderedLine struct {
    eventIndex int
    text       string
}
```

When `verbose` toggles or events change, re-render all visible events into `renderedLines`. Scroll operates on this flat line list.

### Footer Update

```
[Esc]back  [↑↓]scroll  [G]bottom  [g]top  [f]ollow: on  [v]erbose: off
```

## Packages Modified

| File | Action | What |
|------|--------|------|
| `internal/collector/types.go` | **MODIFY** | Add `Content`, `ToolInput`, `ToolOutput`, `Role` to `EventDetail`. Add new `EventType` constants. |
| `internal/providers/claudecode/tailer.go` | **MODIFY** | Parse full message content, tool inputs/outputs from JSONL. Add helper extraction functions. |
| `internal/store/sqlite/sqlite.go` | **MODIFY** | Store/retrieve new content fields. Add `TrimOldContent()`. |
| `internal/store/sqlite/migrations.go` | **MODIFY** | Add migration for new columns. |
| `internal/tui/detail.go` | **MODIFY** | Add verbose toggle, compact/verbose renderers, line-based scrolling. |
| `internal/tui/styles.go` | **MODIFY** | Add styles for role-based coloring, verbose mode separators. |
| `internal/tui/keys.go` | **MODIFY** | Add `v` key binding. |
| `internal/tui/help.go` | **MODIFY** | Add verbose toggle to help text. |
| `internal/tui/format.go` | **NEW** | Shared formatting helpers: `wordWrap`, `eventIcon`, `eventLabel`, `compactSummary`, `verboseContent`. |

## Content Size Limits

| Field | Max Size | Truncation |
|-------|----------|------------|
| `Message` (summary) | 200 chars | Hard truncate with `...` |
| `Content` (full text) | 32 KB | Truncate with `\n[...truncated at 32KB]` |
| `ToolInput` | 32 KB | Same |
| `ToolOutput` | 32 KB | Same |

**Why 32 KB?** Tool outputs (especially `Read` for large files) can be huge. 32 KB is enough to see meaningful content while keeping SQLite performant. The original JSONL files always contain the full data if needed.

## Edge Cases

- **Thinking blocks:** Skip `type: "thinking"` content in assistant messages. These are internal reasoning and very large — they'd overwhelm the log.
- **Binary/non-text tool results:** If tool output contains non-printable chars, show `[binary content, N bytes]`.
- **Empty messages:** Some assistant events are intermediate streaming chunks with no text yet. Show as status indicator only, or skip if content is empty.
- **Duplicate events:** Claude Code JSONL sometimes emits two assistant entries (intermediate + final with `stop_reason`). Deduplicate by `uuid` field if present.
- **Very long conversations:** Line-based scrolling with 32KB content blocks. Lazy render only visible lines to keep TUI responsive.
- **Terminal width < 60:** Verbose mode falls back to compact automatically.

## Verification

```bash
# 1. Start a Claude Code session, run `holo`, open its detail view
#    → Events show with role icons (👤🤖🔧✅)

# 2. In compact mode (default):
#    → User messages show first ~80 chars of prompt
#    → Assistant messages show first ~80 chars of response
#    → Tool events show tool name + target

# 3. Press 'v' to toggle verbose mode:
#    → User messages show full prompt text (word-wrapped)
#    → Assistant messages show full response text
#    → Tool starts show input/arguments
#    → Tool results show output content

# 4. Press 'v' again → back to compact

# 5. Scrolling works correctly in both modes
#    → Compact: one line per event
#    → Verbose: multi-line blocks, scroll by line

# 6. Follow mode works in verbose:
#    → New events appear at bottom with full content

# 7. Content trimming:
#    → After retention_days, old events lose content but keep metadata
#    → Events still appear in compact mode (summary preserved)

# 8. OpenClaw sessions:
#    → Still show status-change events (no regression)
#    → Verbose mode shows extra label/token detail
```

## Build Order

This feature can be built incrementally:

1. **T1: Data model** — Add fields to `EventDetail`, new `EventType` constants, SQLite migration
2. **T2: Claude Code parser** — Enrich `ParseJSONLLine` to extract full content
3. **T3: Storage** — Store/retrieve content fields, implement `TrimOldContent`
4. **T4: TUI compact mode** — Role icons, richer one-liners
5. **T5: TUI verbose mode** — Multi-line renderer, line-based scrolling, `v` toggle
6. **T6: Polish** — Word wrap, edge cases, deduplication, binary detection

## Dependencies

- **F9 (Detail View):** Must be complete ✅
- **F1 (Core Types + Store):** Must be complete ✅
- **F3 (Claude Code Provider):** Must be complete ✅

## Non-Goals (for this spec)

- OpenClaw conversation content (needs API support — deferred to F11b)
- Codex/other provider rich events
- Search within event log (future feature)
- Export conversation log to file
- Streaming/partial message rendering (events are complete by the time we see them in JSONL)
