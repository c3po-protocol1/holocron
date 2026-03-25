# 🔭 Holocron

> "If a record is not in the Holocron, it does not exist."

**`htop` for AI-assisted development.** Real-time visibility into all AI coding sessions running on your machine.

## What is Holocron?

Holocron monitors AI coding tools (Claude Code, Codex, Gemini, etc.) and orchestrators (OpenClaw, etc.) from a single terminal UI. Each source is independent — Holocron just shows you what's happening.

## Features (Planned)

- 🔍 **Auto-discover** running AI coding sessions
- 📊 **Unified view** across Claude Code, Codex, OpenClaw, and more
- 🏷️ **Labels** — link sessions to projects, agents, or tasks at the view layer
- 💾 **SQLite storage** — history survives restarts
- 🖥️ **TUI first** — Bubbletea-powered terminal UI
- 🌐 **Web UI later** — same data, browser renderer

## Quick Start

```bash
# Install
go install github.com/holocron-dev/holocron/cmd/holocron@latest

# Run
holo          # Launch TUI
holo status   # One-shot session summary
```

## Architecture

```
Sources (independent)          Unified Layer          Renderers
┌──────────────┐
│ Claude Code  │──┐
├──────────────┤  │    ┌───────────┐    ┌─────────┐
│ OpenClaw     │──┼───▶│ EventBus  │───▶│ TUI     │
├──────────────┤  │    │ + SQLite  │    ├─────────┤
│ Codex        │──┘    └───────────┘    │ Web(v2) │
├──────────────┤                        └─────────┘
│ (any new)    │──  just add a provider
└──────────────┘
```

## Config

```yaml
# ~/.holocron/config.yaml
sources:
  - type: claude-code
    discover: auto
    watchProcesses: true

  - type: openclaw
    gateway: ws://127.0.0.1:18789
    token: ${OPENCLAW_GATEWAY_TOKEN}

store:
  type: sqlite
  path: ~/.holocron/holocron.db
```

## Roadmap

- [x] Spec v0.1
- [ ] Phase 1: Core + Claude Code Provider + TUI
- [ ] Phase 2: OpenClaw Provider
- [ ] Phase 3: Labels & Linking
- [ ] Phase 4: Daemon + Web UI

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go |
| TUI | Bubbletea + Lipgloss (Charm) |
| Storage | SQLite (pure Go, no CGO) |
| File watching | fsnotify |
| CLI | Cobra |

## License

MIT

---

*Named after the Jedi Holocron — a device that stores knowledge, accessible only to those who seek it.*
