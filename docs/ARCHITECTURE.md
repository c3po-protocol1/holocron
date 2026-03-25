# Holocron — Architecture Guide


> For design principles, see [PRINCIPLES.md](./PRINCIPLES.md).

## Tech Stack

| Component | Choice | Why |
|-----------|--------|-----|
| **Language** | Go 1.22+ | Single binary, goroutines for concurrency, strong AI coding support |
| **TUI** | Bubbletea + Lipgloss + Bubbles | Battle-tested (lazygit, charm tools), Elm architecture |
| **Storage** | SQLite via `modernc.org/sqlite` | Pure Go (no CGO), embedded, zero config |
| **CLI** | `spf13/cobra` | Standard Go CLI framework |
| **Config** | `gopkg.in/yaml.v3` | YAML parsing |
| **File Watching** | `fsnotify/fsnotify` | Cross-platform file system notifications |
| **WebSocket** | `nhooyr.io/websocket` | Modern, context-aware (for OpenClaw Gateway) |
| **Logging** | `log/slog` | Go stdlib, structured logging |
| **Testing** | `testing` + `testify` | Stdlib + assertions |

### Why These Choices

**Pure Go SQLite (`modernc.org/sqlite`)** over `mattn/go-sqlite3`:
- No CGO dependency → simpler cross-compilation
- `go install` just works on any platform
- Slight performance trade-off is irrelevant for our data volume

**Bubbletea** over `tview` or `tcell`:
- Elm architecture (Model → Update → View) is clean and testable
- Charm ecosystem provides styled components (Lipgloss) and widgets (Bubbles)
- Active community, good documentation

**Cobra** over `urfave/cli`:
- More Go projects use it → familiar to contributors
- Built-in help, completion, subcommands

## Folder Structure

```
holocron/
├── cmd/
│   └── holocron/
│       └── main.go              ← Entry point, wiring only
│
├── internal/
│   ├── collector/
│   │   ├── collector.go         ← Collector: starts providers, manages lifecycle
│   │   ├── bus.go               ← EventBus: in-memory pub/sub (channel-based)
│   │   └── types.go             ← MonitorEvent, SessionState, EventType, etc.
│   │
│   ├── store/
│   │   ├── store.go             ← Store interface definition
│   │   └── sqlite/
│   │       ├── sqlite.go        ← SQLite Store implementation
│   │       ├── migrations.go    ← Schema creation / migrations
│   │       └── sqlite_test.go
│   │
│   ├── provider/
│   │   └── provider.go          ← Provider interface + registry
│   │
│   ├── providers/
│   │   ├── claudecode/
│   │   │   ├── claudecode.go    ← Claude Code provider
│   │   │   ├── scanner.go       ← Session file scanner
│   │   │   ├── tailer.go        ← JSONL file tailer
│   │   │   ├── process.go       ← Running process detector
│   │   │   └── claudecode_test.go
│   │   │
│   │   ├── openclaw/
│   │   │   ├── openclaw.go      ← OpenClaw provider (Phase 2)
│   │   │   └── openclaw_test.go
│   │   │
│   │   └── codex/
│   │       ├── codex.go         ← Codex provider (Phase 2+)
│   │       └── codex_test.go
│   │
│   ├── config/
│   │   ├── config.go            ← Config struct + loader
│   │   └── config_test.go
│   │
│   └── tui/
│       ├── app.go               ← Bubbletea main model
│       ├── session_list.go      ← Session list component
│       ├── detail.go            ← Session detail panel
│       ├── styles.go            ← Lipgloss styles
│       ├── keys.go              ← Key bindings
│       └── help.go              ← Help overlay
│
├── docs/
│   ├── SPEC-v0.1.md             ← Full specification
│   └── ARCHITECTURE.md          ← This file
│
├── config.example.yaml          ← Example configuration
├── .gitignore
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
└── README.md
```

### Directory Conventions

- **`cmd/`** — Entry points only. No business logic. Just wiring.
- **`internal/`** — All business logic. Cannot be imported by external packages.
- **`internal/collector/`** — Core orchestration. Depends on provider and store interfaces, not implementations.
- **`internal/providers/`** — Each provider in its own package. Can only import `collector/types.go` and `provider/provider.go`.
- **`internal/tui/`** — Rendering only. Receives `<-chan MonitorEvent`. Never calls providers directly.
- **`docs/`** — Specifications, architecture, decisions.

### Dependency Rules

```
cmd/holocron
  └── imports: collector, config, store/sqlite, providers/*, tui

tui
  └── imports: collector/types (MonitorEvent, SessionState only)

providers/*
  └── imports: collector/types, provider (interface)

store/sqlite
  └── imports: collector/types, store (interface)

collector
  └── imports: provider (interface), store (interface)
```

**The rule:** arrows point inward. `tui` and `providers` depend on `collector/types`. Never the reverse.

## Data Flow (Detailed)

```
1. Startup
   main.go creates: Config → Store → Collector → Providers → TUI
   Collector.Start() → each Provider.Start(ctx, bus)

2. Provider detects a session
   Claude Code Provider scans ~/.claude/projects/
   Finds JSONL file → parses → emits MonitorEvent to EventBus

3. EventBus routes
   bus.Publish(event)
   → Store.Save(event)           // persist
   → TUI subscriber channel      // display

4. TUI renders
   Bubbletea model receives event via channel
   Updates SessionState map
   Re-renders view

5. Shutdown
   TUI exits → ctx cancelled → Providers stop → Store closes
```

## Configuration Loading Order

1. `~/.holocron/config.yaml` (user config)
2. `./holocron.yaml` (project-local override)
3. Environment variables (`HOLOCRON_*`)
4. CLI flags (highest priority)

## Future: Daemon Mode (Phase 4)

When splitting into daemon + client:

```
# Daemon (new binary: holocron-daemon)
Config → Store → Collector → Providers → WebSocket Server

# Client (existing binary: holocron)
WebSocket Client → <-chan MonitorEvent → TUI
```

Code changes needed:
- New `cmd/holocron-daemon/main.go` (wiring only)
- New `internal/api/server.go` (WebSocket event server)
- New `internal/api/client.go` (WebSocket event client that returns `<-chan MonitorEvent`)
- TUI code: **zero changes**
- Provider code: **zero changes**
- Store code: **zero changes**

This is the payoff of separation.
