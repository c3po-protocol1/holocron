package claudecode

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- truncateContent ---

func TestTruncateContent_ShortString(t *testing.T) {
	assert.Equal(t, "hello", truncateContent("hello", 200))
}

func TestTruncateContent_ExactLimit(t *testing.T) {
	s := strings.Repeat("a", 200)
	assert.Equal(t, s, truncateContent(s, 200))
}

func TestTruncateContent_MessageTruncation(t *testing.T) {
	s := strings.Repeat("a", 250)
	result := truncateContent(s, maxMessageSize)
	assert.Equal(t, maxMessageSize+3, len(result)) // 200 chars + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestTruncateContent_LargeTruncation(t *testing.T) {
	s := strings.Repeat("b", maxContentSize+100)
	result := truncateContent(s, maxContentSize)
	assert.True(t, strings.HasSuffix(result, "\n[...truncated at 32KB]"))
	assert.LessOrEqual(t, len(result), maxContentSize+len("\n[...truncated at 32KB]"))
}

// --- extractTextContent ---

func TestExtractTextContent_String(t *testing.T) {
	raw := json.RawMessage(`"hello world"`)
	assert.Equal(t, "hello world", extractTextContent(raw))
}

func TestExtractTextContent_TextBlock(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"hello"},{"type":"text","text":"world"}]`)
	result := extractTextContent(raw)
	assert.Contains(t, result, "hello")
	assert.Contains(t, result, "world")
}

func TestExtractTextContent_SkipsNonText(t *testing.T) {
	raw := json.RawMessage(`[{"type":"image","source":"data"},{"type":"text","text":"visible"}]`)
	result := extractTextContent(raw)
	assert.Equal(t, "visible", result)
}

func TestExtractTextContent_Nil(t *testing.T) {
	assert.Equal(t, "", extractTextContent(nil))
}

func TestExtractTextContent_Empty(t *testing.T) {
	assert.Equal(t, "", extractTextContent(json.RawMessage(`[]`)))
}

// --- extractAssistantText ---

func TestExtractAssistantText_SkipsThinking(t *testing.T) {
	raw := json.RawMessage(`[{"type":"thinking","thinking":"internal"},{"type":"text","text":"visible"}]`)
	result := extractAssistantText(raw)
	assert.Equal(t, "visible", result)
}

func TestExtractAssistantText_MultipleTextBlocks(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"part1"},{"type":"text","text":"part2"}]`)
	result := extractAssistantText(raw)
	assert.Contains(t, result, "part1")
	assert.Contains(t, result, "part2")
}

func TestExtractAssistantText_OnlyThinking(t *testing.T) {
	raw := json.RawMessage(`[{"type":"thinking","thinking":"hidden"}]`)
	result := extractAssistantText(raw)
	assert.Equal(t, "", result)
}

func TestExtractAssistantText_StringFallback(t *testing.T) {
	raw := json.RawMessage(`"plain text response"`)
	result := extractAssistantText(raw)
	assert.Equal(t, "plain text response", result)
}

func TestExtractAssistantText_Nil(t *testing.T) {
	assert.Equal(t, "", extractAssistantText(nil))
}

// --- extractTokenUsage ---

func TestExtractTokenUsage_Basic(t *testing.T) {
	raw := json.RawMessage(`{"input_tokens":100,"output_tokens":50}`)
	tu := extractTokenUsage(raw)
	require.NotNil(t, tu)
	assert.Equal(t, int64(100), tu.Input)
	assert.Equal(t, int64(50), tu.Output)
	assert.Equal(t, int64(0), tu.CacheRead)
}

func TestExtractTokenUsage_WithCacheRead(t *testing.T) {
	raw := json.RawMessage(`{"input_tokens":200,"output_tokens":80,"cache_read_input_tokens":300}`)
	tu := extractTokenUsage(raw)
	require.NotNil(t, tu)
	assert.Equal(t, int64(200), tu.Input)
	assert.Equal(t, int64(80), tu.Output)
	assert.Equal(t, int64(300), tu.CacheRead)
}

func TestExtractTokenUsage_Nil(t *testing.T) {
	assert.Nil(t, extractTokenUsage(nil))
}

func TestExtractTokenUsage_ZeroTokens(t *testing.T) {
	raw := json.RawMessage(`{"input_tokens":0,"output_tokens":0}`)
	assert.Nil(t, extractTokenUsage(raw))
}

func TestExtractTokenUsage_Invalid(t *testing.T) {
	assert.Nil(t, extractTokenUsage(json.RawMessage(`not json`)))
}

// --- extractToolInput ---

func TestExtractToolInput_FormatsJSON(t *testing.T) {
	raw := json.RawMessage(`{"file_path":"main.go","old_string":"foo"}`)
	result := extractToolInput(raw)
	assert.NotEmpty(t, result)
	// Should be valid JSON
	var v interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &v))
}

func TestExtractToolInput_Nil(t *testing.T) {
	assert.Equal(t, "", extractToolInput(nil))
}

// --- extractToolTarget ---

func TestExtractToolTarget_ReadFilePath(t *testing.T) {
	input := json.RawMessage(`{"file_path":"src/main.go"}`)
	assert.Equal(t, "src/main.go", extractToolTarget("Read", input))
}

func TestExtractToolTarget_EditFilePath(t *testing.T) {
	input := json.RawMessage(`{"file_path":"app.py","old_string":"x","new_string":"y"}`)
	assert.Equal(t, "app.py", extractToolTarget("Edit", input))
}

func TestExtractToolTarget_BashCommand(t *testing.T) {
	input := json.RawMessage(`{"command":"go test ./..."}`)
	assert.Equal(t, "go test ./...", extractToolTarget("Bash", input))
}

func TestExtractToolTarget_BashLongCommand(t *testing.T) {
	cmd := strings.Repeat("x", 200)
	input := json.RawMessage(`{"command":"` + cmd + `"}`)
	result := extractToolTarget("Bash", input)
	assert.LessOrEqual(t, len(result), 103) // 100 + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestExtractToolTarget_UnknownTool(t *testing.T) {
	input := json.RawMessage(`{"file_path":"foo.txt"}`)
	// Falls back to generic lookup
	assert.Equal(t, "foo.txt", extractToolTarget("SomeTool", input))
}

func TestExtractToolTarget_Nil(t *testing.T) {
	assert.Equal(t, "", extractToolTarget("Read", nil))
}

// --- extractToolResultContent ---

func TestExtractToolResultContent_String(t *testing.T) {
	raw := json.RawMessage(`"file content here"`)
	assert.Equal(t, "file content here", extractToolResultContent(raw))
}

func TestExtractToolResultContent_TextBlocks(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"output line 1"},{"type":"text","text":"output line 2"}]`)
	result := extractToolResultContent(raw)
	assert.Contains(t, result, "output line 1")
	assert.Contains(t, result, "output line 2")
}

func TestExtractToolResultContent_Nil(t *testing.T) {
	assert.Equal(t, "", extractToolResultContent(nil))
}

// --- extractTokenUsage type check ---

func TestExtractTokenUsage_ReturnsCollectorType(t *testing.T) {
	raw := json.RawMessage(`{"input_tokens":10,"output_tokens":5}`)
	tu := extractTokenUsage(raw)
	require.NotNil(t, tu)
	// Verify it's the right type
	var _ *collector.TokenUsage = tu
}
