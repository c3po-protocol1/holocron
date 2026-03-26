package collector

import (
	"log/slog"
	"sync"
)

const subscriberBufferSize = 256

// EventBus is the interface for in-process event routing.
type EventBus interface {
	Publish(event MonitorEvent)
	Subscribe() <-chan MonitorEvent
	Unsubscribe(ch <-chan MonitorEvent)
}

type eventBus struct {
	mu          sync.RWMutex
	subscribers map[chan MonitorEvent]struct{}
}

// NewEventBus creates a new channel-based EventBus.
func NewEventBus() EventBus {
	return &eventBus{
		subscribers: make(map[chan MonitorEvent]struct{}),
	}
}

func (b *eventBus) Publish(event MonitorEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			slog.Warn("dropping event for slow subscriber",
				"eventID", event.ID,
				"source", event.Source,
			)
		}
	}
}

func (b *eventBus) Subscribe() <-chan MonitorEvent {
	ch := make(chan MonitorEvent, subscriberBufferSize)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *eventBus) Unsubscribe(ch <-chan MonitorEvent) {
	b.mu.Lock()
	for sub := range b.subscribers {
		// Compare the bidirectional channel as receive-only to match the caller's reference.
		if (<-chan MonitorEvent)(sub) == ch {
			delete(b.subscribers, sub)
			b.mu.Unlock()
			close(sub)
			return
		}
	}
	b.mu.Unlock()
}
