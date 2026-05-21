package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeAdapter_Detect(t *testing.T) {
	tests := []struct {
		name  string
		setup func(string)
		want  bool
	}{
		{
			name: "settings file",
			setup: func(home string) {
				writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), "{}")
			},
			want: true,
		},
		{
			name: "projects directory",
			setup: func(home string) {
				if err := os.MkdirAll(filepath.Join(home, ".claude", "projects"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			want: true,
		},
		{name: "empty home", setup: func(string) {}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			tt.setup(home)
			a := &ClaudeAdapter{home: home}
			if got := a.Detect(); got != tt.want {
				t.Fatalf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeAdapter_ListFiles(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"test"}`)
	writeTestFile(t, filepath.Join(home, ".claude", "projects", "demo", "MEMORY.md"), "memory")

	a := &ClaudeAdapter{home: home}
	files := a.ListFiles(ExportOpts{})
	assertArchiveEntry(t, files, "claude-code/config/settings.json", CategoryConfig)
	assertArchiveEntry(t, files, "claude-code/projects/demo/MEMORY.md", CategoryMemory)
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
	_, err := a.RestoreFiles([]FileEntry{{InArchive: "claude-code/memory/-Users-young-App/CLAUDE.md"}}, archiveDir, RestoreOpts{})
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

func assertArchiveEntry(t *testing.T, files []FileEntry, path, category string) {
	t.Helper()
	for _, file := range files {
		if file.InArchive == path {
			if file.Category != category {
				t.Fatalf("%s category = %q, want %q", path, file.Category, category)
			}
			return
		}
	}
	t.Fatalf("archive entry %s not found in %#v", path, files)
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

func TestClaudeAdapter_PlanRestoreRejectsUnknownArchivePath(t *testing.T) {
	a := &ClaudeAdapter{home: t.TempDir()}
	_, err := a.PlanRestore([]FileEntry{{InArchive: "claude-code/unknown/file"}}, RestoreOpts{})
	if err == nil {
		t.Fatal("expected unknown archive path to be rejected")
	}
}

func TestClaudeAdapter_FilterRestoreEntriesAppliesProjectFilter(t *testing.T) {
	a := &ClaudeAdapter{home: t.TempDir()}
	entries := []FileEntry{
		{InArchive: "claude-code/projects/-Users-young-App/memory/MEMORY.md"},
		{InArchive: "claude-code/projects/-Users-young-Other/memory/MEMORY.md"},
	}
	filtered := a.FilterRestoreEntries(entries, RestoreOpts{Project: "-Users-young-App"})
	if len(filtered) != 1 || filtered[0].InArchive != entries[0].InArchive {
		t.Fatalf("unexpected filtered entries: %#v", filtered)
	}
}

func TestClaudeAdapter_DetectsProjectMemoryWithoutSettings(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".claude", "projects", "demo", "MEMORY.md"), "memory")

	a := &ClaudeAdapter{home: home}
	if !a.Detect() {
		t.Fatal("expected project memory directory to detect Claude Code data")
	}
}

func TestClaudeAdapter_ListFilesIncludesProjectRootMemory(t *testing.T) {
	home := t.TempDir()
	memoryPath := filepath.Join(home, ".claude", "projects", "demo", "MEMORY.md")
	writeTestFile(t, memoryPath, "memory")

	a := &ClaudeAdapter{home: home}
	files := a.ListFiles(ExportOpts{})
	for _, file := range files {
		if file.InArchive == "claude-code/projects/demo/MEMORY.md" && file.Category == CategoryMemory {
			return
		}
	}
	t.Fatalf("project root MEMORY.md not exported: %#v", files)
}
