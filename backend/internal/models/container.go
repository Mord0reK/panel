package models

import (
	"database/sql"
	"fmt"
	"strings"
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
	State       string    `json:"state"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

func (c *Container) GetByAgent(db *sql.DB, agentUUID string) ([]Container, error) {
	rows, err := db.Query(`
		SELECT id, agent_uuid, container_id, name, image, project, service, state, first_seen, last_seen
		FROM containers WHERE agent_uuid = ? ORDER BY name ASC`, agentUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []Container
	for rows.Next() {
		var cont Container
		var name, image, project, service, state sql.NullString
		var firstSeen, lastSeen sql.NullTime
		err := rows.Scan(
			&cont.ID, &cont.AgentUUID, &cont.ContainerID, &name,
			&image, &project, &service, &state,
			&firstSeen, &lastSeen,
		)
		if err != nil {
			return nil, err
		}
		cont.Name = name.String
		cont.Image = image.String
		cont.Project = project.String
		cont.Service = service.String
		cont.State = state.String
		if firstSeen.Valid {
			cont.FirstSeen = firstSeen.Time
		}
		if lastSeen.Valid {
			cont.LastSeen = lastSeen.Time
		}
		containers = append(containers, cont)
	}
	return containers, nil
}

func (c *Container) Upsert(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO containers (agent_uuid, container_id, name, image, project, service, state, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(agent_uuid, container_id) DO UPDATE SET
			name=excluded.name,
			image=excluded.image,
			project=excluded.project,
			service=excluded.service,
			state=excluded.state,
			last_seen=CURRENT_TIMESTAMP`,
		c.AgentUUID, c.ContainerID, c.Name, c.Image, c.Project, c.Service, c.State)
	return err
}

// MarkRemovedNotInList marks containers as "removed" for the given agent when
// they are no longer reported by the agent (e.g. docker compose down).
// This keeps them visible in the UI (greyed out) instead of deleting them.
// When the container reappears (docker compose up), Upsert will restore its real state.
func MarkRemovedNotInList(db *sql.DB, agentUUID string, activeIDs []string) error {
	if len(activeIDs) == 0 {
		// Agent sent an empty list — mark all containers for this agent as removed.
		_, err := db.Exec(`UPDATE containers SET state = 'removed' WHERE agent_uuid = ?`, agentUUID)
		return err
	}

	placeholders := strings.Repeat("?,", len(activeIDs))
	placeholders = placeholders[:len(placeholders)-1] // trim trailing comma

	args := make([]interface{}, 0, len(activeIDs)+1)
	args = append(args, agentUUID)
	for _, id := range activeIDs {
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`UPDATE containers SET state = 'removed' WHERE agent_uuid = ? AND container_id NOT IN (%s)`,
		placeholders,
	)
	_, err := db.Exec(query, args...)
	return err
}
