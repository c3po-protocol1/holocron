package claudecode

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

// ProcessInfo represents a detected Claude Code process.
type ProcessInfo struct {
	PID     string
	Command string
}

// DetectProcesses runs ps and returns any Claude Code processes found.
func DetectProcesses() ([]ProcessInfo, error) {
	return detectProcessesFromCmd(exec.Command("ps", "aux"))
}

func detectProcessesFromCmd(cmd *exec.Cmd) ([]ProcessInfo, error) {
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseProcessList(out), nil
}

// parseProcessList parses ps aux output and returns Claude Code processes.
func parseProcessList(output []byte) []ProcessInfo {
	var procs []ProcessInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		lower := strings.ToLower(line)
		// Match claude or claude-code but skip grep itself
		if (strings.Contains(lower, "claude") || strings.Contains(lower, "claude-code")) &&
			!strings.Contains(lower, "grep") {
			fields := strings.Fields(line)
			if len(fields) >= 11 {
				procs = append(procs, ProcessInfo{
					PID:     fields[1],
					Command: strings.Join(fields[10:], " "),
				})
			}
		}
	}
	return procs
}
