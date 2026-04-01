# F11 Tasks: Rich Event Data

> Source: [F11 Spec](../specs/F11-rich-event-data.md) | [F11 Plan](./F11-plan.md)

## T1: Data Model Changes (`internal/collector/types.go`)

- [x] T1.1: Add new `EventType` constants: `EventUserMessage`, `EventAssistantMessage`, `EventToolResult`
- [x] T1.2: Add new fields to `EventDetail`: `Content`, `ToolInput`, `ToolOutput`, `Role`
- [x] T1.3: Write unit tests verifying new types exist and serialize correctly

## T2: Content Extraction Helpers (`internal/providers/claudecode/tailer.go`)

- [x] T2.1: Implement `extractTextContent(raw json.RawMessage) string` — handles string + content-block array formats
- [x] T2.2: Implement `extractAssistantText(raw json.RawMessage) string` — text blocks only, skip thinking blocks
- [x] T2.3: Implement `extractTokenUsage(raw json.RawMessage) *TokenUsage`
- [x] T2.4: Implement `extractToolInput(raw json.RawMessage) string` — readable tool args
- [x] T2.5: Implement `extractToolTarget(name string, input json.RawMessage) string` — filename for Read/Edit, command for Bash
- [x] T2.6: Implement `extractToolResultContent(raw json.RawMessage) string`
- [x] T2.7: Implement `truncateContent(s string, maxLen int) string` helper with truncation markers
- [x] T2.8: Write unit tests for all extraction helpers (various input formats, edge cases)

## T3: Enrich ParseJSONLLine (`internal/providers/claudecode/tailer.go`)

- [x] T3.1: Update `user` case → `EventUserMessage`, extract Content via `extractTextContent`, set Role="user", truncated summary in Message
- [x] T3.2: Update `assistant` case → `EventAssistantMessage`, extract text via `extractAssistantText`, extract TokenUsage, set Role="assistant"
- [x] T3.3: Update `tool_use` case → add `ToolInput` via `extractToolInput`, improve `Target` via `extractToolTarget`
- [x] T3.4: Update `tool_result` case → `EventToolResult`, store output via `extractToolResultContent`, set Role="tool"
- [x] T3.5: Enforce size limits: Message ≤ 200 chars, Content/ToolInput/ToolOutput ≤ 32KB
- [x] T3.6: Write integration tests with real-ish JSONL samples for each case

## T4: SQLite Migration (`internal/store/sqlite/migrations.go`)

- [x] T4.1: Add migration to add `content`, `tool_input`, `tool_output`, `role` columns to `events` table
- [x] T4.2: Ensure migration is backward-compatible (ALTER TABLE ADD COLUMN with defaults)
- [x] T4.3: Write test that migration runs on fresh DB and on existing DB

## T5: Store Changes (`internal/store/sqlite/sqlite.go` + `store.go`)

- [x] T5.1: Update `Save()` to persist new fields (content, tool_input, tool_output, role)
- [x] T5.2: Update `scanEvent()` to read new fields
- [x] T5.3: Add `TrimOldContent(olderThanMs int64) error` to Store interface
- [x] T5.4: Implement `TrimOldContent` in SQLiteStore — clear content fields for old events, preserve metadata
- [x] T5.5: Write unit tests for Save/Get with new fields populated
- [x] T5.6: Write unit test for `TrimOldContent` — verify content cleared, metadata intact

## T6: Edge Cases & Integration

- [x] T6.1: Verify thinking blocks are skipped in assistant messages
- [x] T6.2: Handle binary tool results → `[binary content, N bytes]`
- [x] T6.3: Skip events with empty text content
- [x] T6.4: Test with Claude Code JSONL that has duplicate entries (deduplicate by uuid if present)
- [x] T6.5: Verify OpenClaw provider is unaffected (no regression)
- [x] T6.6: Run full test suite — all existing tests pass
