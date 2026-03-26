# F4: TUI Session List — Development Plan

## Spec References

- **F4 Spec:** [../specs/F4-tui-session-list.md](../specs/F4-tui-session-list.md)
- **F1 Spec (types):** [../specs/F1-core.md](../specs/F1-core.md)
- **Architecture:** [../ARCHITECTURE.md](../ARCHITECTURE.md)
- **Principles:** [../PRINCIPLES.md](../PRINCIPLES.md)
- **Spec Index:** [../SPEC.md](../SPEC.md)

## Overview

Bubbletea TUI that displays all discovered sessions in a navigable list with
live status updates. Receives `<-chan MonitorEvent` — source-independent.

## Tasks (TDD where applicable)

### Task 1: Styles (`internal/tui/styles.go`)
- Define Lipgloss styles for header, session rows, status indicators, footer
- Status indicators: ● green (active), ◌ dim (idle), ✕ red (error), ✓ blue (done)

### Task 2: Key Bindings (`internal/tui/keys.go`)
- Define key bindings: ↑/↓ or j/k (navigate), q/Ctrl+C (quit), ? (help), r (refresh)
- Use Bubbles key binding system

### Task 3: Help Overlay (`internal/tui/help.go`)
- Toggle help overlay showing all key bindings
- Clean overlay that doesn't destroy the session list underneath

### Task 4: Session List Component (`internal/tui/session_list.go`)
- Render session rows: source, session ID (truncated), status indicator, elapsed
- Second line: workspace path
- Third line: current tool + target (if active)
- Scrollable when many sessions
- Handle empty state: "No sessions detected" message
- Elapsed formatting: "2m 13s", "2h 15m" (human-friendly)

### Task 5: Main App Model (`internal/tui/app.go`)
- Elm architecture: Model, Update, View
- New(events <-chan MonitorEvent, sessions []SessionState)
- eventMsg from channel → update matching SessionState or create new
- tickMsg every 1s → update elapsed times
- Terminal resize handling
- Cursor navigation (up/down/j/k)

### Task 6: Tests (`internal/tui/*_test.go`)
- Test session state update from events
- Test elapsed time formatting
- Test empty state rendering
- Test cursor bounds
- Test status indicator mapping

### Task 7: Integration
- All tests pass: `go test ./internal/tui/... -v`
- All prior tests still pass: `go test ./...`

## Definition of Done

- TUI renders session list from events channel
- Key bindings work (navigate, quit, help, refresh)
- Status indicators match spec
- Elapsed time human-readable
- Empty state handled
- All tests pass
- Dependency rule: TUI only imports collector/types
