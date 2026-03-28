# F9: Session Detail View — Tasks

**Plan:** [F9-plan.md](./F9-plan.md)
**Spec:** [docs/specs/F9-session-detail.md](../specs/F9-session-detail.md)

## Tasks

### T1: Store — GetEvents query
- [ ] Add `GetEvents(source, sessionID, limit) ([]MonitorEvent, error)` to store interface
- [ ] Implement in SQLite store
- [ ] Tests: query returns events filtered by source+sessionID, respects limit, ordered by timestamp

### T2: Detail model — internal/tui/detail.go (NEW)
- [ ] Create `DetailModel` struct with session info, events, viewport, follow mode
- [ ] `Init()` — load events from store, set up viewport
- [ ] `Update()` — handle key events (scroll, follow, Esc)
- [ ] `View()` — render info panel + event log
- [ ] Tests: render info panel, render events, scroll behavior, follow mode

### T3: Event formatting
- [ ] Format event rows: `HH:MM:SS [indicator] [event_type] [summary]`
- [ ] Indicators: ● tool, ○ message, ◌ status, ▶ start, ■ end, ✕ error
- [ ] Source-specific fields (OpenClaw: model, context %)
- [ ] Tests: formatting for each event type

### T4: App model integration — internal/tui/app.go (MODIFY)
- [ ] Add `ViewMode` enum (list/detail) to Model
- [ ] Enter key in list view → create DetailModel, switch to detail
- [ ] Esc key in detail view → switch back to list
- [ ] Route new events to detail model if active and matching session
- [ ] Tests: view switching, event routing

### T5: Key bindings — internal/tui/keys.go (MODIFY)
- [ ] Add detail-specific bindings: Esc, j/k, G, g, f
- [ ] Context-aware help (different bindings shown per view)
- [ ] Tests: correct bindings per view mode

### T6: Version bump
- [ ] Update `var version` to `"0.4.0"` in cmd/holo/main.go

### T7: Edge cases
- [ ] No events → "No events recorded for this session yet."
- [ ] Session ends while viewing → keep showing, mark status done
- [ ] Terminal resize → info panel fixed height, event log fills remaining
- [ ] Tests for each edge case
