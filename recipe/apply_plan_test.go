package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestBuildApplyPlanClassifiesWriteSameAndConflict(t *testing.T) {
	dir := t.TempDir()
	missingTarget := filepath.Join(dir, "missing.md")
	sameTarget := filepath.Join(dir, "same.md")
	conflictTarget := filepath.Join(dir, "conflict.md")
	writeRecipeTestFile(t, sameTarget, "same\n")
	writeRecipeTestFile(t, conflictTarget, "local\n")
	entries := []ApplyInput{
		{Entry: adapters.FileEntry{InArchive: "codex-cli/agents/missing.md"}, TargetPath: missingTarget, RecipeContent: []byte("new\n")},
		{Entry: adapters.FileEntry{InArchive: "codex-cli/agents/same.md"}, TargetPath: sameTarget, RecipeContent: []byte("same\n")},
		{Entry: adapters.FileEntry{InArchive: "codex-cli/agents/conflict.md"}, TargetPath: conflictTarget, RecipeContent: []byte("recipe\n")},
	}
	plan, err := BuildApplyPlan(entries)
	if err != nil {
		t.Fatalf("BuildApplyPlan: %v", err)
	}
	want := map[string]string{
		"codex-cli/agents/missing.md":  ApplyActionWrite,
		"codex-cli/agents/same.md":     ApplyActionSame,
		"codex-cli/agents/conflict.md": ApplyActionConflict,
	}
	for _, entry := range plan.Entries {
		if entry.Action != want[entry.Entry.InArchive] {
			t.Fatalf("%s action = %s, want %s", entry.Entry.InArchive, entry.Action, want[entry.Entry.InArchive])
		}
	}
}

func TestResolveConflictDecisionsForNonInteractiveModes(t *testing.T) {
	conflicts := []ApplyPlanEntry{{Action: ApplyActionConflict}, {Action: ApplyActionConflict}}
	keep := ResolveNonInteractiveConflicts(conflicts, ConflictOptions{Yes: true})
	if keep[0].ResolvedAction != ApplyActionSkip || keep[1].ResolvedAction != ApplyActionSkip {
		t.Fatalf("--yes should skip conflicts: %#v", keep)
	}
	overwrite := ResolveNonInteractiveConflicts(conflicts, ConflictOptions{Yes: true, Force: true})
	if overwrite[0].ResolvedAction != ApplyActionOverwrite || overwrite[1].ResolvedAction != ApplyActionOverwrite {
		t.Fatalf("--force --yes should overwrite conflicts: %#v", overwrite)
	}
}

func TestIncomingPathUsesStateDirAndLogicalPath(t *testing.T) {
	got := IncomingPath(filepath.Join("/tmp", "state"), "apply-id", "codex-cli/agents/reviewer.md")
	want := filepath.Join("/tmp", "state", "incoming", "apply-id", "codex-cli", "agents", "reviewer.md")
	if got != want {
		t.Fatalf("IncomingPath = %s, want %s", got, want)
	}
	if err := os.MkdirAll(filepath.Dir(got), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestApplyConflictChoiceAllStrategies(t *testing.T) {
	entries := []ApplyPlanEntry{
		{Action: ApplyActionConflict, ResolvedAction: ApplyActionConflict},
		{Action: ApplyActionConflict, ResolvedAction: ApplyActionConflict},
	}
	resolved := ApplyConflictChoice(entries, ConflictChoiceUseAll)
	if resolved[0].ResolvedAction != ApplyActionOverwrite || resolved[1].ResolvedAction != ApplyActionOverwrite {
		t.Fatalf("use all = %#v", resolved)
	}
	resolved = ApplyConflictChoice(entries, ConflictChoiceSaveAll)
	if resolved[0].ResolvedAction != ApplyActionSave || resolved[1].ResolvedAction != ApplyActionSave {
		t.Fatalf("save all = %#v", resolved)
	}
	resolved = ApplyConflictChoice(entries, ConflictChoiceKeepAll)
	if resolved[0].ResolvedAction != ApplyActionSkip || resolved[1].ResolvedAction != ApplyActionSkip {
		t.Fatalf("keep all = %#v", resolved)
	}
}
