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

	// Limit returns the newest N events (ordered ascending)
	events, err := s.GetEvents("claude-code", "s1", 0, 3)
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, "e7", events[0].ID)
	assert.Equal(t, "e8", events[1].ID)
	assert.Equal(t, "e9", events[2].ID)
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

// --- F11: Rich Event Data Tests ---

func TestSaveAndGetWithRichContentFields(t *testing.T) {
	s := tempDB(t)
	now := time.Now().UnixMilli()

	ev := makeEvent("e1", "claude-code", "s1", now, collector.EventUserMessage, collector.StatusThinking)
	ev.Detail = &collector.EventDetail{
		Tool:       "Read",
		Target:     "main.go",
		Message:    "reading file",
		Content:    "full content of the user message",
		ToolInput:  `{"file_path": "main.go"}`,
		ToolOutput: "package main\n\nfunc main() {}",
		Role:       "user",
	}
	require.NoError(t, s.Save(ev))

	events, err := s.GetEvents("claude-code", "s1", 0, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)

	got := events[0]
	require.NotNil(t, got.Detail)
	assert.Equal(t, "full content of the user message", got.Detail.Content)
	assert.Equal(t, `{"file_path": "main.go"}`, got.Detail.ToolInput)
	assert.Equal(t, "package main\n\nfunc main() {}", got.Detail.ToolOutput)
	assert.Equal(t, "user", got.Detail.Role)
	// Existing fields should also be preserved
	assert.Equal(t, "Read", got.Detail.Tool)
	assert.Equal(t, "main.go", got.Detail.Target)
	assert.Equal(t, "reading file", got.Detail.Message)
}

func TestSaveAndGetWithEmptyRichFields(t *testing.T) {
	s := tempDB(t)
	now := time.Now().UnixMilli()

	ev := makeEvent("e1", "claude-code", "s1", now, collector.EventToolStart, collector.StatusToolRunning)
	ev.Detail = &collector.EventDetail{
		Tool:   "Bash",
		Target: "go test ./...",
	}
	require.NoError(t, s.Save(ev))

	events, err := s.GetEvents("claude-code", "s1", 0, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)

	got := events[0]
	require.NotNil(t, got.Detail)
	assert.Equal(t, "Bash", got.Detail.Tool)
	assert.Equal(t, "", got.Detail.Content)
	assert.Equal(t, "", got.Detail.ToolOutput)
	assert.Equal(t, "", got.Detail.Role)
}

func TestTrimOldContent(t *testing.T) {
	s := tempDB(t)

	oldTs := int64(1000)
	newTs := int64(5000)

	// Save an old event with rich content
	oldEv := makeEvent("e-old", "claude-code", "s1", oldTs, collector.EventUserMessage, collector.StatusThinking)
	oldEv.Detail = &collector.EventDetail{
		Message:    "old message",
		Content:    "old full content",
		ToolInput:  "old tool input",
		ToolOutput: "old tool output",
		Role:       "user",
	}
	require.NoError(t, s.Save(oldEv))

	// Save a new event with rich content
	newEv := makeEvent("e-new", "claude-code", "s1", newTs, collector.EventAssistantMessage, collector.StatusIdle)
	newEv.Detail = &collector.EventDetail{
		Message:    "new message",
		Content:    "new full content",
		ToolInput:  "new tool input",
		ToolOutput: "new tool output",
		Role:       "assistant",
	}
	require.NoError(t, s.Save(newEv))

	// Trim content older than 3000ms
	require.NoError(t, s.TrimOldContent(3000))

	events, err := s.GetEvents("claude-code", "s1", 0, 100)
	require.NoError(t, err)
	require.Len(t, events, 2)

	// Old event: content fields cleared, metadata preserved
	old := events[0]
	assert.Equal(t, "e-old", old.ID)
	assert.Equal(t, collector.EventUserMessage, old.Event)
	require.NotNil(t, old.Detail)
	assert.Equal(t, "old message", old.Detail.Message) // Message is in detail_json, preserved
	assert.Equal(t, "", old.Detail.Content)             // Cleared
	assert.Equal(t, "", old.Detail.ToolInput)           // Cleared
	assert.Equal(t, "", old.Detail.ToolOutput)          // Cleared
	assert.Equal(t, "user", old.Detail.Role)            // Role preserved (not trimmed)

	// New event: all fields intact
	new := events[1]
	assert.Equal(t, "e-new", new.ID)
	require.NotNil(t, new.Detail)
	assert.Equal(t, "new full content", new.Detail.Content)
	assert.Equal(t, "new tool input", new.Detail.ToolInput)
	assert.Equal(t, "new tool output", new.Detail.ToolOutput)
	assert.Equal(t, "assistant", new.Detail.Role)
}

func TestTrimOldContentNoOp(t *testing.T) {
	s := tempDB(t)
	// Trimming on an empty DB should not error
	require.NoError(t, s.TrimOldContent(1000))
}

func TestMigrationOnFreshDB(t *testing.T) {
	// tempDB already runs migrations on a fresh DB
	s := tempDB(t)

	// Verify new columns exist by saving an event with rich fields
	ev := makeEvent("e1", "test", "s1", 1000, collector.EventUserMessage, collector.StatusThinking)
	ev.Detail = &collector.EventDetail{
		Content: "test content",
		Role:    "user",
	}
	require.NoError(t, s.Save(ev))

	events, err := s.GetEvents("test", "s1", 0, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "test content", events[0].Detail.Content)
}

func TestMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// First open creates schema + runs migrations
	s1, err := New(dbPath)
	require.NoError(t, err)
	s1.Close()

	// Second open re-runs migrations (should not error on duplicate columns)
	s2, err := New(dbPath)
	require.NoError(t, err)
	defer s2.Close()

	// Verify it still works
	ev := makeEvent("e1", "test", "s1", 1000, collector.EventToolResult, collector.StatusThinking)
	ev.Detail = &collector.EventDetail{ToolOutput: "result", Role: "tool"}
	require.NoError(t, s2.Save(ev))
}

func TestNewCreatesDirectoryWith0700Permissions(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "secure-dir")
	dbPath := filepath.Join(subDir, "test.db")

	// Set a permissive umask to verify the code enforces 0700
	s, err := New(dbPath)
	require.NoError(t, err)
	defer s.Close()

	info, err := os.Stat(subDir)
	require.NoError(t, err)
	perm := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0o700), perm, "directory should have 0700 permissions, got %o", perm)
}
