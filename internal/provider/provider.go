package provider

import (
	"context"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// Provider is the interface that all session-monitoring sources must implement.
type Provider interface {
	Name() string
	Start(ctx context.Context, bus collector.EventBus) error
	Stop() error
}
