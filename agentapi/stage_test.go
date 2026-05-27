package agentapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
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
	manifest := &backup.Manifest{Version: "1.0.0", Tools: map[string]backup.ToolManifest{"codex-cli": {Included: []string{adapters.CategoryAgents}, FileCount: 1}}}
	result, err := StageImport(StageOptions{ArchiveDir: archiveDir, StageDir: stageDir, Plan: plan, Manifest: manifest})
	if err != nil {
		t.Fatalf("StageImport: %v", err)
	}
	if result.StageDir != stageDir || result.FilesStaged != 1 || result.LocalCopies != 1 || result.ManifestPath == "" {
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
	if _, err := os.Stat(filepath.Join(stageDir, "manifest.json")); err != nil {
		t.Fatalf("missing staged manifest: %v", err)
	}
}

func TestStageImportRejectsUnsafeLogicalPath(t *testing.T) {
	archiveDir := t.TempDir()
	stageDir := filepath.Join(t.TempDir(), "stage")
	plan := ImportPlan{Entries: []ImportPlanEntry{{Path: "../secrets.txt", Action: "write"}}}
	_, err := StageImport(StageOptions{ArchiveDir: archiveDir, StageDir: stageDir, Plan: plan})
	if err == nil || !strings.Contains(err.Error(), "unsafe stage path") {
		t.Fatalf("err = %v", err)
	}
}

func TestStageImportRejectsExistingStageDir(t *testing.T) {
	archiveDir := t.TempDir()
	stageDir := filepath.Join(t.TempDir(), "stage")
	if err := os.Mkdir(stageDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := StageImport(StageOptions{ArchiveDir: archiveDir, StageDir: stageDir, Plan: ImportPlan{}})
	if err == nil || !strings.Contains(err.Error(), "stage dir already exists") {
		t.Fatalf("err = %v", err)
	}
}
