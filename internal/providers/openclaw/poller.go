package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ParseStatusOutput parses the JSON output from `openclaw gateway call status --json`.
func ParseStatusOutput(data []byte) ([]OCSession, error) {
	var resp StatusResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing status output: %w", err)
	}

	var sessions []OCSession
	for _, agent := range resp.Sessions.ByAgent {
		sessions = append(sessions, agent.Recent...)
	}
	return sessions, nil
}

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	Run(ctx context.Context) ([]byte, error)
}

// ExecRunner runs `openclaw gateway call status --json` via os/exec.
type ExecRunner struct{}

// Run executes the openclaw status command and returns stdout.
func (r *ExecRunner) Run(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "openclaw", "gateway", "call", "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("openclaw status command: %w", err)
	}
	return out, nil
}

// Poll executes the command and parses the response into sessions.
func Poll(ctx context.Context, runner CommandRunner) ([]OCSession, error) {
	data, err := runner.Run(ctx)
	if err != nil {
		return nil, err
	}
	return ParseStatusOutput(data)
}
