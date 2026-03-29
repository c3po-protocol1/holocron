# 🔭 Holocron

> "If a record is not in the Holocron, it does not exist."

**`htop` for AI-assisted development.** Real-time visibility into all AI coding sessions running on your machine.

## What is Holocron?

Holocron monitors AI coding tools (Claude Code, Codex, etc.) and orchestrators (OpenClaw) from a single terminal UI. Each source is independent — Holocron just shows you what's happening.

## Features

- 🔍 **Auto-discover** running Claude Code sessions
- 🌐 **OpenClaw provider** — monitor all OpenClaw agents via polling + smart diff
- 📊 **Unified view** across multiple AI coding tools
- 💾 **SQLite storage** — history survives restarts
- 🖥️ **TUI first** — Bubbletea-powered terminal UI
- 📋 **CLI status** — one-shot summary with `holo status`
- 🔎 **Session detail** — press Enter to see full session info + scrollable event log
- 🏷️ **Labels & grouping** — auto-labels from providers, glob-based config rules, `g` key cycles group modes
- ⚡ **Active filter** — press `a` to toggle active-only view
- 📊 **Follow mode** — press `f` in detail view to auto-scroll to new events

## Quick Start

```bash
# Install
go install github.com/c3po-protocol1/holocron/cmd/holo@latest

# Run
holo          # Launch TUI
```

## Usage

```bash
# Launch the interactive TUI
holo

# One-shot session summary
holo status

# JSON output (for scripting)
holo status --json

# Filter by source
holo status --source claude-code
holo status --source openclaw

# Show only active sessions
holo status --active

# Print version
holo version
```

## TUI Key Bindings

### Session List
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate sessions |
| `Enter` | Open session detail view |
| `a` | Toggle active-only filter |
| `g` | Cycle group mode (none → agent → channel) |
| `?` | Toggle help |
| `q` | Quit |

### Session Detail
| Key | Action |
|-----|--------|
| `Esc` | Return to session list |
| `↑/↓` or `j/k` | Scroll event log |
| `G` | Jump to bottom (newest) |
| `g` | Jump to top (oldest) |
| `f` | Toggle follow mode (auto-scroll) |
| `?` | Toggle help |

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
  # Claude Code — auto-discover sessions
  - type: claude-code
    discover: auto
    sessionDir: ~/.claude/projects/
    watchProcesses: true
    tailActive: true
    pollIntervalMs: 2000

  # OpenClaw — monitor agent sessions
  - type: openclaw
    pollIntervalMs: 5000
    idleThresholdMs: 60000

store:
  type: sqlite
  path: ~/.holocron/holocron.db
  retentionDays: 30

view:
  refreshMs: 1000
  showIdle: true
  groupBy: source

# Label rules with glob matching
labels:
  rules:
    - match:
        source: openclaw
        sessionKey: "agent:r2d2:*"
      set:
        agent: r2d2
    - match:
        source: claude-code
        workspace: "*/Projects/holocron"
      set:
        project: holocron
```

## Roadmap

- [x] Phase 1: Core Types + EventBus + SQLite Store (F1)
- [x] Phase 1: Config Loading (F2)
- [x] Phase 1: Claude Code Provider (F3)
- [x] Phase 1: TUI Session List (F4)
- [x] Phase 1: CLI Status Command (F5)
- [x] Phase 1: End-to-End Wiring (F6)
- [x] Phase 2: Active-Only Quick Toggle (F7)
- [x] Phase 2: OpenClaw Provider (F8)
- [x] Phase 2: Session Detail View (F9)
- [x] Phase 2: Labels & Grouping (F10)
- [ ] Phase 3: Daemon + Web UI
- [ ] Phase 3: Codex Provider

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
