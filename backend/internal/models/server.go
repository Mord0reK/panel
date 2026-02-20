package models

import (
	"database/sql"
	"time"
)

// offlineThreshold is the duration after which a server is considered offline.
const offlineThreshold = 30 * time.Second

// Server represents a registered agent server.
type Server struct {
	UUID         string    `json:"uuid"`
	Hostname     string    `json:"hostname"`
	DisplayName  string    `json:"display_name,omitempty"`
	Icon         string    `json:"icon,omitempty"`
	Status       string    `json:"status"` // "active" or "rejected"
	Approved     bool      `json:"approved"`
	Online       bool      `json:"online"` // computed from last_seen, not stored
	CPUModel     string    `json:"cpu_model"`
	CPUCores     int       `json:"cpu_cores"`
	MemoryTotal  uint64    `json:"memory_total"`
	Platform     string    `json:"platform"`
	Kernel       string    `json:"kernel"`
	Architecture string    `json:"architecture"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// isOnline returns true when the server has been seen within the offline threshold.
func isOnline(lastSeen time.Time) bool {
	return !lastSeen.IsZero() && time.Since(lastSeen) < offlineThreshold
}

// GetByUUID retrieves a server by UUID.
func (s *Server) GetByUUID(db *sql.DB, uuid string) (*Server, error) {
	row := db.QueryRow(`
		SELECT uuid, hostname, display_name, icon, status, approved,
		       cpu_model, cpu_cores, memory_total, platform, kernel, architecture, last_seen, created_at
		FROM servers WHERE uuid = ?`, uuid)

	var server Server
	var hostname, cpuModel, platform, kernel, arch sql.NullString
	var displayName, icon, status sql.NullString
	var cpuCores, memTotal sql.NullInt64
	var lastSeen, createdAt sql.NullTime

	err := row.Scan(
		&server.UUID, &hostname, &displayName, &icon, &status, &server.Approved,
		&cpuModel, &cpuCores, &memTotal,
		&platform, &kernel, &arch,
		&lastSeen, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	server.Hostname = hostname.String
	server.DisplayName = displayName.String
	server.Icon = icon.String
	if status.Valid && status.String != "" {
		server.Status = status.String
	} else {
		server.Status = "active"
	}
	server.CPUModel = cpuModel.String
	server.CPUCores = int(cpuCores.Int64)
	server.MemoryTotal = uint64(memTotal.Int64)
	server.Platform = platform.String
	server.Kernel = kernel.String
	server.Architecture = arch.String
	if lastSeen.Valid {
		server.LastSeen = lastSeen.Time
	}
	if createdAt.Valid {
		server.CreatedAt = createdAt.Time
	}
	server.Online = isOnline(server.LastSeen)

	return &server, nil
}

// Upsert registers or updates a server on connection.
func (s *Server) Upsert(db *sql.DB) error {
	existing, err := s.GetByUUID(db, s.UUID)
	if err == nil {
		// Preserve admin-managed fields.
		s.Approved = existing.Approved
		s.Status = existing.Status
		_, err := db.Exec(`
			UPDATE servers SET
				hostname=?, cpu_model=?, cpu_cores=?, memory_total=?,
				platform=?, kernel=?, architecture=?, last_seen=CURRENT_TIMESTAMP
			WHERE uuid=?`,
			s.Hostname, s.CPUModel, s.CPUCores, s.MemoryTotal,
			s.Platform, s.Kernel, s.Architecture, s.UUID)
		return err
	} else if err == sql.ErrNoRows {
		s.Approved = true
		s.Status = "active"
		_, err := db.Exec(`
			INSERT INTO servers (
				uuid, hostname, approved, status, cpu_model, cpu_cores, memory_total,
				platform, kernel, architecture, last_seen
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			s.UUID, s.Hostname, true, "active", s.CPUModel, s.CPUCores, s.MemoryTotal,
			s.Platform, s.Kernel, s.Architecture)
		return err
	}
	return err
}

// GetAll retrieves all servers ordered by hostname.
func (s *Server) GetAll(db *sql.DB) ([]Server, error) {
	rows, err := db.Query(`
		SELECT uuid, hostname, display_name, icon, status, approved,
		       cpu_model, cpu_cores, memory_total, platform, kernel, architecture, last_seen, created_at
		FROM servers ORDER BY hostname ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var srv Server
		var hostname, cpuModel, platform, kernel, arch sql.NullString
		var displayName, icon, status sql.NullString
		var cpuCores, memTotal sql.NullInt64
		var lastSeen, createdAt sql.NullTime

		err := rows.Scan(
			&srv.UUID, &hostname, &displayName, &icon, &status, &srv.Approved,
			&cpuModel, &cpuCores, &memTotal,
			&platform, &kernel, &arch,
			&lastSeen, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		srv.Hostname = hostname.String
		srv.DisplayName = displayName.String
		srv.Icon = icon.String
		if status.Valid && status.String != "" {
			srv.Status = status.String
		} else {
			srv.Status = "active"
		}
		srv.CPUModel = cpuModel.String
		srv.CPUCores = int(cpuCores.Int64)
		srv.MemoryTotal = uint64(memTotal.Int64)
		srv.Platform = platform.String
		srv.Kernel = kernel.String
		srv.Architecture = arch.String
		if lastSeen.Valid {
			srv.LastSeen = lastSeen.Time
		}
		if createdAt.Valid {
			srv.CreatedAt = createdAt.Time
		}
		srv.Online = isOnline(srv.LastSeen)
		servers = append(servers, srv)
	}
	return servers, nil
}

func (s *Server) UpdateLastSeen(db *sql.DB, uuid string) error {
	_, err := db.Exec("UPDATE servers SET last_seen=CURRENT_TIMESTAMP WHERE uuid=?", uuid)
	return err
}

// Approve marks a server as approved and active.
func (s *Server) Approve(db *sql.DB, uuid string) error {
	_, err := db.Exec("UPDATE servers SET approved=1, status='active' WHERE uuid=?", uuid)
	return err
}

// UpdateMeta updates admin-managed display fields.
// approved is derived from status: active=true, rejected=false.
func (s *Server) UpdateMeta(db *sql.DB, uuid, displayName, icon, status string) error {
	approved := status != "rejected"
	_, err := db.Exec(
		"UPDATE servers SET display_name=?, icon=?, status=?, approved=? WHERE uuid=?",
		nullableString(displayName), nullableString(icon), status, approved, uuid,
	)
	return err
}

// Delete removes a server permanently.
func (s *Server) Delete(db *sql.DB, uuid string) error {
	_, err := db.Exec("DELETE FROM servers WHERE uuid=?", uuid)
	return err
}

// nullableString converts an empty string to nil so SQLite stores NULL.
func nullableString(v string) interface{} {
	if v == "" {
		return nil
	}
	return v
}
