package claudecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProcessList_FindsClaude(t *testing.T) {
	psOutput := []byte(`USER       PID  %CPU %MEM      VSZ    RSS   TT  STAT STARTED      TIME COMMAND
c-3po     1234   0.5  1.2  1234567  12345 s000  S    10:00AM   0:01.23 /usr/local/bin/claude --session abc
c-3po     5678   0.1  0.3   234567   3456 s001  S    10:01AM   0:00.45 node /path/to/claude-code/server.js
c-3po     9999   0.0  0.1   123456   1234 s002  S    10:02AM   0:00.01 vim file.txt
`)

	procs := parseProcessList(psOutput)
	assert.Len(t, procs, 2)
	assert.Equal(t, "1234", procs[0].PID)
	assert.Contains(t, procs[0].Command, "claude")
	assert.Equal(t, "5678", procs[1].PID)
	assert.Contains(t, procs[1].Command, "claude-code")
}

func TestParseProcessList_IgnoresGrep(t *testing.T) {
	psOutput := []byte(`USER       PID  %CPU %MEM      VSZ    RSS   TT  STAT STARTED      TIME COMMAND
c-3po     1111   0.0  0.0    12345   1234 s000  S    10:00AM   0:00.00 grep -E claude|claude-code
c-3po     2222   0.5  1.2  1234567  12345 s001  S    10:00AM   0:01.23 /usr/local/bin/claude --session xyz
`)

	procs := parseProcessList(psOutput)
	assert.Len(t, procs, 1)
	assert.Equal(t, "2222", procs[0].PID)
}

func TestParseProcessList_ShortLineNoPanic(t *testing.T) {
	// Lines with "claude" but fewer than 11 fields must not panic.
	psOutput := []byte(`USER       PID  %CPU %MEM      VSZ    RSS   TT  STAT STARTED      TIME COMMAND
claude 1234
claude 5678 0.1 0.2 extra
`)

	procs := parseProcessList(psOutput)
	assert.Empty(t, procs)
}

func TestParseProcessList_Empty(t *testing.T) {
	psOutput := []byte(`USER       PID  %CPU %MEM      VSZ    RSS   TT  STAT STARTED      TIME COMMAND
c-3po     9999   0.0  0.1   123456   1234 s002  S    10:02AM   0:00.01 vim file.txt
`)

	procs := parseProcessList(psOutput)
	assert.Empty(t, procs)
}
