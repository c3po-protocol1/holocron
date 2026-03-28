# F5: CLI Status Command

> Quick glance without entering TUI.

## What This Feature Does

`holo status` prints a one-shot summary of all current sessions to stdout and exits. No interactive UI. Useful for scripting, quick checks, and piping.

## Package

- `cmd/holo/main.go` — Cobra root + subcommands
- (reuses `internal/collector`, `internal/store`, `internal/config`)

## CLI Structure (Cobra)

```
holo                    # default: launch TUI (F4)
holo status             # this feature: one-shot summary
holo version            # print version
holo help               # cobra help
```

## Output Format

**Default (human-readable):**
```
Holocron 🔭 — 3 sessions

● claude-code  a8837a23  editing   2m 13s  ~/Projects/my-app
                         Edit → src/index.ts
◌ claude-code  323ac29b  idle     15m 02s  ~/Projects/agent-monitor
● openclaw     r2d2      running   5m 30s  tokens: 12.5k/1M (1%)
```

**JSON (for scripting):**
```bash
holo status --json
```
```json
[
  {
    "source": "claude-code",
    "sessionId": "a8837a23...",
    "status": "tool_running",
    "workspace": "~/Projects/my-app",
    "elapsedMs": 133000,
    "currentTool": "Edit",
    "currentTarget": "src/index.ts"
  }
]
```

## Behavior

1. Load config
2. Initialize store (SQLite read-only)
3. Load current sessions from store
4. For sources with `watchProcesses: true`, also do a quick process check to update active status
5. Print and exit (exit code 0)

**No providers started.** This reads stored state only + quick process check. Fast.

## Flags

| Flag | Description |
|------|-------------|
| `--json` | Output JSON |
| `--source <type>` | Filter by source type |
| `--active` | Show only active sessions |

## Verification

```bash
# 1. Run `holo status` with no sessions → "No sessions found."
# 2. After using TUI with active sessions → shows stored sessions
# 3. `holo status --json` → valid JSON array
# 4. `holo status --json | jq '.[] | .sessionId'` → works
# 5. `holo status --active` → only non-idle sessions
# 6. Exit code is always 0 (even with no sessions)
```
