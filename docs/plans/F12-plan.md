# F12 Plan: Verbose Event Log

> Role-based icons in compact mode + verbose toggle for full conversation view.

## Source Spec

- [F12 Spec: Verbose Event Log](../specs/F12-verbose-event-log.md)
- [F11 Spec: Rich Event Data](../specs/F11-rich-event-data.md) (dependency — provides Content/ToolInput/ToolOutput fields)
- [Architecture](../ARCHITECTURE.md)
- [Principles](../PRINCIPLES.md)

## Overview

The detail view event log currently shows sparse one-liners with basic indicators. F12 adds:
1. **Compact mode** — role-based emoji icons (👤🤖🔧✅) instead of generic dots
2. **Verbose mode** — press `v` to expand events into multi-line blocks with full content
3. **Line-based scrolling** in verbose mode (events span multiple lines)
4. **New `format.go`** — shared formatting helpers (icons, labels, word wrap)

## Architecture Decisions

- Compact mode replaces existing `EventIndicator` with emoji icons per spec
- Verbose mode uses `renderedLines` flat list for line-based scrolling
- `format.go` is a new file with pure functions — no state, easy to test
- Auto-fallback to compact if terminal width < 60 columns
- Only visible lines + buffer are rendered (performance)
- Existing `FormatEventRow` refactored to use new `formatEventCompact`

## Packages Modified

| File | Change |
|------|--------|
| `internal/tui/format.go` | **NEW** — eventIcon, eventLabel, formatEventCompact, formatEventVerbose, wordWrap, compactSummary |
| `internal/tui/detail.go` | Verbose toggle, renderedLines, line-based scroll in verbose mode |
| `internal/tui/styles.go` | Role-based colors (cyan for assistant, yellow for tool, etc.) |
| `internal/tui/keys.go` | Add `v` key binding (Verbose) |
| `internal/tui/help.go` | Add verbose toggle to detail help text |
| `internal/tui/app.go` | Route `v` key in detail view |

## Dependencies

- F9 (Detail View) ✅
- F11 (Rich Event Data) ✅

## Tasks

See [F12-tasks.md](./F12-tasks.md)
