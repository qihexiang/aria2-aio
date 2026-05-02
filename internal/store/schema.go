package store

import "database/sql"

const schemaSQL = `
CREATE TABLE IF NOT EXISTS app_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS instances (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    rpc_port    INTEGER NOT NULL,
    rpc_secret  TEXT NOT NULL,
    dir         TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'stopped',
    pid         INTEGER NOT NULL DEFAULT 0,
    config_json TEXT NOT NULL DEFAULT '{}',
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS task_history (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    instance_id      TEXT NOT NULL,
    gid              TEXT NOT NULL,
    name             TEXT NOT NULL DEFAULT '',
    uris             TEXT NOT NULL DEFAULT '',
    dir              TEXT NOT NULL DEFAULT '',
    files_json       TEXT NOT NULL DEFAULT '',
    total_length     INTEGER NOT NULL DEFAULT 0,
    completed_length INTEGER NOT NULL DEFAULT 0,
    download_speed   INTEGER NOT NULL DEFAULT 0,
    upload_length    INTEGER NOT NULL DEFAULT 0,
    status           TEXT NOT NULL,
    error_code       INTEGER NOT NULL DEFAULT 0,
    error_message    TEXT NOT NULL DEFAULT '',
    info_hash        TEXT NOT NULL DEFAULT '',
    completed_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_task_history_instance ON task_history(instance_id);
CREATE INDEX IF NOT EXISTS idx_task_history_completed_at ON task_history(completed_at);
CREATE INDEX IF NOT EXISTS idx_task_history_gid ON task_history(gid);
`

// Migration to add dir and files_json columns to existing task_history tables.
const migrationAddFilesColumns = `
ALTER TABLE task_history ADD COLUMN dir TEXT NOT NULL DEFAULT '';
ALTER TABLE task_history ADD COLUMN files_json TEXT NOT NULL DEFAULT '';
`

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return err
	}

	// Try adding new columns (will fail silently if they already exist in SQLite)
	db.Exec(migrationAddFilesColumns)
	return nil
}