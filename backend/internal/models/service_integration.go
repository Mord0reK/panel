package models

import (
	"database/sql"
	"time"
)

type ServiceIntegration struct {
	ServiceKey        string    `json:"service_key"`
	Enabled           bool      `json:"enabled"`
	BaseURL           string    `json:"base_url,omitempty"`
	EncryptedToken    string    `json:"-"`
	EncryptedUsername string    `json:"-"`
	EncryptedPassword string    `json:"-"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

func (s *ServiceIntegration) EnabledMap(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT service_key, enabled FROM service_integrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[string]bool)
	for rows.Next() {
		var serviceKey string
		var enabled bool
		if err := rows.Scan(&serviceKey, &enabled); err != nil {
			return nil, err
		}
		states[serviceKey] = enabled
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return states, nil
}

func (s *ServiceIntegration) GetByKey(db *sql.DB, serviceKey string) (*ServiceIntegration, error) {
	row := db.QueryRow(`
		SELECT service_key, enabled, base_url, encrypted_token, encrypted_username, encrypted_password, created_at, updated_at
		FROM service_integrations
		WHERE service_key = ?
	`, serviceKey)

	var integration ServiceIntegration
	var baseURL, encryptedToken, encryptedUsername, encryptedPassword sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := row.Scan(
		&integration.ServiceKey,
		&integration.Enabled,
		&baseURL,
		&encryptedToken,
		&encryptedUsername,
		&encryptedPassword,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	integration.BaseURL = baseURL.String
	integration.EncryptedToken = encryptedToken.String
	integration.EncryptedUsername = encryptedUsername.String
	integration.EncryptedPassword = encryptedPassword.String
	if createdAt.Valid {
		integration.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		integration.UpdatedAt = updatedAt.Time
	}

	return &integration, nil
}

func (s *ServiceIntegration) Upsert(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO service_integrations (
			service_key, enabled, base_url, encrypted_token, encrypted_username, encrypted_password, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(service_key) DO UPDATE SET
			enabled = excluded.enabled,
			base_url = excluded.base_url,
			encrypted_token = excluded.encrypted_token,
			encrypted_username = excluded.encrypted_username,
			encrypted_password = excluded.encrypted_password,
			updated_at = CURRENT_TIMESTAMP
	`, s.ServiceKey, s.Enabled, nullableString(s.BaseURL), nullableString(s.EncryptedToken), nullableString(s.EncryptedUsername), nullableString(s.EncryptedPassword))

	return err
}
