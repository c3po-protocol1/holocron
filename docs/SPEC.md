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
| F1 | Core Types + EventBus + SQLite Store | [specs/F1-core.md](./specs/F1-core.md) | ✅ Done |
| F2 | Config Loading | [specs/F2-config.md](./specs/F2-config.md) | ✅ Done |
| F3 | Claude Code Provider | [specs/F3-claude-code-provider.md](./specs/F3-claude-code-provider.md) | ✅ Done |
| F4 | TUI Session List | [specs/F4-tui-session-list.md](./specs/F4-tui-session-list.md) | ✅ Done |
| F5 | CLI Status Command | [specs/F5-cli-status.md](./specs/F5-cli-status.md) | ✅ Done |
| F6 | End-to-End Wiring + README Fix | [specs/F6-wiring-and-readme.md](./specs/F6-wiring-and-readme.md) | ✅ Done |
| F7 | Active-Only Quick Toggle | [specs/F7-active-toggle.md](./specs/F7-active-toggle.md) | Draft |
| F8 | OpenClaw Provider | [specs/F8-openclaw-provider.md](./specs/F8-openclaw-provider.md) | Draft |
| F9 | Session Detail View | [specs/F9-detail-view.md](./specs/F9-detail-view.md) | Draft |

### Future Features (not yet spec'd)

| # | Feature | Depends On |
|---|---------|------------|
| F10 | Labels & Linking | F9 |

| F11 | TUI Settings Panel | F9 |
| F12 | Daemon Mode | F9 |
| F13 | Web UI | F12 |

## Build Order

```
F1 → F2 → F3 → F4 → F5 → F6 → F7 (active toggle) → F8 (openclaw) → F9 (detail view) → F10+
```

## References

- [ARCHITECTURE.md](./ARCHITECTURE.md) — tech stack, folder structure, data flow
- [PRINCIPLES.md](./PRINCIPLES.md) — design principles with verification tests
