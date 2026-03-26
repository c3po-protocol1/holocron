# F4: TUI Session List

> The first thing you see when you run `holo`.

## What This Feature Does

A Bubbletea terminal UI that displays all discovered sessions in a navigable list with live status updates.

## Package

- `internal/tui/app.go` — Main Bubbletea model
- `internal/tui/session_list.go` — Session list component
- `internal/tui/styles.go` — Lipgloss styles
- `internal/tui/keys.go` — Key bindings
- `internal/tui/help.go` — Help overlay

## Input

The TUI receives `<-chan MonitorEvent` — it does not know or care where events come from.

```go
func New(events <-chan MonitorEvent, sessions []SessionState) *App
```

- `events`: live event stream (from EventBus or future WebSocket)
- `sessions`: initial state loaded from Store on startup

## Layout

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
├────────────────────────────────────────────────────┤
│ [q]uit  [?]help                                    │
│ 2 sessions │ 1 active │ SQLite: 48 events          │
└────────────────────────────────────────────────────┘
```

## Session Row

Each session shows:
- **Line 1:** Source | Session ID (truncated) | Status indicator | Elapsed time
- **Line 2:** Workspace path (if available)
- **Line 3:** Current activity (tool + target, if running)

**Status indicators:**
- `●` green — active (thinking, tool_running)
- `◌` dim — idle
- `✕` red — error
- `✓` blue — done

## Key Bindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate sessions |
| `q` or `Ctrl+C` | Quit |
| `?` | Toggle help overlay |
| `r` | Force refresh |

**Not in this feature** (deferred to F7/F8):
- `Enter` (detail view), `l` (labels), `g` (group), `f` (filter), `d` (detail panel)

## State Management (Elm Architecture)

```go
type Model struct {
    sessions   []SessionState   // current state per session
    cursor     int              // selected row
    events     <-chan MonitorEvent
    showHelp   bool
    width      int
    height     int
    eventCount int              // total events received
}

// Messages
type eventMsg MonitorEvent     // from event channel
type tickMsg time.Time         // periodic refresh

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

## Live Updates

- TUI polls the event channel via a Bubbletea `Cmd` that reads from the channel
- On receiving an event, update the matching SessionState (or create new)
- Re-render the view
- Elapsed time updates via periodic tick (every 1s)

## Edge Cases

- **No sessions:** Show "No sessions detected. Check your config: ~/.holocron/config.yaml"
- **Many sessions:** Scrollable list (viewport component from Bubbles)
- **Terminal resize:** Responsive layout, truncate paths if needed
- **Long-running:** Elapsed shows "2h 15m" not "8100s"

## Verification

```bash
# Manual test:
# 1. Run `holo` — see empty state message
# 2. Configure claude-code source, run `holo` — see sessions
# 3. Start a Claude Code session — appears in list
# 4. Navigate with j/k — cursor moves
# 5. Press ? — help overlay appears
# 6. Press q — exits cleanly

# Status indicators update in real-time as sessions change state
```
