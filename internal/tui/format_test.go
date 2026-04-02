package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// --- eventIcon tests ---

func TestEventIcon(t *testing.T) {
	tests := []struct {
		event collector.EventType
		icon  string
	}{
		{collector.EventUserMessage, "👤"},
		{collector.EventAssistantMessage, "🤖"},
		{collector.EventToolStart, "🔧"},
		{collector.EventToolResult, "✅"},
		{collector.EventToolEnd, "✅"},
		{collector.EventMessage, "○"},
		{collector.EventStatusChange, "◌"},
		{collector.EventSessionStart, "▶"},
		{collector.EventSessionEnd, "■"},
		{collector.EventError, "✕"},
		{collector.EventType("unknown"), "○"},
	}
	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			assert.Equal(t, tt.icon, eventIcon(tt.event))
		})
	}
}

// --- eventLabel tests ---

func TestEventLabel(t *testing.T) {
	tests := []struct {
		event collector.EventType
		label string
	}{
		{collector.EventUserMessage, "USER"},
		{collector.EventAssistantMessage, "ASSISTANT"},
		{collector.EventToolStart, "TOOL"},
		{collector.EventToolResult, "RESULT"},
		{collector.EventToolEnd, "RESULT"},
		{collector.EventMessage, "MESSAGE"},
		{collector.EventStatusChange, "STATUS"},
		{collector.EventSessionStart, "SESSION START"},
		{collector.EventSessionEnd, "SESSION END"},
		{collector.EventError, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			assert.Equal(t, tt.label, eventLabel(tt.event))
		})
	}
}

// --- compactSummary tests ---

func TestCompactSummary_UserMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventUserMessage,
		Detail: &collector.EventDetail{Content: "Fix the auth bug", Message: "fallback msg"},
	}
	assert.Equal(t, "Fix the auth bug", compactSummary(ev))
}

func TestCompactSummary_UserMessage_FallbackToMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventUserMessage,
		Detail: &collector.EventDetail{Message: "fallback msg"},
	}
	assert.Equal(t, "fallback msg", compactSummary(ev))
}

func TestCompactSummary_AssistantMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventAssistantMessage,
		Detail: &collector.EventDetail{Content: "Looking at the auth handler..."},
	}
	assert.Equal(t, "Looking at the auth handler...", compactSummary(ev))
}

func TestCompactSummary_ToolStart(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolStart,
		Detail: &collector.EventDetail{Tool: "Read", Target: "src/login.go"},
	}
	assert.Equal(t, "Read → src/login.go", compactSummary(ev))
}

func TestCompactSummary_ToolEnd(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolEnd,
		Detail: &collector.EventDetail{Tool: "Read", Target: "src/login.go", Message: "247 lines"},
	}
	assert.Equal(t, "Read → src/login.go (247 lines)", compactSummary(ev))
}

func TestCompactSummary_ToolResult(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolResult,
		Detail: &collector.EventDetail{Tool: "Edit", Target: "foo.go", Message: "ok"},
	}
	assert.Equal(t, "Edit → foo.go (ok)", compactSummary(ev))
}

func TestCompactSummary_StatusChange(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventStatusChange,
		Status: collector.StatusThinking,
	}
	assert.Equal(t, "thinking", compactSummary(ev))
}

func TestCompactSummary_Error(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventError,
		Detail: &collector.EventDetail{Message: "connection lost"},
	}
	assert.Equal(t, "connection lost", compactSummary(ev))
}

func TestCompactSummary_NilDetail(t *testing.T) {
	ev := collector.MonitorEvent{
		Event: collector.EventMessage,
	}
	assert.Equal(t, "", compactSummary(ev))
}

// --- formatEventCompact tests ---

func TestFormatEventCompact_ContainsIconAndSummary(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000, // some timestamp
		Event:     collector.EventUserMessage,
		Detail:    &collector.EventDetail{Content: "Fix the bug"},
	}
	result := formatEventCompact(ev)
	assert.Contains(t, result, "👤")
	assert.Contains(t, result, "user.message")
	assert.Contains(t, result, "Fix the bug")
}

func TestFormatEventCompact_ToolStart(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000,
		Event:     collector.EventToolStart,
		Detail:    &collector.EventDetail{Tool: "Read", Target: "main.go"},
	}
	result := formatEventCompact(ev)
	assert.Contains(t, result, "🔧")
	assert.Contains(t, result, "tool.start")
	assert.Contains(t, result, "Read → main.go")
}

// --- formatEventVerbose tests ---

func TestFormatEventVerbose_UserMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000,
		Event:     collector.EventUserMessage,
		Detail:    &collector.EventDetail{Content: "Fix the auth bug in login.go"},
	}
	result := formatEventVerbose(ev, 60)
	assert.Contains(t, result, "👤")
	assert.Contains(t, result, "USER")
	assert.Contains(t, result, "───")
	assert.Contains(t, result, "Fix the auth bug in login.go")
}

