package rollback

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveLoadAndList(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state"))
	record := RollbackRecord{
		ID:           "20260521-143012-a1b2c3",
		Operation:    OperationRecipeApply,
		Created:      time.Date(2026, 5, 21, 14, 30, 12, 0, time.UTC),
		RecipeName:   "Team Recipe",
		RecipeDigest: "abcdef",
		Entries: []RollbackEntry{{
			LogicalPath: "codex-cli/agents/reviewer.md",
			TargetPath:  filepath.Join(t.TempDir(), "reviewer.md"),
			Action:      ActionWrite,
			Status:      StatusApplied,
			BeforeState: BeforeMissing,
			AfterSHA256: "after",
		}},
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load(record.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ID != record.ID || loaded.Entries[0].Status != StatusApplied {
		t.Fatalf("loaded mismatch: %#v", loaded)
	}
	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != record.ID {
		t.Fatalf("list = %#v", list)
	}
}

func TestStorePathFilterMatchesLogicalAndTargetDirectories(t *testing.T) {
	record := RollbackRecord{Entries: []RollbackEntry{
		{LogicalPath: "codex-cli/agents/reviewer.md", TargetPath: "/tmp/home/.codex/agents/reviewer.md"},
		{LogicalPath: "claude-code/skills/a/SKILL.md", TargetPath: "/tmp/home/.claude/skills/a/SKILL.md"},
	}}
	if got := record.FilterEntries("codex-cli/agents/"); len(got) != 1 || got[0].LogicalPath != "codex-cli/agents/reviewer.md" {
		t.Fatalf("logical dir match = %#v", got)
	}
	if got := record.FilterEntries("/tmp/home/.claude/skills/"); len(got) != 1 || got[0].LogicalPath != "claude-code/skills/a/SKILL.md" {
		t.Fatalf("target dir match = %#v", got)
	}
}

func TestStorePruneDryRun(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state"))
	for _, id := range []string{"20260520-100000-a", "20260521-100000-b", "20260522-100000-c"} {
		if err := store.Save(RollbackRecord{ID: id, Operation: OperationRecipeApply, Created: time.Now()}); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}
	plan, err := store.Prune(PruneOptions{Keep: 1, DryRun: true})
	if err != nil {
		t.Fatalf("Prune dry-run: %v", err)
	}
	if len(plan.DeletedIDs) != 2 {
		t.Fatalf("deleted ids = %#v, want 2", plan.DeletedIDs)
	}
	if _, err := os.Stat(filepath.Join(store.RollbacksDir(), "20260520-100000-a")); err != nil {
		t.Fatalf("dry-run should not delete: %v", err)
	}
}
