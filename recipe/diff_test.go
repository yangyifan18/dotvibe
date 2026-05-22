package recipe

import (
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestDiffArchivesReportsAddedRemovedChangedSame(t *testing.T) {
	left := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/same.md":    {content: "same\n", category: adapters.CategoryAgents},
		"codex-cli/agents/changed.md": {content: "old\n", category: adapters.CategoryAgents},
		"codex-cli/agents/removed.md": {content: "removed\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "left"})
	right := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/same.md":    {content: "same\n", category: adapters.CategoryAgents},
		"codex-cli/agents/changed.md": {content: "new\n", category: adapters.CategoryAgents},
		"codex-cli/agents/added.md":   {content: "added\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "right"})

	diff, err := DiffArchives(left, right, DiffOptions{})
	if err != nil {
		t.Fatalf("DiffArchives: %v", err)
	}
	assertRecipeDiffPaths(t, diff.Added, []string{"codex-cli/agents/added.md"})
	assertRecipeDiffPaths(t, diff.Removed, []string{"codex-cli/agents/removed.md"})
	assertRecipeDiffPaths(t, diff.Changed, []string{"codex-cli/agents/changed.md"})
	if diff.SameCount != 1 {
		t.Fatalf("same count = %d, want 1", diff.SameCount)
	}
}

func TestDiffArchivesContentDiffOnlyWhenRequested(t *testing.T) {
	left := buildTestRecipeArchive(t, map[string]testRecipeFile{"codex-cli/agents/changed.md": {content: "old\n", category: adapters.CategoryAgents}}, ExportOptions{Name: "left"})
	right := buildTestRecipeArchive(t, map[string]testRecipeFile{"codex-cli/agents/changed.md": {content: "new\n", category: adapters.CategoryAgents}}, ExportOptions{Name: "right"})
	withoutContent, err := DiffArchives(left, right, DiffOptions{IncludeContent: false})
	if err != nil {
		t.Fatalf("DiffArchives without content: %v", err)
	}
	if withoutContent.Changed[0].ContentDiffStatus != ContentDiffNotRequested || withoutContent.Changed[0].ContentDiff != "" {
		t.Fatalf("unexpected content diff without --content: %#v", withoutContent.Changed[0])
	}
	withContent, err := DiffArchives(left, right, DiffOptions{IncludeContent: true})
	if err != nil {
		t.Fatalf("DiffArchives with content: %v", err)
	}
	if withContent.Changed[0].ContentDiffStatus != ContentDiffText || !strings.Contains(withContent.Changed[0].ContentDiff, "-old") || !strings.Contains(withContent.Changed[0].ContentDiff, "+new") {
		t.Fatalf("missing content diff: %#v", withContent.Changed[0])
	}
}

func assertRecipeDiffPaths(t *testing.T, entries []DiffEntry, want []string) {
	t.Helper()
	if len(entries) != len(want) {
		t.Fatalf("entries = %#v, want %#v", entries, want)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Path] = true
	}
	for _, path := range want {
		if !seen[path] {
			t.Fatalf("entries = %#v, want path %s", entries, path)
		}
	}
}
