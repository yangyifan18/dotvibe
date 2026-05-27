package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestBuildArchiveCreatesRecipeManifest(t *testing.T) {
	src := t.TempDir()
	skill := filepath.Join(src, "SKILL.md")
	writeRecipeTestFile(t, skill, "# Reviewer\n")
	out := filepath.Join(t.TempDir(), "reviewer.vibe")

	result, err := BuildArchive(out, []adapters.FileEntry{{
		SourcePath: skill,
		InArchive:  "claude-code/skills/reviewer/SKILL.md",
		Category:   adapters.CategorySkills,
	}}, ExportOptions{Name: "Reviewer", Author: "yangyifan", IncludeSettings: true})
	if err != nil {
		t.Fatalf("BuildArchive: %v", err)
	}
	if result.WrittenFiles != 1 || result.RejectedFiles != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}

	ar, err := backup.ReadArchive(out)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ar.Close()
	if ar.Manifest.ArchiveKind != backup.ArchiveKindRecipe {
		t.Fatalf("archive kind = %q", ar.Manifest.ArchiveKind)
	}
	if ar.Manifest.Recipe == nil || ar.Manifest.Recipe.Name != "Reviewer" {
		t.Fatalf("recipe metadata = %#v", ar.Manifest.Recipe)
	}
	data, err := ar.ReadFile("claude-code/skills/reviewer/SKILL.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "# Reviewer\n" {
		t.Fatalf("recipe payload = %q", string(data))
	}
}

func TestBuildArchiveRejectsEmptyRecipe(t *testing.T) {
	out := filepath.Join(t.TempDir(), "empty.vibe")
	_, err := BuildArchive(out, nil, ExportOptions{Name: "Empty"})
	if err == nil {
		t.Fatal("expected empty recipe to fail")
	}
}

func writeRecipeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildArchiveDoesNotCarryProjectMetadata(t *testing.T) {
	dir := t.TempDir()
	agent := filepath.Join(dir, "reviewer.md")
	if err := os.WriteFile(agent, []byte("# reviewer\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "recipe.vibe")
	_, err := BuildArchive(out, []adapters.FileEntry{{SourcePath: agent, InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents}}, ExportOptions{Name: "Recipe"})
	if err != nil {
		t.Fatalf("BuildArchive: %v", err)
	}
	ar, err := backup.ReadArchive(out)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ar.Close()
	if ar.Manifest.SourceHome != "" || ar.Manifest.SourceUser != "" || len(ar.Manifest.Projects) != 0 {
		t.Fatalf("recipe should not carry project metadata: %#v", ar.Manifest)
	}
}
