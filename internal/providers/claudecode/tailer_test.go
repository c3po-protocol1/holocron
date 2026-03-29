package claudecode

import (
	"testing"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSONLLine_UserMessage(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"hello"},"cwd":"/home/dev/project","sessionId":"sess-1","version":"2.1.71","gitBranch":"main"}`)
	event := ParseJSONLLine(line, "fallback-id", "/fallback/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventMessage, event.Event)
	assert.Equal(t, collector.StatusThinking, event.Status)
	assert.Equal(t, sourceName, event.Source)
	assert.Equal(t, "sess-1", event.SessionID)
	assert.Equal(t, "/home/dev/project", event.Workspace)
	assert.Equal(t, "main", event.Labels["git_branch"])
	assert.Equal(t, "2.1.71", event.Labels["claude_version"])
	assert.NotEmpty(t, event.ID)
	assert.NotZero(t, event.Timestamp)
}

func TestParseJSONLLine_AssistantMessage(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":"I'll help"},"sessionId":"sess-2"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventMessage, event.Event)
	assert.Equal(t, collector.StatusIdle, event.Status)
	assert.Equal(t, "sess-2", event.SessionID)
	assert.Equal(t, "/workspace", event.Workspace) // falls back to provided workspace
}

func TestParseJSONLLine_ToolUse(t *testing.T) {
	line := []byte(`{"type":"tool_use","name":"Edit","input":{"file_path":"src/index.ts"},"sessionId":"sess-3"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventToolStart, event.Event)
	assert.Equal(t, collector.StatusToolRunning, event.Status)
	require.NotNil(t, event.Detail)
	assert.Equal(t, "Edit", event.Detail.Tool)
}

func TestParseJSONLLine_ToolResult(t *testing.T) {
	line := []byte(`{"type":"tool_result","content":"file edited","sessionId":"sess-4"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventToolEnd, event.Event)
	assert.Equal(t, collector.StatusThinking, event.Status)
}

func TestParseJSONLLine_QueueOperation(t *testing.T) {
	line := []byte(`{"type":"queue-operation","operation":"enqueue","timestamp":"2025-01-01T00:00:00Z","sessionId":"sess-5"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	assert.Nil(t, event, "queue-operation should be ignored")
}

func TestParseJSONLLine_InvalidJSON(t *testing.T) {
	event := ParseJSONLLine([]byte(`not json`), "id", "/ws")
	assert.Nil(t, event)
}

func TestParseJSONLLine_EmptyLine(t *testing.T) {
	event := ParseJSONLLine([]byte(``), "id", "/ws")
	assert.Nil(t, event)
}

func TestParseJSONLLine_FallbackSessionID(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"hi"}}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, "fallback-id", event.SessionID)
}

func TestParseJSONLLine_ChannelLocalAlwaysSet(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":"ok"},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Labels)
	assert.Equal(t, "local", event.Labels["channel"], "claude-code events must have channel=local")
}

func TestParseJSONLLine_ChannelLocalWithMetadata(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"hi"},"sessionId":"s1","version":"2.1","gitBranch":"main"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	assert.Equal(t, "local", event.Labels["channel"])
	assert.Equal(t, "main", event.Labels["git_branch"])
	assert.Equal(t, "2.1", event.Labels["claude_version"])
}
