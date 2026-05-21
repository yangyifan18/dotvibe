package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestCollectRecipeEntriesFiltersOnlyTools(t *testing.T) {
	entries := []adapters.FileEntry{
		{InArchive: "claude-code/skills/a/SKILL.md", Category: adapters.CategorySkills},
		{InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents},
	}
	filtered := filterRecipeEntriesByOnly(entries, []string{"codex-cli"})
	if len(filtered) != 1 || filtered[0].InArchive != "codex-cli/agents/reviewer.md" {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestDefaultRecipeOutputUsesVibeExtension(t *testing.T) {
	got := defaultRecipeOutput("YYF Vibe Stack")
	if got == "" || filepath.Ext(got) != ".vibe" {
		t.Fatalf("default output = %q, want .vibe", got)
	}
}

func TestCollectRecipeEntriesSkipsSymlinks(t *testing.T) {
	home := t.TempDir()
	source := filepath.Join(home, "secret.txt")
	if err := os.WriteFile(source, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "skill-link.md")
	if err := os.Symlink(source, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	regular := filepath.Join(home, "regular.md")
	if err := os.WriteFile(regular, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	entries := collectRecipeEntries([]adapters.Adapter{fakeRecipeAdapter{entries: []adapters.FileEntry{
		{SourcePath: link, InArchive: "codex-cli/agents/link.md", Category: adapters.CategoryAgents},
		{SourcePath: regular, InArchive: "codex-cli/agents/regular.md", Category: adapters.CategoryAgents},
	}}}, nil, adapters.RecipeOpts{})

	if len(entries) != 1 || entries[0].InArchive != "codex-cli/agents/regular.md" {
		t.Fatalf("entries = %#v, want only regular file", entries)
	}
}

type fakeRecipeAdapter struct {
	entries []adapters.FileEntry
}

func (a fakeRecipeAdapter) Name() string { return "Fake" }
func (a fakeRecipeAdapter) ID() string   { return "codex-cli" }
func (a fakeRecipeAdapter) Detect() bool { return true }
func (a fakeRecipeAdapter) ListFiles(opts adapters.ExportOpts) []adapters.FileEntry {
	return nil
}
func (a fakeRecipeAdapter) ListRecipeFiles(opts adapters.RecipeOpts) []adapters.FileEntry {
	return a.entries
}
func (a fakeRecipeAdapter) Status() adapters.ToolStatus { return adapters.ToolStatus{} }
func (a fakeRecipeAdapter) FilterRestoreEntries(entries []adapters.FileEntry, opts adapters.RestoreOpts) []adapters.FileEntry {
	return entries
}
func (a fakeRecipeAdapter) PlanRestore(entries []adapters.FileEntry, opts adapters.RestoreOpts) ([]adapters.RestorePlanEntry, error) {
	return nil, nil
}
func (a fakeRecipeAdapter) RestoreFiles(entries []adapters.FileEntry, archiveDir string, opts adapters.RestoreOpts) (adapters.RestoreSummary, error) {
	return adapters.RestoreSummary{}, nil
}
