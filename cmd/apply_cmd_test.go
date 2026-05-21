package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/recipe"
)

func TestReadRecipeArchiveRejectsBackupArchive(t *testing.T) {
	backupPath := createDiffArchive(t, map[string]string{"codex-cli/agents/a.md": "agent"})
	_, err := readRecipeArchive(backupPath)
	if err == nil {
		t.Fatal("expected non-recipe archive to be rejected")
	}
}

func TestGroupRecipeFilesByTool(t *testing.T) {
	files := []string{"claude-code/skills/a/SKILL.md", "codex-cli/agents/reviewer.md"}
	grouped := groupRecipeFilesByTool(files)
	if len(grouped["claude-code"]) != 1 || len(grouped["codex-cli"]) != 1 {
		t.Fatalf("grouped = %#v", grouped)
	}
}

func TestBuildApplyPreviewUsesAdapterPlans(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()

	recipePath := filepath.Join(t.TempDir(), "agents.vibe")
	src := t.TempDir()
	agent := filepath.Join(src, "reviewer.md")
	writeFileForImportTest(t, agent, "# Reviewer\n")
	_, err := recipe.BuildArchive(recipePath, []adapters.FileEntry{{SourcePath: agent, InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents}}, recipe.ExportOptions{Name: "Agents"})
	if err != nil {
		t.Fatalf("BuildArchive: %v", err)
	}
	ar, err := readRecipeArchive(recipePath)
	if err != nil {
		t.Fatalf("readRecipeArchive: %v", err)
	}
	defer ar.Close()
	preview, err := buildApplyPreview(groupRecipeFilesByTool(ar.ListFiles()), adapters.RestoreOpts{})
	if err != nil {
		t.Fatalf("buildApplyPreview: %v", err)
	}
	if len(preview) != 1 || preview[0].Action != adapters.RestoreWrite {
		t.Fatalf("preview = %#v", preview)
	}
}

func testSetHome(t *testing.T, home string) func() {
	t.Helper()
	old := os.Getenv("HOME")
	t.Setenv("HOME", home)
	return func() { os.Setenv("HOME", old) }
}
