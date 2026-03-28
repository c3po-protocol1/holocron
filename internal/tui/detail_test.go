package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// --- FormatEventRow tests (T3) ---

func TestFormatEventRow_ToolStart(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 45, 1, 0, time.Local).UnixMilli(),
		Event:     collector.EventToolStart,
		Detail:    &collector.EventDetail{Tool: "Edit", Target: "src/index.ts"},
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "22:45:01")
	assert.Contains(t, row, "●")
	assert.Contains(t, row, "tool.start")
	assert.Contains(t, row, "Edit")
	assert.Contains(t, row, "src/index.ts")
}

func TestFormatEventRow_ToolEnd(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 44, 55, 0, time.Local).UnixMilli(),
		Event:     collector.EventToolEnd,
		Detail:    &collector.EventDetail{Tool: "Read", Target: "package.json"},
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "●")
	assert.Contains(t, row, "tool.end")
	assert.Contains(t, row, "Read")
}

func TestFormatEventRow_Message(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 44, 58, 0, time.Local).UnixMilli(),
		Event:     collector.EventMessage,
		Detail:    &collector.EventDetail{Message: "Updating the auth handler to fix the bug"},
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "○")
	assert.Contains(t, row, "message")
	assert.Contains(t, row, "Updating the auth")
}

func TestFormatEventRow_StatusChange(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 44, 50, 0, time.Local).UnixMilli(),
		Event:     collector.EventStatusChange,
		Status:    collector.StatusThinking,
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "◌")
	assert.Contains(t, row, "status.change")
	assert.Contains(t, row, "thinking")
}

func TestFormatEventRow_SessionStart(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 44, 48, 0, time.Local).UnixMilli(),
		Event:     collector.EventSessionStart,
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "▶")
	assert.Contains(t, row, "session.start")
}

func TestFormatEventRow_SessionEnd(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 50, 0, 0, time.Local).UnixMilli(),
		Event:     collector.EventSessionEnd,
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "■")
	assert.Contains(t, row, "session.end")
}

func TestFormatEventRow_Error(t *testing.T) {
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 50, 0, 0, time.Local).UnixMilli(),
		Event:     collector.EventError,
		Detail:    &collector.EventDetail{Message: "connection lost"},
	}
	row := FormatEventRow(ev)
	assert.Contains(t, row, "✕")
	assert.Contains(t, row, "error")
	assert.Contains(t, row, "connection lost")
}

func TestFormatEventRow_MessageTruncation(t *testing.T) {
	longMsg := strings.Repeat("x", 100)
	ev := collector.MonitorEvent{
		Timestamp: time.Date(2026, 3, 28, 22, 0, 0, 0, time.Local).UnixMilli(),
		Event:     collector.EventMessage,
		Detail:    &collector.EventDetail{Message: longMsg},
	}
	row := FormatEventRow(ev)
	// Should be truncated
	assert.True(t, len(row) < len(longMsg)+50, "row should be truncated")
}

// --- EventIndicator tests ---

func TestEventIndicator(t *testing.T) {
	tests := []struct {
		event    collector.EventType
		expected string
	}{
		{collector.EventToolStart, "●"},
		{collector.EventToolEnd, "●"},
		{collector.EventMessage, "○"},
		{collector.EventStatusChange, "◌"},
		{collector.EventSessionStart, "▶"},
		{collector.EventSessionEnd, "■"},
		{collector.EventError, "✕"},
	}
	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			assert.Equal(t, tt.expected, EventIndicator(tt.event))
		})
	}
}

// --- RenderInfoPanel tests ---

func TestRenderInfoPanel_ClaudeCode(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:      "claude-code",
		SessionID:   "a8837a23-595d-49de-83c1-0d95ebab",
		Workspace:   "~/Projects/my-app",
		Status:      collector.StatusToolRunning,
		CurrentTool: "Edit",
		StartedAt:   now.Add(-12*time.Minute - 34*time.Second).UnixMilli(),
		LastEventAt: now.UnixMilli(),
		EventCount:  247,
		TokenUsage:  &collector.TokenUsage{Input: 45200, Output: 12100, CacheRead: 89000},
		Labels:      map[string]string{"git_branch": "feature/auth"},
	}

	result := RenderInfoPanel(session, now)
	assert.Contains(t, result, "claude-code")
	assert.Contains(t, result, "a8837a23-595d-49de-83c1-0d95ebab")
	assert.Contains(t, result, "~/Projects/my-app")
	assert.Contains(t, result, "tool_running")
	assert.Contains(t, result, "Edit")
	assert.Contains(t, result, "12m 34s")
	assert.Contains(t, result, "45.2k in")
	assert.Contains(t, result, "12.1k out")
	assert.Contains(t, result, "89k cache")
	assert.Contains(t, result, "247")
}

