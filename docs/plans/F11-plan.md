# F11 Plan: Rich Event Data

> Enrich the data model and Claude Code parser to capture full conversation content.

## Source Spec

- [F11 Spec: Rich Event Data](../specs/F11-rich-event-data.md)
- [Architecture](../ARCHITECTURE.md)
- [Principles](../PRINCIPLES.md)

## Overview

Currently, Holocron events contain sparse metadata (tool name, target, short message). Claude Code JSONL files contain full conversations — prompts, responses, tool arguments, results — but the parser discards them.

F11 enriches:
1. **Data model** — new fields on `EventDetail` (Content, ToolInput, ToolOutput, Role) + new event types
2. **Claude Code parser** — extract full content from JSONL entries
3. **SQLite storage** — new columns + migration + content trimming for retention
4. **Store interface** — new `TrimOldContent` method

## Architecture Decisions

- New fields go on existing `EventDetail` struct (not a new struct) — keeps the event pipeline unchanged
- Content size limits enforced at parse time (32KB max per field)
- `Message` field stays as short summary (200 chars), `Content` holds full text
- Thinking blocks are explicitly skipped (large, internal reasoning)
- Retention trimming clears content fields but preserves metadata
- OpenClaw provider unaffected — only Claude Code parser changes

## Packages Modified

| File | Change |
|------|--------|
| `internal/collector/types.go` | Add fields to `EventDetail`, new `EventType` constants |
| `internal/providers/claudecode/tailer.go` | Enrich parser, add extraction helpers |
| `internal/store/sqlite/sqlite.go` | Store/retrieve new fields, add `TrimOldContent` |
| `internal/store/sqlite/migrations.go` | Add migration for new columns |
| `internal/store/store.go` | Add `TrimOldContent` to interface |

## Dependencies

- F1 (Core Types + Store) ✅
- F3 (Claude Code Provider) ✅

## Tasks

See [F11-tasks.md](./F11-tasks.md)
