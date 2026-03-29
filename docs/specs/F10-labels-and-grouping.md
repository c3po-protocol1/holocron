# F10: Labels & Grouping

> See your sessions organized by who and where.

## What This Feature Does

Auto-labels sessions using provider defaults + config rules, then groups them in the TUI with visual headers. `g` key cycles: none → agent → channel.

## Label Sources

**1. OpenClaw provider (automatic):** Parse session key to set labels.
```
agent:r2d2:discord:direct:1088...  → agent=r2d2, channel=discord
agent:yoda:cron:a38a9d61...        → agent=yoda, channel=cron
```

**2. Claude Code provider (automatic):** `channel=local` (always).

**3. Config rules (user-defined):**
```yaml
labels:
  rules:
    - match:
        source: claude-code
        workspace: "*/workspace-r2d2*"
      set:
        agent: r2d2
    - match:
        source: claude-code
        workspace: "*/Projects/holocron*"
      set:
        project: holocron
```

### Rule Matching

Matchable fields: `source`, `workspace`, `sessionKey`, `sessionId`.
Glob matching with `*` wildcard. Rules evaluated in order — later overrides earlier. Provider-set labels applied first, then config rules.

```go
func ApplyLabels(s *SessionState, rules []LabelRule)
```

## Grouping

### `g` Key — Cycle Group Mode

`none → agent → channel → none → ...`

### Grouped Layout

```
─── r2d2 (2 sessions, 1 active) ──────────────────
  openclaw      r2d2         ● active    0m 30s
                discord:direct
  claude-code   a8837a23     ● editing   2m 13s
                ~/workspace-r2d2

─── yoda (2 sessions, 1 active) ──────────────────
  openclaw      yoda         ● active    0m 05s
                discord:direct
  claude-code   323ac29b     ◌ idle      15m
                ~/workspace-yoda

─── unlabeled (1 session) ────────────────────────
  claude-code   f4da5833     ◌ idle      1h 02m
                ~/Projects/personal
```

Sessions without the grouping label go under `unlabeled`. Groups with active sessions sort first. `unlabeled` always last.

### Group by Channel

```
─── discord (3 sessions, 2 active) ───────────────
─── cron (2 sessions) ────────────────────────────
─── local (2 sessions, 1 active) ─────────────────
```

## Implementation

### New File: `internal/labels/labels.go`

```go
type SessionGroup struct {
    Label    string
    Sessions []SessionState
    Active   int
}

func ApplyLabels(s *SessionState, rules []LabelRule)
func GroupSessions(sessions []SessionState, mode GroupMode) []SessionGroup
```

### Files Modified

| File | Change |
|------|--------|
| `internal/tui/app.go` | Add `groupMode`, handle `g` key |
| `internal/tui/session_list.go` | Render group headers |
| `internal/tui/styles.go` | Group header style |
| `internal/tui/keys.go` | Add `g` binding |
| `internal/tui/help.go` | Update help |
| `internal/providers/openclaw/openclaw.go` | Ensure agent/channel labels |
| `internal/providers/claudecode/claudecode.go` | Set channel=local |

## Interaction with Active Filter

Active filter applies BEFORE grouping. Empty groups are hidden.

## Verification

```bash
# 1. Run `holo` with OpenClaw → sessions show agent/channel labels
# 2. Press `g` → group by agent with headers
# 3. Press `g` → group by channel
# 4. Press `g` → back to no grouping
# 5. Add label rule in config → label appears on matching sessions
# 6. Press `a` while grouped → empty groups disappear
# 7. Status bar shows current group mode
```
