package backup

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestManifestJSON(t *testing.T) {
	m := Manifest{
		Version:  "1.0.0",
		Created:  time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
		Hostname: "test-mac",
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

func TestManifestV2MetadataJSON(t *testing.T) {
	created := time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC)
	m := &Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindIncremental,
		Created:       created,
		Hostname:      "new-mac",
		Base: &BaseArchiveRef{
			FileName:       "dotvibe-2026-05-19.tar.gz",
			Created:        created.Add(-24 * time.Hour),
			ManifestSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Tools: map[string]ToolManifest{
			"claude-code": {Included: []string{"memory"}, FileCount: 1},
		},
		Files: []FileManifest{
			{
				Path:       "claude-code/memory/project/MEMORY.md",
				ToolID:     "claude-code",
				Size:       7,
				SHA256:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Category:   "memory",
				Storage:    FileStorageBase,
				StoredPath: "",
			},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.FormatVersion != 2 || got.ArchiveKind != ArchiveKindIncremental {
		t.Fatalf("unexpected metadata: %#v", got)
	}
	if got.Base == nil || got.Base.ManifestSHA256 != m.Base.ManifestSHA256 {
		t.Fatalf("base ref not preserved: %#v", got.Base)
	}
	if got.Files[0].Storage != FileStorageBase || got.Files[0].ToolID != "claude-code" {
		t.Fatalf("file storage metadata not preserved: %#v", got.Files[0])
	}
}

func TestRecipeManifestMetadataJSON(t *testing.T) {
	created := time.Date(2026, 5, 21, 8, 0, 0, 0, time.UTC)
	m := &Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindRecipe,
		Created:       created,
		Hostname:      "source-mac",
		Recipe: &RecipeMetadata{
			Name:        "YYF Vibe Stack",
			Description: "Claude skills and Codex agents for daily coding",
			Author:      "yangyifan",
			Schema:      RecipeSchemaV1,
			SharePolicy: "shareable-only",
		},
		Tools: map[string]ToolManifest{
			"claude-code": {Included: []string{"settings", "rules", "skills"}, FileCount: 2},
		},
		Files: []FileManifest{
			{Path: "claude-code/skills/reviewer/SKILL.md", ToolID: "claude-code", Category: "skills", Size: 8, SHA256: strings.Repeat("a", 64), Storage: FileStorageInline, StoredPath: "objects/sha256/aa"},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got.Normalize()
	if got.ArchiveKind != ArchiveKindRecipe {
		t.Fatalf("archive kind = %q, want %q", got.ArchiveKind, ArchiveKindRecipe)
	}
	if got.Recipe == nil || got.Recipe.Name != "YYF Vibe Stack" || got.Recipe.Schema != RecipeSchemaV1 {
		t.Fatalf("recipe metadata not preserved: %#v", got.Recipe)
	}
}

func TestLegacyManifestDefaultsToFullArchive(t *testing.T) {
	var m Manifest
	data := []byte(`{"version":"1.0.0","tools":{},"files":[{"path":"tool/config.json","size":2,"sha256":"abc","category":"config"}]}`)
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal legacy manifest: %v", err)
	}
	m.Normalize()
	if m.FormatVersion != 1 || m.ArchiveKind != ArchiveKindFull {
		t.Fatalf("legacy defaults = (%d,%q), want (1,%q)", m.FormatVersion, m.ArchiveKind, ArchiveKindFull)
	}
	if m.Files[0].Storage != FileStorageInline || m.Files[0].StoredPath != m.Files[0].Path {
		t.Fatalf("legacy file storage = %#v", m.Files[0])
	}
}

func TestReadManifest_MissingFile(t *testing.T) {
	_, err := ReadManifest("/nonexistent/manifest.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
