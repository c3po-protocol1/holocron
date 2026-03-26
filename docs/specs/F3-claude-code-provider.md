# F3: Claude Code Provider

> The first source. Discovers and monitors Claude Code sessions.

## What This Feature Does

Scans `~/.claude/projects/` for session files, detects running Claude Code processes, and tails active JSONL files to emit real-time events.

## Package

- `internal/providers/claudecode/claudecode.go` — Provider implementation
- `internal/providers/claudecode/scanner.go` — Session file discovery
- `internal/providers/claudecode/tailer.go` — JSONL file tailing
- `internal/providers/claudecode/process.go` — Running process detection

## Provider Interface

```go
type Provider interface {
    Name() string                                    // "claude-code"
    Start(ctx context.Context, bus EventBus) error
    Stop() error
}
```

## Discovery: Session File Scan

**Location:** `~/.claude/projects/<workspace-slug>/`

Each directory contains JSONL files named `<sessionId>.jsonl`.

**Workspace slug format** (observed): path with `/` replaced by `-`
- `/Users/c-3po/Projects/holocron` → `-Users-c-3po-Projects-holocron`

**On Start:**
1. Scan all directories in `~/.claude/projects/`
2. For each directory, list `.jsonl` files
3. Read last N lines to determine session status
4. Emit `session.start` event for each discovered session

## Discovery: Process Detection

**Method:** Periodically run `ps` and filter for Claude Code processes.

```bash
ps aux | grep -E "claude|claude-code" | grep -v grep
```

**Poll interval:** configurable, default 2000ms

**Result:** Mark sessions as "active" if a matching process exists. Cross-reference with session files via working directory or session ID.

## Discovery: JSONL Tailing

For sessions detected as active, tail their `.jsonl` file using `fsnotify`.

**Claude Code JSONL format** (observed):

```jsonl
{"type":"queue-operation","operation":"enqueue","timestamp":"...","sessionId":"..."}
{"type":"user","message":{"role":"user","content":"..."},"cwd":"/path","sessionId":"...","version":"2.1.71","gitBranch":"main"}
{"type":"assistant","message":{"role":"assistant","content":"..."},"sessionId":"..."}
{"type":"tool_use","name":"Edit","input":{"file_path":"src/index.ts",...},"sessionId":"..."}
{"type":"tool_result","content":"...","sessionId":"..."}
```

**Event mapping:**

| Claude Code type | MonitorEvent |
|-----------------|--------------|
| `user` | `event: message, status: thinking` |
| `assistant` | `event: message, status: idle` |
| `tool_use` | `event: tool.start, status: tool_running, detail.tool: <name>` |
| `tool_result` | `event: tool.end, status: thinking` |

**Metadata extracted:**
- `cwd` → `workspace`
- `gitBranch` → `labels["git_branch"]`
- `sessionId` → `sessionId`
- `version` → `labels["claude_version"]`

## Lifecycle

```
Start()
  ├── Initial scan (all sessions in ~/.claude/projects/)
  ├── Start process poller (goroutine, every pollIntervalMs)
  ├── Start file watchers for active sessions (fsnotify)
  └── Watch for new session directories (fsnotify on ~/.claude/projects/)

Stop()
  ├── Cancel context
  ├── Stop all file watchers
  └── Stop process poller
```

## Config

```yaml
sources:
  - type: claude-code
    discover: auto
    sessionDir: ~/.claude/projects/   # override
    watchProcesses: true
    tailActive: true
    pollIntervalMs: 2000
```

## Verification

```bash
go test ./internal/providers/claudecode/...

# Manual integration test:
# 1. Run `holo` with claude-code source configured
# 2. Open Claude Code in another terminal, start a session
# 3. Verify: session appears in Holocron
# 4. Use a tool in Claude Code (e.g. edit a file)
# 5. Verify: tool event appears in Holocron
# 6. Close Claude Code session
# 7. Verify: session status changes to idle/done

# Unit test checks:
# 1. Scanner finds .jsonl files in mock directory
# 2. Tailer parses each JSONL type correctly
# 3. Process detector identifies running claude processes
# 4. Events emitted have correct source="claude-code"
# 5. Workspace extracted from directory slug
```
