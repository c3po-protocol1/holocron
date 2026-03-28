package openclaw

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// DefaultPollInterval is the default polling interval.
const DefaultPollInterval = 5 * time.Second

// DefaultIdleThresholdMs is the default idle threshold (1 minute).
const DefaultIdleThresholdMs = 60000

// Options configures the OpenClaw provider.
type Options struct {
	PollInterval    time.Duration
	IdleThresholdMs int64
	Runner          CommandRunner // nil → uses ExecRunner
}

// Provider monitors OpenClaw agent sessions via polling.
type Provider struct {
	pollInterval    time.Duration
	idleThresholdMs int64
	runner          CommandRunner

	bus    collector.EventBus
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new OpenClaw provider.
func New(opts Options) *Provider {
	interval := opts.PollInterval
	if interval == 0 {
		interval = DefaultPollInterval
	}
	threshold := opts.IdleThresholdMs
	if threshold == 0 {
		threshold = DefaultIdleThresholdMs
	}
	runner := opts.Runner
	if runner == nil {
		runner = &ExecRunner{}
	}
	return &Provider{
		pollInterval:    interval,
		idleThresholdMs: threshold,
		runner:          runner,
	}
}

func (p *Provider) Name() string { return sourceName }

func (p *Provider) Start(ctx context.Context, bus collector.EventBus) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.bus = bus

	differ := NewDiffer(p.idleThresholdMs)

	// Initial poll
	sessions, err := Poll(ctx, p.runner)
	if err != nil {
		slog.Warn("openclaw initial poll failed", "error", err)
	} else {
		events := differ.Diff(sessions)
		for _, ev := range events {
			bus.Publish(ev)
		}
	}

	// Start polling goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.pollLoop(ctx, differ)
	}()

	return nil
}

func (p *Provider) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	return nil
}

func (p *Provider) pollLoop(ctx context.Context, differ *Differ) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sessions, err := Poll(ctx, p.runner)
			if err != nil {
				slog.Warn("openclaw poll failed", "error", err)
				continue
			}
			events := differ.Diff(sessions)
			for _, ev := range events {
				p.bus.Publish(ev)
			}
		}
	}
}
