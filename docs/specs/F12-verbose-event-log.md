# F12: Verbose Event Log

> See what the agent thinks, says, and does — compact by default, verbose on demand.

## Problem

The detail view event log shows sparse one-liners. With F11's rich event data now available, this feature adds role-based icons in compact mode and a verbose toggle (`v` key) that expands events into full conversation view.

## Compact Mode (default)

One line per event with role icons:

```
22:44:48  👤 user            Fix the auth bug in login.go
22:44:50  🤖 assistant       Looking at the auth handler...
22:44:52  🔧 tool.start      Read → src/login.go
22:44:55  ✅ tool.end        Read → src/login.go (247 lines)
22:44:58  🤖 assistant       Found the issue. The token...
22:45:01  🔧 tool.start      Edit → src/login.go
22:45:03  ✅ tool.end        Edit → src/login.go (ok)
```

### Icon Map

| Event | Icon | Color |
|-------|------|-------|
| `user.message` | 👤 | white/bright |
| `assistant.message` | 🤖 | cyan |
| `tool.start` | 🔧 | yellow |
| `tool.result` / `tool.end` | ✅ | green |
| `message` (generic) | ○ | dim |
| `status.change` | ◌ | dim |
| `session.start` | ▶ | green |
| `session.end` | ■ | dim |
| `error` | ✕ | red |

## Verbose Mode (press `v`)

Multi-line blocks with headers and full content:

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
      token, err := jwt.Parse(tokenStr, ...

22:45:01  🔧 TOOL: Edit ──────────────────────────────
  Target: src/login.go
  Operation: replace lines 45-52
  ┌─ old ──────────────────
  │ type Claims struct {
  │     jwt.StandardClaims
  │ }
  ├─ new ──────────────────
  │ type Claims struct {
  │     jwt.StandardClaims
  │     Custom map[string]interface{} `json:"custom"`
  │ }
  └────────────────────────
```

### Content Selection Logic

```go
func verboseContent(ev MonitorEvent) string {
    d := ev.Detail
    switch ev.Event {
    case EventUserMessage, EventAssistantMessage:
        return coalesce(d.Content, d.Message)
    case EventToolStart:
        return joinNonEmpty("\n", "Target: "+d.Target, d.ToolInput)
    case EventToolResult:
        return coalesce(d.ToolOutput, d.Message)
    default:
        return d.Message
    }
}
```

## Scrolling Changes

Compact mode: scroll by event index (unchanged).

Verbose mode: events span multiple lines. Switch to line-based scrolling:

```go
type DetailModel struct {
    verbose       bool
    renderedLines []renderedLine  // flattened rendered output
}

type renderedLine struct {
    eventIndex int
    text       string
}
```

Re-render into `renderedLines` when verbose toggles or events change. Viewport scrolls this flat list.

**Performance:** Only render visible lines + small buffer. For terminals < 60 columns wide, verbose falls back to compact automatically.

## Formatting Helpers (`format.go` — new file)

```go
func eventIcon(t EventType) string        // event → emoji
func eventLabel(t EventType) string       // event → short label
func compactSummary(ev MonitorEvent) string   // one-liner from Message
func formatEventCompact(ev MonitorEvent) string
func formatEventVerbose(ev MonitorEvent, width int) string
func wordWrap(text string, width int) string
```

## Key Binding

`v` toggles verbose mode. Footer updates:

```
[Esc]back  [↑↓]scroll  [G]bottom  [g]top  [f]ollow: on  [v]erbose: off
```

## Packages Modified

| File | Change |
|------|--------|
| `internal/tui/detail.go` | Verbose toggle, compact/verbose renderers, line-based scroll |
| `internal/tui/format.go` | **NEW** — shared formatting helpers |
| `internal/tui/styles.go` | Role-based coloring, verbose separators |
| `internal/tui/keys.go` | Add `v` key binding |
| `internal/tui/help.go` | Add verbose toggle to help text |

## Verification

1. Open detail view → compact mode shows role icons (👤🤖🔧✅)
2. Press `v` → verbose mode shows full content, word-wrapped
3. Press `v` again → back to compact
4. Scrolling works in both modes (event-based vs line-based)
5. Follow mode works in verbose — new events appear at bottom with full content
6. Terminal < 60 cols → verbose auto-falls back to compact
7. Events without rich content (OpenClaw status changes) still render correctly
8. No regression on existing detail view functionality

## Dependencies

- F9 (Detail View) ✅
- **F11 (Rich Event Data)** — must be complete (provides the content fields)
