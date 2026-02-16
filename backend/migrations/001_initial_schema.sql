CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS servers (
    uuid TEXT PRIMARY KEY,
    hostname TEXT,
    approved BOOLEAN DEFAULT 0,
    cpu_model TEXT,
    cpu_cores INTEGER,
    memory_total INTEGER,
    platform TEXT,
    kernel TEXT,
    architecture TEXT,
    last_seen DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS containers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_uuid TEXT,
    container_id TEXT,
    name TEXT,
    image TEXT,
    project TEXT,
    service TEXT,
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME,
    UNIQUE(agent_uuid, container_id),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS container_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_uuid TEXT,
    container_id TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    event_type TEXT,
    old_value TEXT,
    new_value TEXT,
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

-- Metrics 1s (Raw data)
CREATE TABLE IF NOT EXISTS metrics_1s (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_percent REAL,
    mem_used INTEGER,
    mem_percent REAL,
    disk_used INTEGER,
    disk_percent REAL,
    net_rx_bytes INTEGER,
    net_tx_bytes INTEGER,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

-- Aggregated metrics tables
CREATE TABLE IF NOT EXISTS metrics_5s (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_15s (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_30s (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_1m (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_5m (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_15m (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_30m (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_1h (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_6h (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS metrics_12h (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_avg REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, container_id, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

-- Host metrics 1s (Raw data)
CREATE TABLE IF NOT EXISTS host_metrics_1s (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_percent REAL,
    mem_used INTEGER,
    mem_percent REAL,
    disk_read_bytes_per_sec REAL,
    disk_write_bytes_per_sec REAL,
    net_rx_bytes_per_sec REAL,
    net_tx_bytes_per_sec REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

-- Aggregated host metrics tables
CREATE TABLE IF NOT EXISTS host_metrics_5s (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_15s (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_30s (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_1m (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_5m (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_15m (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_30m (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_1h (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_6h (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS host_metrics_12h (
    agent_uuid TEXT,
    timestamp INTEGER,
    cpu_avg REAL, cpu_min REAL, cpu_max REAL,
    mem_avg REAL, mem_min REAL, mem_max REAL,
    disk_read_avg REAL, disk_read_min REAL, disk_read_max REAL,
    disk_write_avg REAL, disk_write_min REAL, disk_write_max REAL,
    net_rx_avg REAL, net_rx_min REAL, net_rx_max REAL,
    net_tx_avg REAL, net_tx_min REAL, net_tx_max REAL,
    PRIMARY KEY(agent_uuid, timestamp),
    FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_servers_approved ON servers(approved);
CREATE INDEX IF NOT EXISTS idx_containers_agent ON containers(agent_uuid);
CREATE INDEX IF NOT EXISTS idx_events_container ON container_events(agent_uuid, container_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_host_metrics_1s_agent_ts ON host_metrics_1s(agent_uuid, timestamp);
