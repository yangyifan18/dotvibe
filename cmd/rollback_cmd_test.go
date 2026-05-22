package cmd

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yangyifan18/dotvibe/rollback"
)

func TestRunRollbackRestoresFileAndDeletesNewFile(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	targetExisting := filepath.Join(t.TempDir(), "existing.md")
	targetNew := filepath.Join(t.TempDir(), "new.md")
	writeFileForImportTest(t, targetExisting, "after\n")
	writeFileForImportTest(t, targetNew, "new\n")
	record := rollback.RollbackRecord{ID: "apply1", Operation: rollback.OperationRecipeApply, Created: time.Now()}
	oldSHA, oldBlob, err := rollback.WriteBlob(store.RecordDir(record.ID), []byte("before\n"))
	if err != nil {
		t.Fatalf("WriteBlob: %v", err)
	}
	record.Entries = []rollback.RollbackEntry{
		{LogicalPath: "codex-cli/agents/existing.md", TargetPath: targetExisting, Action: rollback.ActionOverwrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeFile, BeforeSHA256: oldSHA, BeforeBlob: oldBlob, AfterSHA256: fileSHAForCmdTest(t, targetExisting)},
		{LogicalPath: "codex-cli/agents/new.md", TargetPath: targetNew, Action: rollback.ActionWrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeMissing, AfterSHA256: fileSHAForCmdTest(t, targetNew)},
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollback("apply1", rollbackRunOptions{StateRoot: state, Yes: true}, &out); err != nil {
		t.Fatalf("runRollback: %v", err)
	}
	data, _ := os.ReadFile(targetExisting)
	if string(data) != "before\n" {
		t.Fatalf("existing = %q", string(data))
	}
	if _, err := os.Stat(targetNew); !os.IsNotExist(err) {
		t.Fatalf("new file should be deleted, stat err=%v", err)
	}
}

func TestRunRollbackRefusesChangedTargetWithoutForce(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	target := filepath.Join(t.TempDir(), "file.md")
	writeFileForImportTest(t, target, "changed-after-apply\n")
	record := rollback.RollbackRecord{ID: "apply1", Operation: rollback.OperationRecipeApply, Created: time.Now(), Entries: []rollback.RollbackEntry{{LogicalPath: "codex-cli/agents/file.md", TargetPath: target, Action: rollback.ActionWrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeMissing, AfterSHA256: strings.Repeat("0", 64)}}}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollback("apply1", rollbackRunOptions{StateRoot: state, Yes: true}, &out); err == nil {
		t.Fatal("expected changed target to be refused")
	}
}

func TestRunRollbackForceAllowsChangedNewFileTarget(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	target := filepath.Join(t.TempDir(), "file.md")
	writeFileForImportTest(t, target, "changed-after-apply\n")
	record := rollback.RollbackRecord{ID: "apply1", Operation: rollback.OperationRecipeApply, Created: time.Now(), Entries: []rollback.RollbackEntry{{LogicalPath: "codex-cli/agents/file.md", TargetPath: target, Action: rollback.ActionWrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeMissing, AfterSHA256: strings.Repeat("0", 64)}}}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollback("apply1", rollbackRunOptions{StateRoot: state, Yes: true, Force: true}, &out); err != nil {
		t.Fatalf("force rollback should remove changed new-file target: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("force rollback should delete target, stat err=%v", err)
	}
}

func fileSHAForCmdTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func TestRunRollbackBeforeMissingAllowsAlreadyDeletedTarget(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	target := filepath.Join(t.TempDir(), "new.md")
	writeFileForImportTest(t, target, "new\n")
	after := fileSHAForCmdTest(t, target)
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	record := rollback.RollbackRecord{ID: "apply1", Operation: rollback.OperationRecipeApply, Created: time.Now(), Entries: []rollback.RollbackEntry{{LogicalPath: "codex-cli/agents/new.md", TargetPath: target, Action: rollback.ActionWrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeMissing, AfterSHA256: after}}}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollback("apply1", rollbackRunOptions{StateRoot: state, Yes: true}, &out); err != nil {
		t.Fatalf("already deleted target should be treated as rolled back: %v", err)
	}
}

func TestRunRollbackRefusesDeletedOverwriteTargetWithoutForce(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	target := filepath.Join(t.TempDir(), "existing.md")
	writeFileForImportTest(t, target, "after\n")
	after := fileSHAForCmdTest(t, target)
	oldSHA, oldBlob, err := rollback.WriteBlob(store.RecordDir("apply1"), []byte("before\n"))
	if err != nil {
		t.Fatalf("WriteBlob: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	record := rollback.RollbackRecord{ID: "apply1", Operation: rollback.OperationRecipeApply, Created: time.Now(), Entries: []rollback.RollbackEntry{{LogicalPath: "codex-cli/agents/existing.md", TargetPath: target, Action: rollback.ActionOverwrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeFile, BeforeSHA256: oldSHA, BeforeBlob: oldBlob, AfterSHA256: after}}}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollback("apply1", rollbackRunOptions{StateRoot: state, Yes: true}, &out); err == nil {
		t.Fatal("expected deleted overwrite target to be refused")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("refused rollback should not restore deleted target, stat err=%v", err)
	}
}

func TestRunRollbackVerifiesBeforeBlobSHA(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	target := filepath.Join(t.TempDir(), "existing.md")
	writeFileForImportTest(t, target, "after\n")
	_, oldBlob, err := rollback.WriteBlob(store.RecordDir("apply1"), []byte("before\n"))
	if err != nil {
		t.Fatalf("WriteBlob: %v", err)
	}
	record := rollback.RollbackRecord{ID: "apply1", Operation: rollback.OperationRecipeApply, Created: time.Now(), Entries: []rollback.RollbackEntry{{LogicalPath: "codex-cli/agents/existing.md", TargetPath: target, Action: rollback.ActionOverwrite, Status: rollback.StatusApplied, BeforeState: rollback.BeforeFile, BeforeSHA256: strings.Repeat("0", 64), BeforeBlob: oldBlob, AfterSHA256: fileSHAForCmdTest(t, target)}}}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollback("apply1", rollbackRunOptions{StateRoot: state, Yes: true, Force: true}, &out); err == nil {
		t.Fatal("expected before blob hash mismatch to be refused")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "after\n" {
		t.Fatalf("hash mismatch should leave target unchanged, got %q", string(data))
	}
}

func TestRunRollbackListShowsDigestAndCounts(t *testing.T) {
	state := t.TempDir()
	store := rollback.NewStore(state)
	record := rollback.RollbackRecord{
		ID:           "apply1",
		Operation:    rollback.OperationRecipeApply,
		Created:      time.Now(),
		RecipeName:   "Recipe",
		RecipeDigest: strings.Repeat("a", 64),
		Entries: []rollback.RollbackEntry{
			{Action: rollback.ActionWrite, Status: rollback.StatusApplied},
			{Action: rollback.ActionOverwrite, Status: rollback.StatusFailed},
		},
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var out bytes.Buffer
	if err := runRollbackList(state, &out); err != nil {
		t.Fatalf("runRollbackList: %v", err)
	}
	output := out.String()
	for _, want := range []string{"digest=aaaaaaaaaaaa", "actions=write=1,overwrite=1", "statuses=applied=1,failed=1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("rollback list output missing %q:\n%s", want, output)
		}
	}
}
