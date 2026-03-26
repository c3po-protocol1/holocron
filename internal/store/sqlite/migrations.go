package sqlite

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS events (
    id          TEXT PRIMARY KEY,
    source      TEXT NOT NULL,
    session_id  TEXT NOT NULL,
    workspace   TEXT,
    timestamp   INTEGER NOT NULL,
    event       TEXT NOT NULL,
    status      TEXT NOT NULL,
    detail_json TEXT,
    labels_json TEXT,
    created_at  INTEGER DEFAULT (strftime('%s','now') * 1000)
);
CREATE INDEX IF NOT EXISTS idx_events_session ON events(source, session_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);

CREATE TABLE IF NOT EXISTS sessions (
    source        TEXT NOT NULL,
    session_id    TEXT NOT NULL,
    workspace     TEXT,
    status        TEXT NOT NULL,
    started_at    INTEGER,
    last_event_at INTEGER,
    event_count   INTEGER DEFAULT 0,
    labels_json   TEXT,
    token_json    TEXT,
    PRIMARY KEY (source, session_id)
);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
