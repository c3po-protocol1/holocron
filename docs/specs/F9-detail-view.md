# F9: Session Detail View

> Press Enter to see what a session is actually doing.

## What This Feature Does

When a user selects a session and presses Enter, the TUI switches to a detail view showing a session info panel at the top and a scrollable event log below.

## Layout

```
┌─ Holocron 🔭 ─ Session Detail ───────────────────┐
│                                                    │
│  ┌─ Info ────────────────────────────────────────┐ │
│  │ Source:    claude-code                         │ │
│  │ Session:   a8837a23-595d-49de-83c1-0d95ebab   │ │
│  │ Workspace: ~/Projects/my-app                   │ │
│  │ Status:    ● tool_running (Edit)               │ │
│  │ Duration:  12m 34s                             │ │
│  │ Model:     claude-opus-4 (openclaw only)       │ │
│  │ Tokens:    45.2k in / 12.1k out / 89k cache   │ │
│  │ Context:   112k / 1M (11%)                     │ │
│  │ Events:    247                                 │ │
│  └────────────────────────────────────────────────┘ │
│                                                    │
│  ┌─ Event Log ───────────────────────────────────┐ │
│  │ 22:45:01  ● tool.start   Edit src/index.ts    │ │
│  │ 22:44:58  ○ message      "Updating the..."    │ │
│  │ 22:44:55  ● tool.end     Read package.json    │ │
│  │ 22:44:52  ● tool.start   Read package.json    │ │
│  │ 22:44:50  ○ status       thinking             │ │
│  │ 22:44:48  ○ message      (user) "Fix the..."  │ │
│  │ 22:44:30  ◌ status       idle                 │ │
│  │ 22:43:15  ● tool.end     Bash npm test        │ │
│  │ ...                                            │ │
│  └────────────────────────────────────────────────┘ │
│                                                    │
├────────────────────────────────────────────────────┤
│ [Esc]back  [↑↓]scroll  [G]bottom  [g]top          │
└────────────────────────────────────────────────────┘
```

## Navigation

| Key | Action |
|-----|--------|
| `Enter` (from session list) | Open detail view for selected session |
| `Esc` | Return to session list |
| `↑/↓` or `j/k` | Scroll event log |
| `G` | Jump to bottom (newest events) |
| `g` | Jump to top (oldest loaded events) |
| `f` | Follow mode — auto-scroll to new events (toggle) |

## Info Panel

Shows session metadata. Updates live as new events arrive.

```go
type DetailInfo struct {
    Source      string
    SessionID   string
    Workspace   string
    Status      SessionStatus
    CurrentTool string
    Duration    time.Duration
    Model       string          // openclaw only
    TokenUsage  *TokenUsage
    ContextPct  int             // openclaw only
    EventCount  int
    Labels      map[string]string
}
```

**Source-specific fields:**
- **Claude Code:** workspace, git branch (from labels)
- **OpenClaw:** model, context percentage, agent name, session type

## Event Log

Scrollable list of events for this session, newest at bottom.

**Data source:** `Store.GetEvents(source, sessionID, since, limit)`

**Initial load:** last 200 events from SQLite.
**Live updates:** new events from EventBus appended to bottom.

**Event row format:**
```
HH:MM:SS  [indicator] [event_type]  [summary]
```

**Examples:**
```
22:45:01  ● tool.start    Edit → src/index.ts
22:44:58  ○ message       "Updating the auth handler to..."
22:44:55  ● tool.end      Read → package.json (ok)
22:44:50  ◌ status.change thinking
22:44:48  ▶ session.start
```

**Indicators:**
- `●` — tool activity (start/end)
- `○` — message
- `◌` — status change
- `▶` — session start
- `■` — session end
- `✕` — error

## State Management

```go
type DetailModel struct {
    session    SessionState       // header info
    events     []MonitorEvent     // loaded events
    viewport   viewport.Model     // Bubbles scrollable viewport
    follow     bool               // auto-scroll to bottom
    eventChan  <-chan MonitorEvent // filtered for this session
}
```

**Event filtering:** The detail view subscribes to the same EventBus channel but only renders events matching the current `source + sessionID`.

## Integration with App Model

```go
type Model struct {
    // existing fields...
    view       ViewMode  // "list" | "detail"
    detail     *DetailModel
}

type ViewMode string
const (
    ViewList   ViewMode = "list"
    ViewDetail ViewMode = "detail"
)
```

**In Update():**
- `ViewList` + `Enter` → load events from store, create DetailModel, switch to `ViewDetail`
- `ViewDetail` + `Esc` → set `view = ViewList`, clear detail
- `ViewDetail` + new event → if matches session, append to detail events

## Packages

| File | Action |
|------|--------|
| `internal/tui/detail.go` | **NEW** — DetailModel component |
| `internal/tui/app.go` | **MODIFY** — add ViewMode, wire Enter/Esc |
| `internal/tui/keys.go` | **MODIFY** — add detail key bindings |
| `internal/tui/help.go` | **MODIFY** — context-aware help |

## Edge Cases

- **No events in store:** Show "No events recorded for this session yet."
- **Very long event list:** Viewport handles scrolling. Only load last 200, could add "load more" later.
- **Session ends while viewing:** Keep showing, mark status as done.
- **Terminal resize:** Info panel fixed height, event log fills remaining space.

## Verification

```bash
# 1. Run `holo`, select a session, press Enter → detail view appears
# 2. Info panel shows correct metadata (source, session, workspace, tokens)
# 3. Event log shows chronological events
# 4. Press Esc → returns to session list
# 5. While in detail view, trigger activity in the session
#    → New events appear at bottom of log
# 6. Press 'f' → follow mode, auto-scrolls to new events
# 7. Scroll up → follow mode disables
# 8. Press 'G' → jump to bottom, 'g' → jump to top
# 9. OpenClaw session detail shows model and context percentage
```
