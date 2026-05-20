package cmd

import (
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestDiffArchivesReportsAddedRemovedChangedUnchanged(t *testing.T) {
	left := createDiffArchive(t, map[string]string{
		"tool/same.txt":    "same",
		"tool/changed.txt": "old",
		"tool/removed.txt": "removed",
	})
	right := createDiffArchive(t, map[string]string{
		"tool/same.txt":    "same",
		"tool/changed.txt": "new",
		"tool/added.txt":   "added",
	})

	diff, err := diffArchives(left, right)
	if err != nil {
		t.Fatalf("diffArchives: %v", err)
	}
	assertStringSet(t, diff.Added, []string{"tool/added.txt"})
	assertStringSet(t, diff.Removed, []string{"tool/removed.txt"})
	assertStringSet(t, diff.Changed, []string{"tool/changed.txt"})
	assertStringSet(t, diff.Unchanged, []string{"tool/same.txt"})
}

func createDiffArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	src := t.TempDir()
	var entries []adapters.FileEntry
	for name, content := range files {
		path := filepath.Join(src, filepath.Base(name))
		writeFileForImportTest(t, path, content)
		entries = append(entries, adapters.FileEntry{SourcePath: path, InArchive: name, Category: adapters.CategoryConfig})
	}
	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	manifest := &backup.Manifest{Version: "1.0.0", Tools: map[string]backup.ToolManifest{"tool": {Included: []string{"config"}, FileCount: len(entries)}}}
	if err := backup.CreateArchive(archivePath, manifest, entries); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	return archivePath
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}
