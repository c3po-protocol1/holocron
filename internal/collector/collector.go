package collector

import (
	"context"
	"log/slog"
	"sync"
)

// CollectorProvider is the interface for providers used by the Collector.
// This mirrors provider.Provider but avoids a circular import.
type CollectorProvider interface {
	Name() string
	Start(ctx context.Context, bus EventBus) error
	Stop() error
}

// Store is the interface for persisting events.
// This mirrors store.Store but avoids a circular import.
type Store interface {
	Save(event MonitorEvent) error
	ListSessions() ([]SessionState, error)
	GetSession(source, sessionID string) (*SessionState, error)
	GetEvents(source, sessionID string, since int64, limit int) ([]MonitorEvent, error)
	Close() error
}

// Collector wires providers → EventBus → Store.
type Collector struct {
	bus       EventBus
	store     Store
	providers []CollectorProvider
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New creates a new Collector. Store may be nil (no persistence).
func New(s Store) *Collector {
	return &Collector{
		bus:   NewEventBus(),
		store: s,
	}
}

// AddProvider registers a provider to be started by the Collector.
func (c *Collector) AddProvider(p CollectorProvider) {
	c.providers = append(c.providers, p)
}

// Start starts all providers and the store subscriber goroutine.
func (c *Collector) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Start store subscriber goroutine
	if c.store != nil {
		storeCh := c.bus.Subscribe()
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			defer c.bus.Unsubscribe(storeCh)
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-storeCh:
					if !ok {
						return
					}
					if err := c.store.Save(ev); err != nil {
						slog.Warn("failed to persist event", "error", err, "eventID", ev.ID)
					}
				}
			}
		}()
	}

	// Start all providers
	for _, p := range c.providers {
		if err := p.Start(ctx, c.bus); err != nil {
			slog.Warn("provider failed to start", "provider", p.Name(), "error", err)
		}
	}

	return nil
}

// Subscribe returns a channel that receives all events from the bus.
func (c *Collector) Subscribe() <-chan MonitorEvent {
	return c.bus.Subscribe()
}

// Stop cancels the context and stops all providers.
func (c *Collector) Stop() {
	if c.cancel != nil {
		c.cancel()
	}

	for _, p := range c.providers {
		if err := p.Stop(); err != nil {
			slog.Warn("provider failed to stop", "provider", p.Name(), "error", err)
		}
	}

	c.wg.Wait()
}
