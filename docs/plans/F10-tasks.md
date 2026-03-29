# F10: Labels & Grouping — Tasks

**Plan:** [F10-plan.md](./F10-plan.md)
**Spec:** [docs/specs/F10-labels-and-grouping.md](../specs/F10-labels-and-grouping.md)

## Tasks

### T1: Label engine — `internal/labels/labels.go` (NEW)
- [x] Define `GroupMode` type: `GroupNone`, `GroupByAgent`, `GroupByChannel`
- [x] Define `SessionGroup` struct: Label, Sessions, Active count
- [x] `ApplyLabels(s *SessionState, rules []LabelRule)` — glob matching on source/workspace/sessionKey/sessionId
- [x] `GroupSessions(sessions []SessionState, mode GroupMode) []SessionGroup` — group by label key, sort active-first, unlabeled last
- [x] Tests: glob matching, rule ordering (later overrides earlier), provider labels not overwritten by empty rules

### T2: Label engine tests — `internal/labels/labels_test.go` (NEW)
- [x] Test ApplyLabels with single rule match
- [x] Test ApplyLabels with multiple rules (later overrides)
- [x] Test ApplyLabels with glob patterns (wildcards)
- [x] Test ApplyLabels with no matching rules
- [x] Test GroupSessions by agent — sessions grouped, active-first, unlabeled last
- [x] Test GroupSessions by channel — correct grouping
- [x] Test GroupSessions with GroupNone — returns single group with all sessions
- [x] Test empty sessions input
- [x] Test active filter interaction — empty groups not returned

### T3: OpenClaw provider — extract labels
- [x] Parse session key format: `agent:<name>:<channel>:<type>:<id>` → set agent + channel labels
- [x] Set labels on MonitorEvent.Labels and propagated to SessionState
- [x] Tests: key parsing for various formats, graceful handling of unexpected formats

### T4: Claude Code provider — set channel=local
- [x] Set `channel=local` label on all Claude Code sessions
- [x] Tests: label present on emitted events

### T5: TUI — group rendering (`internal/tui/session_list.go` MODIFY)
- [x] `RenderGroupedList()` — render session groups with headers
- [x] Group header style: `─── r2d2 (2 sessions, 1 active) ──────`
- [x] Indented sessions under group headers
- [x] Tests: grouped rendering output, header format

### T6: TUI — app model integration (`internal/tui/app.go` MODIFY)
- [x] Add `groupMode GroupMode` to Model
- [x] Handle `g` key: cycle none → agent → channel → none
- [x] In View: apply labels, then group if mode != none
- [x] Active filter applies BEFORE grouping
- [x] Cursor navigation works across group boundaries
- [x] Footer shows current group mode
- [x] Tests: g key cycling, cursor across groups, footer text

### T7: TUI — key bindings & help
- [x] Add `Group` binding (`g` key) to KeyMap
- [x] Update help overlay with `g` description
- [x] Tests: binding exists, help text includes grouping

### T8: App wiring — pass config labels to TUI
- [x] `cmd/holo/main.go` — pass `cfg.Labels.Rules` to TUI model
- [x] TUI Model stores label rules, applies them in View
- [x] Tests: labels applied from config rules

### T9: Version bump + integration
- [x] Update `var version` to `"0.5.0"` in cmd/holo/main.go
- [x] All tests pass: `go test ./... -v`
- [x] All prior tests still pass
- [x] `go build ./cmd/holo` compiles clean
