package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempDB(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func makeEvent(id, source, sessionID string, ts int64, evt collector.EventType, status collector.SessionStatus) collector.MonitorEvent {
	return collector.MonitorEvent{
		ID:        id,
		Source:    source,
		SessionID: sessionID,
		Workspace: "/projects/test",
		Timestamp: ts,
		Event:     evt,
		Status:    status,
	}
}

func TestSaveAndGetEvents(t *testing.T) {
	s := tempDB(t)
	now := time.Now().UnixMilli()

	ev := makeEvent("e1", "claude-code", "s1", now, collector.EventSessionStart, collector.StatusThinking)
	ev.Detail = &collector.EventDetail{
		Tool:    "Read",
		Target:  "main.go",
		Message: "reading file",
	}
	ev.Labels = map[string]string{"env": "test"}

	require.NoError(t, s.Save(ev))

	events, err := s.GetEvents("claude-code", "s1", 0, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)

	got := events[0]
	assert.Equal(t, "e1", got.ID)
	assert.Equal(t, "claude-code", got.Source)
	assert.Equal(t, "s1", got.SessionID)
	assert.Equal(t, "/projects/test", got.Workspace)
	assert.Equal(t, now, got.Timestamp)
	assert.Equal(t, collector.EventSessionStart, got.Event)
	assert.Equal(t, collector.StatusThinking, got.Status)
	require.NotNil(t, got.Detail)
	assert.Equal(t, "Read", got.Detail.Tool)
	assert.Equal(t, "main.go", got.Detail.Target)
	assert.Equal(t, "reading file", got.Detail.Message)
	require.NotNil(t, got.Labels)
	assert.Equal(t, "test", got.Labels["env"])
}

func TestSaveUpsertsSessionTable(t *testing.T) {
	s := tempDB(t)
	now := time.Now().UnixMilli()

	// First event starts a session
	ev1 := makeEvent("e1", "claude-code", "s1", now, collector.EventSessionStart, collector.StatusThinking)
	require.NoError(t, s.Save(ev1))

	sess, err := s.GetSession("claude-code", "s1")
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, collector.StatusThinking, sess.Status)
	assert.Equal(t, now, sess.StartedAt)
	assert.Equal(t, 1, sess.EventCount)

	// Second event updates session
	ev2 := makeEvent("e2", "claude-code", "s1", now+1000, collector.EventToolStart, collector.StatusToolRunning)
	ev2.Detail = &collector.EventDetail{Tool: "Edit", Target: "foo.go"}
	require.NoError(t, s.Save(ev2))

	sess, err = s.GetSession("claude-code", "s1")
	require.NoError(t, err)
	assert.Equal(t, collector.StatusToolRunning, sess.Status)
	assert.Equal(t, now, sess.StartedAt)      // unchanged
	assert.Equal(t, now+1000, sess.LastEventAt) // updated
	assert.Equal(t, 2, sess.EventCount)
}

func TestListSessions(t *testing.T) {
	s := tempDB(t)
	now := time.Now().UnixMilli()

	require.NoError(t, s.Save(makeEvent("e1", "claude-code", "s1", now, collector.EventSessionStart, collector.StatusThinking)))
	require.NoError(t, s.Save(makeEvent("e2", "openclaw", "s2", now+100, collector.EventSessionStart, collector.StatusIdle)))

	sessions, err := s.ListSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	// Should be ordered by last_event_at descending
	assert.Equal(t, "s2", sessions[0].SessionID)
	assert.Equal(t, "s1", sessions[1].SessionID)
}

func TestGetSessionNotFound(t *testing.T) {
	s := tempDB(t)

	sess, err := s.GetSession("nonexistent", "no-such-session")
	require.NoError(t, err)
	assert.Nil(t, sess)
}

func TestGetEventsWithSinceFilter(t *testing.T) {
	s := tempDB(t)
	base := int64(1000000)

	for i := 0; i < 5; i++ {
		ev := makeEvent(fmt.Sprintf("e%d", i), "claude-code", "s1", base+int64(i)*100, collector.EventMessage, collector.StatusThinking)
		require.NoError(t, s.Save(ev))
	}

	// Get events since timestamp 1000200 (should exclude first two)
	events, err := s.GetEvents("claude-code", "s1", base+200, 100)
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, "e2", events[0].ID)
}

func TestGetEventsWithLimit(t *testing.T) {
	s := tempDB(t)
	base := int64(1000000)

	for i := 0; i < 10; i++ {
		ev := makeEvent(fmt.Sprintf("e%d", i), "claude-code", "s1", base+int64(i)*100, collector.EventMessage, collector.StatusThinking)
		require.NoError(t, s.Save(ev))
	}

	events, err := s.GetEvents("claude-code", "s1", 0, 3)
	require.NoError(t, err)
	require.Len(t, events, 3)
	// Should be ordered by timestamp ascending
	assert.Equal(t, "e0", events[0].ID)
	assert.Equal(t, "e1", events[1].ID)
	assert.Equal(t, "e2", events[2].ID)
}

func TestGetEventsOrderedByTimestamp(t *testing.T) {
	s := tempDB(t)

	// Insert out of order
	require.NoError(t, s.Save(makeEvent("e3", "claude-code", "s1", 3000, collector.EventMessage, collector.StatusThinking)))
	require.NoError(t, s.Save(makeEvent("e1", "claude-code", "s1", 1000, collector.EventSessionStart, collector.StatusThinking)))
	require.NoError(t, s.Save(makeEvent("e2", "claude-code", "s1", 2000, collector.EventToolStart, collector.StatusToolRunning)))

	events, err := s.GetEvents("claude-code", "s1", 0, 100)
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, "e1", events[0].ID)
	assert.Equal(t, "e2", events[1].ID)
	assert.Equal(t, "e3", events[2].ID)
}

func TestTokenUsagePersistence(t *testing.T) {
	s := tempDB(t)
	now := time.Now().UnixMilli()

	ev := makeEvent("e1", "claude-code", "s1", now, collector.EventSessionEnd, collector.StatusDone)
	ev.Detail = &collector.EventDetail{
		TokenUsage: &collector.TokenUsage{
			Input:     1500,
			Output:    500,
			CacheRead: 200,
		},
	}
	require.NoError(t, s.Save(ev))

	events, err := s.GetEvents("claude-code", "s1", 0, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.NotNil(t, events[0].Detail)
	require.NotNil(t, events[0].Detail.TokenUsage)
	assert.Equal(t, int64(1500), events[0].Detail.TokenUsage.Input)
	assert.Equal(t, int64(500), events[0].Detail.TokenUsage.Output)
	assert.Equal(t, int64(200), events[0].Detail.TokenUsage.CacheRead)
}

func TestCloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
	require.NoError(t, err)

	require.NoError(t, s.Close())
	require.NoError(t, s.Close()) // second close should not error
}

func TestNewCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")
	s, err := New(dbPath)
	require.NoError(t, err)
	defer s.Close()

	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "database file should exist")
}
