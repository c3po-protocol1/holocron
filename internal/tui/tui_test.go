package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// --- FormatElapsed tests ---

func TestFormatElapsed_Seconds(t *testing.T) {
	assert.Equal(t, "0s", FormatElapsed(0))
	assert.Equal(t, "5s", FormatElapsed(5*time.Second))
	assert.Equal(t, "59s", FormatElapsed(59*time.Second))
}

func TestFormatElapsed_Minutes(t *testing.T) {
	assert.Equal(t, "1m 00s", FormatElapsed(60*time.Second))
	assert.Equal(t, "2m 13s", FormatElapsed(2*time.Minute+13*time.Second))
	assert.Equal(t, "59m 59s", FormatElapsed(59*time.Minute+59*time.Second))
}

func TestFormatElapsed_Hours(t *testing.T) {
	assert.Equal(t, "1h 00m", FormatElapsed(time.Hour))
	assert.Equal(t, "2h 15m", FormatElapsed(2*time.Hour+15*time.Minute))
	assert.Equal(t, "10h 05m", FormatElapsed(10*time.Hour+5*time.Minute))
}

func TestFormatElapsed_Negative(t *testing.T) {
	assert.Equal(t, "0s", FormatElapsed(-5*time.Second))
}

// --- StatusIndicator tests ---

func TestStatusIndicator_Active(t *testing.T) {
	for _, status := range []collector.SessionStatus{collector.StatusThinking, collector.StatusToolRunning} {
		result := StatusIndicator(status)
		assert.Contains(t, result, StatusDotActive, "status %s should show active dot", status)
	}
}

func TestStatusIndicator_Idle(t *testing.T) {
	for _, status := range []collector.SessionStatus{collector.StatusIdle, collector.StatusWaiting} {
		result := StatusIndicator(status)
		assert.Contains(t, result, StatusDotIdle, "status %s should show idle dot", status)
	}
}

func TestStatusIndicator_Error(t *testing.T) {
	result := StatusIndicator(collector.StatusError)
	assert.Contains(t, result, StatusDotError)
}

func TestStatusIndicator_Done(t *testing.T) {
	result := StatusIndicator(collector.StatusDone)
	assert.Contains(t, result, StatusDotDone)
}

// --- TruncateID tests ---

func TestTruncateID_Short(t *testing.T) {
	assert.Equal(t, "abc", TruncateID("abc"))
	assert.Equal(t, "1234567890", TruncateID("1234567890"))
}

func TestTruncateID_Long(t *testing.T) {
	assert.Equal(t, "a8837a23cd..", TruncateID("a8837a23cdef1234"))
}

// --- RenderEmptyState tests ---

func TestRenderEmptyState(t *testing.T) {
	result := RenderEmptyState()
	assert.Contains(t, result, "No sessions detected")
	assert.Contains(t, result, "config")
}

// --- RenderSessionList tests ---

func TestRenderSessionList_Empty(t *testing.T) {
	result := RenderSessionList(nil, 0, time.Now())
	assert.Contains(t, result, "No sessions detected")
}

func TestRenderSessionList_WithSessions(t *testing.T) {
	now := time.Now()
	sessions := []collector.SessionState{
		{
			Source:    "claude-code",
			SessionID: "a8837a23cdef1234",
			Workspace: "~/Projects/my-app",
			Status:    collector.StatusToolRunning,
			StartedAt: now.Add(-2*time.Minute - 13*time.Second).UnixMilli(),
			CurrentTool:   "Edit",
			CurrentTarget: "src/index.ts",
		},
		{
			Source:    "claude-code",
			SessionID: "323ac29b5678",
			Workspace: "~/Projects/agent-monitor",
			Status:    collector.StatusIdle,
			StartedAt: now.Add(-15*time.Minute - 2*time.Second).UnixMilli(),
		},
	}

	result := RenderSessionList(sessions, 0, now)
	assert.Contains(t, result, "claude-code")
	assert.Contains(t, result, "a8837a23cd..")
	assert.Contains(t, result, "~/Projects/my-app")
	assert.Contains(t, result, "Edit → src/index.ts")
	assert.Contains(t, result, "▶") // cursor on first row
}

