package recipe

import (
	"encoding/json"
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

func TestDiffArchivesRejectsRecipeKindArchiveMissingRecipeMetadata(t *testing.T) {
	missingMetadata := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/a.md": {content: "a\n", category: adapters.CategoryAgents},
	}, nil)
	valid := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/a.md": {content: "a\n", category: adapters.CategoryAgents},
	}, validRecipeMetadata("Valid"))

	if _, err := DiffArchives(missingMetadata, valid, DiffOptions{}); err == nil {
		t.Fatal("expected diff to reject recipe-kind archive with nil recipe metadata")
	}
}

func TestDiffArchivesRejectsUnsupportedRecipeSchema(t *testing.T) {
	unsupportedMeta := validRecipeMetadata("Unsupported")
	unsupportedMeta.Schema = "dotvibe.recipe.v0"
	unsupported := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/a.md": {content: "a\n", category: adapters.CategoryAgents},
	}, unsupportedMeta)
	valid := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/a.md": {content: "a\n", category: adapters.CategoryAgents},
	}, validRecipeMetadata("Valid"))

	if _, err := DiffArchives(unsupported, valid, DiffOptions{}); err == nil {
		t.Fatal("expected diff to reject unsupported recipe schema")
	}
}

func TestDiffArchivesJSONUsesEmptyArrays(t *testing.T) {
	left := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/same.md": {content: "same\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "left"})
	right := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/same.md": {content: "same\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "right"})

	diff, err := DiffArchives(left, right, DiffOptions{})
	if err != nil {
		t.Fatalf("DiffArchives: %v", err)
	}
	if diff.Added == nil || diff.Removed == nil || diff.Changed == nil {
		t.Fatalf("diff slices should be non-nil: %#v", diff)
	}
	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Marshal RecipeDiff: %v", err)
	}
	got := string(data)
	for _, want := range []string{`"added":[]`, `"removed":[]`, `"changed":[]`} {
		if !strings.Contains(got, want) {
			t.Fatalf("diff should marshal %s, got %s", want, got)
		}
	}
}

func TestUnifiedTextDiffTreatsInsertedLineAsInsertion(t *testing.T) {
	diff := UnifiedTextDiff("old.txt", "new.txt", []byte("a\nb\nc\n"), []byte("a\nx\nb\nc\n"))
	if !strings.Contains(diff, "\n+x\n b\n c\n") {
		t.Fatalf("insertion should keep following lines unchanged:\n%s", diff)
	}
	if strings.Contains(diff, "\n-b\n+x\n") {
		t.Fatalf("insertion should not be represented as replacing b:\n%s", diff)
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
