# F7: Active-Only Quick Toggle

> Press `a` to hide idle sessions. Press again to show all.

## What This Feature Does

A single-key toggle (`a`) that filters the session list to show only active sessions (non-idle). Pressing `a` again shows all sessions including idle ones.

## Behavior

| State | What shows | Status bar hint |
|-------|-----------|----------------|
| **All** (default) | Every session regardless of status | `[a]ctive: off` |
| **Active only** | Sessions with status ≠ `idle` and ≠ `done` | `[a]ctive: on · (3 hidden)` |

**Active statuses shown:** `thinking`, `tool_running`, `waiting`, `error`
**Hidden when active-only:** `idle`, `done`

## Changes

### `internal/tui/app.go` — Model

Add to Model struct:
```go
activeOnly bool  // false by default
```

### `internal/tui/app.go` — Update

Handle `a` key:
```go
case "a":
    m.activeOnly = !m.activeOnly
    // Re-filter the visible session list
```

### `internal/tui/session_list.go` — View

Before rendering, filter sessions:
```go
visible := m.sessions
if m.activeOnly {
    visible = filterActive(m.sessions)
}
```

```go
func filterActive(sessions []SessionState) []SessionState {
    var out []SessionState
    for _, s := range sessions {
        if s.Status != StatusIdle && s.Status != StatusDone {
            out = append(out, s)
        }
    }
    return out
}
```

### `internal/tui/app.go` — Status bar

Show hidden count when filter is active:
```
│ 2 sessions │ 2 active │ [a]ctive: on (4 hidden) │
```

### `internal/tui/keys.go`

Add `a` to key bindings and help overlay.

### `internal/tui/help.go`

Add line: `a  toggle active-only filter`

## Edge Cases

- **All sessions idle + active filter on:** Show "No active sessions. Press 'a' to show all."
- **Cursor position:** If cursor is on a session that becomes hidden, reset cursor to 0.
- **New active session appears:** Automatically visible even with filter on.

## Verification

```bash
# 1. Run `holo` with mix of active and idle sessions
# 2. Press `a` → idle sessions disappear, status bar shows "active: on (N hidden)"
# 3. Press `a` again → all sessions reappear
# 4. With active filter on, all sessions go idle → "No active sessions" message
# 5. Start a new Claude Code session → appears despite filter being on
# 6. Press `?` → help shows `a` key binding
```
