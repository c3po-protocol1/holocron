package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
sources:
  - type: claude-code
    discover: auto
    sessionDir: /tmp/sessions
    watchProcesses: true
    tailActive: true
    pollIntervalMs: 1000
  - type: openclaw
    gateway: ws://localhost:9090
    token: my-token
store:
  type: sqlite
  path: /tmp/test.db
  retentionDays: 14
view:
  refreshMs: 500
  showIdle: false
  groupBy: workspace
labels:
  rules:
    - match:
        workspace: myproject
      set:
        team: backend
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := LoadFile(cfgPath)
	require.NoError(t, err)

	require.Len(t, cfg.Sources, 2)
	assert.Equal(t, "claude-code", cfg.Sources[0].Type)
	assert.Equal(t, "auto", cfg.Sources[0].Discover)
	assert.Equal(t, "/tmp/sessions", cfg.Sources[0].SessionDir)
	assert.True(t, cfg.Sources[0].WatchProcesses)
	assert.True(t, cfg.Sources[0].TailActive)
	assert.Equal(t, 1000, cfg.Sources[0].PollIntervalMs)

	assert.Equal(t, "openclaw", cfg.Sources[1].Type)
	assert.Equal(t, "ws://localhost:9090", cfg.Sources[1].Gateway)
	assert.Equal(t, "my-token", cfg.Sources[1].Token)

	assert.Equal(t, "sqlite", cfg.Store.Type)
	assert.Equal(t, "/tmp/test.db", cfg.Store.Path)
	assert.Equal(t, 14, cfg.Store.RetentionDays)

	assert.Equal(t, 500, cfg.View.RefreshMs)
	assert.False(t, cfg.View.ShowIdle)
	assert.Equal(t, "workspace", cfg.View.GroupBy)

	require.Len(t, cfg.Labels.Rules, 1)
	assert.Equal(t, "myproject", cfg.Labels.Rules[0].Match["workspace"])
	assert.Equal(t, "backend", cfg.Labels.Rules[0].Set["team"])
}

func TestExpandEnvVarInToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	t.Setenv("OPENCLAW_GATEWAY_TOKEN", "secret-123")

	yaml := `
sources:
  - type: openclaw
    gateway: ws://localhost:9090
    token: ${OPENCLAW_GATEWAY_TOKEN}
    pollIntervalMs: 500
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := LoadFile(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "secret-123", cfg.Sources[0].Token)
}

func TestMergeUserAndLocalConfig(t *testing.T) {
	userDir := t.TempDir()
	localDir := t.TempDir()

	userYAML := `
sources:
  - type: claude-code
    discover: auto
    pollIntervalMs: 2000
store:
  type: sqlite
  path: /user/path.db
  retentionDays: 30
view:
  refreshMs: 1000
  showIdle: true
  groupBy: source
`
	localYAML := `
sources:
  - type: openclaw
    gateway: ws://localhost:9090
    token: local-token
    pollIntervalMs: 500
store:
  path: /local/path.db
  retentionDays: 7
view:
  refreshMs: 500
`
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "config.yaml"), []byte(userYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "holocron.yaml"), []byte(localYAML), 0644))

	cfg, err := Load(userDir, localDir)
	require.NoError(t, err)

	// Local sources replace user sources
	require.Len(t, cfg.Sources, 1)
	assert.Equal(t, "openclaw", cfg.Sources[0].Type)

	// Local store fields win
	assert.Equal(t, "/local/path.db", cfg.Store.Path)
	assert.Equal(t, 7, cfg.Store.RetentionDays)
	// User store type preserved when local doesn't set it
	assert.Equal(t, "sqlite", cfg.Store.Type)

	// Local view fields win
	assert.Equal(t, 500, cfg.View.RefreshMs)
	// User view fields preserved when local doesn't set them
	assert.True(t, cfg.View.ShowIdle)
	assert.Equal(t, "source", cfg.View.GroupBy)
}

func TestApplyDefaults(t *testing.T) {
	cfg := Defaults()

	assert.Equal(t, "sqlite", cfg.Store.Type)
	assert.Contains(t, cfg.Store.Path, "holocron.db")
	assert.Equal(t, 30, cfg.Store.RetentionDays)

	assert.Equal(t, 1000, cfg.View.RefreshMs)
	assert.True(t, cfg.View.ShowIdle)
	assert.Equal(t, "source", cfg.View.GroupBy)

	assert.Empty(t, cfg.Sources)
}

func TestInvalidSourceType(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
sources:
  - type: unknown-source
    pollIntervalMs: 500
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := LoadFile(cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-source")
	assert.Contains(t, err.Error(), "invalid source type")
}

func TestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	require.NoError(t, os.WriteFile(cfgPath, []byte("{{not yaml at all:"), 0644))

	_, err := LoadFile(cfgPath)
	require.Error(t, err)
}

func TestNoConfigFile(t *testing.T) {
	userDir := t.TempDir()
	localDir := t.TempDir()
	// No files written — both dirs are empty

	cfg, err := Load(userDir, localDir)
	require.NoError(t, err)

	// Should return defaults
	assert.Equal(t, "sqlite", cfg.Store.Type)
	assert.Equal(t, 30, cfg.Store.RetentionDays)
	assert.Equal(t, 1000, cfg.View.RefreshMs)
	assert.True(t, cfg.View.ShowIdle)
	assert.Equal(t, "source", cfg.View.GroupBy)
	assert.Empty(t, cfg.Sources)
}

func TestPollIntervalTooLow(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
sources:
  - type: claude-code
    pollIntervalMs: 100
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := LoadFile(cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pollIntervalMs")
}

func TestRetentionDaysTooLow(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
store:
  retentionDays: 0
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := LoadFile(cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retentionDays")
}

func TestEnvVarExpansionInAllStringFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	t.Setenv("MY_GATEWAY", "ws://expanded:9090")
	t.Setenv("MY_TOKEN", "expanded-token")

	yaml := `
sources:
  - type: openclaw
    gateway: ${MY_GATEWAY}
    token: ${MY_TOKEN}
    pollIntervalMs: 500
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := LoadFile(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "ws://expanded:9090", cfg.Sources[0].Gateway)
	assert.Equal(t, "expanded-token", cfg.Sources[0].Token)
}

func TestValidSourceTypes(t *testing.T) {
	validTypes := []string{"claude-code", "openclaw", "codex", "file-watch"}
	for _, st := range validTypes {
		t.Run(st, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.yaml")

			yaml := "sources:\n  - type: " + st + "\n    pollIntervalMs: 500\n"
			require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

			_, err := LoadFile(cfgPath)
			assert.NoError(t, err)
		})
	}
}
