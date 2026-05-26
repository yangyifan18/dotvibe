package agentapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestStageImportWritesArchiveAndLocalConflictFiles(t *testing.T) {
	archiveDir := t.TempDir()
	archiveFile := filepath.Join(archiveDir, "codex-cli", "agents", "reviewer.md")
	if err := writeAgentAPITestFile(archiveFile, "archive\n"); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(t.TempDir(), "reviewer.md")
	if err := writeAgentAPITestFile(local, "local\n"); err != nil {
		t.Fatal(err)
	}
	stageDir := filepath.Join(t.TempDir(), "stage")
	plan := ImportPlan{ArchivePath: "backup.tar.gz", Entries: []ImportPlanEntry{{Path: "codex-cli/agents/reviewer.md", ToolID: "codex-cli", Category: adapters.CategoryAgents, TargetPath: local, Action: "conflict", NeedsReview: true}}}
	result, err := StageImport(StageOptions{ArchiveDir: archiveDir, StageDir: stageDir, Plan: plan})
	if err != nil {
		t.Fatalf("StageImport: %v", err)
	}
	if result.StageDir != stageDir || result.FilesStaged != 1 || result.LocalCopies != 1 {
		t.Fatalf("result = %#v", result)
	}
	if data, err := os.ReadFile(filepath.Join(stageDir, "files", "codex-cli", "agents", "reviewer.md")); err != nil || string(data) != "archive\n" {
		t.Fatalf("staged archive file = %q err=%v", string(data), err)
	}
	if data, err := os.ReadFile(filepath.Join(stageDir, "local", "codex-cli", "agents", "reviewer.md")); err != nil || string(data) != "local\n" {
		t.Fatalf("staged local file = %q err=%v", string(data), err)
	}
	var written ImportPlan
	data, err := os.ReadFile(filepath.Join(stageDir, "import-plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatal(err)
	}
}
