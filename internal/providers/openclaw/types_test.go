package openclaw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSessionKey_DirectDiscord(t *testing.T) {
	info := ParseSessionKey("agent:r2d2:discord:direct:1088123456")
	assert.Equal(t, "r2d2", info.Agent)
	assert.Equal(t, "direct", info.SessionType)
	assert.Equal(t, "discord", info.Channel)
}

func TestParseSessionKey_GroupDiscord(t *testing.T) {
	info := ParseSessionKey("agent:yoda:discord:group:9991234")
	assert.Equal(t, "yoda", info.Agent)
	assert.Equal(t, "group", info.SessionType)
	assert.Equal(t, "discord", info.Channel)
}

func TestParseSessionKey_Cron(t *testing.T) {
	info := ParseSessionKey("agent:r2d2:cron:4b62933d")
	assert.Equal(t, "r2d2", info.Agent)
	assert.Equal(t, "cron", info.SessionType)
	assert.Equal(t, "", info.Channel)
}

func TestParseSessionKey_Subagent(t *testing.T) {
	info := ParseSessionKey("agent:r2d2:subagent:550e8400-e29b-41d4-a716-446655440000")
	assert.Equal(t, "r2d2", info.Agent)
	assert.Equal(t, "subagent", info.SessionType)
	assert.Equal(t, "", info.Channel)
}

func TestParseSessionKey_CronRun(t *testing.T) {
	info := ParseSessionKey("agent:r2d2:cron:4b62933d:run:550e8400-uuid")
	assert.Equal(t, "r2d2", info.Agent)
	assert.Equal(t, "cron_run", info.SessionType)
	assert.Equal(t, "", info.Channel)
}

func TestParseSessionKey_Invalid(t *testing.T) {
	info := ParseSessionKey("invalid-key")
	assert.Equal(t, "unknown", info.SessionType)
}

func TestParseSessionKey_TooShort(t *testing.T) {
	info := ParseSessionKey("agent:r2d2")
	assert.Equal(t, "unknown", info.SessionType)
}

func TestSessionKeyInfo_ToLabels(t *testing.T) {
	info := SessionKeyInfo{
		Agent:       "r2d2",
		SessionType: "direct",
		Channel:     "discord",
	}
	labels := info.ToLabels()
	assert.Equal(t, "r2d2", labels["agent"])
	assert.Equal(t, "direct", labels["session_type"])
	assert.Equal(t, "discord", labels["channel"])
}

func TestSessionKeyInfo_ToLabels_NoChannel(t *testing.T) {
	info := SessionKeyInfo{
		Agent:       "r2d2",
		SessionType: "cron",
	}
	labels := info.ToLabels()
	assert.Equal(t, "r2d2", labels["agent"])
	assert.Equal(t, "cron", labels["session_type"])
	_, hasChannel := labels["channel"]
	assert.False(t, hasChannel)
}
