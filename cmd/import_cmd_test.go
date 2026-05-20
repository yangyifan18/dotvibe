package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestFilterImportEntriesByProjectKeepsNonClaudeTools(t *testing.T) {
	toolFiles := map[string][]adapters.FileEntry{
		"claude-code": {
			{InArchive: "claude-code/projects/-Users-young-App/memory/MEMORY.md"},
			{InArchive: "claude-code/projects/-Users-young-Other/memory/MEMORY.md"},
			{InArchive: "claude-code/config/settings.json"},
		},
		"codex-cli": {{InArchive: "codex-cli/config/config.toml"}},
		"opencode":  {{InArchive: "opencode/xdg-config/opencode.json"}},
	}

	filtered := filterImportEntriesByProject(toolFiles, "-Users-young-App")
	if len(filtered) != 3 {
		t.Fatalf("filtered tools = %d, want 3", len(filtered))
	}
	entries := filtered["claude-code"]
	if len(entries) != 1 {
		t.Fatalf("claude entries = %d, want 1", len(entries))
	}
	if entries[0].InArchive != "claude-code/projects/-Users-young-App/memory/MEMORY.md" {
		t.Fatalf("entry = %q", entries[0].InArchive)
	}
	if len(filtered["codex-cli"]) != 1 || len(filtered["opencode"]) != 1 {
		t.Fatalf("non-Claude tools were not preserved: %#v", filtered)
	}
}

func TestImportDryRunReturnsBeforeConfirmation(t *testing.T) {
	archivePath := makeImportTestArchive(t)
	oldDryRun, oldYes, oldOnly, oldProject, oldForce := importDryRun, importYes, importOnly, importProject, importForce
	defer func() {
		importDryRun, importYes, importOnly, importProject, importForce = oldDryRun, oldYes, oldOnly, oldProject, oldForce
	}()
	importDryRun = true
	importYes = false
	importOnly = ""
	importProject = ""
	importForce = false

	if err := importCmd.RunE(importCmd, []string{archivePath}); err != nil {
		t.Fatalf("dry-run import failed: %v", err)
	}
}

func makeImportTestArchive(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	filePath := filepath.Join(src, "MEMORY.md")
	writeFileForImportTest(t, filePath, "# memory")
	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	manifest := &backup.Manifest{
		Version: "1.0.0",
		Tools: map[string]backup.ToolManifest{
			"claude-code": {Included: []string{"memory"}, FileCount: 1},
		},
	}
	entries := []adapters.FileEntry{{
		SourcePath: filePath,
		InArchive:  "claude-code/projects/-Users-young-App/memory/MEMORY.md",
		Category:   adapters.CategoryMemory,
	}}
	if err := backup.CreateArchive(archivePath, manifest, entries); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	return archivePath
}

func writeFileForImportTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFilterImportEntriesByProjectUsesAdapterFilters(t *testing.T) {
	toolFiles := map[string][]adapters.FileEntry{
		"claude-code": {
			{InArchive: "claude-code/projects/-Users-young-App/memory/MEMORY.md"},
			{InArchive: "claude-code/projects/-Users-young-Other/memory/MEMORY.md"},
		},
		"codex-cli": {{InArchive: "codex-cli/config/config.toml"}},
	}
	filtered := filterImportEntriesByProject(toolFiles, "-Users-young-App")
	if len(filtered["claude-code"]) != 1 {
		t.Fatalf("claude filter count = %d, want 1", len(filtered["claude-code"]))
	}
	if len(filtered["codex-cli"]) != 1 {
		t.Fatalf("codex entries should be preserved")
	}
}
