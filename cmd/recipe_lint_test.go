package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/recipe"
)

func TestRunRecipeLintReturnsErrorForSecret(t *testing.T) {
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/leak.md": "sk-proj-abcdefghijklmnopqrstuvwxyz123456\n"})
	var buf bytes.Buffer
	err := runRecipeLint(path, recipeLintOptions{JSON: false, Strict: false, ScanContent: true}, &buf)
	if err == nil {
		t.Fatal("expected lint error for secret")
	}
	if !strings.Contains(buf.String(), "openai_key") {
		t.Fatalf("lint output missing code: %s", buf.String())
	}
}

func TestRunRecipeLintJSON(t *testing.T) {
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/reviewer.md": "# Reviewer\n"})
	var buf bytes.Buffer
	if err := runRecipeLint(path, recipeLintOptions{JSON: true, ScanContent: true}, &buf); err != nil {
		t.Fatalf("runRecipeLint: %v", err)
	}
	if !strings.Contains(buf.String(), `"findings"`) {
		t.Fatalf("JSON output missing findings: %s", buf.String())
	}
}

func buildRecipeCommandFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	src := t.TempDir()
	entries := make([]adapters.FileEntry, 0, len(files))
	for archivePath, content := range files {
		source := filepath.Join(src, filepath.Base(archivePath))
		writeFileForImportTest(t, source, content)
		entries = append(entries, adapters.FileEntry{SourcePath: source, InArchive: archivePath, Category: adapters.CategoryAgents})
	}
	out := filepath.Join(t.TempDir(), "recipe.vibe")
	if _, err := recipe.BuildArchive(out, entries, recipe.ExportOptions{Name: "Recipe"}); err != nil {
		t.Fatalf("BuildArchive: %v", err)
	}
	return out
}
