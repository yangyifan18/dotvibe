package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	oldDryRun, oldYes, oldOnly, oldProject, oldForce, oldBases := importDryRun, importYes, importOnly, importProject, importForce, importBases
	defer func() {
		importDryRun, importYes, importOnly, importProject, importForce = oldDryRun, oldYes, oldOnly, oldProject, oldForce
		importBases = oldBases
	}()
	importDryRun = true
	importYes = false
	importOnly = ""
	importProject = ""
	importForce = false
	importBases = nil

	if err := importCmd.RunE(importCmd, []string{archivePath}); err != nil {
		t.Fatalf("dry-run import failed: %v", err)
	}
}

func TestImportMissingBaseFailsBeforeRestoreWrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	archivePath, _, expectedDigest := makeIncrementalImportArchivePair(t)
	oldDryRun, oldYes, oldOnly, oldProject, oldForce, oldBases := importDryRun, importYes, importOnly, importProject, importForce, importBases
	defer func() {
		importDryRun, importYes, importOnly, importProject, importForce = oldDryRun, oldYes, oldOnly, oldProject, oldForce
		importBases = oldBases
	}()
	importDryRun = false
	importYes = true
	importOnly = ""
	importProject = ""
	importForce = false
	importBases = nil

	err := importCmd.RunE(importCmd, []string{archivePath})
	if err == nil {
		t.Fatal("expected missing base import to fail")
	}
	if !strings.Contains(err.Error(), expectedDigest) {
		t.Fatalf("error = %q, want expected base fingerprint %s", err.Error(), expectedDigest)
	}
	target := filepath.Join(home, ".claude", "projects", "-Users-young-App", "memory", "MEMORY.md")
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("restore wrote target before validating base chain: %v", statErr)
	}
}

func TestImportWithMatchingBaseChainRestoresBaseBackedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	archivePath, basePath, _ := makeIncrementalImportArchivePair(t)
	oldDryRun, oldYes, oldOnly, oldProject, oldForce, oldBases := importDryRun, importYes, importOnly, importProject, importForce, importBases
	defer func() {
		importDryRun, importYes, importOnly, importProject, importForce = oldDryRun, oldYes, oldOnly, oldProject, oldForce
		importBases = oldBases
	}()
	importDryRun = false
	importYes = true
	importOnly = "claude-code"
	importProject = "-Users-young-App"
	importForce = false
	importBases = []string{basePath}

	if err := importCmd.RunE(importCmd, []string{archivePath}); err != nil {
		t.Fatalf("import with base chain failed: %v", err)
	}

	target := filepath.Join(home, ".claude", "projects", "-Users-young-App", "memory", "MEMORY.md")
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(got) != "# base memory" {
		t.Fatalf("restored content = %q, want base content", string(got))
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

func makeBaseBackedImportArchive(t *testing.T) string {
	t.Helper()
	created := time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC)
	manifest := backup.Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   backup.ArchiveKindIncremental,
		Created:       created,
		Hostname:      "new-mac",
		Base: &backup.BaseArchiveRef{
			FileName:       "base.tar.gz",
			Created:        created.Add(-24 * time.Hour),
			ManifestSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Tools: map[string]backup.ToolManifest{
			"claude-code": {Included: []string{"memory"}, FileCount: 1},
		},
		Files: []backup.FileManifest{
			{
				Path:     "claude-code/projects/-Users-young-App/memory/MEMORY.md",
				ToolID:   "claude-code",
				Size:     8,
				SHA256:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Category: adapters.CategoryMemory,
				Storage:  backup.FileStorageBase,
			},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "incremental.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0644, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func makeIncrementalImportArchivePair(t *testing.T) (headPath, basePath, baseDigest string) {
	t.Helper()
	dir := t.TempDir()
	basePayload := filepath.Join(dir, "MEMORY.md")
	writeFileForImportTest(t, basePayload, "# base memory")
	basePath = filepath.Join(dir, "base.tar.gz")
	baseManifest := &backup.Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   backup.ArchiveKindFull,
		Tools: map[string]backup.ToolManifest{
			"claude-code": {Included: []string{"memory"}, FileCount: 1},
		},
	}
	logicalPath := "claude-code/projects/-Users-young-App/memory/MEMORY.md"
	if err := backup.CreateArchive(basePath, baseManifest, []adapters.FileEntry{{
		SourcePath: basePayload,
		InArchive:  logicalPath,
		Category:   adapters.CategoryMemory,
	}}); err != nil {
		t.Fatalf("CreateArchive base: %v", err)
	}
	baseReader, err := backup.ReadArchive(basePath)
	if err != nil {
		t.Fatalf("ReadArchive base: %v", err)
	}
	baseDigest = baseReader.ManifestDigest()
	if err := baseReader.Close(); err != nil {
		t.Fatalf("Close base: %v", err)
	}

	created := time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC)
	headManifest := backup.Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   backup.ArchiveKindIncremental,
		Created:       created,
		Hostname:      "new-mac",
		Base: &backup.BaseArchiveRef{
			FileName:       "base.tar.gz",
			Created:        created.Add(-24 * time.Hour),
			ManifestSHA256: baseDigest,
		},
		Tools: map[string]backup.ToolManifest{
			"claude-code": {Included: []string{"memory"}, FileCount: 1},
		},
		Files: []backup.FileManifest{{
			Path:     logicalPath,
			ToolID:   "claude-code",
			Size:     int64(len("# base memory")),
			SHA256:   mustSHA256ForImportTest(t, basePayload),
			Category: adapters.CategoryMemory,
			Storage:  backup.FileStorageBase,
		}},
	}
	data, err := json.Marshal(headManifest)
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}
	headPath = filepath.Join(dir, "delta.tar.gz")
	writeRawArchiveForImportTest(t, headPath, map[string]string{"manifest.json": string(data)})
	return headPath, basePath, baseDigest
}

func mustSHA256ForImportTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func writeRawArchiveForImportTest(t *testing.T, archivePath string, files map[string]string) {
	t.Helper()
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
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