// --- RenderSessionRow tests ---

func TestRenderSessionRow_Selected(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "claude-code",
		SessionID: "abc123",
		Status:    collector.StatusThinking,
		StartedAt: now.Add(-30 * time.Second).UnixMilli(),
	}

	result := RenderSessionRow(s, true, now)
	assert.Contains(t, result, "▶")
}

func TestRenderSessionRow_NotSelected(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "claude-code",
		SessionID: "abc123",
		Status:    collector.StatusIdle,
		StartedAt: now.Add(-30 * time.Second).UnixMilli(),
	}

	result := RenderSessionRow(s, false, now)
	assert.NotContains(t, result, "▶")
}

// --- Model tests ---

func TestNew(t *testing.T) {
	ch := make(chan collector.MonitorEvent)
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "claude-code", Status: collector.StatusIdle},
	}

	m := New(ch, sessions)
	assert.Len(t, m.sessions, 1)
	assert.Equal(t, 0, m.cursor)
	assert.False(t, m.showHelp)
}

func TestNew_NilSessions(t *testing.T) {
	m := New(nil, nil)
	assert.Empty(t, m.sessions)
}

func TestModel_CursorBounds(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "test"},
		{SessionID: "s2", Source: "test"},
		{SessionID: "s3", Source: "test"},
	}
	m := New(nil, sessions)

	// Move down through all sessions
	var model tea.Model = m
	for i := 0; i < 5; i++ {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	assert.Equal(t, 2, model.(Model).cursor, "cursor should not exceed last index")

	// Move up past the beginning
	for i := 0; i < 5; i++ {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	}
	assert.Equal(t, 0, model.(Model).cursor, "cursor should not go below 0")
}

func TestModel_ToggleHelp(t *testing.T) {
	m := New(nil, nil)
	assert.False(t, m.showHelp)

	// Press ?
	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.True(t, model.(Model).showHelp)

	// Press any key to close help (not quit)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.False(t, model.(Model).showHelp)
}

func TestModel_Quit(t *testing.T) {
	m := New(nil, nil)

	var model tea.Model = m
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	// tea.Quit returns a special command
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestModel_WindowResize(t *testing.T) {
	m := New(nil, nil)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.Equal(t, 120, model.(Model).width)
	assert.Equal(t, 40, model.(Model).height)
}

func TestModel_ApplyEvent_NewSession(t *testing.T) {
	m := New(nil, nil)

	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "new-session",
		Workspace: "~/Projects/test",
		Status:    collector.StatusThinking,
		Timestamp: time.Now().UnixMilli(),
		Event:     collector.EventSessionStart,
	}

	m.applyEvent(ev)

	assert.Len(t, m.sessions, 1)
	assert.Equal(t, "new-session", m.sessions[0].SessionID)
	assert.Equal(t, collector.StatusThinking, m.sessions[0].Status)
	assert.Equal(t, "~/Projects/test", m.sessions[0].Workspace)
	assert.Equal(t, 1, m.eventCount)
}

func TestModel_ApplyEvent_UpdateExisting(t *testing.T) {
	sessions := []collector.SessionState{
		{
			Source:    "claude-code",
			SessionID: "existing",
			Status:    collector.StatusIdle,
		},
	}
	m := New(nil, sessions)

	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "existing",
		Status:    collector.StatusToolRunning,
		Timestamp: time.Now().UnixMilli(),
		Detail: &collector.EventDetail{
			Tool:   "Edit",
			Target: "main.go",
		},
	}

	m.applyEvent(ev)

	assert.Len(t, m.sessions, 1)
	assert.Equal(t, collector.StatusToolRunning, m.sessions[0].Status)
	assert.Equal(t, "Edit", m.sessions[0].CurrentTool)
	assert.Equal(t, "main.go", m.sessions[0].CurrentTarget)
}

