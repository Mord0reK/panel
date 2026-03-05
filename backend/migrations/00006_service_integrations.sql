-- +goose Up

CREATE TABLE IF NOT EXISTS service_integrations (
    service_key TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT 0,
    base_url TEXT,
    encrypted_token TEXT,
    encrypted_username TEXT,
    encrypted_password TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down

DROP TABLE IF EXISTS service_integrations;
