package claudecode

import (
	"encoding/json"
	"strings"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

const (
	maxContentSize   = 32 * 1024 // 32 KB
	maxMessageSize   = 200
	truncationSuffix = "\n[...truncated at 32KB]"
)

// truncateContent truncates s to maxLen. For maxMessageSize, appends "...".
// For larger limits (32KB), appends truncationSuffix.
func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen == maxMessageSize {
		return s[:maxLen] + "..."
	}
	cut := maxLen - len(truncationSuffix)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + truncationSuffix
}

// extractTextContent handles both plain string and content-block array formats.
// It returns the concatenated text of all "text" blocks.
func extractTextContent(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// Try as plain string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try as array of content blocks.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(b.Text)
			}
		}
		return sb.String()
	}
	return ""
}

// extractAssistantText extracts text blocks from assistant message content,
// skipping "thinking" blocks.
func extractAssistantText(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(b.Text)
			}
			// Explicitly skip "thinking" and other block types.
		}
		return sb.String()
	}
	// Fallback: try plain string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

// extractTokenUsage parses a Claude API usage object into a TokenUsage.
// Returns nil if usage is zero or unparseable.
func extractTokenUsage(raw json.RawMessage) *collector.TokenUsage {
	if raw == nil {
		return nil
	}
	var u struct {
		InputTokens          int64 `json:"input_tokens"`
		OutputTokens         int64 `json:"output_tokens"`
		CacheReadInputTokens int64 `json:"cache_read_input_tokens"`
	}
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil
	}
	if u.InputTokens == 0 && u.OutputTokens == 0 {
		return nil
	}
	return &collector.TokenUsage{
		Input:     u.InputTokens,
		Output:    u.OutputTokens,
		CacheRead: u.CacheReadInputTokens,
	}
}

// extractToolInput returns a formatted (pretty-printed) JSON string of the tool's input.
func extractToolInput(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}

// extractToolTarget extracts a human-readable target (file path, command, URL)
// from tool input based on the tool name.
func extractToolTarget(name string, input json.RawMessage) string {
	if input == nil {
		return ""
	}
	var params map[string]interface{}
	if err := json.Unmarshal(input, &params); err != nil {
		return ""
	}

	switch name {
	case "Read", "Write", "Edit", "MultiEdit", "NotebookEdit", "Glob":
		if v, ok := params["file_path"].(string); ok && v != "" {
			return v
		}
		if v, ok := params["path"].(string); ok && v != "" {
			return v
		}
	case "Bash":
		if v, ok := params["command"].(string); ok && v != "" {
			if len(v) > 100 {
				return v[:100] + "..."
			}
			return v
		}
	case "WebFetch":
		if v, ok := params["url"].(string); ok && v != "" {
			return v
		}
	case "WebSearch":
		if v, ok := params["query"].(string); ok && v != "" {
			return v
		}
	}

	// Generic fallback: look for common target keys in order.
	for _, key := range []string{"file_path", "path", "command", "url", "query"} {
		if v, ok := params[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// extractToolResultContent extracts the text content from a tool result,
// which can be a plain string or an array of content blocks.
func extractToolResultContent(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// Try plain string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try array of content blocks.
	var blocks []struct {
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var sb strings.Builder
		for _, b := range blocks {
			switch b.Type {
			case "text":
				if b.Text != "" {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(b.Text)
				}
			default:
				// Binary or unknown block type.
				if len(b.Content) > 0 {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString("[binary content, " + itoa(len(b.Content)) + " bytes]")
				}
			}
		}
		return sb.String()
	}
	return ""
}

// extractUserContent extracts text content from a user message envelope.
// The message field is {"role":"user","content":...} where content is string or content-block array.
func extractUserContent(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return ""
	}
	return extractTextContent(msg.Content)
}

// extractAssistantContent extracts text from an assistant message envelope,
// skipping thinking blocks.
func extractAssistantContent(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return ""
	}
	return extractAssistantText(msg.Content)
}

// extractAssistantTokenUsage extracts token usage from an assistant message envelope.
func extractAssistantTokenUsage(raw json.RawMessage) *collector.TokenUsage {
	if raw == nil {
		return nil
	}
	var msg struct {
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}
	return extractTokenUsage(msg.Usage)
}

// itoa converts an int to its decimal string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
