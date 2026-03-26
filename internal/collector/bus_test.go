package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEvent(id string) MonitorEvent {
	return MonitorEvent{
		ID:        id,
		Source:    "claude-code",
		SessionID: "sess-1",
		Timestamp: time.Now().UnixMilli(),
		Event:     EventStatusChange,
		Status:    StatusThinking,
	}
}

func TestSubscribeReceivesPublishedEvent(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe()

	ev := makeEvent("evt-1")
	bus.Publish(ev)

	select {
	case got := <-ch:
		assert.Equal(t, "evt-1", got.ID)
		assert.Equal(t, EventStatusChange, got.Event)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestMultipleSubscribersEachGetEveryEvent(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	ev := makeEvent("evt-2")
	bus.Publish(ev)

	for i, ch := range []<-chan MonitorEvent{ch1, ch2} {
		select {
		case got := <-ch:
			assert.Equal(t, "evt-2", got.ID, "subscriber %d", i)
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out", i)
		}
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe()
	bus.Unsubscribe(ch)

	bus.Publish(makeEvent("evt-3"))

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("should not receive event after unsubscribe")
		}
		// channel closed — expected
	case <-time.After(100 * time.Millisecond):
		// no event received — also acceptable
	}
}

func TestSlowSubscriberDoesNotBlockPublisher(t *testing.T) {
	bus := NewEventBus()
	_ = bus.Subscribe() // subscribe but never read

	done := make(chan struct{})
	go func() {
		// Publish more events than the buffer size (256)
		for i := 0; i < 300; i++ {
			bus.Publish(makeEvent("flood"))
		}
		close(done)
	}()

	select {
	case <-done:
		// publisher was not blocked — pass
	case <-time.After(2 * time.Second):
		t.Fatal("publisher blocked on slow subscriber")
	}
}

func TestUnsubscribeOnlyAffectsTargetChannel(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()
	bus.Unsubscribe(ch1)

	bus.Publish(makeEvent("evt-4"))

	select {
	case got := <-ch2:
		assert.Equal(t, "evt-4", got.ID)
	case <-time.After(time.Second):
		t.Fatal("remaining subscriber should still receive events")
	}
}

func TestSubscribeReturnsDistinctChannels(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()
	require.NotEqual(t, ch1, ch2, "each subscription should return a unique channel")
}
