# Holocron — Specification Index

> "If a record is not in the Holocron, it does not exist."

## Vision

A standalone developer tool that provides **real-time visibility** into all AI coding sessions running on your machine — regardless of which tool or orchestrator is used.

Think `htop` for AI-assisted development.

**Name:** Holocron (`holo` CLI)
**Language:** Go | **TUI:** Bubbletea | **Storage:** SQLite

## Feature Specs

Each spec is a self-contained, testable unit. Build and verify independently.

| # | Feature | Spec | Status |
|---|---------|------|--------|
| F1 | Core Types + EventBus + SQLite Store | [specs/F1-core.md](./specs/F1-core.md) | Draft |
| F2 | Config Loading | [specs/F2-config.md](./specs/F2-config.md) | Draft |
| F3 | Claude Code Provider | [specs/F3-claude-code-provider.md](./specs/F3-claude-code-provider.md) | Draft |
| F4 | TUI Session List | [specs/F4-tui-session-list.md](./specs/F4-tui-session-list.md) | Draft |
| F5 | CLI Status Command | [specs/F5-cli-status.md](./specs/F5-cli-status.md) | Draft |

### Future Features (not yet spec'd)

| # | Feature | Depends On |
|---|---------|------------|
| F6 | OpenClaw Provider | F1, F2 |
| F7 | Labels & Linking | F1, F4 |
| F8 | TUI Detail View | F4 |
| F9 | Daemon Mode | F1 |
| F10 | Web UI | F9 |

## Build Order

```
F1 (core) → F2 (config) → F3 (claude code) → F4 (TUI) → F5 (CLI status)
```

Each feature is testable on its own:
- **F1**: unit tests — events save/load, bus pub/sub works
- **F2**: unit tests — config parses, validates, env vars expand
- **F3**: integration test — point at real `~/.claude/projects/`, see sessions discovered
- **F4**: manual test — launch TUI, see sessions rendered live
- **F5**: manual test — run `holo status`, see one-shot output

## References

- [ARCHITECTURE.md](./ARCHITECTURE.md) — tech stack, folder structure, data flow
- [PRINCIPLES.md](./PRINCIPLES.md) — design principles with verification tests