func TestModel_ApplyEvent_ClearsToolOnIdle(t *testing.T) {
	sessions := []collector.SessionState{
		{
			Source:      "claude-code",
			SessionID:   "s1",
			Status:      collector.StatusToolRunning,
			CurrentTool: "Edit",
			CurrentTarget: "file.go",
		},
	}
	m := New(nil, sessions)

	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusIdle,
		Timestamp: time.Now().UnixMilli(),
	}

	m.applyEvent(ev)

	assert.Equal(t, "", m.sessions[0].CurrentTool)
	assert.Equal(t, "", m.sessions[0].CurrentTarget)
}

func TestModel_ApplyEvent_FromChannel(t *testing.T) {
	ch := make(chan collector.MonitorEvent, 1)
	m := New(ch, nil)

	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "ch-session",
		Status:    collector.StatusThinking,
		Timestamp: time.Now().UnixMilli(),
	}
	ch <- ev

	// The waitForEvent cmd should read from the channel
	cmd := m.waitForEvent()
	require.NotNil(t, cmd)
	msg := cmd()
	require.NotNil(t, msg)

	evMsg, ok := msg.(eventMsg)
	require.True(t, ok)
	assert.Equal(t, "ch-session", collector.MonitorEvent(evMsg).SessionID)
}

func TestModel_View_EmptyState(t *testing.T) {
	m := New(nil, nil)
	view := m.View()

	assert.Contains(t, view, "Holocron")
	assert.Contains(t, view, "No sessions detected")
	assert.Contains(t, view, "0 sessions")
}

func TestModel_View_WithSessions(t *testing.T) {
	now := time.Now()
	sessions := []collector.SessionState{
		{
			Source:    "claude-code",
			SessionID: "abc123",
			Status:    collector.StatusThinking,
			StartedAt: now.Add(-5 * time.Minute).UnixMilli(),
		},
	}
	m := New(nil, sessions)
	view := m.View()

	assert.Contains(t, view, "Holocron")
	assert.Contains(t, view, "claude-code")
	assert.Contains(t, view, "1 sessions")
	assert.Contains(t, view, "1 active")
}

func TestModel_View_HelpOverlay(t *testing.T) {
	m := New(nil, nil)
	m.showHelp = true
	view := m.View()

	assert.Contains(t, view, "Key Bindings")
	assert.Contains(t, view, "Move up")
	assert.Contains(t, view, "Quit")
}

// --- RenderHelp tests ---

func TestRenderHelp(t *testing.T) {
	keys := DefaultKeyMap()
	result := RenderHelp(keys, 80)

	assert.Contains(t, result, "Key Bindings")
	assert.Contains(t, result, "Move up")
	assert.Contains(t, result, "Move down")
	assert.Contains(t, result, "Quit")
	assert.Contains(t, result, "Toggle help")
	assert.Contains(t, result, "Force refresh")
}

// --- KeyMap tests ---

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Quit", km.Quit},
		{"Help", km.Help},
		{"Refresh", km.Refresh},
	}

	for _, b := range bindings {
		h := b.binding.Help()
		assert.NotEmpty(t, h.Key, "%s should have a key", b.name)
		assert.NotEmpty(t, h.Desc, "%s should have a description", b.name)
	}
}

// --- Style constant tests ---

func TestStatusDotConstants(t *testing.T) {
	assert.Equal(t, "●", StatusDotActive)
	assert.Equal(t, "◌", StatusDotIdle)
	assert.Equal(t, "✕", StatusDotError)
	assert.Equal(t, "✓", StatusDotDone)
}

// --- filterActive tests ---