func TestRenderInfoPanel_OpenClaw(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:    "openclaw",
		SessionID: "sess-abc",
		Status:    collector.StatusThinking,
		StartedAt: now.Add(-3 * time.Minute).UnixMilli(),
		EventCount: 42,
		Labels: map[string]string{
			"model":        "claude-opus-4",
			"percent_used": "11",
			"total_tokens": "1000000",
			"agent":        "r2d2",
		},
		TokenUsage: &collector.TokenUsage{Input: 100000, Output: 12000},
	}

	result := RenderInfoPanel(session, now)
	assert.Contains(t, result, "openclaw")
	assert.Contains(t, result, "claude-opus-4")
	assert.Contains(t, result, "11%")
	assert.Contains(t, result, "42")
}

func TestRenderInfoPanel_NoTokens(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusIdle,
		StartedAt: now.Add(-10 * time.Second).UnixMilli(),
	}

	result := RenderInfoPanel(session, now)
	assert.Contains(t, result, "claude-code")
	assert.Contains(t, result, "idle")
}

// --- DetailModel tests ---

func TestNewDetailModel(t *testing.T) {
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusThinking,
	}
	events := []collector.MonitorEvent{
		{ID: "e1", Event: collector.EventSessionStart, Timestamp: 1000},
		{ID: "e2", Event: collector.EventToolStart, Timestamp: 2000},
	}

	dm := NewDetailModel(session, events, 80, 24)
	assert.Equal(t, "s1", dm.session.SessionID)
	assert.Len(t, dm.events, 2)
	assert.True(t, dm.follow)
}

func TestDetailModel_View_ShowsInfoAndEvents(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:     "claude-code",
		SessionID:  "test-sess",
		Workspace:  "~/test",
		Status:     collector.StatusThinking,
		StartedAt:  now.Add(-1 * time.Minute).UnixMilli(),
		EventCount: 2,
	}
	events := []collector.MonitorEvent{
		{
			ID:        "e1",
			Timestamp: now.Add(-1 * time.Minute).UnixMilli(),
			Event:     collector.EventSessionStart,
			Status:    collector.StatusThinking,
		},
		{
			ID:        "e2",
			Timestamp: now.Add(-30 * time.Second).UnixMilli(),
			Event:     collector.EventToolStart,
			Status:    collector.StatusToolRunning,
			Detail:    &collector.EventDetail{Tool: "Read", Target: "main.go"},
		},
	}

	dm := NewDetailModel(session, events, 80, 24)
	view := dm.View()
	assert.Contains(t, view, "Session Detail")
	assert.Contains(t, view, "claude-code")
	assert.Contains(t, view, "test-sess")
	assert.Contains(t, view, "Event Log")
}

func TestDetailModel_AppendEvent(t *testing.T) {
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
	}
	dm := NewDetailModel(session, nil, 80, 24)
	assert.Empty(t, dm.events)

	ev := collector.MonitorEvent{
		ID:        "e1",
		Source:    "claude-code",
		SessionID: "s1",
		Event:     collector.EventToolStart,
		Timestamp: time.Now().UnixMilli(),
		Detail:    &collector.EventDetail{Tool: "Edit", Target: "foo.go"},
	}

	dm.AppendEvent(ev)
	assert.Len(t, dm.events, 1)
	assert.Equal(t, "e1", dm.events[0].ID)
}

func TestDetailModel_AppendEvent_UpdatesSession(t *testing.T) {
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusIdle,
	}
	dm := NewDetailModel(session, nil, 80, 24)

	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusToolRunning,
		Timestamp: time.Now().UnixMilli(),
		Event:     collector.EventToolStart,
		Detail:    &collector.EventDetail{Tool: "Edit"},
	}

	dm.AppendEvent(ev)
	assert.Equal(t, collector.StatusToolRunning, dm.session.Status)
	assert.Equal(t, "Edit", dm.session.CurrentTool)
}

