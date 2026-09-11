// Package database provides the persistence layer: SQLite by default via
// the pure-Go modernc driver, with a migration runner and repositories
// (spec sections 3.3, 8.7).
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB handle.
type DB struct {
	sql    *sql.DB
	driver string
}

// OpenSQLite opens (creating if needed) the SQLite database with sane
// pragmas: WAL journaling, foreign keys, busy timeout.
func OpenSQLite(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite is single-writer; bound the pool.
	handle.SetMaxOpenConns(1)
	handle.SetConnMaxLifetime(0)
	if err := handle.Ping(); err != nil {
		handle.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return &DB{sql: handle, driver: "sqlite"}, nil
}

// IntegCheck runs PRAGMA quick_check; non-empty result other than "ok" is
// a corruption signal (spec section 8.7).
func (d *DB) IntegCheck() error {
	var out string
	if err := d.sql.QueryRow("PRAGMA quick_check").Scan(&out); err != nil {
		return err
	}
	if out != "ok" {
		return fmt.Errorf("sqlite integrity: %s", out)
	}
	return nil
}

// Close releases the handle.
func (d *DB) Close() error { return d.sql.Close() }

// SQL exposes the raw handle to repositories.
func (d *DB) SQL() *sql.DB { return d.sql }

// Migrate applies embedded schema migrations in order, tracking applied
// versions in schema_migrations. Backup before destructive migration is the
// caller's job (db backup happens in app startup path, spec section 8.7).
func (d *DB) Migrate(migrations []Migration) error {
	if _, err := d.sql.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return err
	}
	var current int
	if err := d.sql.QueryRow("SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&current); err != nil {
		return err
	}
	for _, m := range migrations {
		if m.Version <= current {
			continue
		}
		tx, err := d.sql.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d (%s): %w", m.Version, m.Name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name, applied_at) VALUES (?,?,?)",
			m.Version, m.Name, time.Now().UTC().Format(time.RFC3339)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d record: %w", m.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Migration is a single ordered schema step.
type Migration struct {
	Version int
	Name    string
	SQL     string
}
