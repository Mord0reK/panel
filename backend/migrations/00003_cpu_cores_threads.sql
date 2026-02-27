-- +goose Up

-- ensure existing databases also receive the new column
ALTER TABLE servers ADD COLUMN cpu_threads INTEGER DEFAULT 0;

-- +goose Down
-- SQLite doesn’t support DROP COLUMN; this migration is irreversible
