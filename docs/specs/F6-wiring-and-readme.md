# F6: End-to-End Wiring + README Fix

> Make `holo` actually work as advertised.

## Problem

F1–F5 built all the pieces, but they are not wired together:

1. **Providers never started.** `main.go` loads config/store but never creates Claude Code provider. TUI receives `nil` event channel → always empty.
2. **No Collector.** The orchestrator that connects providers → EventBus → Store does not exist as runnable code.
3. **README wrong install path.** Says `holocron-dev` but module is `c3po-protocol1`.
4. **README outdated.** Says "Planned" but F1–F5 are done.
5. **Tilde not expanded.** User-supplied `~/` paths in YAML config are not expanded.

## Changes Required

### 1. Create Collector (`internal/collector/collector.go`)

```go
type Collector struct {
    bus       *Bus
    store     store.Store
    providers []provider.Provider
    cancel    context.CancelFunc
}

func New(s store.Store) *Collector
func (c *Collector) AddProvider(p provider.Provider)
func (c *Collector) Start(ctx context.Context) error  // starts all providers, subscribes store
func (c *Collector) Subscribe() <-chan MonitorEvent     // for TUI
func (c *Collector) Stop()
```

### 2. Wire `runTUI()` in `main.go`

Replace `tui.New(nil, sessions)` with:

```go
c := collector.New(st)
// Create providers from config sources
for _, src := range cfg.Sources {
    if src.Type == "claude-code" {
        dir := expandTilde(src.SessionDir)
        if dir == "" { dir = defaultClaudeDir() }
        c.AddProvider(claudecode.New(dir, pollDuration(src)))
    }
}
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
c.Start(ctx)
defer c.Stop()

sessions, _ := st.ListSessions()
model := tui.New(c.Subscribe(), sessions)
```

### 3. Wire `runStatus()` in `main.go`

Start providers briefly so `holo status` discovers live sessions:

```go
c := collector.New(st)
// ... add providers from config
c.Start(ctx)
time.Sleep(500 * time.Millisecond)  // allow initial scan
c.Stop()
sessions, _ := st.ListSessions()    // now includes fresh data
```

### 4. Tilde expansion (`internal/config/config.go`)

```go
func expandTilde(path string) string {
    if strings.HasPrefix(path, "~/") {
        home, _ := os.UserHomeDir()
        return filepath.Join(home, path[2:])
    }
    return path
}
```

Apply after loading to: `store.path`, `sources[].sessionDir`, `sources[].path`.

### 5. Update README.md

- **Install path:** `go install github.com/c3po-protocol1/holocron/cmd/holo@latest`
- **Alias:** `alias holo="holocron"`
- **Roadmap:** Mark Phase 1 as complete
- **Features:** Change "Planned" to actual working features
- **Add Usage section** with `holocron`, `holocron status`, `holocron status --json`

## Files Changed

| File | Action |
|------|--------|
| `internal/collector/collector.go` | **NEW** |
| `cmd/holo/main.go` | **MODIFY** — wire collector + providers |
| `internal/config/config.go` | **MODIFY** — tilde expansion |
| `README.md` | **MODIFY** — fix paths, update status |

## Verification

```bash
# 1. Build and run TUI — shows Claude Code sessions if any exist
go build -o holocron ./cmd/holo/ && ./holo

# 2. Open Claude Code elsewhere → session appears in Holocron live

# 3. Quick status shows discovered sessions
./holo status

# 4. JSON output is valid
./holo status --json | jq .

# 5. No config → helpful message, no crash
mv ~/.holocron/config.yaml /tmp/ && ./holo; mv /tmp/config.yaml ~/.holocron/

# 6. Version
./holo version
```

## Out of Scope

- OpenClaw/Codex providers, labels, detail view — future features only
- This spec is purely making existing F1–F5 code work end-to-end
