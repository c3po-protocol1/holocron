package collector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProvider implements provider.Provider for testing.
type fakeProvider struct {
	name    string
	started bool
	stopped bool
	bus     EventBus
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Start(ctx context.Context, bus EventBus) error {
	f.started = true
	f.bus = bus
	return nil
}

func (f *fakeProvider) Stop() error {
	f.stopped = true
	return nil
}

// fakeStore implements store.Store for testing.
type fakeStore struct {
	events []MonitorEvent
}

func (s *fakeStore) Save(event MonitorEvent) error {
	s.events = append(s.events, event)
	return nil
}

func (s *fakeStore) ListSessions() ([]SessionState, error) {
	return nil, nil
}

func (s *fakeStore) GetSession(source, sessionID string) (*SessionState, error) {
	return nil, nil
}

func (s *fakeStore) GetEvents(source, sessionID string, since int64, limit int) ([]MonitorEvent, error) {
	return nil, nil
}

func (s *fakeStore) TrimOldContent(olderThanMs int64) error {
	return nil
}

func (s *fakeStore) Close() error {
	return nil
}

func TestCollectorStartsProviders(t *testing.T) {
	st := &fakeStore{}
	c := New(st)

	p1 := &fakeProvider{name: "p1"}
	p2 := &fakeProvider{name: "p2"}
	c.AddProvider(p1)
	c.AddProvider(p2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)
	defer c.Stop()

	assert.True(t, p1.started, "provider p1 should be started")
	assert.True(t, p2.started, "provider p2 should be started")
}

func TestCollectorStopsProviders(t *testing.T) {
	st := &fakeStore{}
	c := New(st)

	p1 := &fakeProvider{name: "p1"}
	c.AddProvider(p1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)

	c.Stop()
	assert.True(t, p1.stopped, "provider should be stopped")
}

func TestCollectorPersistsEventsToStore(t *testing.T) {
	st := &fakeStore{}
	c := New(st)

	p := &fakeProvider{name: "test"}
	c.AddProvider(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)
	defer c.Stop()

	// Provider publishes an event through the bus
	ev := MonitorEvent{
		ID:        "evt-1",
		Source:    "test",
		SessionID: "sess-1",
		Timestamp: time.Now().UnixMilli(),
		Event:     EventStatusChange,
		Status:    StatusThinking,
	}
	p.bus.Publish(ev)

	// Give the store subscriber goroutine time to process
	time.Sleep(50 * time.Millisecond)

	require.Len(t, st.events, 1)
	assert.Equal(t, "evt-1", st.events[0].ID)
}

func TestCollectorSubscribeReceivesEvents(t *testing.T) {
	st := &fakeStore{}
	c := New(st)

	p := &fakeProvider{name: "test"}
	c.AddProvider(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)
	defer c.Stop()

	ch := c.Subscribe()

	ev := MonitorEvent{
		ID:        "evt-2",
		Source:    "test",
		SessionID: "sess-2",
		Timestamp: time.Now().UnixMilli(),
		Event:     EventSessionStart,
		Status:    StatusIdle,
	}
	p.bus.Publish(ev)

	select {
	case got := <-ch:
		assert.Equal(t, "evt-2", got.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event on subscriber channel")
	}
}

func TestCollectorNewWithNilStore(t *testing.T) {
	// Collector should work even with nil store (no persistence)
	c := New(nil)
	p := &fakeProvider{name: "test"}
	c.AddProvider(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)

	ch := c.Subscribe()

	ev := MonitorEvent{
		ID:        "evt-3",
		Source:    "test",
		SessionID: "sess-3",
		Timestamp: time.Now().UnixMilli(),
		Event:     EventSessionStart,
		Status:    StatusIdle,
	}
	p.bus.Publish(ev)

	select {
	case got := <-ch:
		assert.Equal(t, "evt-3", got.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	c.Stop()
	assert.True(t, p.stopped)
}
