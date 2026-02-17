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
-- Indexes
CREATE INDEX IF NOT EXISTS idx_servers_approved ON servers(approved);
CREATE INDEX IF NOT EXISTS idx_containers_agent ON containers(agent_uuid);
CREATE INDEX IF NOT EXISTS idx_events_container ON container_events(agent_uuid, container_id, timestamp);
