# F10: Labels & Grouping — Development Plan

**Spec:** [docs/specs/F10-labels-and-grouping.md](../specs/F10-labels-and-grouping.md)
**Branch:** feature/f10-labels-grouping
**Status:** In Progress

## Overview

Auto-label sessions (from providers + config rules), then group them in the TUI
with visual headers. `g` key cycles group mode: none → agent → channel → none.

## Architecture Decisions

- New package `internal/labels/` — pure logic, no TUI or provider imports
- Labels stored in `SessionState.Labels` map (already exists in types.go)
- Provider labels applied first (openclaw parses session key, claude-code sets channel=local)
- Config label rules applied second (glob matching on source/workspace/sessionKey/sessionId)
- Grouping is a view-layer concern — `labels.GroupSessions()` returns `[]SessionGroup`
- Active filter applies BEFORE grouping; empty groups hidden

## Dependencies

- F1: SessionState.Labels (already in types.go)
- F4/F7: TUI session list + active filter
- F8: OpenClaw provider (needs label extraction from session key)
- Config: LabelsConfig + LabelRule (already in config.go)

## Task List

See [F10-tasks.md](./F10-tasks.md)
