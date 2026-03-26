# F2: Config Loading

> Tell Holocron where to look.

## What This Feature Does

Loads, validates, and merges configuration from files, env vars, and CLI flags. Determines which sources to activate and how the store/view behave.

## Package

- `internal/config/config.go`
- `internal/config/config_test.go`

## Config Structure

```go
type Config struct {
    Sources []SourceConfig `yaml:"sources"`
    Store   StoreConfig    `yaml:"store"`
    View    ViewConfig     `yaml:"view"`
    Labels  LabelsConfig   `yaml:"labels"`
}

type SourceConfig struct {
    Type           string `yaml:"type"`             // "claude-code" | "openclaw" | "codex" | "file-watch"
    Discover       string `yaml:"discover"`          // "auto" | "manual"
    SessionDir     string `yaml:"sessionDir"`        // override default paths
    WatchProcesses bool   `yaml:"watchProcesses"`
    TailActive     bool   `yaml:"tailActive"`
    PollIntervalMs int    `yaml:"pollIntervalMs"`
    Gateway        string `yaml:"gateway"`           // OpenClaw WS URL
    Token          string `yaml:"token"`             // supports ${ENV_VAR}
    Path           string `yaml:"path"`              // file-watch glob
    Format         string `yaml:"format"`            // file-watch format
}

type StoreConfig struct {
    Type          string `yaml:"type"`              // "sqlite"
    Path          string `yaml:"path"`              // db file path
    RetentionDays int    `yaml:"retentionDays"`
}

type ViewConfig struct {
    RefreshMs int    `yaml:"refreshMs"`
    ShowIdle  bool   `yaml:"showIdle"`
    GroupBy   string `yaml:"groupBy"`              // "source" | "workspace" | "label"
}

type LabelsConfig struct {
    Rules []LabelRule `yaml:"rules"`
}

type LabelRule struct {
    Match map[string]string `yaml:"match"`
    Set   map[string]string `yaml:"set"`
}
```

## Loading Order (highest priority last)

1. `~/.holocron/config.yaml` (user config)
2. `./holocron.yaml` (project-local override)
3. Environment variables (`HOLOCRON_STORE_PATH`, etc.)
4. CLI flags

## Env Var Expansion

Token fields support `${ENV_VAR}` syntax:
```yaml
token: ${OPENCLAW_GATEWAY_TOKEN}
```

## Defaults

```yaml
store:
  type: sqlite
  path: ~/.holocron/holocron.db
  retentionDays: 30
view:
  refreshMs: 1000
  showIdle: true
  groupBy: source
```

If no config file exists, Holocron runs with defaults + no sources (shows empty TUI with helpful message).

## Validation

- `sources[].type` must be a known type
- `store.path` parent directory must exist (or be creatable)
- `pollIntervalMs` minimum: 500
- `retentionDays` minimum: 1

## Verification

```bash
go test ./internal/config/...

# Specific checks:
# 1. Parses valid YAML config
# 2. Expands ${ENV_VAR} in token fields
# 3. Merges user + local configs (local wins)
# 4. Applies defaults for missing fields
# 5. Returns clear error for invalid source type
# 6. Returns clear error for malformed YAML
# 7. Works with no config file (defaults only)
```
