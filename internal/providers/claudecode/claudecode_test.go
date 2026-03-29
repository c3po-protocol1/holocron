package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderName(t *testing.T) {
	p := New("/tmp/fake", 0)
	assert.Equal(t, "claude-code", p.Name())
}

// testBus collects published events for assertions.
type testBus struct {
	mu     sync.Mutex
	events []collector.MonitorEvent
}

func (b *testBus) Publish(event collector.MonitorEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

func (b *testBus) Subscribe() <-chan collector.MonitorEvent {
	return make(chan collector.MonitorEvent)
}

func (b *testBus) Unsubscribe(_ <-chan collector.MonitorEvent) {}

func (b *testBus) Events() []collector.MonitorEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]collector.MonitorEvent, len(b.events))
	copy(cp, b.events)
	return cp
}

func TestProviderStartStop(t *testing.T) {
	// Create mock session directory
	baseDir := t.TempDir()
	wsDir := filepath.Join(baseDir, "-Users-c-3po-Projects-test")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(wsDir, "sess-1.jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":"hi"},"sessionId":"sess-1"}`+"\n"),
		0o644,
	))

	bus := &testBus{}
	p := New(baseDir, 100*time.Millisecond)

	ctx := context.Background()
	err := p.Start(ctx, bus)
	require.NoError(t, err)

	// Wait for initial scan to emit session.start
	time.Sleep(200 * time.Millisecond)

	err = p.Stop()
	require.NoError(t, err)

	events := bus.Events()
	require.NotEmpty(t, events)

	// First event should be session.start
	assert.Equal(t, collector.EventSessionStart, events[0].Event)
	assert.Equal(t, "claude-code", events[0].Source)
	assert.Equal(t, "sess-1", events[0].SessionID)
}

func TestProviderDetectsNewFile(t *testing.T) {
	baseDir := t.TempDir()
	wsDir := filepath.Join(baseDir, "-Users-c-3po-Projects-test")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	bus := &testBus{}
	p := New(baseDir, 5*time.Second) // long poll so we don't get noise

	ctx := context.Background()
	err := p.Start(ctx, bus)
	require.NoError(t, err)

	// Give watcher time to start
	time.Sleep(200 * time.Millisecond)

	// Create a new session file — should be detected by fsnotify
	require.NoError(t, os.WriteFile(
		filepath.Join(wsDir, "new-sess.jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":"hello"},"sessionId":"new-sess"}`+"\n"),
		0o644,
	))

	// Wait for detection
	time.Sleep(500 * time.Millisecond)

	err = p.Stop()
	require.NoError(t, err)

	events := bus.Events()
	found := false
	for _, e := range events {
		if e.SessionID == "new-sess" && e.Event == collector.EventSessionStart {
			found = true
			break
		}
	}
	assert.True(t, found, "should detect new session file via fsnotify")
}

func TestProviderSessionStartHasChannelLocal(t *testing.T) {
	baseDir := t.TempDir()
	wsDir := filepath.Join(baseDir, "-Users-c-3po-Projects-test")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(wsDir, "sess-1.jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":"hi"},"sessionId":"sess-1"}`+"\n"),
		0o644,
	))

	bus := &testBus{}
	p := New(baseDir, 100*time.Millisecond)

	ctx := context.Background()
	require.NoError(t, p.Start(ctx, bus))
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, p.Stop())

	events := bus.Events()
	require.NotEmpty(t, events)

	// session.start event must have channel=local
	startEvent := events[0]
	assert.Equal(t, collector.EventSessionStart, startEvent.Event)
	require.NotNil(t, startEvent.Labels)
	assert.Equal(t, "local", startEvent.Labels["channel"], "session.start must have channel=local")
}

func TestProviderAllEventsHaveSource(t *testing.T) {
	baseDir := t.TempDir()
	wsDir := filepath.Join(baseDir, "-test")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(wsDir, "s1.jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":"hi"},"sessionId":"s1"}`+"\n"),
		0o644,
	))

	bus := &testBus{}
	p := New(baseDir, 5*time.Second)
	ctx := context.Background()
	require.NoError(t, p.Start(ctx, bus))
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, p.Stop())

	for _, e := range bus.Events() {
		assert.Equal(t, "claude-code", e.Source, "all events must have source=claude-code")
	}
}
