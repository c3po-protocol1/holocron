# F9: Session Detail View — Development Plan

**Spec:** [docs/specs/F9-session-detail.md](../specs/F9-session-detail.md)
**Branch:** feature/f9-session-detail
**Status:** In Progress

## Overview

Add a detail view to the TUI that shows full session info + scrollable event log when user presses Enter on a session.

## Architecture Decisions

- Use Bubbletea's `viewport.Model` (from Bubbles) for scrollable event log
- ViewMode enum on app Model to switch between list and detail
- Detail view subscribes to same EventBus but filters for selected session
- Events loaded from SQLite store (last 200), live updates appended

## Dependencies

- F1: Store (GetEvents query)
- F4: TUI Session List (base app model)
- F6: Collector (EventBus subscription)
- Bubbles viewport package

## Task List

See [F9-tasks.md](./F9-tasks.md)
