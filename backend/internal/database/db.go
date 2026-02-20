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

// RunMigrations executes all SQL files found in the migrations directory.
// Applied migrations are tracked in the schema_migrations table so each file
// runs exactly once, making the process idempotent across restarts.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Bootstrap the tracking table using a direct Exec – this must always run.
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name       TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

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
		var exists int
		if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE name = ?`, file).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", file, err)
		}
		if exists > 0 {
			continue // already applied
		}

		path := filepath.Join(migrationsDir, file)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		if err := execMigration(db, string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, file); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}
	}

	return nil
}

// execMigration executes a migration file statement-by-statement, tolerating
// idempotent errors (duplicate column, table already exists) that occur when
// a migration was applied before the schema_migrations tracking was introduced.
func execMigration(db *sql.DB, content string) error {
	for _, stmt := range strings.Split(content, ";") {
		stmt = stripLineComments(stmt)
		if stmt == "" {
			continue
		}
		if err := execWithRetry(db, stmt); err != nil {
			msg := err.Error()
			// These errors mean the DDL change was already applied — safe to skip.
			if strings.Contains(msg, "duplicate column name") ||
				strings.Contains(msg, "already exists") {
				continue
			}
			return err
		}
	}
	return nil
}

// stripLineComments removes SQL line-comment lines (starting with --) and
// returns the trimmed result. Inline comments within SQL are not stripped.
func stripLineComments(sql string) string {
	var lines []string
	for _, line := range strings.Split(sql, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
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
