# F12 Tasks: Verbose Event Log

> Source: [F12 Spec](../specs/F12-verbose-event-log.md) | [F12 Plan](./F12-plan.md)

## T1: Formatting Helpers (`internal/tui/format.go` — NEW)

- [x] T1.1: Create `format.go` with `eventIcon(t EventType) string` — returns emoji per spec icon map
- [x] T1.2: Implement `eventLabel(t EventType) string` — short label for verbose headers
- [x] T1.3: Implement `compactSummary(ev MonitorEvent) string` — one-liner from Message/Tool/Target
- [x] T1.4: Implement `formatEventCompact(ev MonitorEvent) string` — timestamp + icon + type + summary
- [x] T1.5: Implement `formatEventVerbose(ev MonitorEvent, width int) string` — multi-line block with header, content, word-wrap
- [x] T1.6: Implement `wordWrap(text string, width int) string` — wraps text at word boundaries
- [x] T1.7: Implement `verboseContent(ev MonitorEvent) string` — content selection logic per spec
- [x] T1.8: Write unit tests for all formatting helpers

## T2: Styles Updates (`internal/tui/styles.go`)

- [x] T2.1: Add role-based color styles: cyan for assistant, yellow for tool, green for tool result, white for user
- [x] T2.2: Add verbose separator style (dim horizontal rule with label)
- [x] T2.3: Write tests verifying style objects are non-nil

## T3: Key Binding (`internal/tui/keys.go` + `help.go`)

- [x] T3.1: Add `Verbose` key binding (`v` key) to KeyMap struct
- [x] T3.2: Update `DefaultKeyMap()` to include Verbose binding
- [x] T3.3: Update `detailBindings()` in help.go to include verbose toggle
- [x] T3.4: Update footer in detail view to show `[v]erbose: off/on`

## T4: Detail Model — Verbose Toggle (`internal/tui/detail.go`)

- [x] T4.1: Add `verbose bool` field to DetailModel
- [x] T4.2: Add `renderedLines []renderedLine` struct and field for line-based scrolling
- [x] T4.3: Implement `ToggleVerbose()` method — toggle flag, re-render lines, adjust scroll
- [x] T4.4: Implement `rebuildRenderedLines()` — flatten events into renderedLines based on mode
- [x] T4.5: Auto-fallback: if width < 60, force compact even if verbose is on
- [x] T4.6: Write unit tests for ToggleVerbose and rebuildRenderedLines

## T5: Detail Model — Rendering Updates (`internal/tui/detail.go`)

- [x] T5.1: Refactor `View()` to use `formatEventCompact` in compact mode (replace existing FormatEventRow)
- [x] T5.2: In verbose mode, render from `renderedLines` with line-based scroll
- [x] T5.3: Update `maxScroll()` — event count for compact, line count for verbose
- [x] T5.4: Update `ScrollUp/Down/ToTop/ToBottom` to work with line-based scroll in verbose
- [x] T5.5: Update `AppendEvent` to rebuild renderedLines when verbose is active
- [x] T5.6: Write unit tests for View output in both modes

## T6: App Integration (`internal/tui/app.go`)

- [x] T6.1: Add `v` key routing in `updateDetail()` to call `detail.ToggleVerbose()`
- [x] T6.2: Verify `WindowSizeMsg` triggers renderedLines rebuild if verbose
- [x] T6.3: Write integration test: key press toggles verbose mode

## T7: Edge Cases & Full Test Suite

- [x] T7.1: Events without rich content (OpenClaw status changes) render correctly in both modes
- [x] T7.2: Follow mode works in verbose — new events appear at bottom with full content
- [x] T7.3: Empty event log renders correctly in verbose mode
- [x] T7.4: Very long content word-wraps properly at terminal width
- [x] T7.5: Run full test suite — all existing tests pass, no regressions
