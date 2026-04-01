package collector

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventTypeConstants(t *testing.T) {
	assert.Equal(t, EventType("user.message"), EventUserMessage)
	assert.Equal(t, EventType("assistant.message"), EventAssistantMessage)
	assert.Equal(t, EventType("tool.result"), EventToolResult)
}

func TestEventDetailNewFieldsSerialization(t *testing.T) {
	d := EventDetail{
		Tool:       "Read",
		Target:     "main.go",
		Message:    "reading file",
		Content:    "full file content here",
		ToolInput:  `{"file_path": "main.go"}`,
		ToolOutput: "package main\n\nfunc main() {}",
		Role:       "user",
	}

	b, err := json.Marshal(d)
	require.NoError(t, err)

	var got EventDetail
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, "Read", got.Tool)
	assert.Equal(t, "main.go", got.Target)
	assert.Equal(t, "reading file", got.Message)
	assert.Equal(t, "full file content here", got.Content)
	assert.Equal(t, `{"file_path": "main.go"}`, got.ToolInput)
	assert.Equal(t, "package main\n\nfunc main() {}", got.ToolOutput)
	assert.Equal(t, "user", got.Role)
}

func TestEventDetailOmitsEmptyNewFields(t *testing.T) {
	d := EventDetail{Tool: "Read"}
	b, err := json.Marshal(d)
	require.NoError(t, err)

	// Content, ToolInput, ToolOutput, Role should be omitted when empty
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))

	assert.NotContains(t, m, "content")
	assert.NotContains(t, m, "toolInput")
	assert.NotContains(t, m, "toolOutput")
	assert.NotContains(t, m, "role")
}
