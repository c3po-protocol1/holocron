# F1: Core Types + EventBus + SQLite Store — Development Plan

## Spec References

- **F1 Spec:** [../specs/F1-core.md](../specs/F1-core.md)
- **Architecture:** [../ARCHITECTURE.md](../ARCHITECTURE.md)
- **Principles:** [../PRINCIPLES.md](../PRINCIPLES.md)
- **Spec Index:** [../SPEC.md](../SPEC.md)

## Overview

Build the data backbone: event types, in-process pub/sub EventBus, and SQLite persistence.
No UI, no providers — just the foundation.

## Tasks (TDD — write tests first, then implement)

### Task 1: Initialize Go Module & Folder Structure
- `go mod init github.com/c3po-protocol1/holocron`
- Create directory skeleton per ARCHITECTURE.md:
  - `cmd/holo/`
  - `internal/collector/`
  - `internal/store/`
  - `internal/store/sqlite/`
  - `internal/provider/`

### Task 2: Core Types (`internal/collector/types.go`)
- Define MonitorEvent struct with all fields
- Define EventType constants (session.start, session.end, etc.)
- Define SessionStatus constants (idle, thinking, etc.)
- Define EventDetail and TokenUsage structs
- Define SessionState struct

### Task 3: EventBus (`internal/collector/bus.go`)
- Implement EventBus interface: Publish, Subscribe, Unsubscribe
- Channel-based pub/sub, buffer size 256
- Non-blocking publish (drop + log warning if subscriber slow)
- Multiple subscribers supported
- **Tests (bus_test.go):**
  - Publish → subscriber receives event
  - Multiple subscribers each get every event
  - Unsubscribe stops delivery
  - Slow subscriber doesn't block publisher

### Task 4: Store Interface (`internal/store/store.go`)
- Define Store interface: Save, ListSessions, GetSession, GetEvents, Close

### Task 5: SQLite Implementation (`internal/store/sqlite/`)
- `migrations.go` — schema creation (events + sessions tables with indexes)
- `sqlite.go` — Implement Store interface
  - Save: insert event + upsert sessions table
  - ListSessions: query sessions table
  - GetSession: query by source + sessionID
  - GetEvents: filter by source, sessionID, since, limit
  - Close: close DB connection
- **Tests (sqlite_test.go):**
  - Save event → persists to DB
  - Save event → sessions table updated (upsert)
  - ListSessions returns aggregated state
  - GetSession returns correct session
  - GetEvents returns filtered, ordered events
  - GetEvents with since/limit works correctly

### Task 6: Integration Verification
- Combine EventBus + Store: publish event → store saves it
- Verify all unit tests pass: `go test ./internal/collector/... ./internal/store/...`

## Definition of Done

- All unit tests pass
- All types match the F1 spec exactly
- Code follows ARCHITECTURE.md folder structure and dependency rules
- No provider or TUI code exists — only core data layer

## Build Command

```bash
go test ./internal/collector/... ./internal/store/... -v
```
