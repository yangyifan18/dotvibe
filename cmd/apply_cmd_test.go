package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/recipe"
	"github.com/yangyifan18/dotvibe/rollback"
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

func TestRecipeApplyBlocksLintErrorsUnlessAllowRisk(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/leak.md": "sk-proj-abcdefghijklmnopqrstuvwxyz123456\n"})
	var out bytes.Buffer
	err := runRecipeApply(path, recipeApplyOptions{Yes: true, DryRun: true, AllowRisk: false, ScanContent: true}, &out)
	if err == nil {
		t.Fatal("expected lint error to block apply")
	}
	if err := runRecipeApply(path, recipeApplyOptions{Yes: true, DryRun: true, AllowRisk: true, ScanContent: true}, &out); err != nil {
		t.Fatalf("allow-risk dry-run should continue: %v", err)
	}
}

func TestRecipeApplyDryRunDoesNotCreateRollback(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	state := t.TempDir()
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/reviewer.md": "# Reviewer\n"})
	var out bytes.Buffer
	if err := runRecipeApply(path, recipeApplyOptions{Yes: true, DryRun: true, StateRoot: state, ScanContent: true}, &out); err != nil {
		t.Fatalf("runRecipeApply dry-run: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "rollbacks")); len(entries) != 0 {
		t.Fatalf("dry-run created rollback entries: %#v", entries)
	}
}

func TestRecipeApplyCreatesRollbackForNewFile(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	state := t.TempDir()
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/reviewer.md": "# Reviewer\n"})
	var out bytes.Buffer
	if err := runRecipeApply(path, recipeApplyOptions{Yes: true, StateRoot: state, ScanContent: true}, &out); err != nil {
		t.Fatalf("runRecipeApply: %v", err)
	}
	store := rollback.NewStore(state)
	records, err := store.List()
	if err != nil {
		t.Fatalf("List rollbacks: %v", err)
	}
	if len(records) != 1 || len(records[0].Entries) != 1 {
		t.Fatalf("records = %#v", records)
	}
	entry := records[0].Entries[0]
	if entry.BeforeState != rollback.BeforeMissing || entry.Status != rollback.StatusApplied || entry.Action != rollback.ActionWrite {
		t.Fatalf("unexpected rollback entry: %#v", entry)
	}
}

func TestRecipeApplyForceOverwriteStoresBeforeBlob(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	target := filepath.Join(home, ".codex", "agents", "reviewer.md")
	writeFileForImportTest(t, target, "local\n")
	state := t.TempDir()
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/reviewer.md": "recipe\n"})
	var out bytes.Buffer
	if err := runRecipeApply(path, recipeApplyOptions{Yes: true, Force: true, StateRoot: state, ScanContent: true}, &out); err != nil {
		t.Fatalf("runRecipeApply: %v", err)
	}
	store := rollback.NewStore(state)
	records, _ := store.List()
	entry := records[0].Entries[0]
	if entry.BeforeState != rollback.BeforeFile || entry.BeforeBlob == "" || entry.Action != rollback.ActionOverwrite {
		t.Fatalf("expected before blob for overwrite: %#v", entry)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "recipe\n" {
		t.Fatalf("target = %q", string(data))
	}
}

func TestRecipeApplyPromptsWhenYesIsNotSet(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	state := t.TempDir()
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/reviewer.md": "# Reviewer\n"})
	oldInput := recipeApplyInput
	recipeApplyInput = strings.NewReader("n\n")
	defer func() { recipeApplyInput = oldInput }()
	var out bytes.Buffer
	if err := runRecipeApply(path, recipeApplyOptions{StateRoot: state, ScanContent: true}, &out); err == nil {
		t.Fatal("expected cancelled apply to return an error")
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "agents", "reviewer.md")); !os.IsNotExist(err) {
		t.Fatalf("apply wrote target despite cancellation: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "rollbacks")); len(entries) != 0 {
		t.Fatalf("cancelled apply created rollback entries: %#v", entries)
	}
}