func TestFilterActive_MixedStatuses(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Status: collector.StatusThinking},
		{SessionID: "s2", Status: collector.StatusIdle},
		{SessionID: "s3", Status: collector.StatusToolRunning},
		{SessionID: "s4", Status: collector.StatusDone},
		{SessionID: "s5", Status: collector.StatusWaiting},
		{SessionID: "s6", Status: collector.StatusError},
	}

	result := filterActive(sessions)
	assert.Len(t, result, 4)
	assert.Equal(t, "s1", result[0].SessionID)
	assert.Equal(t, "s3", result[1].SessionID)
	assert.Equal(t, "s5", result[2].SessionID)
	assert.Equal(t, "s6", result[3].SessionID)
}

func TestFilterActive_AllIdle(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Status: collector.StatusIdle},
		{SessionID: "s2", Status: collector.StatusDone},
	}

	result := filterActive(sessions)
	assert.Empty(t, result)
}

func TestFilterActive_AllActive(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Status: collector.StatusThinking},
		{SessionID: "s2", Status: collector.StatusToolRunning},
	}

	result := filterActive(sessions)
	assert.Len(t, result, 2)
}

func TestFilterActive_Empty(t *testing.T) {
	result := filterActive(nil)
	assert.Empty(t, result)
}

// --- Active toggle tests ---

func TestModel_ToggleActiveOnly(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "test", Status: collector.StatusThinking},
		{SessionID: "s2", Source: "test", Status: collector.StatusIdle},
	}
	m := New(nil, sessions)
	assert.False(t, m.activeOnly)

	// Press 'a' to enable
	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.True(t, model.(Model).activeOnly)

	// Press 'a' again to disable
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.False(t, model.(Model).activeOnly)
}

func TestModel_ActiveFilter_CursorReset(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "test", Status: collector.StatusIdle},
		{SessionID: "s2", Source: "test", Status: collector.StatusIdle},
		{SessionID: "s3", Source: "test", Status: collector.StatusThinking},
	}
	m := New(nil, sessions)

	// Move cursor to s3 (index 2)
	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, model.(Model).cursor)

	// Enable active filter — only s3 visible, cursor should reset to 0
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, 0, model.(Model).cursor)
}

func TestModel_ActiveFilter_EmptyStateMessage(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "test", Status: collector.StatusIdle},
		{SessionID: "s2", Source: "test", Status: collector.StatusDone},
	}
	m := New(nil, sessions)
	m.activeOnly = true

	view := m.View()
	assert.Contains(t, view, "No active sessions")
	assert.Contains(t, view, "'a'")
}

func TestModel_ActiveFilter_FooterShowsHiddenCount(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "test", Status: collector.StatusThinking},
		{SessionID: "s2", Source: "test", Status: collector.StatusIdle},
		{SessionID: "s3", Source: "test", Status: collector.StatusDone},
	}
	m := New(nil, sessions)
	m.activeOnly = true

	view := m.View()
	assert.Contains(t, view, "[a]ctive: on")
	assert.Contains(t, view, "2 hidden")
}

func TestModel_ActiveFilter_FooterShowsOff(t *testing.T) {
	m := New(nil, nil)
	m.activeOnly = false

	view := m.View()
	assert.Contains(t, view, "[a]ctive: off")
}

func TestModel_ActiveFilter_CursorBoundsWithFilter(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Source: "test", Status: collector.StatusThinking},
	}
	m := New(nil, sessions)
	m.activeOnly = true

	// Try to move cursor down past the single visible session
	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 0, model.(Model).cursor)
}

// --- Help shows active binding ---

func TestRenderHelp_ShowsActiveBinding(t *testing.T) {
	keys := DefaultKeyMap()
	result := RenderHelp(keys, 80)
	assert.Contains(t, result, "toggle active-only filter")
}

// --- KeyMap includes Active ---

func TestDefaultKeyMap_IncludesActive(t *testing.T) {
	km := DefaultKeyMap()
	h := km.Active.Help()
	assert.NotEmpty(t, h.Key)
	assert.NotEmpty(t, h.Desc)
}

// --- OpenClaw rendering tests ---

