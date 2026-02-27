-- +goose Up

ALTER TABLE containers ADD COLUMN state TEXT NOT NULL DEFAULT 'unknown';

-- +goose Down

-- SQLite does not support DROP COLUMN in older versions; migration is intentionally left empty.
