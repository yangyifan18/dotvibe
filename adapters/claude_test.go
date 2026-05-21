package adapters

import (
	"os"
	"path/filepath"
	"strings"
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

func TestClaudeAdapter_ListRecipeFilesExcludesProjectMemory(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark"}`)
	writeTestFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "# Global rule\n")
	writeTestFile(t, filepath.Join(home, ".claude", "skills", "reviewer", "SKILL.md"), "# Skill\n")
	writeTestFile(t, filepath.Join(home, ".claude", "agents", "planner.md"), "# Planner\n")
	writeTestFile(t, filepath.Join(home, ".claude", "commands", "ship.md"), "/ship\n")
	writeTestFile(t, filepath.Join(home, ".claude", "projects", "secret", "MEMORY.md"), "private project\n")
	writeTestFile(t, filepath.Join(home, ".claude", "transcripts", "session.jsonl"), "private transcript\n")

	adapter := &ClaudeAdapter{home: home}
	entries := adapter.ListRecipeFiles(RecipeOpts{IncludeSettings: true})
	paths := entryArchivePathsForTest(entries)

	assertContainsString(t, paths, "claude-code/config/settings.json")
	assertContainsString(t, paths, "claude-code/rules/CLAUDE.md")
	assertContainsString(t, paths, "claude-code/skills/reviewer/SKILL.md")
	assertContainsString(t, paths, "claude-code/agents/planner.md")
	assertContainsString(t, paths, "claude-code/commands/ship.md")
	assertNotContainsPrefix(t, paths, "claude-code/projects/")
	assertNotContainsPrefix(t, paths, "claude-code/transcripts/")
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

func TestClaudeAdapter_RecipePathsRestoreToShareableRoots(t *testing.T) {
	home := t.TempDir()
	adapter := &ClaudeAdapter{home: home}
	cases := map[string]string{
		"claude-code/rules/CLAUDE.md":           filepath.Join(home, ".claude", "CLAUDE.md"),
		"claude-code/agents/planner.md":         filepath.Join(home, ".claude", "agents", "planner.md"),
		"claude-code/commands/ship.md":          filepath.Join(home, ".claude", "commands", "ship.md"),
		"claude-code/skills/reviewer/SKILL.md":  filepath.Join(home, ".claude", "skills", "reviewer", "SKILL.md"),
		"claude-code/plugins/codex/plugin.json": filepath.Join(home, ".claude", "plugins", "codex", "plugin.json"),
		"claude-code/config/settings.json":      filepath.Join(home, ".claude", "settings.json"),
	}
	for archivePath, want := range cases {
		got, err := adapter.adaptPath(archivePath)
		if err != nil {
			t.Fatalf("adaptPath(%s): %v", archivePath, err)
		}
		if got != want {
			t.Fatalf("adaptPath(%s) = %s, want %s", archivePath, got, want)
		}
	}
}

func TestCodexAdapter_RecipeRulesRestoreToConfigRoot(t *testing.T) {
	home := t.TempDir()
	adapter := &CodexAdapter{home: home}
	got, err := adapter.adaptPath("codex-cli/rules/AGENTS.md")
	if err != nil {
		t.Fatalf("adaptPath: %v", err)
	}
	want := filepath.Join(home, ".codex", "AGENTS.md")
	if got != want {
		t.Fatalf("adaptPath = %s, want %s", got, want)
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

func entryArchivePathsForTest(entries []FileEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.InArchive)
	}
	return paths
}

func assertContainsString(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("%q not found in %#v", want, items)
}

func assertNotContainsPrefix(t *testing.T, items []string, prefix string) {
	t.Helper()
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			t.Fatalf("unexpected path with prefix %q in %#v", prefix, items)
		}
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
