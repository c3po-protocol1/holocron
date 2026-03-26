# F2: Config Loading — Development Plan

## Spec References

- **F2 Spec:** [../specs/F2-config.md](../specs/F2-config.md)
- **Architecture:** [../ARCHITECTURE.md](../ARCHITECTURE.md)
- **Principles:** [../PRINCIPLES.md](../PRINCIPLES.md)
- **Spec Index:** [../SPEC.md](../SPEC.md)

## Overview

Load, validate, and merge configuration from YAML files, env vars, and CLI flags.
Determines which sources to activate and how store/view behave.

## Tasks (TDD — write tests first, then implement)

### Task 1: Config Types (`internal/config/config.go`)
- Define Config, SourceConfig, StoreConfig, ViewConfig, LabelsConfig, LabelRule structs
- All fields with yaml tags as specified in F2 spec

### Task 2: Defaults
- Apply sensible defaults when fields are missing:
  - store.type: "sqlite"
  - store.path: "~/.holocron/holocron.db"
  - store.retentionDays: 30
  - view.refreshMs: 1000
  - view.showIdle: true
  - view.groupBy: "source"

### Task 3: Config Loading & Merging
- Load from ~/.holocron/config.yaml (user config)
- Load from ./holocron.yaml (project-local override)
- Merge: local wins over user config
- Apply defaults for any missing fields
- Works with no config file (returns defaults)

### Task 4: Env Var Expansion
- Expand ${ENV_VAR} syntax in token fields
- Support in all string fields of SourceConfig

### Task 5: Validation
- sources[].type must be known ("claude-code", "openclaw", "codex", "file-watch")
- pollIntervalMs minimum: 500
- retentionDays minimum: 1
- Clear error messages for invalid config

### Task 6: Tests (`internal/config/config_test.go`)
- Parses valid YAML config
- Expands ${ENV_VAR} in token fields
- Merges user + local configs (local wins)
- Applies defaults for missing fields
- Returns clear error for invalid source type
- Returns clear error for malformed YAML
- Works with no config file (defaults only)

## Definition of Done

- All unit tests pass
- Config types match F2 spec exactly
- `go test ./internal/config/... -v` passes
