package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sources []SourceConfig `yaml:"sources"`
	Store   StoreConfig    `yaml:"store"`
	View    ViewConfig     `yaml:"view"`
	Labels  LabelsConfig   `yaml:"labels"`
}

type SourceConfig struct {
	Type           string `yaml:"type"`
	Discover       string `yaml:"discover"`
	SessionDir     string `yaml:"sessionDir"`
	WatchProcesses bool   `yaml:"watchProcesses"`
	TailActive     bool   `yaml:"tailActive"`
	PollIntervalMs int    `yaml:"pollIntervalMs"`
	Gateway        string `yaml:"gateway"`
	Token          string `yaml:"token"`
	Path           string `yaml:"path"`
	Format         string `yaml:"format"`
}

type StoreConfig struct {
	Type          string `yaml:"type"`
	Path          string `yaml:"path"`
	RetentionDays int    `yaml:"retentionDays"`
}

type ViewConfig struct {
	RefreshMs int    `yaml:"refreshMs"`
	ShowIdle  bool   `yaml:"showIdle"`
	GroupBy   string `yaml:"groupBy"`
}

type LabelsConfig struct {
	Rules []LabelRule `yaml:"rules"`
}

type LabelRule struct {
	Match map[string]string `yaml:"match"`
	Set   map[string]string `yaml:"set"`
}

var validSourceTypes = map[string]bool{
	"claude-code": true,
	"openclaw":    true,
	"codex":       true,
	"file-watch":  true,
}

var envVarRe = regexp.MustCompile(`\$\{([^}]+)\}`)

// Defaults returns a Config with all default values applied.
func Defaults() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Store: StoreConfig{
			Type:          "sqlite",
			Path:          filepath.Join(home, ".holocron", "holocron.db"),
			RetentionDays: 30,
		},
		View: ViewConfig{
			RefreshMs: 1000,
			ShowIdle:  true,
			GroupBy:   "source",
		},
	}
}

// Load loads and merges config from userDir/config.yaml and localDir/holocron.yaml.
func Load(userDir, localDir string) (Config, error) {
	cfg := Defaults()

	userPath := filepath.Join(userDir, "config.yaml")
	if _, err := os.Stat(userPath); err == nil {
		userCfg, err := parseFile(userPath)
		if err != nil {
			return Config{}, fmt.Errorf("user config: %w", err)
		}
		cfg = merge(cfg, userCfg)
	}

	localPath := filepath.Join(localDir, "holocron.yaml")
	if _, err := os.Stat(localPath); err == nil {
		localCfg, err := parseFile(localPath)
		if err != nil {
			return Config{}, fmt.Errorf("local config: %w", err)
		}
		cfg = merge(cfg, localCfg)
	}

	expandEnvVars(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// LoadFile loads config from a single file, applying defaults.
func LoadFile(path string) (Config, error) {
	cfg := Defaults()

	parsed, err := parseFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg = merge(cfg, parsed)
	expandEnvVars(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// parsedConfig mirrors Config but uses pointers to detect which fields were set.
type parsedConfig struct {
	Sources []SourceConfig `yaml:"sources"`
	Store   parsedStore    `yaml:"store"`
	View    parsedView     `yaml:"view"`
	Labels  LabelsConfig   `yaml:"labels"`
}

type parsedStore struct {
	Type          *string `yaml:"type"`
	Path          *string `yaml:"path"`
	RetentionDays *int    `yaml:"retentionDays"`
}

type parsedView struct {
	RefreshMs *int    `yaml:"refreshMs"`
	ShowIdle  *bool   `yaml:"showIdle"`
	GroupBy   *string `yaml:"groupBy"`
}

func parseFile(path string) (parsedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return parsedConfig{}, fmt.Errorf("reading config file: %w", err)
	}

	var p parsedConfig
	if err := yaml.Unmarshal(data, &p); err != nil {
		return parsedConfig{}, fmt.Errorf("parsing config file: %w", err)
	}

	return p, nil
}

func merge(base Config, overlay parsedConfig) Config {
	if len(overlay.Sources) > 0 {
		base.Sources = overlay.Sources
	}

	if overlay.Store.Type != nil {
		base.Store.Type = *overlay.Store.Type
	}
	if overlay.Store.Path != nil {
		base.Store.Path = *overlay.Store.Path
	}
	if overlay.Store.RetentionDays != nil {
		base.Store.RetentionDays = *overlay.Store.RetentionDays
	}

	if overlay.View.RefreshMs != nil {
		base.View.RefreshMs = *overlay.View.RefreshMs
	}
	if overlay.View.ShowIdle != nil {
		base.View.ShowIdle = *overlay.View.ShowIdle
	}
	if overlay.View.GroupBy != nil {
		base.View.GroupBy = *overlay.View.GroupBy
	}

	if len(overlay.Labels.Rules) > 0 {
		base.Labels.Rules = overlay.Labels.Rules
	}

	return base
}

func expandEnvVars(cfg *Config) {
	for i := range cfg.Sources {
		cfg.Sources[i].Token = expandString(cfg.Sources[i].Token)
		cfg.Sources[i].Gateway = expandString(cfg.Sources[i].Gateway)
		cfg.Sources[i].SessionDir = expandString(cfg.Sources[i].SessionDir)
		cfg.Sources[i].Path = expandString(cfg.Sources[i].Path)
	}
}

func expandString(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})
}

func validate(cfg Config) error {
	for i, src := range cfg.Sources {
		if !validSourceTypes[src.Type] {
			return fmt.Errorf("sources[%d]: invalid source type %q", i, src.Type)
		}
		if src.PollIntervalMs > 0 && src.PollIntervalMs < 500 {
			return fmt.Errorf("sources[%d]: pollIntervalMs must be >= 500 (got %d)", i, src.PollIntervalMs)
		}
	}

	if cfg.Store.RetentionDays < 1 {
		return fmt.Errorf("store: retentionDays must be >= 1 (got %d)", cfg.Store.RetentionDays)
	}

	return nil
}
