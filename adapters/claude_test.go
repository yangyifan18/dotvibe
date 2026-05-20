package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeAdapter_Detect(t *testing.T) {
	a := &ClaudeAdapter{}

	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	_, err := os.Stat(settingsPath)
	expected := err == nil

	if got := a.Detect(); got != expected {
		t.Errorf("Detect() = %v, want %v (settings.json exists: %v)", got, expected, err == nil)
	}
}

func TestClaudeAdapter_ListFiles(t *testing.T) {
	a := &ClaudeAdapter{}
	if !a.Detect() {
		t.Skip("Claude Code not installed")
	}

	files := a.ListFiles(ExportOpts{})
	if len(files) == 0 {
		t.Error("ListFiles returned empty — expected at least settings.json")
	}

	found := false
	for _, f := range files {
		if filepath.Base(f.InArchive) == "settings.json" {
			found = true
			if f.Category != CategoryConfig {
				t.Errorf("settings.json category = %q, want %q", f.Category, CategoryConfig)
			}
		}
	}
	if !found {
		t.Error("settings.json not found in file list")
	}
}

func TestClaudeAdapter_Status(t *testing.T) {
	a := &ClaudeAdapter{}
	if !a.Detect() {
		t.Skip("Claude Code not installed")
	}

	s := a.Status()
	if s.Name != "Claude Code" {
		t.Errorf("Name = %q, want %q", s.Name, "Claude Code")
	}
	if s.Path == "" {
		t.Error("Path is empty")
	}
}

func TestClaudeAdapter_RestoreLegacyClaudeMarkdownToProjectRoot(t *testing.T) {
	home := t.TempDir()
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "claude-code", "memory", "-Users-young-App", "CLAUDE.md")
	writeTestFile(t, archivePath, "# rules")

	a := &ClaudeAdapter{home: home}
	err := a.RestoreFiles([]FileEntry{{InArchive: "claude-code/memory/-Users-young-App/CLAUDE.md"}}, archiveDir, RestoreOpts{})
	if err != nil {
		t.Fatalf("RestoreFiles: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(home, ".claude", "projects", "-Users-young-App", "CLAUDE.md"))
	if err != nil {
		t.Fatalf("expected CLAUDE.md at project root: %v", err)
	}
	if string(got) != "# rules" {
		t.Fatalf("restored content = %q", got)
	}
}

func TestClaudeAdapter_ProjectFilterKeepsOnlyRequestedProject(t *testing.T) {
	entries := []FileEntry{
		{InArchive: "claude-code/projects/-Users-young-App/memory/MEMORY.md"},
		{InArchive: "claude-code/projects/-Users-young-Other/memory/MEMORY.md"},
		{InArchive: "claude-code/config/settings.json"},
	}

	filtered := FilterProjectEntries(entries, "/Users/young/App")
	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1", len(filtered))
	}
	if filtered[0].InArchive != "claude-code/projects/-Users-young-App/memory/MEMORY.md" {
		t.Fatalf("filtered path = %q", filtered[0].InArchive)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
