package backup

import (
	"encoding/json"
	"testing"
	"time"
)

func TestManifestJSON(t *testing.T) {
	m := Manifest{
		Version:   "1.0.0",
		Created:   time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
		Hostname:  "test-mac",
		Tools: map[string]ToolManifest{
			"claude-code": {
				Included:     []string{"config", "memory", "skills"},
				ProjectCount: 12,
				FileCount:    245,
			},
			"codex-cli": {
				Included:   []string{"config", "agents"},
				AgentCount: 24,
			},
		},
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", decoded.Version, "1.0.0")
	}
	if decoded.Hostname != "test-mac" {
		t.Errorf("Hostname = %q, want %q", decoded.Hostname, "test-mac")
	}
	if len(decoded.Tools) != 2 {
		t.Errorf("Tools count = %d, want 2", len(decoded.Tools))
	}
	claude := decoded.Tools["claude-code"]
	if claude.ProjectCount != 12 {
		t.Errorf("ProjectCount = %d, want 12", claude.ProjectCount)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	tmp := t.TempDir() + "/manifest.json"

	m := Manifest{
		Version:  "1.0.0",
		Created:  time.Now().UTC(),
		Hostname: "test",
		Tools:    map[string]ToolManifest{},
	}

	if err := WriteManifest(tmp, &m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	loaded, err := ReadManifest(tmp)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if loaded.Version != m.Version {
		t.Errorf("Version mismatch: %q vs %q", loaded.Version, m.Version)
	}
	if loaded.Hostname != m.Hostname {
		t.Errorf("Hostname mismatch: %q vs %q", loaded.Hostname, m.Hostname)
	}
}

func TestReadManifest_MissingFile(t *testing.T) {
	_, err := ReadManifest("/nonexistent/manifest.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
