package tests

import (
	"database/sql"
	"testing"

	"backend/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory database for testing
	db, err := database.New(":memory:")
	require.NoError(t, err)

	// Run migrations
	// Assuming the test is run from the backend/ directory or subdirectories
	// We need to find the migrations directory
	err = database.RunMigrations(db, "../migrations")
	require.NoError(t, err)

	return db
}

func TestDatabaseInitAndMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check if tables exist
	tables := []string{
		"users", "servers", "containers", "container_events",
		"metrics_1s", "metrics_5s", "metrics_15s", "metrics_30s",
		"metrics_1m", "metrics_5m", "metrics_15m", "metrics_30m",
		"metrics_1h", "metrics_6h", "metrics_12h",
		"host_metrics_1s", "host_metrics_5s", "host_metrics_15s", "host_metrics_30s",
		"host_metrics_1m", "host_metrics_5m", "host_metrics_15m", "host_metrics_30m",
		"host_metrics_1h", "host_metrics_6h", "host_metrics_12h",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		assert.NoError(t, err, "Table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestDatabaseInsertAndRead(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test Users
	_, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", "testuser", "hash123")
	assert.NoError(t, err)

	var username string
	err = db.QueryRow("SELECT username FROM users WHERE username=?", "testuser").Scan(&username)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", username)

	// Test Servers
	_, err = db.Exec("INSERT INTO servers (uuid, hostname, approved) VALUES (?, ?, ?)", "server1", "host1", true)
	assert.NoError(t, err)

	var approved bool
	err = db.QueryRow("SELECT approved FROM servers WHERE uuid=?", "server1").Scan(&approved)
	assert.NoError(t, err)
	assert.True(t, approved)

	// Test Containers
	_, err = db.Exec("INSERT INTO containers (agent_uuid, container_id, name) VALUES (?, ?, ?)", "server1", "cont1", "nginx")
	assert.NoError(t, err)

	var containerName string
	err = db.QueryRow("SELECT name FROM containers WHERE container_id=?", "cont1").Scan(&containerName)
	assert.NoError(t, err)
	assert.Equal(t, "nginx", containerName)

	// Test Metrics
	_, err = db.Exec(`INSERT INTO metrics_1s 
		(agent_uuid, container_id, timestamp, cpu_percent, mem_used, mem_percent, disk_used, disk_percent, net_rx_bytes, net_tx_bytes) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"server1", "cont1", 1234567890, 10.5, 1024, 50.0, 2048, 10.0, 100, 200)
	assert.NoError(t, err)

	var cpuPercent float64
	err = db.QueryRow("SELECT cpu_percent FROM metrics_1s WHERE timestamp=?", 1234567890).Scan(&cpuPercent)
	assert.NoError(t, err)
	assert.Equal(t, 10.5, cpuPercent)
}
