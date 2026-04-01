# F11: Rich Event Data

> Capture full conversation content from agent sessions — messages, tool inputs, tool outputs.

## Problem

Holocron events contain only sparse metadata (tool name, target, short message). The underlying data sources (e.g., Claude Code JSONL) contain full conversation content — prompts, responses, tool arguments, results — but the parser discards it all.

This feature enriches the data model and parser to capture that content, making it available for display (F12) and future features (search, export).

## Data Model Changes

### Extended EventDetail

```go
type EventDetail struct {
    // Existing
    Tool       string      `json:"tool,omitempty"`
    Target     string      `json:"target,omitempty"`
    Message    string      `json:"message,omitempty"`
    ElapsedMs  int64       `json:"elapsedMs,omitempty"`
    TokenUsage *TokenUsage `json:"tokenUsage,omitempty"`

    // New
    Content    string `json:"content,omitempty"`    // full message text
    ToolInput  string `json:"toolInput,omitempty"`  // tool call arguments
    ToolOutput string `json:"toolOutput,omitempty"` // tool call result
    Role       string `json:"role,omitempty"`       // user|assistant|system|tool
}
```

### New Event Types

```go
const (
    EventUserMessage      EventType = "user.message"
    EventAssistantMessage EventType = "assistant.message"
    EventToolResult       EventType = "tool.result"
)
```

Existing `EventMessage` stays valid — providers that can't distinguish roles keep using it.

### Content Size Limits

| Field | Max | Truncation |
|-------|-----|------------|
| `Message` | 200 chars | Hard truncate with `...` |
| `Content` | 32 KB | Truncate with `\n[...truncated at 32KB]` |
| `ToolInput` | 32 KB | Same |
| `ToolOutput` | 32 KB | Same |

## Claude Code Parser Changes (`tailer.go`)

JSONL entries already contain everything. Enrich `ParseJSONLLine`:

- **`type: "user"`** → `EventUserMessage`, extract text from `message.content` (string or content-block array), store in `Content`, truncated summary in `Message`
- **`type: "assistant"`** → `EventAssistantMessage`, extract text blocks (skip `type: "thinking"`), extract `usage` into `TokenUsage`
- **`type: "tool_use"`** → `EventToolStart` (existing), add `ToolInput` with formatted arguments, `Target` from input (filename for Read/Edit, command for Bash)
- **`type: "tool_result"`** → `EventToolResult` (new), store output in `ToolOutput`

Helper functions needed:

```go
func extractTextContent(raw json.RawMessage) string      // handles string + content-block formats
func extractAssistantText(raw json.RawMessage) string     // text blocks only, skip thinking
func extractTokenUsage(raw json.RawMessage) *TokenUsage
func extractToolInput(raw json.RawMessage) string         // readable tool args
func extractToolTarget(name string, input json.RawMessage) string
func extractToolResultContent(raw json.RawMessage) string
```

## Storage Changes

### SQLite Migration

```sql
ALTER TABLE events ADD COLUMN content TEXT DEFAULT '';
ALTER TABLE events ADD COLUMN tool_input TEXT DEFAULT '';
ALTER TABLE events ADD COLUMN tool_output TEXT DEFAULT '';
ALTER TABLE events ADD COLUMN role TEXT DEFAULT '';
```

### Content Trimming

Events older than `retention_days` (default: 7) have content fields cleared to save space. Metadata preserved. Runs at startup + hourly.

```go
// New method on Store interface
TrimOldContent(olderThanMs int64) error
```

## Edge Cases

- **Thinking blocks:** Skip — internal reasoning, very large
- **Binary tool results:** Show `[binary content, N bytes]`
- **Empty messages:** Skip events with no text content
- **Duplicate entries:** Claude Code sometimes emits intermediate + final assistant entries. Deduplicate by uuid if present

## Packages Modified

| File | Change |
|------|--------|
| `internal/collector/types.go` | Add fields to `EventDetail`, new `EventType` constants |
| `internal/providers/claudecode/tailer.go` | Enrich parser, add extraction helpers |
| `internal/store/sqlite/sqlite.go` | Store/retrieve new fields, add `TrimOldContent` |
| `internal/store/sqlite/migrations.go` | Add migration for new columns |

## Verification

1. Start Claude Code session, run `holo` → events stored with full content in SQLite
2. `SELECT content, tool_input, tool_output, role FROM events` shows populated fields
3. Content > 32KB is truncated with marker
4. After retention period, old content fields cleared, metadata intact
5. No regression on existing event flow or OpenClaw provider

## Dependencies

- F1 (Core Types + Store) ✅
- F3 (Claude Code Provider) ✅
