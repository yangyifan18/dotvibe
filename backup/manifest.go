package backup

import (
	"encoding/json"
	"os"
	"time"
)

type Manifest struct {
	Version  string                  `json:"version"`
	Created  time.Time               `json:"created"`
	Hostname string                  `json:"hostname"`
	Tools    map[string]ToolManifest `json:"tools"`
	Files    []FileManifest          `json:"files,omitempty"`
}

type ToolManifest struct {
	Version      string   `json:"version,omitempty"`
	Included     []string `json:"included"`
	ProjectCount int      `json:"project_count,omitempty"`
	FileCount    int      `json:"file_count,omitempty"`
	AgentCount   int      `json:"agent_count,omitempty"`
}

type FileManifest struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Category string `json:"category,omitempty"`
}

func WriteManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
