-- +goose Up

ALTER TABLE containers ADD COLUMN health TEXT NOT NULL DEFAULT '';
ALTER TABLE containers ADD COLUMN status TEXT NOT NULL DEFAULT '';

-- +goose Down

-- SQLite does not support DROP COLUMN in older versions; left intentionally empty.
