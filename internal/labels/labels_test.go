package labels

import (
	"testing"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/c3po-protocol1/holocron/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ApplyLabels tests ---

func TestApplyLabels_SingleRuleMatch(t *testing.T) {
	s := collector.SessionState{
		Source:    "claude-code",
		SessionID: "abc123",
		Workspace: "/home/user/Projects/holocron",
	}
	rules := []config.LabelRule{
		{
			Match: map[string]string{"source": "claude-code"},
			Set:   map[string]string{"project": "holocron"},
		},
	}

	ApplyLabels(&s, rules)

	assert.Equal(t, "holocron", s.Labels["project"])
}

func TestApplyLabels_MultipleRules_LaterOverrides(t *testing.T) {
	s := collector.SessionState{
		Source:    "claude-code",
		SessionID: "abc123",
		Workspace: "/home/user/Projects/holocron",
	}
	rules := []config.LabelRule{
		{
			Match: map[string]string{"source": "claude-code"},
			Set:   map[string]string{"agent": "default-agent"},
		},
		{
			Match: map[string]string{"workspace": "*/holocron*"},
			Set:   map[string]string{"agent": "holocron-agent"},
		},
	}

	ApplyLabels(&s, rules)

	assert.Equal(t, "holocron-agent", s.Labels["agent"])
}

func TestApplyLabels_GlobPatterns(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		pattern   string
		value     string
		wantMatch bool
	}{
		{"star prefix", "workspace", "*/holocron*", "/home/user/Projects/holocron", true},
		{"star suffix", "workspace", "/home/*", "/home/user/Projects", true},
		{"exact match", "source", "claude-code", "claude-code", true},
		{"no match", "source", "openclaw", "claude-code", false},
		{"star in middle", "workspace", "*/Projects/*", "/home/user/Projects/holocron", true},
		{"question mark", "sessionId", "abc?23", "abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := collector.SessionState{Labels: map[string]string{}}
			switch tt.field {
			case "source":
				s.Source = tt.value
			case "workspace":
				s.Workspace = tt.value
			case "sessionId":
				s.SessionID = tt.value
			}

			rules := []config.LabelRule{
				{
					Match: map[string]string{tt.field: tt.pattern},
					Set:   map[string]string{"matched": "yes"},
				},
			}

			ApplyLabels(&s, rules)

			if tt.wantMatch {
				assert.Equal(t, "yes", s.Labels["matched"])
			} else {
				assert.NotEqual(t, "yes", s.Labels["matched"])
			}
		})
	}
}

func TestApplyLabels_NoMatchingRules(t *testing.T) {
	s := collector.SessionState{
		Source:    "claude-code",
		SessionID: "abc123",
		Labels:   map[string]string{"existing": "value"},
	}
	rules := []config.LabelRule{
		{
			Match: map[string]string{"source": "openclaw"},
			Set:   map[string]string{"agent": "r2d2"},
		},
	}

	ApplyLabels(&s, rules)

	assert.Equal(t, "value", s.Labels["existing"])
	assert.Empty(t, s.Labels["agent"])
}

func TestApplyLabels_ProviderLabelsNotOverwrittenByEmptyRules(t *testing.T) {
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "abc123",
		Labels:   map[string]string{"agent": "r2d2", "channel": "discord"},
	}
	// Rule that matches but sets a different label — should not clear existing
	rules := []config.LabelRule{
		{
			Match: map[string]string{"source": "openclaw"},
			Set:   map[string]string{"project": "myproject"},
		},
	}

	ApplyLabels(&s, rules)

	assert.Equal(t, "r2d2", s.Labels["agent"])
	assert.Equal(t, "discord", s.Labels["channel"])
	assert.Equal(t, "myproject", s.Labels["project"])
}

func TestApplyLabels_NilLabelsMapInitialized(t *testing.T) {
	s := collector.SessionState{
		Source:    "claude-code",
		SessionID: "abc123",
	}
	rules := []config.LabelRule{
		{
			Match: map[string]string{"source": "claude-code"},
			Set:   map[string]string{"project": "test"},
		},
	}

	ApplyLabels(&s, rules)

	require.NotNil(t, s.Labels)
	assert.Equal(t, "test", s.Labels["project"])
}

func TestApplyLabels_SessionKeyMatch(t *testing.T) {
	s := collector.SessionState{
		Source:    "openclaw",
		SessionID: "sess-123",
		Labels:   map[string]string{"sessionKey": "agent:r2d2:discord:direct:1088"},
	}
	rules := []config.LabelRule{
		{
			Match: map[string]string{"sessionKey": "agent:r2d2:*"},
			Set:   map[string]string{"owner": "r2d2"},
		},
	}

	ApplyLabels(&s, rules)

	assert.Equal(t, "r2d2", s.Labels["owner"])
}

