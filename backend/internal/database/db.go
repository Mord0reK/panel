package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"backend/migrations"
)

// New opens a connection to the SQLite database at dbPath.
// Use ":memory:" for an ephemeral in-memory database (for tests).
func New(dbPath string) (*sql.DB, error) {
	var dsn string
	if dbPath == ":memory:" {
		dsn = "file::memory:?cache=shared&_foreign_keys=1"
	} else {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		dsn = fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=1", dbPath)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// SQLite supports only one concurrent writer; cap the pool to prevent SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	// modernc.org/sqlite ignores _foreign_keys in the DSN — enforce it explicitly.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return db, nil
}

// RunMigrations applies all pending SQL migrations using goose.
// Migrations are embedded in the binary at build time via embed.FS —
// no migrations directory needs to be present on the filesystem at runtime.
func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
