package repository

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schemaVersion = 3

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// NewDB creates a new SQLite database connection with WAL mode enabled.
func NewDB(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(1) // SQLite only supports one writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0) // Don't expire connections

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) migrate() error {
	// Create schema_migrations table
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	// Run migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{1, migrationV1},
		{2, migrationV2},
		{3, migrationV3},
	}

	for _, m := range migrations {
		if m.version > currentVersion {
			log.Printf("[DB] Applying migration v%d", m.version)
			if _, err := db.conn.Exec(m.sql); err != nil {
				return fmt.Errorf("migration v%d failed: %w", m.version, err)
			}
			if _, err := db.conn.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
				return fmt.Errorf("failed to record migration v%d: %w", m.version, err)
			}
		}
	}

	return nil
}

// StartPurgeLoop starts a goroutine that purges old records every hour.
func (db *DB) StartPurgeLoop(retentionDays int) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run immediately on start
		db.purgeOldRecords(retentionDays)

		for range ticker.C {
			db.purgeOldRecords(retentionDays)
		}
	}()
}

func (db *DB) purgeOldRecords(retentionDays int) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result, err := db.conn.Exec("DELETE FROM spike_events WHERE created_at < ?", cutoff)
	if err != nil {
		log.Printf("[DB] Error purging old records: %v", err)
		return
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[DB] Purged %d spike events older than %d days", rows, retentionDays)
	}

	// Purge old metrics cache
	result, err = db.conn.Exec("DELETE FROM metrics_cache WHERE created_at < ?", cutoff)
	if err != nil {
		log.Printf("[DB] Error purging metrics cache: %v", err)
		return
	}

	rows, _ = result.RowsAffected()
	if rows > 0 {
		log.Printf("[DB] Purged %d metrics cache entries older than %d days", rows, retentionDays)
	}
}

// Migration V1: Initial schema
const migrationV1 = `
CREATE TABLE IF NOT EXISTS spike_events (
	id TEXT PRIMARY KEY,
	timestamp DATETIME NOT NULL,
	deployment_name TEXT NOT NULL,
	namespace TEXT NOT NULL,
	cpu_usage_percent REAL NOT NULL,
	cpu_limit_percent REAL DEFAULT 0,
	ram_usage_percent REAL NOT NULL,
	ram_limit_percent REAL DEFAULT 0,
	threshold_percent REAL NOT NULL,
	moving_average_percent REAL NOT NULL,
	route_name TEXT,
	trace_id TEXT,
	culprit_function TEXT,
	culprit_file_path TEXT,
	alert_sent BOOLEAN DEFAULT FALSE,
	cooldown_end DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_spike_events_timestamp ON spike_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_spike_events_deployment ON spike_events(deployment_name);
CREATE INDEX IF NOT EXISTS idx_spike_events_namespace ON spike_events(namespace);
CREATE INDEX IF NOT EXISTS idx_spike_events_created_at ON spike_events(created_at);

CREATE TABLE IF NOT EXISTS config (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// Migration V2: Add metrics cache table
const migrationV2 = `
CREATE TABLE IF NOT EXISTS metrics_cache (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	deployment_name TEXT NOT NULL,
	namespace TEXT NOT NULL,
	cpu_percent REAL NOT NULL,
	ram_percent REAL NOT NULL,
	timestamp DATETIME NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_metrics_cache_deployment ON metrics_cache(deployment_name);
CREATE INDEX IF NOT EXISTS idx_metrics_cache_timestamp ON metrics_cache(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_cache_created_at ON metrics_cache(created_at);
`

// Migration V3: Add datasource support
const migrationV3 = `
ALTER TABLE spike_events ADD COLUMN datasource TEXT NOT NULL DEFAULT 'default';
CREATE INDEX IF NOT EXISTS idx_spike_events_datasource ON spike_events(datasource);
`
