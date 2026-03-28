# F7: Active-Only Quick Toggle — Development Plan

## Spec References

- **F7 Spec:** [../specs/F7-active-toggle.md](../specs/F7-active-toggle.md)
- **F4 Spec (TUI base):** [../specs/F4-tui-session-list.md](../specs/F4-tui-session-list.md)
- **Architecture:** [../ARCHITECTURE.md](../ARCHITECTURE.md)
- **Principles:** [../PRINCIPLES.md](../PRINCIPLES.md)

## Overview

Add `a` key toggle to filter TUI session list to active-only sessions.
Press `a` to hide idle/done sessions, press again to show all.

## Tasks (TDD)

### Task 1: Add `a` key binding (`internal/tui/keys.go`)
- Add `Active` binding to KeyMap struct
- Key: `a`, Help: "toggle active filter"

### Task 2: Update help overlay (`internal/tui/help.go`)
- Add `a  toggle active-only filter` line

### Task 3: Add filterActive function (`internal/tui/session_list.go`)
- `filterActive(sessions []SessionState) []SessionState`
- Keeps: thinking, tool_running, waiting, error
- Hides: idle, done

### Task 4: Update Model (`internal/tui/app.go`)
- Add `activeOnly bool` to Model struct
- Handle `a` key in Update: toggle activeOnly
- In View: filter sessions before rendering when activeOnly=true
- Update footer: show `[a]ctive: on (N hidden)` or `[a]ctive: off`
- Empty active state: "No active sessions. Press 'a' to show all."
- Cursor reset: if cursor > len(visible)-1 after filter, reset to 0

### Task 5: Tests (`internal/tui/tui_test.go`)
- Test filterActive function
- Test `a` key toggles activeOnly
- Test cursor reset when filtered list is shorter
- Test empty active state message
- Test footer shows hidden count
- Test status bar reflects filter state

### Task 6: Integration
- All tests pass: `go test ./... -v`
- All prior tests still pass

## Definition of Done

- `a` toggles active-only filter
- Footer shows filter state and hidden count
- Empty active state shows helpful message
- Cursor resets when out of bounds
- Help overlay shows `a` binding
- All tests pass
