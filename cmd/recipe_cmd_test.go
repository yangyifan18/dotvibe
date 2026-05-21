package cmd

import (
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
