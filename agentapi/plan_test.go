package agentapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestBuildExportPlanForRecipe(t *testing.T) {
	plan, err := BuildExportPlan(ExportPlanOptions{Profile: "recipe", Output: "team.vibe", Name: "Team", Author: "yyf", OnlyTools: []string{"codex-cli"}})
	if err != nil {
		t.Fatalf("BuildExportPlan: %v", err)
	}
	if plan.Command[0] != "dotvibe" || plan.Command[1] != "recipe" || plan.Command[2] != "export" {
		t.Fatalf("command = %#v", plan.Command)
	}
	if !plan.SafeToRun || plan.WritesArchive != true || plan.Risk != "shareable" {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestBuildExportPlanRejectsUnknownProfile(t *testing.T) {
	_, err := BuildExportPlan(ExportPlanOptions{Profile: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "unsupported export profile") {
		t.Fatalf("err = %v", err)
	}
}

func TestBuildImportPlanDetectsConflicts(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".codex", "agents", "reviewer.md")
	if err := writeAgentAPITestFile(target, "local\n"); err != nil {
		t.Fatal(err)
	}
	manifest := &backup.Manifest{ArchiveKind: backup.ArchiveKindFull, Tools: map[string]backup.ToolManifest{"codex-cli": {Included: []string{adapters.CategoryAgents}, FileCount: 1}}, Files: []backup.FileManifest{{Path: "codex-cli/agents/reviewer.md", ToolID: "codex-cli", Category: adapters.CategoryAgents, Size: 7, SHA256: "archive-sha"}}}
	plan, err := BuildImportPlan(ImportPlanOptions{Manifest: manifest, RestorePlan: []adapters.RestorePlanEntry{{FileEntry: adapters.FileEntry{InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents}, TargetPath: target, Action: adapters.RestoreSkip, Reason: "target exists"}}})
	if err != nil {
		t.Fatalf("BuildImportPlan: %v", err)
	}
	if plan.Summary.Conflicts != 1 || plan.RecommendedNextAction != "stage-or-choose-conflict-policy" {
		t.Fatalf("plan = %#v", plan)
	}
}

func writeAgentAPITestFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
