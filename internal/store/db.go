// Package store provides the SQLite + sqlite-vec storage layer.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"

	"github.com/sgx-labs/statelessagent/internal/config"
)

func init() {
	sqlite_vec.Auto()
}

// DB wraps a SQLite connection with sqlite-vec support.
type DB struct {
	conn *sql.DB
	mu   sync.Mutex // serialize writes
}

// Open opens or creates the database at the configured path.
func Open() (*DB, error) {
	return OpenPath(config.DBPath())
}

// OpenPath opens or creates the database at the given path.
func OpenPath(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Verify sqlite-vec is loaded
	var vecVersion string
	if err := conn.QueryRow("SELECT vec_version()").Scan(&vecVersion); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sqlite-vec not available: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// OpenMemory opens an in-memory database for testing.
func OpenMemory() (*DB, error) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying sql.DB for direct queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS vault_notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			title TEXT NOT NULL,
			tags TEXT DEFAULT '[]',
			domain TEXT DEFAULT '',
			workstream TEXT DEFAULT '',
			chunk_id INTEGER NOT NULL,
			chunk_heading TEXT NOT NULL,
			text TEXT NOT NULL,
			modified REAL NOT NULL,
			content_hash TEXT NOT NULL,
			content_type TEXT DEFAULT 'note',
			review_by TEXT DEFAULT '',
			confidence REAL DEFAULT 0.5,
			access_count INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_notes_path ON vault_notes(path)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_notes_content_hash ON vault_notes(content_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_notes_content_type ON vault_notes(content_type)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_notes_domain ON vault_notes(domain)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_notes_workstream ON vault_notes(workstream)`,

		fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vault_notes_vec USING vec0(
			note_id INTEGER PRIMARY KEY,
			embedding float[%d]
		)`, config.EmbeddingDim),

		`CREATE TABLE IF NOT EXISTS session_log (
			session_id TEXT PRIMARY KEY,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			handoff_path TEXT DEFAULT '',
			machine TEXT DEFAULT '',
			files_changed TEXT DEFAULT '[]',
			summary TEXT DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS context_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			hook_name TEXT NOT NULL,
			injected_paths TEXT DEFAULT '[]',
			estimated_tokens INTEGER DEFAULT 0,
			was_referenced INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_context_usage_session ON context_usage(session_id)`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}
