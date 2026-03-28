package openclaw

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatusOutput_Valid(t *testing.T) {
	resp := StatusResponse{
		RuntimeVersion: "1.2.3",
		Sessions: SessionsBlock{
			Count: 2,
			ByAgent: []AgentSessions{
				{
					AgentID: "r2d2",
					Count:   1,
					Recent: []OCSession{
						{
							AgentID:   "r2d2",
							Key:       "agent:r2d2:discord:direct:123",
							Kind:      "direct",
							SessionID: "sess-1",
							UpdatedAt: 1711000000000,
							Age:       5000,
							Model:     "opus-4",
						},
					},
				},
				{
					AgentID: "yoda",
					Count:   1,
					Recent: []OCSession{
						{
							AgentID:   "yoda",
							Key:       "agent:yoda:cron:abc",
							Kind:      "direct",
							SessionID: "sess-2",
							UpdatedAt: 1711000000000,
							Age:       120000,
							Model:     "opus-4",
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	sessions, err := ParseStatusOutput(data)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	assert.Equal(t, "r2d2", sessions[0].AgentID)
	assert.Equal(t, "sess-1", sessions[0].SessionID)
	assert.Equal(t, "yoda", sessions[1].AgentID)
	assert.Equal(t, "sess-2", sessions[1].SessionID)
}

func TestParseStatusOutput_Empty(t *testing.T) {
	resp := StatusResponse{
		RuntimeVersion: "1.0.0",
		Sessions: SessionsBlock{
			Count:   0,
			ByAgent: []AgentSessions{},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	sessions, err := ParseStatusOutput(data)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestParseStatusOutput_InvalidJSON(t *testing.T) {
	_, err := ParseStatusOutput([]byte("not json"))
	assert.Error(t, err)
}

func TestParseStatusOutput_NullableFields(t *testing.T) {
	total := int64(100000)
	remaining := int64(90000)
	pct := 10

	resp := StatusResponse{
		Sessions: SessionsBlock{
			Count: 1,
			ByAgent: []AgentSessions{
				{
					AgentID: "r2d2",
					Count:   1,
					Recent: []OCSession{
						{
							AgentID:         "r2d2",
							Key:             "agent:r2d2:discord:direct:123",
							SessionID:       "sess-1",
							TotalTokens:     &total,
							RemainingTokens: &remaining,
							PercentUsed:     &pct,
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	sessions, err := ParseStatusOutput(data)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	assert.Equal(t, int64(100000), *sessions[0].TotalTokens)
	assert.Equal(t, int64(90000), *sessions[0].RemainingTokens)
	assert.Equal(t, 10, *sessions[0].PercentUsed)
}