func TestRenderSessionRow_OpenClaw_ShowsAgentName(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "sess-1",
		Status:    collector.StatusThinking,
		StartedAt: now.Add(-30 * time.Second).UnixMilli(),
		Labels: map[string]string{
			"agent":        "r2d2",
			"session_type": "direct",
			"channel":      "discord",
			"model":        "opus-4",
		},
		TokenUsage: &collector.TokenUsage{Input: 100000, Output: 11000},
	}

	result := RenderSessionRow(s, false, now)
	assert.Contains(t, result, "openclaw")
	assert.Contains(t, result, "r2d2")
	assert.Contains(t, result, "active")
	assert.Contains(t, result, "discord:direct")
	assert.Contains(t, result, "opus-4")
	assert.Contains(t, result, "tokens:")
}

func TestRenderSessionRow_OpenClaw_IdleSession(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "sess-2",
		Status:    collector.StatusIdle,
		StartedAt: now.Add(-42 * time.Minute).UnixMilli(),
		Labels: map[string]string{
			"agent":        "r2d2",
			"session_type": "cron",
		},
	}

	result := RenderSessionRow(s, false, now)
	assert.Contains(t, result, "idle")
	assert.Contains(t, result, "cron")
}

func TestRenderSessionRow_OpenClaw_WithTokenBudget(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "sess-3",
		Status:    collector.StatusThinking,
		StartedAt: now.Add(-2 * time.Second).UnixMilli(),
		Labels: map[string]string{
			"agent":        "yoda",
			"session_type": "direct",
			"channel":      "discord",
			"model":        "opus-4",
			"total_tokens":  "1000000",
			"percent_used":  "11",
		},
		TokenUsage: &collector.TokenUsage{Input: 100000, Output: 12000},
	}

	result := RenderSessionRow(s, false, now)
	assert.Contains(t, result, "112k / 1M (11%)")
	assert.Contains(t, result, "opus-4")
}

func TestFormatTokenCount(t *testing.T) {
	assert.Equal(t, "0", formatTokenCount(0))
	assert.Equal(t, "500", formatTokenCount(500))
	assert.Equal(t, "1k", formatTokenCount(1000))
	assert.Equal(t, "111k", formatTokenCount(111000))
	assert.Equal(t, "1M", formatTokenCount(1000000))
	assert.Equal(t, "2M", formatTokenCount(2500000))
}

func TestRenderSessionRow_OpenClaw_Selected(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "sess-1",
		Status:    collector.StatusThinking,
		StartedAt: now.Add(-10 * time.Second).UnixMilli(),
		Labels:    map[string]string{"agent": "r2d2", "session_type": "direct"},
	}

	result := RenderSessionRow(s, true, now)
	assert.Contains(t, result, "▶")
}

func TestRenderSessionRow_OpenClaw_FallbackSessionID(t *testing.T) {
	now := time.Now()
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "abcdefghijklmnop",
		Status:    collector.StatusIdle,
		StartedAt: now.Add(-10 * time.Second).UnixMilli(),
		Labels:    map[string]string{},
	}

	result := RenderSessionRow(s, false, now)
	assert.Contains(t, result, "abcdefghij..")
}

// --- ViewMode and detail integration tests (T4) ---

// mockStore implements EventLoader for testing.
type mockStore struct {
	events []collector.MonitorEvent
}