func TestFormatEventVerbose_AssistantMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000,
		Event:     collector.EventAssistantMessage,
		Detail:    &collector.EventDetail{Content: "Looking at the auth handler..."},
	}
	result := formatEventVerbose(ev, 60)
	assert.Contains(t, result, "🤖")
	assert.Contains(t, result, "ASSISTANT")
	assert.Contains(t, result, "Looking at the auth handler...")
}

func TestFormatEventVerbose_ToolStart(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000,
		Event:     collector.EventToolStart,
		Detail:    &collector.EventDetail{Tool: "Read", Target: "src/login.go", ToolInput: "some input"},
	}
	result := formatEventVerbose(ev, 60)
	assert.Contains(t, result, "🔧")
	assert.Contains(t, result, "TOOL: Read")
	assert.Contains(t, result, "Target: src/login.go")
	assert.Contains(t, result, "some input")
}

func TestFormatEventVerbose_ToolResult(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000,
		Event:     collector.EventToolResult,
		Detail:    &collector.EventDetail{Tool: "Read", ToolOutput: "file contents here"},
	}
	result := formatEventVerbose(ev, 60)
	assert.Contains(t, result, "✅")
	assert.Contains(t, result, "RESULT: Read")
	assert.Contains(t, result, "file contents here")
}

func TestFormatEventVerbose_NoContent(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: 1711666800000,
		Event:     collector.EventStatusChange,
		Status:    collector.StatusThinking,
	}
	result := formatEventVerbose(ev, 60)
	assert.Contains(t, result, "◌")
	assert.Contains(t, result, "STATUS")
}

// --- verboseContent tests ---

func TestVerboseContent_UserMessage_PrefersContent(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventUserMessage,
		Detail: &collector.EventDetail{Content: "content", Message: "msg"},
	}
	assert.Equal(t, "content", verboseContent(ev))
}

func TestVerboseContent_UserMessage_FallsBackToMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventUserMessage,
		Detail: &collector.EventDetail{Message: "msg"},
	}
	assert.Equal(t, "msg", verboseContent(ev))
}

func TestVerboseContent_AssistantMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventAssistantMessage,
		Detail: &collector.EventDetail{Content: "assistant content"},
	}
	assert.Equal(t, "assistant content", verboseContent(ev))
}

func TestVerboseContent_ToolStart(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolStart,
		Detail: &collector.EventDetail{Target: "file.go", ToolInput: "input data"},
	}
	result := verboseContent(ev)
	assert.Contains(t, result, "Target: file.go")
	assert.Contains(t, result, "input data")
}

func TestVerboseContent_ToolStart_NoInput(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolStart,
		Detail: &collector.EventDetail{Target: "file.go"},
	}
	result := verboseContent(ev)
	assert.Equal(t, "Target: file.go", result)
}

func TestVerboseContent_ToolResult_PrefersToolOutput(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolResult,
		Detail: &collector.EventDetail{ToolOutput: "output", Message: "msg"},
	}
	assert.Equal(t, "output", verboseContent(ev))
}

func TestVerboseContent_ToolResult_FallsBackToMessage(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventToolResult,
		Detail: &collector.EventDetail{Message: "msg"},
	}
	assert.Equal(t, "msg", verboseContent(ev))
}

func TestVerboseContent_NilDetail(t *testing.T) {
	ev := collector.MonitorEvent{Event: collector.EventMessage}
	assert.Equal(t, "", verboseContent(ev))
}

func TestVerboseContent_Default(t *testing.T) {
	ev := collector.MonitorEvent{
		Event:  collector.EventSessionStart,
		Detail: &collector.EventDetail{Message: "session started"},
	}
	assert.Equal(t, "session started", verboseContent(ev))
}

// --- wordWrap tests ---

func TestWordWrap_ShortText(t *testing.T) {
	assert.Equal(t, "hello world", wordWrap("hello world", 40))
}

func TestWordWrap_ExactWidth(t *testing.T) {
	text := "hello world"
	assert.Equal(t, text, wordWrap(text, len(text)))
}

func TestWordWrap_WrapsAtWord(t *testing.T) {
	result := wordWrap("hello world foo bar", 12)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 2, len(lines))
	assert.Equal(t, "hello world", lines[0])
	assert.Equal(t, "foo bar", lines[1])
}

func TestWordWrap_LongWord(t *testing.T) {
	result := wordWrap("superlongwordthatexceedswidth", 10)
	// Long words that exceed width should be kept intact (not broken mid-word)
	assert.Contains(t, result, "superlongwordthatexceedswidth")
}

func TestWordWrap_MultipleLines(t *testing.T) {
	result := wordWrap("aaa bbb ccc ddd eee", 8)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		// Each line (excluding long single words) should be <= width
		assert.LessOrEqual(t, len(line), 8, "line too long: %q", line)
	}
}

func TestWordWrap_EmptyString(t *testing.T) {
	assert.Equal(t, "", wordWrap("", 40))
}

func TestWordWrap_PreservesExistingNewlines(t *testing.T) {
	result := wordWrap("line one\nline two", 40)
	assert.Contains(t, result, "line one\nline two")
}

func TestWordWrap_WidthZero(t *testing.T) {
	// Should not panic
	result := wordWrap("hello", 0)
	assert.NotEmpty(t, result)
}
