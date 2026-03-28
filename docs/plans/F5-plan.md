# F5: CLI Status Command — Development Plan

## Spec References

- **F5 Spec:** [../specs/F5-cli-status.md](../specs/F5-cli-status.md)
- **F4 Spec (TUI):** [../specs/F4-tui-session-list.md](../specs/F4-tui-session-list.md)
- **Architecture:** [../ARCHITECTURE.md](../ARCHITECTURE.md)
- **Principles:** [../PRINCIPLES.md](../PRINCIPLES.md)
- **Spec Index:** [../SPEC.md](../SPEC.md)

## Overview

Create the Cobra CLI entry point (`cmd/holo/main.go`) with two modes:
- `holo` (default) → launch TUI (F4)
- `holo status` → one-shot session summary and exit
- `holo version` → print version

This is the first feature that creates `cmd/` and wires everything together.

## Tasks (TDD where applicable)

### Task 1: CLI Entry Point (`cmd/holo/main.go`)
- Create Cobra root command (`holo`)
- Default action: load config → open store → load sessions → launch TUI (F4)
- Add `version` subcommand (hardcoded version string for now)
- Wire: config → store → collector types → TUI

### Task 2: Status Formatter (`internal/cli/status.go`)
- Human-readable formatter: prints session list to stdout
- Format: `● source  sessionID  status  elapsed  workspace`
- Second line: `tool → target` (if active)
- Empty state: "No sessions found."
- Reuse `tui.FormatElapsed` and `tui.StatusIndicator` logic or extract shared utils

### Task 3: JSON Formatter (`internal/cli/status_json.go`)
- JSON output mode for `--json` flag
- Output: JSON array of session objects
- Fields: source, sessionId, status, workspace, elapsedMs, currentTool, currentTarget

### Task 4: Status Command (`cmd/holo/status.go` or in main.go)
- Cobra subcommand `status`
- Flags: `--json`, `--source <type>`, `--active`
- Behavior: load config → open store (read-only) → load sessions → filter → format → print → exit 0
- No providers started — reads stored state only

### Task 5: Tests (`internal/cli/*_test.go`)
- Test human-readable format output
- Test JSON format output (valid JSON, correct fields)
- Test `--active` filter
- Test `--source` filter
- Test empty state
- Test elapsed formatting in status output

### Task 6: Build & Integration
- `go build ./cmd/holo` compiles successfully
- All tests pass: `go test ./... -v`
- All prior tests still pass (F1–F4)

## Definition of Done

- `holo` launches TUI
- `holo status` prints one-shot summary
- `holo status --json` outputs valid JSON
- `holo status --active` filters active only
- `holo status --source claude-code` filters by source
- `holo version` prints version
- All tests pass
- Dependency rules respected