func TestDetailModel_FollowMode(t *testing.T) {
	session := collector.SessionState{Source: "test", SessionID: "s1"}
	dm := NewDetailModel(session, nil, 80, 24)
	assert.True(t, dm.follow, "follow mode should be on by default")

	dm.ToggleFollow()
	assert.False(t, dm.follow)

	dm.ToggleFollow()
	assert.True(t, dm.follow)
}

func TestDetailModel_SetSize(t *testing.T) {
	session := collector.SessionState{Source: "test", SessionID: "s1"}
	dm := NewDetailModel(session, nil, 80, 24)

	dm.SetSize(120, 40)
	assert.Equal(t, 120, dm.width)
	assert.Equal(t, 40, dm.height)
}

// --- Detail view empty state (T7) ---

func TestDetailModel_View_NoEvents(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "empty-sess",
		Status:    collector.StatusIdle,
		StartedAt: now.Add(-5 * time.Second).UnixMilli(),
	}

	dm := NewDetailModel(session, nil, 80, 24)
	view := dm.View()
	assert.Contains(t, view, "No events recorded")
}

// --- formatTokenDetailed tests ---

func TestFormatTokenDetailed(t *testing.T) {
	tu := &collector.TokenUsage{Input: 45200, Output: 12100, CacheRead: 89000}
	result := formatTokenDetailed(tu)
	assert.Contains(t, result, "45.2k in")
	assert.Contains(t, result, "12.1k out")
	assert.Contains(t, result, "89k cache")
}

func TestFormatTokenDetailed_Small(t *testing.T) {
	tu := &collector.TokenUsage{Input: 500, Output: 100}
	result := formatTokenDetailed(tu)
	assert.Contains(t, result, "500 in")
	assert.Contains(t, result, "100 out")
}

func TestFormatTokenDetailed_NoCacheRead(t *testing.T) {
	tu := &collector.TokenUsage{Input: 1000, Output: 200}
	result := formatTokenDetailed(tu)
	assert.NotContains(t, result, "cache")
}

// --- Edge cases (T7) ---

func TestDetailModel_SessionEndsWhileViewing(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusToolRunning,
		StartedAt: now.Add(-5 * time.Minute).UnixMilli(),
	}
	dm := NewDetailModel(session, nil, 80, 24)

	// Session ends
	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusDone,
		Timestamp: now.UnixMilli(),
		Event:     collector.EventSessionEnd,
	}
	dm.AppendEvent(ev)

	assert.Equal(t, collector.StatusDone, dm.session.Status)
	assert.Equal(t, "", dm.session.CurrentTool)
	assert.Len(t, dm.events, 1)

	// View still renders
	view := dm.View()
	assert.Contains(t, view, "done")
	assert.Contains(t, view, "session.end")
}

func TestDetailModel_ResizeAdjustsEventLog(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
		StartedAt: now.UnixMilli(),
	}
	events := make([]collector.MonitorEvent, 50)
	for i := range events {
		events[i] = collector.MonitorEvent{
			Source: "claude-code", SessionID: "s1",
			Event: collector.EventMessage, Timestamp: now.Add(time.Duration(i) * time.Second).UnixMilli(),
		}
	}

	dm := NewDetailModel(session, events, 80, 24)
	h1 := dm.eventLogHeight()

	// Resize to larger terminal
	dm.SetSize(120, 50)
	h2 := dm.eventLogHeight()
	assert.Greater(t, h2, h1, "larger terminal should show more event lines")

	// View should still render without panic
	view := dm.View()
	assert.Contains(t, view, "Event Log")
}

// --- Ensure MatchesSession works ---

func TestDetailModel_MatchesSession(t *testing.T) {
	session := collector.SessionState{Source: "claude-code", SessionID: "s1"}
	dm := NewDetailModel(session, nil, 80, 24)

	require.True(t, dm.MatchesSession("claude-code", "s1"))
	require.False(t, dm.MatchesSession("claude-code", "s2"))
	require.False(t, dm.MatchesSession("openclaw", "s1"))
}
