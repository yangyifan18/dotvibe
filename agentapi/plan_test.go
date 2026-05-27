package agentapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/projectmeta"
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

func TestBuildImportPlanReportsOverwriteSeparately(t *testing.T) {
	manifest := &backup.Manifest{ArchiveKind: backup.ArchiveKindFull, Tools: map[string]backup.ToolManifest{"codex-cli": {Included: []string{adapters.CategoryAgents}, FileCount: 1}}, Files: []backup.FileManifest{{Path: "codex-cli/agents/reviewer.md", ToolID: "codex-cli", Category: adapters.CategoryAgents, Size: 7, SHA256: "archive-sha"}}}
	plan, err := BuildImportPlan(ImportPlanOptions{Manifest: manifest, RestorePlan: []adapters.RestorePlanEntry{{FileEntry: adapters.FileEntry{InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents}, TargetPath: "/tmp/reviewer.md", Action: adapters.RestoreOverwrite, Reason: "target exists and --force is set"}}})
	if err != nil {
		t.Fatalf("BuildImportPlan: %v", err)
	}
	if plan.Summary.Overwrites != 1 || plan.Summary.Conflicts != 0 || plan.Entries[0].Action != "overwrite" || plan.RecommendedNextAction != "confirm-overwrite" {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestBuildImportPlanAddsUnsupportedSelectedFiles(t *testing.T) {
	manifest := &backup.Manifest{ArchiveKind: backup.ArchiveKindFull, Tools: map[string]backup.ToolManifest{"unknown-tool": {Included: []string{adapters.CategoryConfig}, FileCount: 1}}, Files: []backup.FileManifest{{Path: "unknown-tool/config.json", ToolID: "unknown-tool", Category: adapters.CategoryConfig, Size: 2, SHA256: "sha"}}}
	plan, err := BuildImportPlan(ImportPlanOptions{Manifest: manifest, SelectedFiles: []string{"unknown-tool/config.json"}})
	if err != nil {
		t.Fatalf("BuildImportPlan: %v", err)
	}
	if plan.Summary.Unsupported != 1 || plan.Entries[0].Action != "unsupported" || plan.RecommendedNextAction != "fix-unsupported-paths" {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestBuildImportPlanIncludesProjectRelocations(t *testing.T) {
	destHome := filepath.Join(t.TempDir(), "youtopia")
	manifest := &backup.Manifest{
		ArchiveKind: backup.ArchiveKindFull,
		Tools:       map[string]backup.ToolManifest{"claude-code": {Included: []string{adapters.CategoryMemory}, FileCount: 1}},
		Projects: []backup.ProjectManifest{{
			ToolID:         "claude-code",
			ProjectKey:     "-Users-young-Softwares-dotvibe",
			SourcePath:     "/Users/young/Softwares/dotvibe",
			SourceHome:     "/Users/young",
			RelativeToHome: "Softwares/dotvibe",
			PathScope:      backup.ProjectPathScopeHome,
			Git: backup.ProjectGitMetadata{IsRepo: true, Remotes: []backup.ProjectGitRemote{{
				Name: "origin", URL: "git@github.com:yangyifan18/dotvibe.git", Sanitized: true, Cloneable: true,
			}}},
		}},
	}
	plan, err := BuildImportPlan(ImportPlanOptions{Manifest: manifest, DestinationHome: destHome, DestinationUser: "youtopia", EnableHomeRemap: true})
	if err != nil {
		t.Fatalf("BuildImportPlan: %v", err)
	}
	if plan.Destination.Home != destHome || len(plan.ProjectRelocations) != 1 {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.ProjectRelocations[0].Clone.Command[0] != "git" || plan.ProjectRelocations[0].RecommendedNextAction != projectmeta.ActionConfirmCloneThenStageMemory {
		t.Fatalf("relocation = %#v", plan.ProjectRelocations[0])
	}
}

func writeAgentAPITestFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func TestBuildExportPlanUsesSingleOnlyFlagForMultipleTools(t *testing.T) {
	plan, err := BuildExportPlan(ExportPlanOptions{Profile: "full", Output: "backup.tar.gz", OnlyTools: []string{"codex-cli", "claude-code"}})
	if err != nil {
		t.Fatalf("BuildExportPlan: %v", err)
	}
	var onlyCount int
	for i, arg := range plan.Command {
		if arg == "--only" {
			onlyCount++
			if i+1 >= len(plan.Command) || plan.Command[i+1] != "codex-cli,claude-code" {
				t.Fatalf("--only args = %#v", plan.Command)
			}
		}
	}
	if onlyCount != 1 {
		t.Fatalf("--only count = %d, command = %#v", onlyCount, plan.Command)
	}
}

func TestBuildExportPlanRejectsConflictingProjectMemoryOnlyTools(t *testing.T) {
	_, err := BuildExportPlan(ExportPlanOptions{Profile: "project-memory", Output: "backup.tar.gz", OnlyTools: []string{"codex-cli"}})
	if err == nil || !strings.Contains(err.Error(), "project-memory profile only supports claude-code") {
		t.Fatalf("err = %v", err)
	}
}
