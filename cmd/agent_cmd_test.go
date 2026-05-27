package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/agentapi"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/projectmeta"
)

func TestRunAgentDoctorJSON(t *testing.T) {
	var out bytes.Buffer
	err := runAgentDoctor(agentDoctorOptions{JSON: true, Version: "test"}, &out)
	if err != nil {
		t.Fatalf("runAgentDoctor: %v", err)
	}
	var report agentapi.DoctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if report.DotvibeVersion != "test" || !report.Capabilities.AgentDoctor {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunAgentDoctorHuman(t *testing.T) {
	var out bytes.Buffer
	if err := runAgentDoctor(agentDoctorOptions{JSON: false, Version: "test"}, &out); err != nil {
		t.Fatalf("runAgentDoctor human: %v", err)
	}
	if !strings.Contains(out.String(), "dotvibe agent doctor") || !strings.Contains(out.String(), "Capabilities") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRunAgentInventoryJSON(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	writeFileForImportTest(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "# Reviewer\n")
	var out bytes.Buffer
	if err := runAgentInventory(agentInventoryOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentInventory: %v", err)
	}
	if !strings.Contains(out.String(), `"recommended_profiles"`) || !strings.Contains(out.String(), `"codex-cli"`) {
		t.Fatalf("unexpected inventory JSON: %s", out.String())
	}
}

func TestRunAgentExportPlanJSON(t *testing.T) {
	var out bytes.Buffer
	if err := runAgentExportPlan(agentExportPlanOptions{JSON: true, Profile: "recipe", Output: "team.vibe", Name: "Team", Author: "yyf"}, &out); err != nil {
		t.Fatalf("runAgentExportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"profile": "recipe"`) || !strings.Contains(out.String(), `"recipe"`) {
		t.Fatalf("unexpected export plan: %s", out.String())
	}
}

func TestRunAgentImportPlanJSON(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	writeFileForImportTest(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "local\n")
	archive := createAgentPlanArchive(t, map[string]string{"codex-cli/agents/reviewer.md": "archive\n"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"conflicts": 1`) || !strings.Contains(out.String(), `"recommended_next_action": "stage-or-choose-conflict-policy"`) {
		t.Fatalf("unexpected import plan: %s", out.String())
	}
}

func TestRunAgentImportPlanForceReportsOverwrite(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	writeFileForImportTest(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "local\n")
	archive := createAgentPlanArchive(t, map[string]string{"codex-cli/agents/reviewer.md": "archive\n"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true, Force: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"overwrites": 1`) || !strings.Contains(out.String(), `"action": "overwrite"`) || strings.Contains(out.String(), `"conflicts": 1`) {
		t.Fatalf("unexpected force import plan: %s", out.String())
	}
}

func TestRunAgentImportPlanDetectsIdenticalFiles(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	writeFileForImportTest(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "same\n")
	archive := createAgentPlanArchive(t, map[string]string{"codex-cli/agents/reviewer.md": "same\n"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"identical": 1`) || !strings.Contains(out.String(), `"action": "same"`) {
		t.Fatalf("unexpected identical import plan: %s", out.String())
	}
}

func TestRunAgentImportPlanReportsUnsupportedFiles(t *testing.T) {
	archive := createAgentPlanArchive(t, map[string]string{"unknown-tool/config.json": "{}"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"unsupported": 1`) || !strings.Contains(out.String(), `"recommended_next_action": "fix-unsupported-paths"`) {
		t.Fatalf("unexpected unsupported import plan: %s", out.String())
	}
}

func createAgentPlanArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	return createDiffArchive(t, files)
}

func TestRunAgentImportPlanValidatesRequiredBases(t *testing.T) {
	archive, _, expectedDigest := makeIncrementalImportArchivePair(t)
	var out bytes.Buffer
	err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out)
	if err != nil {
		t.Fatalf("runAgentImportPlan should emit JSON issue for missing base: %v", err)
	}
	if !strings.Contains(out.String(), `"code": "missing_base_archive"`) || !strings.Contains(out.String(), expectedDigest) || !strings.Contains(out.String(), `"recommended_next_action": "provide-base-archives"`) {
		t.Fatalf("unexpected missing base plan: %s", out.String())
	}
}

func TestRunAgentImportPlanJSONIncludesProjectRelocation(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	archive := createAgentPlanArchiveWithManifest(t, &backup.Manifest{
		Version:     "1.0.0",
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
	})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"project_relocations"`) || !strings.Contains(out.String(), `"clone"`) || !strings.Contains(out.String(), `"git"`) {
		t.Fatalf("unexpected JSON: %s", out.String())
	}
}

func TestRunAgentImportPlanRemapsClaudeTargetPath(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	archive := createDiffArchiveWithManifestAndFiles(t, &backup.Manifest{
		Version:     "1.0.0",
		ArchiveKind: backup.ArchiveKindFull,
		Tools:       map[string]backup.ToolManifest{"claude-code": {Included: []string{adapters.CategoryMemory}, FileCount: 1}},
		Projects:    []backup.ProjectManifest{{ToolID: "claude-code", ProjectKey: "-Users-young-Softwares-dotvibe", RelativeToHome: "Softwares/dotvibe", PathScope: backup.ProjectPathScopeHome}},
	}, map[string]string{"claude-code/projects/-Users-young-Softwares-dotvibe/CLAUDE.md": "old memory\n"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	wantKey := adapters.ClaudeProjectKey(filepath.Join(home, "Softwares", "dotvibe"))
	if !strings.Contains(out.String(), wantKey) || strings.Contains(out.String(), filepath.Join(home, ".claude", "projects", "-Users-young-Softwares-dotvibe")) {
		t.Fatalf("unexpected remapped plan: %s", out.String())
	}
}

func TestRunAgentImportPlanFlagsAssociationReviewRequired(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	targetPath := filepath.Join(home, "Softwares", "dotvibe")
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatal(err)
	}
	runGitForExportTest(t, targetPath, "init")
	runGitForExportTest(t, targetPath, "remote", "add", "origin", "git@github.com:someone/fork.git")
	archive := createAgentPlanArchiveWithManifest(t, &backup.Manifest{
		Version:     "1.0.0",
		ArchiveKind: backup.ArchiveKindFull,
		Tools:       map[string]backup.ToolManifest{"claude-code": {Included: []string{adapters.CategoryMemory}, FileCount: 1}},
		Projects: []backup.ProjectManifest{{
			ToolID:         "claude-code",
			ProjectKey:     "-Users-young-Softwares-dotvibe",
			RelativeToHome: "Softwares/dotvibe",
			PathScope:      backup.ProjectPathScopeHome,
			Git: backup.ProjectGitMetadata{IsRepo: true, Remotes: []backup.ProjectGitRemote{{
				Name: "origin", URL: "git@github.com:yangyifan18/dotvibe.git", Sanitized: true, Cloneable: true,
			}}},
		}},
	})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"association_review_required": true`) || !strings.Contains(out.String(), projectmeta.ActionAssociationReviewRequired) {
		t.Fatalf("unexpected JSON: %s", out.String())
	}
}

func TestAgentProjectRelocationEndToEndPlanJSON(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	archive := createDiffArchiveWithManifestAndFiles(t, &backup.Manifest{
		Version:     "1.0.0",
		ArchiveKind: backup.ArchiveKindFull,
		SourceHome:  "/Users/young",
		SourceUser:  "young",
		Tools:       map[string]backup.ToolManifest{"claude-code": {Included: []string{adapters.CategoryMemory}, FileCount: 1}},
		Projects: []backup.ProjectManifest{{
			ToolID:         "claude-code",
			ProjectKey:     "-Users-young-Softwares-dotvibe",
			SourcePath:     "/Users/young/Softwares/dotvibe",
			SourceHome:     "/Users/young",
			RelativeToHome: "Softwares/dotvibe",
			PathScope:      backup.ProjectPathScopeHome,
			MemoryFiles:    []string{"claude-code/projects/-Users-young-Softwares-dotvibe/CLAUDE.md"},
			Git: backup.ProjectGitMetadata{IsRepo: true, Remotes: []backup.ProjectGitRemote{{
				Name: "origin", URL: "git@github.com:yangyifan18/dotvibe.git", Sanitized: true, Cloneable: true,
			}}},
		}},
	}, map[string]string{"claude-code/projects/-Users-young-Softwares-dotvibe/CLAUDE.md": "archive memory\n"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	var plan agentapi.ImportPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if plan.Destination.Home != home || len(plan.ProjectRelocations) != 1 {
		t.Fatalf("plan = %#v", plan)
	}
	if !plan.ProjectRelocations[0].Clone.Recommended || plan.ProjectRelocations[0].Clone.Command[0] != "git" {
		t.Fatalf("clone plan = %#v", plan.ProjectRelocations[0].Clone)
	}
}

func createAgentPlanArchiveWithManifest(t *testing.T, manifest *backup.Manifest) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if manifest.Tools == nil {
		manifest.Tools = map[string]backup.ToolManifest{}
	}
	if err := backup.CreateArchive(path, manifest, nil); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	return path
}

func createDiffArchiveWithManifestAndFiles(t *testing.T, manifest *backup.Manifest, files map[string]string) string {
	t.Helper()
	src := t.TempDir()
	var entries []adapters.FileEntry
	for name, content := range files {
		path := filepath.Join(src, strings.ReplaceAll(name, "/", "_"))
		writeFileForImportTest(t, path, content)
		entries = append(entries, adapters.FileEntry{SourcePath: path, InArchive: name, Category: adapters.CategoryMemory})
	}
	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if manifest.Tools == nil {
		manifest.Tools = map[string]backup.ToolManifest{"claude-code": {Included: []string{adapters.CategoryMemory}, FileCount: len(entries)}}
	}
	if err := backup.CreateArchive(archivePath, manifest, entries); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	return archivePath
}
