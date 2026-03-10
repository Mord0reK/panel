-- +goose Up

-- Indexes on metrics tables to speed up:
-- 1. DELETE WHERE timestamp < ? (cleanup)
-- 2. SELECT WHERE agent_uuid = ? AND timestamp < ? ORDER BY timestamp (aggregation fetch)
-- 3. SELECT WHERE agent_uuid = ? AND timestamp BETWEEN ? AND ? (history queries)

CREATE INDEX IF NOT EXISTS idx_metrics_5s_timestamp      ON metrics_5s  (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_5s_agent_ts       ON metrics_5s  (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_15s_timestamp     ON metrics_15s (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_15s_agent_ts      ON metrics_15s (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_30s_timestamp     ON metrics_30s (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_30s_agent_ts      ON metrics_30s (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_1m_timestamp      ON metrics_1m  (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_1m_agent_ts       ON metrics_1m  (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_5m_timestamp      ON metrics_5m  (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_5m_agent_ts       ON metrics_5m  (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_15m_timestamp     ON metrics_15m (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_15m_agent_ts      ON metrics_15m (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_30m_timestamp     ON metrics_30m (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_30m_agent_ts      ON metrics_30m (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_1h_timestamp      ON metrics_1h  (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_1h_agent_ts       ON metrics_1h  (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_6h_timestamp      ON metrics_6h  (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_6h_agent_ts       ON metrics_6h  (agent_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_metrics_12h_timestamp     ON metrics_12h (timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_12h_agent_ts      ON metrics_12h (agent_uuid, timestamp);

-- +goose Down

DROP INDEX IF EXISTS idx_metrics_5s_timestamp;
DROP INDEX IF EXISTS idx_metrics_5s_agent_ts;
DROP INDEX IF EXISTS idx_metrics_15s_timestamp;
DROP INDEX IF EXISTS idx_metrics_15s_agent_ts;
DROP INDEX IF EXISTS idx_metrics_30s_timestamp;
DROP INDEX IF EXISTS idx_metrics_30s_agent_ts;
DROP INDEX IF EXISTS idx_metrics_1m_timestamp;
DROP INDEX IF EXISTS idx_metrics_1m_agent_ts;
DROP INDEX IF EXISTS idx_metrics_5m_timestamp;
DROP INDEX IF EXISTS idx_metrics_5m_agent_ts;
DROP INDEX IF EXISTS idx_metrics_15m_timestamp;
DROP INDEX IF EXISTS idx_metrics_15m_agent_ts;
DROP INDEX IF EXISTS idx_metrics_30m_timestamp;
DROP INDEX IF EXISTS idx_metrics_30m_agent_ts;
DROP INDEX IF EXISTS idx_metrics_1h_timestamp;
DROP INDEX IF EXISTS idx_metrics_1h_agent_ts;
DROP INDEX IF EXISTS idx_metrics_6h_timestamp;
DROP INDEX IF EXISTS idx_metrics_6h_agent_ts;
DROP INDEX IF EXISTS idx_metrics_12h_timestamp;
DROP INDEX IF EXISTS idx_metrics_12h_agent_ts;
