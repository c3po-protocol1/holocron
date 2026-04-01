package store

import "github.com/c3po-protocol1/holocron/internal/collector"

// Store persists and queries monitor events and session state.
type Store interface {
	Save(event collector.MonitorEvent) error
	ListSessions() ([]collector.SessionState, error)
	GetSession(source, sessionID string) (*collector.SessionState, error)
	GetEvents(source, sessionID string, since int64, limit int) ([]collector.MonitorEvent, error)
	TrimOldContent(olderThanMs int64) error
	Close() error
}