func (m *mockStore) GetEvents(source, sessionID string, since int64, limit int) ([]collector.MonitorEvent, error) {
	var result []collector.MonitorEvent
	for _, ev := range m.events {
		if ev.Source == source && ev.SessionID == sessionID {
			result = append(result, ev)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func TestModel_ViewMode_DefaultIsList(t *testing.T) {
	m := New(nil, nil)
	assert.Equal(t, ViewList, m.view)
	assert.Nil(t, m.detail)
}

func TestModel_EnterOpensDetail(t *testing.T) {
	now := time.Now()
	sessions := []collector.SessionState{
		{
			Source:    "claude-code",
			SessionID: "s1",
			Status:    collector.StatusThinking,
			StartedAt: now.Add(-1 * time.Minute).UnixMilli(),
		},
	}
	store := &mockStore{
		events: []collector.MonitorEvent{
			{Source: "claude-code", SessionID: "s1", Event: collector.EventSessionStart, Timestamp: now.Add(-1 * time.Minute).UnixMilli()},
		},
	}
	m := NewWithStore(nil, sessions, store)

	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	result := model.(Model)
	assert.Equal(t, ViewDetail, result.view)
	require.NotNil(t, result.detail)
	assert.Equal(t, "s1", result.detail.session.SessionID)
	assert.Len(t, result.detail.events, 1)
}

func TestModel_EnterNoSessions(t *testing.T) {
	m := New(nil, nil)

	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	result := model.(Model)
	assert.Equal(t, ViewList, result.view)
	assert.Nil(t, result.detail)
}

func TestModel_EscReturnsToList(t *testing.T) {
	now := time.Now()
	sessions := []collector.SessionState{
		{Source: "claude-code", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.UnixMilli()},
	}
	m := NewWithStore(nil, sessions, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(sessions[0], nil, 80, 24)

	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})

	result := model.(Model)
	assert.Equal(t, ViewList, result.view)
	assert.Nil(t, result.detail)
}

func TestModel_DetailView_EventRouting(t *testing.T) {
	now := time.Now()
	sessions := []collector.SessionState{
		{Source: "claude-code", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.UnixMilli()},
	}
	m := NewWithStore(nil, sessions, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(sessions[0], nil, 80, 24)

	// Matching event should be routed to detail
	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusToolRunning,
		Timestamp: now.UnixMilli(),
		Event:     collector.EventToolStart,
		Detail:    &collector.EventDetail{Tool: "Edit", Target: "foo.go"},
	}

	var model tea.Model = m
	model, _ = model.Update(eventMsg(ev))
	result := model.(Model)

	assert.Len(t, result.detail.events, 1)
	assert.Equal(t, "Edit", result.detail.events[0].Detail.Tool)
}

func TestModel_DetailView_NonMatchingEventNotRouted(t *testing.T) {
	now := time.Now()
	sessions := []collector.SessionState{
		{Source: "claude-code", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.UnixMilli()},
	}
	m := NewWithStore(nil, sessions, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(sessions[0], nil, 80, 24)

	// Event for different session should NOT be routed
	ev := collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "s2",
		Status:    collector.StatusThinking,
		Timestamp: now.UnixMilli(),
	}

	var model tea.Model = m
	model, _ = model.Update(eventMsg(ev))
	result := model.(Model)

	assert.Empty(t, result.detail.events)
}

func TestModel_DetailView_ScrollKeys(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{Source: "claude-code", SessionID: "s1", StartedAt: now.UnixMilli()}
	events := make([]collector.MonitorEvent, 50)
	for i := range events {
		events[i] = collector.MonitorEvent{
			Source: "claude-code", SessionID: "s1",
			Event: collector.EventMessage, Timestamp: now.Add(time.Duration(i) * time.Second).UnixMilli(),
		}
	}

	m := NewWithStore(nil, []collector.SessionState{session}, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(session, events, 80, 24)

	// 'g' should jump to top
	var model tea.Model = m
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	result := model.(Model)
	assert.Equal(t, 0, result.detail.scroll)
	assert.False(t, result.detail.follow)

	// 'G' should jump to bottom
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	result = model.(Model)
	assert.True(t, result.detail.follow)

	// 'f' should toggle follow
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	result = model.(Model)
	assert.False(t, result.detail.follow)
}

func TestModel_DetailView_Renders(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{
		Source:    "claude-code",
		SessionID: "s1",
		Status:    collector.StatusThinking,
		StartedAt: now.Add(-30 * time.Second).UnixMilli(),
	}
	m := NewWithStore(nil, []collector.SessionState{session}, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(session, nil, 80, 24)

	view := m.View()
	assert.Contains(t, view, "Session Detail")
	assert.Contains(t, view, "claude-code")
	assert.Contains(t, view, "[Esc]back")
}

func TestModel_DetailView_HelpOverlay(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{Source: "claude-code", SessionID: "s1", StartedAt: now.UnixMilli()}
	m := NewWithStore(nil, []collector.SessionState{session}, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(session, nil, 80, 24)
	m.showHelp = true

	view := m.View()
	assert.Contains(t, view, "Key Bindings")
	assert.Contains(t, view, "Back to session list")
	assert.Contains(t, view, "Toggle follow")
}

func TestModel_DetailView_WindowResize(t *testing.T) {
	now := time.Now()
	session := collector.SessionState{Source: "claude-code", SessionID: "s1", StartedAt: now.UnixMilli()}
	m := NewWithStore(nil, []collector.SessionState{session}, &mockStore{})
	m.view = ViewDetail
	m.detail = NewDetailModel(session, nil, 80, 24)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := model.(Model)

	assert.Equal(t, 120, result.detail.width)
	assert.Equal(t, 40, result.detail.height)
}

// --- Key bindings tests (T5) ---

func TestDefaultKeyMap_IncludesDetailBindings(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Enter", km.Enter},
		{"Back", km.Back},
		{"Top", km.Top},
		{"Bottom", km.Bottom},
		{"Follow", km.Follow},
	}

	for _, b := range bindings {
		h := b.binding.Help()
		assert.NotEmpty(t, h.Key, "%s should have a key", b.name)
		assert.NotEmpty(t, h.Desc, "%s should have a description", b.name)
	}
}

func TestRenderDetailHelp(t *testing.T) {
	keys := DefaultKeyMap()
	result := RenderDetailHelp(keys, 80)
	assert.Contains(t, result, "Back to session list")
	assert.Contains(t, result, "Scroll up")
	assert.Contains(t, result, "Scroll down")
	assert.Contains(t, result, "Jump to top")
	assert.Contains(t, result, "Jump to bottom")
	assert.Contains(t, result, "Toggle follow")
}

func TestRenderHelp_ShowsEnterBinding(t *testing.T) {
	keys := DefaultKeyMap()
	result := RenderHelp(keys, 80)
	assert.Contains(t, result, "Open session detail")
}

// --- Integration-like tests ---

func TestModel_FullEventFlow(t *testing.T) {
	ch := make(chan collector.MonitorEvent, 10)
	m := New(ch, nil)

	// Send session start
	ch <- collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "flow-test",
		Workspace: "~/test",
		Status:    collector.StatusThinking,
		Timestamp: time.Now().UnixMilli(),
		Event:     collector.EventSessionStart,
	}

	// Read event through cmd
	cmd := m.waitForEvent()
	msg := cmd()
	var model tea.Model = m
	model, _ = model.Update(msg)

	result := model.(Model)
	assert.Len(t, result.sessions, 1)
	assert.Equal(t, "flow-test", result.sessions[0].SessionID)

	// Send tool start
	ch <- collector.MonitorEvent{
		Source:    "claude-code",
		SessionID: "flow-test",
		Status:    collector.StatusToolRunning,
		Timestamp: time.Now().UnixMilli(),
		Event:     collector.EventToolStart,
		Detail:    &collector.EventDetail{Tool: "Read", Target: "main.go"},
	}

	cmd = result.waitForEvent()
	msg = cmd()
	model, _ = model.Update(msg)
	result = model.(Model)

	assert.Equal(t, collector.StatusToolRunning, result.sessions[0].Status)
	assert.Equal(t, "Read", result.sessions[0].CurrentTool)
	assert.Equal(t, "main.go", result.sessions[0].CurrentTarget)

	// Verify view renders
	view := result.View()
	assert.True(t, strings.Contains(view, "claude-code"))
	assert.True(t, strings.Contains(view, "1 active"))
}
