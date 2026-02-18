package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// New opens a connection to the SQLite database
func New(dbPath string) (*sql.DB, error) {
	var dsn string
	if dbPath == ":memory:" {
		dsn = "file::memory:?cache=shared&_foreign_keys=1"
	} else {
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		dsn = fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=1", dbPath)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// SQLite supports only one concurrent writer; cap the pool to prevent SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	return db, nil
}

// RunMigrations executes all SQL files found in the migrations directory
func RunMigrations(db *sql.DB, migrationsDir string) error {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		path := filepath.Join(migrationsDir, file)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		if err := execWithRetry(db, string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}

// execWithRetry executes a query with exponential backoff retry for "database is locked" errors
func execWithRetry(db *sql.DB, query string) error {
	var err error
	maxRetries := 5
	baseDelay := 10 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		_, err = db.Exec(query)
		if err == nil {
			return nil
		}

		if strings.Contains(err.Error(), "database is locked") {
			time.Sleep(baseDelay * time.Duration(1<<i))
			continue
		}

		return err
	}
	return fmt.Errorf("query failed after %d retries: %w", maxRetries, err)
}
