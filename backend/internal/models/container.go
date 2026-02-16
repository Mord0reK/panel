package models

import (
	"database/sql"
	"time"
)

type Container struct {
	ID          int       `json:"id"`
	AgentUUID   string    `json:"agent_uuid"`
	ContainerID string    `json:"container_id"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Project     string    `json:"project"`
	Service     string    `json:"service"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

func (c *Container) GetByAgent(db *sql.DB, agentUUID string) ([]Container, error) {
	rows, err := db.Query(`
		SELECT id, agent_uuid, container_id, name, image, project, service, first_seen, last_seen
		FROM containers WHERE agent_uuid = ? ORDER BY name ASC`, agentUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []Container
	for rows.Next() {
		var cont Container
		var name, image, project, service sql.NullString
		var firstSeen, lastSeen sql.NullTime
		err := rows.Scan(
			&cont.ID, &cont.AgentUUID, &cont.ContainerID, &name,
			&image, &project, &service,
			&firstSeen, &lastSeen,
		)
		if err != nil {
			return nil, err
		}
		cont.Name = name.String
		cont.Image = image.String
		cont.Project = project.String
		cont.Service = service.String
		if firstSeen.Valid { cont.FirstSeen = firstSeen.Time }
		if lastSeen.Valid { cont.LastSeen = lastSeen.Time }
		containers = append(containers, cont)
	}
	return containers, nil
}

func (c *Container) Upsert(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO containers (agent_uuid, container_id, name, image, project, service, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(agent_uuid, container_id) DO UPDATE SET
			name=excluded.name,
			image=excluded.image,
			project=excluded.project,
			service=excluded.service,
			last_seen=CURRENT_TIMESTAMP`,
		c.AgentUUID, c.ContainerID, c.Name, c.Image, c.Project, c.Service)
	return err
}