// --- GroupSessions tests ---

func TestGroupSessions_ByAgent(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Labels: map[string]string{"agent": "r2d2"}, Status: collector.StatusThinking},
		{SessionID: "s2", Labels: map[string]string{"agent": "yoda"}, Status: collector.StatusIdle},
		{SessionID: "s3", Labels: map[string]string{"agent": "r2d2"}, Status: collector.StatusIdle},
		{SessionID: "s4", Status: collector.StatusIdle}, // no agent label
	}

	groups := GroupSessions(sessions, GroupByAgent)

	require.Len(t, groups, 3) // r2d2, yoda, unlabeled

	// r2d2 has active session → should be first
	assert.Equal(t, "r2d2", groups[0].Label)
	assert.Len(t, groups[0].Sessions, 2)
	assert.Equal(t, 1, groups[0].Active)

	// yoda — no active sessions
	assert.Equal(t, "yoda", groups[1].Label)
	assert.Len(t, groups[1].Sessions, 1)
	assert.Equal(t, 0, groups[1].Active)

	// unlabeled always last
	assert.Equal(t, "unlabeled", groups[2].Label)
	assert.Len(t, groups[2].Sessions, 1)
	assert.Equal(t, 0, groups[2].Active)
}

func TestGroupSessions_ByChannel(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Labels: map[string]string{"channel": "discord"}, Status: collector.StatusThinking},
		{SessionID: "s2", Labels: map[string]string{"channel": "local"}, Status: collector.StatusIdle},
		{SessionID: "s3", Labels: map[string]string{"channel": "discord"}, Status: collector.StatusIdle},
		{SessionID: "s4", Labels: map[string]string{"channel": "cron"}, Status: collector.StatusThinking},
	}

	groups := GroupSessions(sessions, GroupByChannel)

	require.Len(t, groups, 3)

	// Groups with active sessions first (discord has 1 active, cron has 1 active)
	activeLabels := []string{groups[0].Label, groups[1].Label}
	assert.Contains(t, activeLabels, "discord")
	assert.Contains(t, activeLabels, "cron")

	// local has no active → last (no unlabeled here)
	assert.Equal(t, "local", groups[2].Label)
}

func TestGroupSessions_GroupNone(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Labels: map[string]string{"agent": "r2d2"}},
		{SessionID: "s2", Labels: map[string]string{"agent": "yoda"}},
	}

	groups := GroupSessions(sessions, GroupNone)

	require.Len(t, groups, 1)
	assert.Equal(t, "", groups[0].Label)
	assert.Len(t, groups[0].Sessions, 2)
}

func TestGroupSessions_EmptyInput(t *testing.T) {
	groups := GroupSessions(nil, GroupByAgent)
	assert.Empty(t, groups)
}

func TestGroupSessions_ActiveFirst_UnlabeledLast(t *testing.T) {
	sessions := []collector.SessionState{
		{SessionID: "s1", Labels: map[string]string{"agent": "yoda"}, Status: collector.StatusIdle},
		{SessionID: "s2", Labels: map[string]string{"agent": "r2d2"}, Status: collector.StatusThinking},
		{SessionID: "s3", Status: collector.StatusThinking}, // unlabeled but active
	}

	groups := GroupSessions(sessions, GroupByAgent)

	require.Len(t, groups, 3)
	// r2d2 active → first
	assert.Equal(t, "r2d2", groups[0].Label)
	// yoda not active → second
	assert.Equal(t, "yoda", groups[1].Label)
	// unlabeled always last regardless of activity
	assert.Equal(t, "unlabeled", groups[2].Label)
}

func TestGroupSessions_EmptyGroupsNotReturned(t *testing.T) {
	// When all sessions in a potential group are already filtered out,
	// that group shouldn't appear
	sessions := []collector.SessionState{
		{SessionID: "s1", Labels: map[string]string{"agent": "r2d2"}, Status: collector.StatusThinking},
	}

	groups := GroupSessions(sessions, GroupByAgent)

	require.Len(t, groups, 1)
	assert.Equal(t, "r2d2", groups[0].Label)
}

// --- CycleGroupMode tests ---

func TestCycleGroupMode(t *testing.T) {
	assert.Equal(t, GroupByAgent, CycleGroupMode(GroupNone))
	assert.Equal(t, GroupByChannel, CycleGroupMode(GroupByAgent))
	assert.Equal(t, GroupNone, CycleGroupMode(GroupByChannel))
}

// --- GroupMode String tests ---

func TestGroupMode_String(t *testing.T) {
	assert.Equal(t, "none", string(GroupNone))
	assert.Equal(t, "agent", string(GroupByAgent))
	assert.Equal(t, "channel", string(GroupByChannel))
}
