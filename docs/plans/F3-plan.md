# F3: Claude Code Provider — Development Plan

## Spec References

- **F3 Spec:** [../specs/F3-claude-code-provider.md](../specs/F3-claude-code-provider.md)
- **F1 Spec (types):** [../specs/F1-core.md](../specs/F1-core.md)
- **Architecture:** [../ARCHITECTURE.md](../ARCHITECTURE.md)
- **Principles:** [../PRINCIPLES.md](../PRINCIPLES.md)
- **Spec Index:** [../SPEC.md](../SPEC.md)

## Overview

First source provider. Discovers and monitors Claude Code sessions by scanning
~/.claude/projects/, detecting running processes, and tailing active JSONL files.

## Tasks (TDD — write tests first, then implement)

### Task 1: Provider Interface (`internal/provider/provider.go`)
- Define Provider interface: Name(), Start(ctx, bus), Stop()

### Task 2: Scanner (`internal/providers/claudecode/scanner.go`)
- Scan ~/.claude/projects/ for session directories
- List .jsonl files in each directory
- Extract workspace from directory slug (reverse slug: -Users-c-3po-X → /Users/c-3po/X)
- **Tests:** scanner finds .jsonl files in mock directory, workspace extracted correctly

### Task 3: Tailer (`internal/providers/claudecode/tailer.go`)
- Tail JSONL files using fsnotify
- Parse Claude Code JSONL format (user, assistant, tool_use, tool_result)
- Map JSONL types to MonitorEvent (see spec event mapping table)
- Extract metadata: cwd→workspace, gitBranch→label, sessionId, version→label
- **Tests:** each JSONL type parsed correctly, events have correct fields

### Task 4: Process Detector (`internal/providers/claudecode/process.go`)
- Run ps to detect Claude Code processes
- Poll at configurable interval (default 2000ms)
- Cross-reference with session files
- **Tests:** identifies running processes from mock ps output

### Task 5: Provider (`internal/providers/claudecode/claudecode.go`)
- Implement Provider interface
- Start(): initial scan + process poller + file watchers + directory watcher
- Stop(): cancel context, stop watchers, stop poller
- All events emitted with source="claude-code"
- **Tests:** provider lifecycle, events emitted with correct source

### Task 6: Integration Verification
- All unit tests pass: `go test ./internal/providers/claudecode/... -v`
- All prior tests still pass: `go test ./... -v`

## Definition of Done

- All unit tests pass
- Provider follows Provider interface from ARCHITECTURE.md
- Dependency rule: only imports collector/types and provider interface
- Commit all changes when done
