package openclaw

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// fakeRunner returns canned JSON responses, cycling through them.
type fakeRunner struct {
	mu        sync.Mutex
	responses []StatusResponse
	callCount int
}

func (r *fakeRunner) Run(_ context.Context) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	idx := r.callCount
	if idx >= len(r.responses) {
		idx = len(r.responses) - 1
	}
	r.callCount++
	return json.Marshal(r.responses[idx])
}

func (r *fakeRunner) CallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.callCount
}

func TestProviderName(t *testing.T) {
	p := New(Options{})
	assert.Equal(t, "openclaw", p.Name())
}

func TestProviderStartStop(t *testing.T) {
	runner := &fakeRunner{
		responses: []StatusResponse{
			{
				Sessions: SessionsBlock{
					Count: 1,
					ByAgent: []AgentSessions{
						{
							AgentID: "r2d2",
							Count:   1,
							Recent: []OCSession{
								{
									AgentID:   "r2d2",
									Key:       "agent:r2d2:discord:direct:123",
									SessionID: "sess-1",
									UpdatedAt: 1000,
									Age:       500,
									Model:     "opus-4",
								},
							},
						},
					},
				},
			},
		},
	}

	bus := &testBus{}
	p := New(Options{
		PollInterval:    100 * time.Millisecond,
		IdleThresholdMs: 60000,
		Runner:          runner,
	})

	ctx := context.Background()
	err := p.Start(ctx, bus)
	require.NoError(t, err)

	// Wait for initial poll + at least one tick
	time.Sleep(250 * time.Millisecond)

	err = p.Stop()
	require.NoError(t, err)

	events := bus.Events()
	require.NotEmpty(t, events)

	// First event should be session.start
	assert.Equal(t, collector.EventSessionStart, events[0].Event)
	assert.Equal(t, "openclaw", events[0].Source)
	assert.Equal(t, "sess-1", events[0].SessionID)
	assert.Equal(t, "r2d2", events[0].Labels["agent"])
}

func TestProviderDetectsChanges(t *testing.T) {
	runner := &fakeRunner{
		responses: []StatusResponse{
			// Poll 1: one session
			{
				Sessions: SessionsBlock{
					Count: 1,
					ByAgent: []AgentSessions{
						{AgentID: "r2d2", Count: 1, Recent: []OCSession{
							{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
						}},
					},
				},
			},
			// Poll 2: session updated
			{
				Sessions: SessionsBlock{
					Count: 1,
					ByAgent: []AgentSessions{
						{AgentID: "r2d2", Count: 1, Recent: []OCSession{
							{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 2000, Age: 200},
						}},
					},
				},
			},
		},
	}

	bus := &testBus{}
	p := New(Options{
		PollInterval:    100 * time.Millisecond,
		IdleThresholdMs: 60000,
		Runner:          runner,
	})

	ctx := context.Background()
	require.NoError(t, p.Start(ctx, bus))
	time.Sleep(350 * time.Millisecond)
	require.NoError(t, p.Stop())

	events := bus.Events()

	// Should have session.start and then status.change
	var hasStart, hasChange bool
	for _, e := range events {
		if e.Event == collector.EventSessionStart && e.SessionID == "sess-1" {
			hasStart = true
		}
		if e.Event == collector.EventStatusChange && e.SessionID == "sess-1" {
			hasChange = true
		}
	}
	assert.True(t, hasStart, "should emit session.start")
	assert.True(t, hasChange, "should emit status.change on updatedAt change")
}

func TestProviderAllEventsHaveSource(t *testing.T) {
	runner := &fakeRunner{
		responses: []StatusResponse{
			{
				Sessions: SessionsBlock{
					Count: 1,
					ByAgent: []AgentSessions{
						{AgentID: "r2d2", Count: 1, Recent: []OCSession{
							{AgentID: "r2d2", Key: "agent:r2d2:discord:direct:123", SessionID: "sess-1", UpdatedAt: 1000, Age: 500},
						}},
					},
				},
			},
		},
	}

	bus := &testBus{}
	p := New(Options{
		PollInterval:    100 * time.Millisecond,
		IdleThresholdMs: 60000,
		Runner:          runner,
	})

	ctx := context.Background()
	require.NoError(t, p.Start(ctx, bus))
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, p.Stop())

	for _, e := range bus.Events() {
		assert.Equal(t, "openclaw", e.Source, "all events must have source=openclaw")
	}
}

func TestProviderStopIsIdempotent(t *testing.T) {
	p := New(Options{})
	// Stop before start should not panic
	assert.NoError(t, p.Stop())
}

func TestProviderNewSessionAppears(t *testing.T) {
	runner := &fakeRunner{
		responses: []StatusResponse{
			// Poll 1: empty
			{Sessions: SessionsBlock{Count: 0, ByAgent: []AgentSessions{}}},
			// Poll 2: new session appears
			{
				Sessions: SessionsBlock{
					Count: 1,
					ByAgent: []AgentSessions{
						{AgentID: "yoda", Count: 1, Recent: []OCSession{
							{AgentID: "yoda", Key: "agent:yoda:discord:direct:789", SessionID: "sess-new", UpdatedAt: 5000, Age: 100},
						}},
					},
				},
			},
		},
	}

	bus := &testBus{}
	p := New(Options{
		PollInterval:    100 * time.Millisecond,
		IdleThresholdMs: 60000,
		Runner:          runner,
	})

	ctx := context.Background()
	require.NoError(t, p.Start(ctx, bus))
	time.Sleep(350 * time.Millisecond)
	require.NoError(t, p.Stop())

	events := bus.Events()
	found := false
	for _, e := range events {
		if e.SessionID == "sess-new" && e.Event == collector.EventSessionStart {
			found = true
			assert.Equal(t, "yoda", e.Labels["agent"])
		}
	}
	assert.True(t, found, "should detect new session appearing")
}
