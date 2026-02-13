package output

import (
	"encoding/json"
	"fmt"
	"os"
)

type AgentMetrics struct {
	SystemMetrics interface{} `json:"system,omitempty"`
	DockerMetrics interface{} `json:"docker,omitempty"`
}

func WriteToJSON(filename string, systemMetrics, dockerMetrics interface{}) error {
	metrics := AgentMetrics{
		SystemMetrics: systemMetrics,
		DockerMetrics: dockerMetrics,
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
