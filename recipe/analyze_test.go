package recipe

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestAnalyzeArchiveReturnsRecipeInfo(t *testing.T) {
	archivePath := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"claude-code/skills/reviewer/SKILL.md": {content: "# Reviewer\n", category: adapters.CategorySkills},
		"codex-cli/agents/reviewer.md":         {content: "# Agent\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "Team Recipe", Author: "yangyifan", Description: "shared agents"})

	info, err := AnalyzeArchive(archivePath, AnalyzeOptions{IncludeRisks: false})
	if err != nil {
		t.Fatalf("AnalyzeArchive: %v", err)
	}
	if info.Name != "Team Recipe" || info.Author != "yangyifan" || info.Schema != backup.RecipeSchemaV1 {
		t.Fatalf("metadata mismatch: %#v", info)
	}
	if info.Digest == "" || info.TotalSize == 0 {
		t.Fatalf("missing digest/size: %#v", info)
	}
	if len(info.Tools) != 2 {
		t.Fatalf("tools = %#v, want claude-code and codex-cli", info.Tools)
	}
	if len(info.Files) != 2 {
		t.Fatalf("files = %#v, want 2", info.Files)
	}
}

func TestAnalyzeArchiveRejectsNonRecipe(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "not-recipe.tar.gz")
	if err := backup.CreateArchive(archivePath, &backup.Manifest{Version: "1.0.0", Tools: map[string]backup.ToolManifest{}}, nil); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	_, err := AnalyzeArchive(archivePath, AnalyzeOptions{})
	if err == nil {
		t.Fatal("expected non-recipe archive to fail")
	}
}

func TestAnalyzeArchiveIncludesEmptyRisksArrayWhenRequested(t *testing.T) {
	meta := validRecipeMetadata("Clean")
	meta.Author = "yangyifan"
	meta.Description = "clean recipe"
	archivePath := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/safe.md": {content: "safe\n", category: adapters.CategoryAgents},
	}, meta)

	info, err := AnalyzeArchive(archivePath, AnalyzeOptions{IncludeRisks: true, LintOptions: LintOptions{ScanContent: true}})
	if err != nil {
		t.Fatalf("AnalyzeArchive: %v", err)
	}
	if info.Risks == nil {
		t.Fatal("risks should be a non-nil empty slice when IncludeRisks=true")
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal RecipeInfo: %v", err)
	}
	if !strings.Contains(string(data), `"risks":[]`) {
		t.Fatalf("risks should marshal as [] when requested, got %s", string(data))
	}
}

type testRecipeFile struct {
	content  string
	category string
}

func buildTestRecipeArchive(t *testing.T, files map[string]testRecipeFile, opts ExportOptions) string {
	t.Helper()
	src := t.TempDir()
	entries := make([]adapters.FileEntry, 0, len(files))
	for archivePath, file := range files {
		sourcePath := filepath.Join(src, filepath.Base(archivePath))
		writeRecipeTestFile(t, sourcePath, file.content)
		entries = append(entries, adapters.FileEntry{SourcePath: sourcePath, InArchive: archivePath, Category: file.category})
	}
	out := filepath.Join(t.TempDir(), "recipe.vibe")
	if _, err := BuildArchive(out, entries, opts); err != nil {
		t.Fatalf("BuildArchive: %v", err)
	}
	return out
}

func validRecipeMetadata(name string) *backup.RecipeMetadata {
	return &backup.RecipeMetadata{Name: name, Schema: backup.RecipeSchemaV1, SharePolicy: "shareable-only"}
}

func buildRecipeArchiveWithMetadata(t *testing.T, files map[string]testRecipeFile, meta *backup.RecipeMetadata) string {
	t.Helper()
	src := t.TempDir()
	entries := make([]adapters.FileEntry, 0, len(files))
	for archivePath, file := range files {
		sourcePath := filepath.Join(src, filepath.Base(archivePath))
		writeRecipeTestFile(t, sourcePath, file.content)
		entries = append(entries, adapters.FileEntry{SourcePath: sourcePath, InArchive: archivePath, Category: file.category})
	}
	manifest := &backup.Manifest{
		Version: "1.0.0",
		Created: time.Now().UTC(),
		Recipe:  meta,
		Tools:   map[string]backup.ToolManifest{},
	}
	plan, err := backup.BuildRecipeArchivePlan(manifest, entries)
	if err != nil {
		t.Fatalf("BuildRecipeArchivePlan: %v", err)
	}
	out := filepath.Join(t.TempDir(), "recipe.vibe")
	if err := backup.CreateArchiveWithStoredEntries(out, plan.Manifest, plan.StoredEntries); err != nil {
		t.Fatalf("CreateArchiveWithStoredEntries: %v", err)
	}
	return out
}
