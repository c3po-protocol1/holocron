package claudecode

import (
	"strings"
	"testing"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSONLLine_UserMessage(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":"hello"},"cwd":"/home/dev/project","sessionId":"sess-1","version":"2.1.71","gitBranch":"main"}`)
	event := ParseJSONLLine(line, "fallback-id", "/fallback/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventUserMessage, event.Event)
	assert.Equal(t, collector.StatusThinking, event.Status)
	assert.Equal(t, sourceName, event.Source)
	assert.Equal(t, "sess-1", event.SessionID)
	assert.Equal(t, "/home/dev/project", event.Workspace)
	assert.Equal(t, "main", event.Labels["git_branch"])
	assert.Equal(t, "2.1.71", event.Labels["claude_version"])
	assert.NotEmpty(t, event.ID)
	assert.NotZero(t, event.Timestamp)

	// Rich content fields
	require.NotNil(t, event.Detail)
	assert.Equal(t, "hello", event.Detail.Content)
	assert.Equal(t, "hello", event.Detail.Message)
	assert.Equal(t, "user", event.Detail.Role)
}

func TestParseJSONLLine_UserMessage_ContentBlockArray(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"first"},{"type":"text","text":"second"}]},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventUserMessage, event.Event)
	require.NotNil(t, event.Detail)
	assert.Contains(t, event.Detail.Content, "first")
	assert.Contains(t, event.Detail.Content, "second")
	assert.Equal(t, "user", event.Detail.Role)
}

func TestParseJSONLLine_UserMessage_EmptyContent(t *testing.T) {
	line := []byte(`{"type":"user","message":{"role":"user","content":""},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")
	assert.Nil(t, event, "empty user messages should be skipped")
}

func TestParseJSONLLine_AssistantMessage(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll help you"}]},"sessionId":"sess-2"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventAssistantMessage, event.Event)
	assert.Equal(t, collector.StatusIdle, event.Status)
	assert.Equal(t, "sess-2", event.SessionID)
	assert.Equal(t, "/workspace", event.Workspace)

	require.NotNil(t, event.Detail)
	assert.Equal(t, "I'll help you", event.Detail.Content)
	assert.Equal(t, "I'll help you", event.Detail.Message)
	assert.Equal(t, "assistant", event.Detail.Role)
}

func TestParseJSONLLine_AssistantMessage_SkipsThinking(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"internal reasoning"},{"type":"text","text":"visible output"}]},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Detail)
	assert.Equal(t, "visible output", event.Detail.Content)
	assert.NotContains(t, event.Detail.Content, "internal reasoning")
}

func TestParseJSONLLine_AssistantMessage_WithTokenUsage(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"response"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":200}},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Detail)
	require.NotNil(t, event.Detail.TokenUsage)
	assert.Equal(t, int64(100), event.Detail.TokenUsage.Input)
	assert.Equal(t, int64(50), event.Detail.TokenUsage.Output)
	assert.Equal(t, int64(200), event.Detail.TokenUsage.CacheRead)
}

func TestParseJSONLLine_ToolUse(t *testing.T) {
	line := []byte(`{"type":"tool_use","name":"Edit","input":{"file_path":"src/index.ts","old_string":"foo","new_string":"bar"},"sessionId":"sess-3"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventToolStart, event.Event)
	assert.Equal(t, collector.StatusToolRunning, event.Status)
	require.NotNil(t, event.Detail)
	assert.Equal(t, "Edit", event.Detail.Tool)
	assert.Equal(t, "src/index.ts", event.Detail.Target)
	assert.NotEmpty(t, event.Detail.ToolInput)
}

func TestParseJSONLLine_ToolResult(t *testing.T) {
	line := []byte(`{"type":"tool_result","content":"file edited successfully","sessionId":"sess-4"}`)
	event := ParseJSONLLine(line, "fallback-id", "/workspace")

	require.NotNil(t, event)
	assert.Equal(t, collector.EventToolResult, event.Event)
	assert.Equal(t, collector.StatusThinking, event.Status)
	require.NotNil(t, event.Detail)
	assert.Equal(t, "file edited successfully", event.Detail.ToolOutput)
	assert.Equal(t, "tool", event.Detail.Role)
}

func TestParseJSONLLine_ToolResult_ContentBlocks(t *testing.T) {
	line := []byte(`{"type":"tool_result","content":[{"type":"text","text":"output line 1"},{"type":"text","text":"output line 2"}],"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Detail)
	assert.Contains(t, event.Detail.ToolOutput, "output line 1")
	assert.Contains(t, event.Detail.ToolOutput, "output line 2")
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
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]},"sessionId":"s1"}`)
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

func TestParseJSONLLine_MessageTruncatedAt200(t *testing.T) {
	longText := strings.Repeat("x", 300)
	line := []byte(`{"type":"user","message":{"role":"user","content":"` + longText + `"},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Detail)
	assert.LessOrEqual(t, len(event.Detail.Message), 203) // 200 + "..."
	assert.True(t, strings.HasSuffix(event.Detail.Message, "..."))
	assert.Equal(t, longText, event.Detail.Content) // Content not truncated (under 32KB)
}

func TestParseJSONLLine_ContentTruncatedAt32KB(t *testing.T) {
	longText := strings.Repeat("y", 40*1024) // 40KB
	line := []byte(`{"type":"user","message":{"role":"user","content":"` + longText + `"},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Detail)
	assert.True(t, strings.HasSuffix(event.Detail.Content, "\n[...truncated at 32KB]"))
	assert.LessOrEqual(t, len(event.Detail.Content), maxContentSize+len("\n[...truncated at 32KB]"))
}

func TestParseJSONLLine_ToolInputTruncatedAt32KB(t *testing.T) {
	longValue := strings.Repeat("z", 40*1024)
	line := []byte(`{"type":"tool_use","name":"Write","input":{"file_path":"big.txt","content":"` + longValue + `"},"sessionId":"s1"}`)
	event := ParseJSONLLine(line, "s1", "/ws")

	require.NotNil(t, event)
	require.NotNil(t, event.Detail)
	assert.True(t, strings.HasSuffix(event.Detail.ToolInput, "\n[...truncated at 32KB]"))
}
