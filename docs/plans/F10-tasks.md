# F10: Labels & Grouping — Tasks

**Plan:** [F10-plan.md](./F10-plan.md)
**Spec:** [docs/specs/F10-labels-and-grouping.md](../specs/F10-labels-and-grouping.md)

## Tasks

### T1: Label engine — `internal/labels/labels.go` (NEW)
- [ ] Define `GroupMode` type: `GroupNone`, `GroupByAgent`, `GroupByChannel`
- [ ] Define `SessionGroup` struct: Label, Sessions, Active count
- [ ] `ApplyLabels(s *SessionState, rules []LabelRule)` — glob matching on source/workspace/sessionKey/sessionId
- [ ] `GroupSessions(sessions []SessionState, mode GroupMode) []SessionGroup` — group by label key, sort active-first, unlabeled last
- [ ] Tests: glob matching, rule ordering (later overrides earlier), provider labels not overwritten by empty rules

### T2: Label engine tests — `internal/labels/labels_test.go` (NEW)
- [ ] Test ApplyLabels with single rule match
- [ ] Test ApplyLabels with multiple rules (later overrides)
- [ ] Test ApplyLabels with glob patterns (wildcards)
- [ ] Test ApplyLabels with no matching rules
- [ ] Test GroupSessions by agent — sessions grouped, active-first, unlabeled last
- [ ] Test GroupSessions by channel — correct grouping
- [ ] Test GroupSessions with GroupNone — returns single group with all sessions
- [ ] Test empty sessions input
- [ ] Test active filter interaction — empty groups not returned

### T3: OpenClaw provider — extract labels
- [ ] Parse session key format: `agent:<name>:<channel>:<type>:<id>` → set agent + channel labels
- [ ] Set labels on MonitorEvent.Labels and propagated to SessionState
- [ ] Tests: key parsing for various formats, graceful handling of unexpected formats

### T4: Claude Code provider — set channel=local
- [ ] Set `channel=local` label on all Claude Code sessions
- [ ] Tests: label present on emitted events

### T5: TUI — group rendering (`internal/tui/session_list.go` MODIFY)
- [ ] `RenderGroupedList()` — render session groups with headers
- [ ] Group header style: `─── r2d2 (2 sessions, 1 active) ──────`
- [ ] Indented sessions under group headers
- [ ] Tests: grouped rendering output, header format

### T6: TUI — app model integration (`internal/tui/app.go` MODIFY)
- [ ] Add `groupMode GroupMode` to Model
- [ ] Handle `g` key: cycle none → agent → channel → none
- [ ] In View: apply labels, then group if mode != none
- [ ] Active filter applies BEFORE grouping
- [ ] Cursor navigation works across group boundaries
- [ ] Footer shows current group mode
- [ ] Tests: g key cycling, cursor across groups, footer text

### T7: TUI — key bindings & help
- [ ] Add `Group` binding (`g` key) to KeyMap
- [ ] Update help overlay with `g` description
- [ ] Tests: binding exists, help text includes grouping

### T8: App wiring — pass config labels to TUI
- [ ] `cmd/holo/main.go` — pass `cfg.Labels.Rules` to TUI model
- [ ] TUI Model stores label rules, applies them in View
- [ ] Tests: labels applied from config rules

### T9: Version bump + integration
- [ ] Update `var version` to `"0.5.0"` in cmd/holo/main.go
- [ ] All tests pass: `go test ./... -v`
- [ ] All prior tests still pass
- [ ] `go build ./cmd/holo` compiles clean
