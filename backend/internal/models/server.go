package models

import (
	"database/sql"
	"time"
)

type Server struct {
	UUID         string    `json:"uuid"`
	Hostname     string    `json:"hostname"`
	Approved     bool      `json:"approved"`
	CPUModel     string    `json:"cpu_model"`
	CPUCores     int       `json:"cpu_cores"`
	MemoryTotal  uint64    `json:"memory_total"`
	Platform     string    `json:"platform"`
	Kernel       string    `json:"kernel"`
	Architecture string    `json:"architecture"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetByUUID retrieves a server by UUID
func (s *Server) GetByUUID(db *sql.DB, uuid string) (*Server, error) {
	row := db.QueryRow(`
		SELECT uuid, hostname, approved, cpu_model, cpu_cores, memory_total, platform, kernel, architecture, last_seen, created_at
		FROM servers WHERE uuid = ?`, uuid)

	var server Server
	var hostname, cpuModel, platform, kernel, arch sql.NullString
	var cpuCores, memTotal sql.NullInt64
	var lastSeen, createdAt sql.NullTime

	err := row.Scan(
		&server.UUID, &hostname, &server.Approved,
		&cpuModel, &cpuCores, &memTotal,
		&platform, &kernel, &arch,
		&lastSeen, &createdAt,
	)

	if err != nil {
		return nil, err
	}

	server.Hostname = hostname.String
	server.CPUModel = cpuModel.String
	server.CPUCores = int(cpuCores.Int64)
	server.MemoryTotal = uint64(memTotal.Int64)
	server.Platform = platform.String
	server.Kernel = kernel.String
	server.Architecture = arch.String
	if lastSeen.Valid { server.LastSeen = lastSeen.Time }
	if createdAt.Valid { server.CreatedAt = createdAt.Time }

	return &server, nil
}

// Upsert registers or updates a server
func (s *Server) Upsert(db *sql.DB) error {
	// Check if exists to preserve 'approved' status if it does
	existing, err := s.GetByUUID(db, s.UUID)
	if err == nil {
		s.Approved = existing.Approved
		// Update details
		_, err := db.Exec(`
			UPDATE servers SET 
				hostname=?, cpu_model=?, cpu_cores=?, memory_total=?, 
				platform=?, kernel=?, architecture=?, last_seen=CURRENT_TIMESTAMP
			WHERE uuid=?`,
			s.Hostname, s.CPUModel, s.CPUCores, s.MemoryTotal,
			s.Platform, s.Kernel, s.Architecture, s.UUID)
		return err
	} else if err == sql.ErrNoRows {
		// Insert new
		_, err := db.Exec(`
			INSERT INTO servers (
				uuid, hostname, approved, cpu_model, cpu_cores, memory_total, 
				platform, kernel, architecture, last_seen
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			s.UUID, s.Hostname, false, s.CPUModel, s.CPUCores, s.MemoryTotal,
			s.Platform, s.Kernel, s.Architecture)
		return err
	}
	return err
}

// GetAll retrieves all servers
func (s *Server) GetAll(db *sql.DB) ([]Server, error) {
	rows, err := db.Query(`
		SELECT uuid, hostname, approved, cpu_model, cpu_cores, memory_total, platform, kernel, architecture, last_seen, created_at
		FROM servers ORDER BY hostname ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var srv Server
		var hostname, cpuModel, platform, kernel, arch sql.NullString
		var cpuCores, memTotal sql.NullInt64
		var lastSeen, createdAt sql.NullTime
		err := rows.Scan(
			&srv.UUID, &hostname, &srv.Approved,
			&cpuModel, &cpuCores, &memTotal,
			&platform, &kernel, &arch,
			&lastSeen, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		srv.Hostname = hostname.String
		srv.CPUModel = cpuModel.String
		srv.CPUCores = int(cpuCores.Int64)
		srv.MemoryTotal = uint64(memTotal.Int64)
		srv.Platform = platform.String
		srv.Kernel = kernel.String
		srv.Architecture = arch.String
		if lastSeen.Valid { srv.LastSeen = lastSeen.Time }
		if createdAt.Valid { srv.CreatedAt = createdAt.Time }
		servers = append(servers, srv)
	}
	return servers, nil
}

func (s *Server) UpdateLastSeen(db *sql.DB, uuid string) error {
	_, err := db.Exec("UPDATE servers SET last_seen=CURRENT_TIMESTAMP WHERE uuid=?", uuid)
	return err
}

func (s *Server) Approve(db *sql.DB, uuid string) error {
	_, err := db.Exec("UPDATE servers SET approved=1 WHERE uuid=?", uuid)
	return err
}

func (s *Server) Delete(db *sql.DB, uuid string) error {
	_, err := db.Exec("DELETE FROM servers WHERE uuid=?", uuid)
	return err
}
