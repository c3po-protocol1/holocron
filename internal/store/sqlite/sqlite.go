package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/c3po-protocol1/holocron/internal/collector"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements store.Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at dbPath and runs migrations.
func New(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Save(event collector.MonitorEvent) error {
	detailJSON, err := marshalNullable(event.Detail)
	if err != nil {
		return fmt.Errorf("marshaling detail: %w", err)
	}
	labelsJSON, err := marshalNullable(event.Labels)
	if err != nil {
		return fmt.Errorf("marshaling labels: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Extract rich content fields from detail
	var content, toolInput, toolOutput, role string
	if event.Detail != nil {
		content = event.Detail.Content
		toolInput = event.Detail.ToolInput
		toolOutput = event.Detail.ToolOutput
		role = event.Detail.Role
	}

	// Insert event
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO events (id, source, session_id, workspace, timestamp, event, status, detail_json, labels_json, content, tool_input, tool_output, role)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.Source, event.SessionID, event.Workspace,
		event.Timestamp, string(event.Event), string(event.Status),
		detailJSON, labelsJSON, content, toolInput, toolOutput, role,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}

	// Upsert session
	var tokenJSON *string
	if event.Detail != nil && event.Detail.TokenUsage != nil {
		b, err := json.Marshal(event.Detail.TokenUsage)
		if err != nil {
			return fmt.Errorf("marshaling token usage: %w", err)
		}
		s := string(b)
		tokenJSON = &s
	}

	_, err = tx.Exec(`
		INSERT INTO sessions (source, session_id, workspace, status, started_at, last_event_at, event_count, labels_json, token_json)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(source, session_id) DO UPDATE SET
			status        = excluded.status,
			workspace     = COALESCE(excluded.workspace, sessions.workspace),
			last_event_at = excluded.last_event_at,
			event_count   = sessions.event_count + 1,
			labels_json   = COALESCE(excluded.labels_json, sessions.labels_json),
			token_json    = COALESCE(excluded.token_json, sessions.token_json)`,
		event.Source, event.SessionID, event.Workspace,
		string(event.Status), event.Timestamp, event.Timestamp,
		labelsJSON, tokenJSON,
	)
	if err != nil {
		return fmt.Errorf("upserting session: %w", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) ListSessions() ([]collector.SessionState, error) {
	rows, err := s.db.Query(`
		SELECT source, session_id, workspace, status, started_at, last_event_at, event_count, labels_json, token_json
		FROM sessions
		ORDER BY last_event_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []collector.SessionState
	for rows.Next() {
		ss, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, ss)
	}
	return sessions, rows.Err()
}

func (s *SQLiteStore) GetSession(source, sessionID string) (*collector.SessionState, error) {
	row := s.db.QueryRow(`
		SELECT source, session_id, workspace, status, started_at, last_event_at, event_count, labels_json, token_json
		FROM sessions
		WHERE source = ? AND session_id = ?`,
		source, sessionID)

	ss, err := scanSessionRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

func (s *SQLiteStore) GetEvents(source, sessionID string, since int64, limit int) ([]collector.MonitorEvent, error) {
	rows, err := s.db.Query(`
		SELECT * FROM (
			SELECT id, source, session_id, workspace, timestamp, event, status, detail_json, labels_json, content, tool_input, tool_output, role
			FROM events
			WHERE source = ? AND session_id = ? AND timestamp >= ?
			ORDER BY timestamp DESC
			LIMIT ?
		) ORDER BY timestamp ASC`,
		source, sessionID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []collector.MonitorEvent
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

// TrimOldContent clears content, tool_input, and tool_output fields for events
// older than olderThanMs (Unix milliseconds). Metadata (id, type, role, etc.) is preserved.
func (s *SQLiteStore) TrimOldContent(olderThanMs int64) error {
	_, err := s.db.Exec(`
		UPDATE events SET content = '', tool_input = '', tool_output = ''
		WHERE timestamp < ? AND (content != '' OR tool_input != '' OR tool_output != '')`,
		olderThanMs,
	)
	return err
}

func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// --- helpers ---

func marshalNullable(v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSession(s scanner) (collector.SessionState, error) {
	var ss collector.SessionState
	var workspace sql.NullString
	var labelsJSON, tokenJSON sql.NullString
	var startedAt, lastEventAt sql.NullInt64

	err := s.Scan(
		&ss.Source, &ss.SessionID, &workspace,
		&ss.Status, &startedAt, &lastEventAt,
		&ss.EventCount, &labelsJSON, &tokenJSON,
	)
	if err != nil {
		return ss, err
	}

	ss.Workspace = workspace.String
	if startedAt.Valid {
		ss.StartedAt = startedAt.Int64
	}
	if lastEventAt.Valid {
		ss.LastEventAt = lastEventAt.Int64
	}
	if lastEventAt.Valid && startedAt.Valid {
		ss.ElapsedMs = lastEventAt.Int64 - startedAt.Int64
	}
	if labelsJSON.Valid {
		if err := json.Unmarshal([]byte(labelsJSON.String), &ss.Labels); err != nil {
			slog.Warn("corrupt session labels JSON", "session_id", ss.SessionID, "field", "labels_json", "err", err)
		}
	}
	if tokenJSON.Valid {
		var tu collector.TokenUsage
		if err := json.Unmarshal([]byte(tokenJSON.String), &tu); err != nil {
			slog.Warn("corrupt session token JSON", "session_id", ss.SessionID, "field", "token_json", "err", err)
		} else {
			ss.TokenUsage = &tu
		}
	}

	return ss, nil
}

func scanSessionRow(row *sql.Row) (collector.SessionState, error) {
	return scanSession(row)
}

func scanEvent(s scanner) (collector.MonitorEvent, error) {
	var ev collector.MonitorEvent
	var workspace sql.NullString
	var detailJSON, labelsJSON sql.NullString
	var content, toolInput, toolOutput, role string

	err := s.Scan(
		&ev.ID, &ev.Source, &ev.SessionID, &workspace,
		&ev.Timestamp, &ev.Event, &ev.Status,
		&detailJSON, &labelsJSON,
		&content, &toolInput, &toolOutput, &role,
	)
	if err != nil {
		return ev, err
	}

	ev.Workspace = workspace.String
	if detailJSON.Valid {
		var d collector.EventDetail
		if err := json.Unmarshal([]byte(detailJSON.String), &d); err != nil {
			slog.Warn("corrupt event detail JSON", "event_id", ev.ID, "field", "detail_json", "err", err)
		} else {
			ev.Detail = &d
		}
	}
	// Overlay rich content fields from dedicated columns onto Detail.
	if content != "" || toolInput != "" || toolOutput != "" || role != "" {
		if ev.Detail == nil {
			ev.Detail = &collector.EventDetail{}
		}
		ev.Detail.Content = content
		ev.Detail.ToolInput = toolInput
		ev.Detail.ToolOutput = toolOutput
		ev.Detail.Role = role
	}
	if labelsJSON.Valid {
		if err := json.Unmarshal([]byte(labelsJSON.String), &ev.Labels); err != nil {
			slog.Warn("corrupt event labels JSON", "event_id", ev.ID, "field", "labels_json", "err", err)
		}
	}

	return ev, nil
}
