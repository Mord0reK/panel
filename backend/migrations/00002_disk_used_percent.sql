-- +goose Up
ALTER TABLE metrics_5s ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_5s ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_5s ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_15s ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_15s ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_15s ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_30s ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_30s ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_30s ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_1m ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_1m ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_1m ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_5m ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_5m ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_5m ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_15m ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_15m ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_15m ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_30m ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_30m ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_30m ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_1h ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_1h ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_1h ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_6h ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_6h ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_6h ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

ALTER TABLE metrics_12h ADD COLUMN disk_used_percent_avg REAL DEFAULT 0;
ALTER TABLE metrics_12h ADD COLUMN disk_used_percent_min REAL DEFAULT 0;
ALTER TABLE metrics_12h ADD COLUMN disk_used_percent_max REAL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; migration is intentionally irreversible
