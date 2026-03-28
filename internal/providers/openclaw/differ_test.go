package openclaw

import (
	"testing"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffer_NewSession(t *testing.T) {
	d := NewDiffer(60000)

	sessions := []OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
	}

	events := d.Diff(sessions)
	require.Len(t, events, 1)
	assert.Equal(t, collector.EventSessionStart, events[0].Event)
	assert.Equal(t, "openclaw", events[0].Source)
	assert.Equal(t, "sess-1", events[0].SessionID)
	assert.Equal(t, "r2d2", events[0].Labels["agent"])
}

func TestDiffer_SessionDisappears(t *testing.T) {
	d := NewDiffer(60000)

	// First poll: session exists
	d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
	})

	// Second poll: session gone
	events := d.Diff([]OCSession{})
	require.Len(t, events, 1)
	assert.Equal(t, collector.EventSessionEnd, events[0].Event)
	assert.Equal(t, "sess-1", events[0].SessionID)
}

func TestDiffer_UpdatedAtChanged(t *testing.T) {
	d := NewDiffer(60000)

	// First poll
	d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
	})

	// Second poll: updatedAt changed, low age → thinking
	events := d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 2000, Age: 200},
	})

	require.Len(t, events, 1)
	assert.Equal(t, collector.EventStatusChange, events[0].Event)
	assert.Equal(t, collector.StatusThinking, events[0].Status)
}

func TestDiffer_BecameIdle(t *testing.T) {
	d := NewDiffer(60000) // 60s threshold

	// First poll: active
	d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
	})

	// Second poll: same updatedAt, age exceeds threshold
	events := d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 70000},
	})

	require.Len(t, events, 1)
	assert.Equal(t, collector.EventStatusChange, events[0].Event)
	assert.Equal(t, collector.StatusIdle, events[0].Status)
}

func TestDiffer_AbortedLastRun(t *testing.T) {
	d := NewDiffer(60000)

	// First poll: not aborted
	d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, AbortedLastRun: false},
	})

	// Second poll: aborted
	events := d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, AbortedLastRun: true},
	})

	require.Len(t, events, 1)
	assert.Equal(t, collector.EventError, events[0].Event)
	assert.Equal(t, collector.StatusError, events[0].Status)
}

func TestDiffer_TokenIncrease(t *testing.T) {
	d := NewDiffer(60000)

	total1 := int64(1000)
	total2 := int64(2000)

	// First poll
	d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1",
			UpdatedAt: 1000, Age: 500, TotalTokens: &total1, InputTokens: 800, OutputTokens: 200},
	})

	// Second poll: tokens increased, updatedAt also changed
	events := d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1",
			UpdatedAt: 2000, Age: 200, TotalTokens: &total2, InputTokens: 1500, OutputTokens: 500},
	})

	// Should emit status.change with token usage in detail
	require.NotEmpty(t, events)
	found := false
	for _, e := range events {
		if e.Detail != nil && e.Detail.TokenUsage != nil {
			found = true
			assert.Equal(t, int64(1500), e.Detail.TokenUsage.Input)
			assert.Equal(t, int64(500), e.Detail.TokenUsage.Output)
		}
	}
	assert.True(t, found, "should have event with token usage")
}

func TestDiffer_NoChangeNoEvent(t *testing.T) {
	d := NewDiffer(60000)

	sess := OCSession{
		AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1",
		UpdatedAt: 1000, Age: 500,
	}

	d.Diff([]OCSession{sess})

	// Same data, no change
	events := d.Diff([]OCSession{sess})
	assert.Empty(t, events)
}

func TestDiffer_MultipleSessionsMixed(t *testing.T) {
	d := NewDiffer(60000)

	// First poll: two sessions
	d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
		{AgentID: "yoda", Key: "agent:yoda:discord:direct:456", SessionID: "sess-2", UpdatedAt: 1000, Age: 500},
	})

	// Second poll: sess-1 updated, sess-2 gone, sess-3 new
	events := d.Diff([]OCSession{
		{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 2000, Age: 200},
		{AgentID: "yoda", Key: "agent:yoda:cron:abc123", SessionID: "sess-3", UpdatedAt: 3000, Age: 100},
	})

	// Expect: sess-2 ended, sess-1 status change, sess-3 started
	var hasEnd, hasChange, hasStart bool
	for _, e := range events {
		switch {
		case e.SessionID == "sess-2" && e.Event == collector.EventSessionEnd:
			hasEnd = true
		case e.SessionID == "sess-1" && e.Event == collector.EventStatusChange:
			hasChange = true
		case e.SessionID == "sess-3" && e.Event == collector.EventSessionStart:
			hasStart = true
		}
	}
	assert.True(t, hasEnd, "sess-2 should end")
	assert.True(t, hasChange, "sess-1 should have status change")
	assert.True(t, hasStart, "sess-3 should start")
}
