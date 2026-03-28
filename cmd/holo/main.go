package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/c3po-protocol1/holocron/internal/cli"
	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/c3po-protocol1/holocron/internal/config"
	claudecode "github.com/c3po-protocol1/holocron/internal/providers/claudecode"
	"github.com/c3po-protocol1/holocron/internal/store/sqlite"
	"github.com/c3po-protocol1/holocron/internal/tui"
)

var version = "0.2.0"

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "holo",
		Short: "Holocron — AI session monitor",
		Long:  "Holocron monitors AI coding sessions (Claude Code, OpenClaw, Codex) in a terminal UI.",
		RunE:  runTUI,
		// Silence cobra's default error/usage printing — we handle it
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(statusCmd())
	root.AddCommand(versionCmd())

	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("holocron %s\n", version)
		},
	}
}

func statusCmd() *cobra.Command {
	var (
		jsonOutput bool
		source     string
		activeOnly bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show one-shot session summary",
		Long:  "Print a summary of all current sessions and exit. No interactive UI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(jsonOutput, source, activeOnly)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source type")
	cmd.Flags().BoolVar(&activeOnly, "active", false, "Show only active sessions")

	return cmd
}

func loadConfigAndStore() (config.Config, *sqlite.SQLiteStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("home dir: %w", err)
	}

	userDir := filepath.Join(home, ".holocron")
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cfg, err := config.Load(userDir, cwd)
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("config: %w", err)
	}

	st, err := sqlite.New(cfg.Store.Path)
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("store: %w", err)
	}

	return cfg, st, nil
}

func defaultClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

func pollDuration(src config.SourceConfig) time.Duration {
	if src.PollIntervalMs > 0 {
		return time.Duration(src.PollIntervalMs) * time.Millisecond
	}
	return 0 // provider uses its own default
}

func buildCollector(cfg config.Config, st collector.Store) *collector.Collector {
	c := collector.New(st)
	for _, src := range cfg.Sources {
		if src.Type == "claude-code" {
			dir := config.ExpandTilde(src.SessionDir)
			if dir == "" {
				dir = defaultClaudeDir()
			}
			c.AddProvider(claudecode.New(dir, pollDuration(src)))
		}
	}
	return c
}

func runStatus(jsonOutput bool, source string, activeOnly bool) error {
	cfg, st, err := loadConfigAndStore()
	if err != nil {
		return err
	}
	defer st.Close()

	// Start providers briefly for initial discovery
	c := buildCollector(cfg, st)
	ctx, cancel := context.WithCancel(context.Background())
	if err := c.Start(ctx); err != nil {
		cancel()
		return fmt.Errorf("collector: %w", err)
	}

	// Allow initial scan to complete
	time.Sleep(500 * time.Millisecond)
	cancel()
	c.Stop()

	sessions, err := st.ListSessions()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	sessions = cli.FilterSessions(sessions, activeOnly, source)

	if jsonOutput {
		output, err := cli.FormatStatusJSON(sessions)
		if err != nil {
			return fmt.Errorf("formatting JSON: %w", err)
		}
		fmt.Print(output)
	} else {
		fmt.Println(cli.FormatStatus(sessions, time.Now()))
	}

	return nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, st, err := loadConfigAndStore()
	if err != nil {
		return err
	}
	defer st.Close()

	c := buildCollector(cfg, st)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.Start(ctx); err != nil {
		return fmt.Errorf("collector: %w", err)
	}
	defer c.Stop()

	sessions, err := st.ListSessions()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	model := tui.New(c.Subscribe(), sessions)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI: %w", err)
	}

	return nil
}
