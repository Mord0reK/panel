package models

import (
	"database/sql"
	"time"
)

type ServiceConfig struct {
	ID                  int       `json:"id"`
	Slug                string    `json:"slug"`
	ConfigJSONEncrypted string    `json:"-"`
	IsEnabled           bool      `json:"is_enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (m *ServiceConfig) GetBySlug(db *sql.DB, slug string) (*ServiceConfig, error) {
	row := db.QueryRow("SELECT id, slug, config_json_encrypted, is_enabled, created_at, updated_at FROM service_configs WHERE slug = ?", slug)

	var config ServiceConfig
	err := row.Scan(&config.ID, &config.Slug, &config.ConfigJSONEncrypted, &config.IsEnabled, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (m *ServiceConfig) GetAll(db *sql.DB) ([]ServiceConfig, error) {
	rows, err := db.Query("SELECT id, slug, config_json_encrypted, is_enabled, created_at, updated_at FROM service_configs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []ServiceConfig
	for rows.Next() {
		var config ServiceConfig
		err := rows.Scan(&config.ID, &config.Slug, &config.ConfigJSONEncrypted, &config.IsEnabled, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func (m *ServiceConfig) Save(db *sql.DB, slug string, configJSON string, isEnabled bool) error {
	_, err := db.Exec(`
		INSERT INTO service_configs (slug, config_json_encrypted, is_enabled, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(slug) DO UPDATE SET
			config_json_encrypted = excluded.config_json_encrypted,
			is_enabled = excluded.is_enabled,
			updated_at = CURRENT_TIMESTAMP
	`, slug, configJSON, isEnabled)
	return err
}
