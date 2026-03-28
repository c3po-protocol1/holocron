# 🔭 Holocron

> "If a record is not in the Holocron, it does not exist."

**`htop` for AI-assisted development.** Real-time visibility into all AI coding sessions running on your machine.

## What is Holocron?

Holocron monitors AI coding tools (Claude Code, Codex, Gemini, etc.) and orchestrators (OpenClaw, etc.) from a single terminal UI. Each source is independent — Holocron just shows you what's happening.

## Features

- 🔍 **Auto-discover** running Claude Code sessions
- 📊 **Unified view** across multiple AI coding tools
- 💾 **SQLite storage** — history survives restarts
- 🖥️ **TUI first** — Bubbletea-powered terminal UI
- 📋 **CLI status** — one-shot summary with `holo status`
- 🏷️ **Labels** — link sessions to projects, agents, or tasks (coming soon)
- 🌐 **Web UI** — same data, browser renderer (coming soon)

## Quick Start

```bash
# Install
go install github.com/c3po-protocol1/holocron/cmd/holocron@latest

# Alias (optional)
alias holo="holocron"

# Run
holo          # Launch TUI
```

## Usage

```bash
# Launch the interactive TUI
holocron

# One-shot session summary
holocron status

# JSON output (for scripting)
holocron status --json

# Filter by source
holocron status --source claude-code

# Show only active sessions
holocron status --active

# Print version
holocron version
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

- [x] Phase 1: Core + Claude Code Provider + TUI + CLI Status + E2E Wiring
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
