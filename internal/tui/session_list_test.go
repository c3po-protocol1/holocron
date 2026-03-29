package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/c3po-protocol1/holocron/internal/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderGroupedList_SingleGroup(t *testing.T) {
	now := time.Now()
	groups := []labels.SessionGroup{
		{
			Label: "r2d2",
			Sessions: []collector.SessionState{
				{Source: "openclaw", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.Add(-30 * time.Second).UnixMilli(), Labels: map[string]string{"agent": "r2d2"}},
				{Source: "claude-code", SessionID: "s2", Status: collector.StatusIdle, StartedAt: now.Add(-2 * time.Minute).UnixMilli(), Labels: map[string]string{"agent": "r2d2"}},
			},
			Active: 1,
		},
	}

	result := RenderGroupedList(groups, 0, now, 60)

	assert.Contains(t, result, "r2d2")
	assert.Contains(t, result, "2 sessions")
	assert.Contains(t, result, "1 active")
	assert.Contains(t, result, "───")
}

func TestRenderGroupedList_MultipleGroups(t *testing.T) {
	now := time.Now()
	groups := []labels.SessionGroup{
		{
			Label: "r2d2",
			Sessions: []collector.SessionState{
				{Source: "claude-code", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.Add(-30 * time.Second).UnixMilli()},
			},
			Active: 1,
		},
		{
			Label: "yoda",
			Sessions: []collector.SessionState{
				{Source: "claude-code", SessionID: "s2", Status: collector.StatusIdle, StartedAt: now.Add(-5 * time.Minute).UnixMilli()},
			},
			Active: 0,
		},
	}

	result := RenderGroupedList(groups, 0, now, 60)

	assert.Contains(t, result, "r2d2")
	assert.Contains(t, result, "yoda")
	// Cursor on first session
	assert.Contains(t, result, "▶")
}

func TestRenderGroupedList_CursorAcrossGroups(t *testing.T) {
	now := time.Now()
	groups := []labels.SessionGroup{
		{
			Label: "r2d2",
			Sessions: []collector.SessionState{
				{Source: "claude-code", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.UnixMilli()},
			},
			Active: 1,
		},
		{
			Label: "yoda",
			Sessions: []collector.SessionState{
				{Source: "claude-code", SessionID: "s2", Status: collector.StatusIdle, StartedAt: now.UnixMilli()},
			},
			Active: 0,
		},
	}

	// Cursor on second session (index 1) which is in the second group
	result := RenderGroupedList(groups, 1, now, 60)

	lines := strings.Split(result, "\n")
	// Find the line with the cursor
	found := false
	for _, line := range lines {
		if strings.Contains(line, "▶") && strings.Contains(line, "s2") {
			found = true
			break
		}
	}
	assert.True(t, found, "cursor should be on s2 in second group")
}

func TestRenderGroupedList_HeaderFormat(t *testing.T) {
	now := time.Now()
	groups := []labels.SessionGroup{
		{
			Label: "r2d2",
			Sessions: []collector.SessionState{
				{Source: "claude-code", SessionID: "s1", Status: collector.StatusThinking, StartedAt: now.UnixMilli()},
				{Source: "claude-code", SessionID: "s2", Status: collector.StatusIdle, StartedAt: now.UnixMilli()},
			},
			Active: 1,
		},
	}

	result := RenderGroupedList(groups, 0, now, 60)

	// Header should contain: ─── r2d2 (2 sessions, 1 active) ───
	assert.Contains(t, result, "r2d2 (2 sessions, 1 active)")
}

func TestRenderGroupedList_HeaderNoActive(t *testing.T) {
	now := time.Now()
	groups := []labels.SessionGroup{
		{
			Label: "unlabeled",
			Sessions: []collector.SessionState{
				{Source: "claude-code", SessionID: "s1", Status: collector.StatusIdle, StartedAt: now.UnixMilli()},
			},
			Active: 0,
		},
	}

	result := RenderGroupedList(groups, 0, now, 60)

	// No active count when 0
	assert.Contains(t, result, "unlabeled (1 session)")
	assert.NotContains(t, result, "0 active")
}

func TestRenderGroupedList_EmptyGroups(t *testing.T) {
	result := RenderGroupedList(nil, 0, time.Now(), 60)
	assert.Contains(t, result, "No sessions detected")
}

func TestFlattenGroups(t *testing.T) {
	groups := []labels.SessionGroup{
		{
			Label: "r2d2",
			Sessions: []collector.SessionState{
				{SessionID: "s1"},
				{SessionID: "s2"},
			},
		},
		{
			Label: "yoda",
			Sessions: []collector.SessionState{
				{SessionID: "s3"},
			},
		},
	}

	flat := FlattenGroups(groups)

	require.Len(t, flat, 3)
	assert.Equal(t, "s1", flat[0].SessionID)
	assert.Equal(t, "s2", flat[1].SessionID)
	assert.Equal(t, "s3", flat[2].SessionID)
}

func TestFlattenGroups_Empty(t *testing.T) {
	flat := FlattenGroups(nil)
	assert.Empty(t, flat)
}
