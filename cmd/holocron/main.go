package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/c3po-protocol1/holocron/internal/cli"
	"github.com/c3po-protocol1/holocron/internal/config"
	"github.com/c3po-protocol1/holocron/internal/store/sqlite"
	"github.com/c3po-protocol1/holocron/internal/tui"
)

var version = "0.1.0"

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

func runStatus(jsonOutput bool, source string, activeOnly bool) error {
	_, st, err := loadConfigAndStore()
	if err != nil {
		return err
	}
	defer st.Close()

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
	_, st, err := loadConfigAndStore()
	if err != nil {
		return err
	}
	defer st.Close()

	sessions, err := st.ListSessions()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	// TUI receives a nil event channel — no live provider updates in this mode yet.
	// When collector/providers are wired, this will become a real channel.
	model := tui.New(nil, sessions)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI: %w", err)
	}

	return nil
}
