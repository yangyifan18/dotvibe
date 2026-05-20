package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestDiffArchivesFiltersByToolAndCategory(t *testing.T) {
	left := createDiffArchiveWithCategories(t, map[string]diffFixtureFile{
		"claude-code/memory/same.md":       {Content: "same", Category: adapters.CategoryMemory},
		"claude-code/memory/changed.md":    {Content: "old", Category: adapters.CategoryMemory},
		"claude-code/memory/removed.md":    {Content: "removed", Category: adapters.CategoryMemory},
		"claude-code/config/settings.json": {Content: "old-config", Category: adapters.CategoryConfig},
		"codex/memory/session.json":        {Content: "old-codex", Category: adapters.CategoryMemory},
	})
	right := createDiffArchiveWithCategories(t, map[string]diffFixtureFile{
		"claude-code/memory/same.md":       {Content: "same", Category: adapters.CategoryMemory},
		"claude-code/memory/changed.md":    {Content: "new", Category: adapters.CategoryMemory},
		"claude-code/memory/added.md":      {Content: "added", Category: adapters.CategoryMemory},
		"claude-code/config/settings.json": {Content: "new-config", Category: adapters.CategoryConfig},
		"codex/memory/session.json":        {Content: "new-codex", Category: adapters.CategoryMemory},
	})

	diff, err := diffArchivesWithOptions(left, right, diffOptions{OnlyTool: "claude-code", Category: adapters.CategoryMemory})
	if err != nil {
		t.Fatalf("diffArchivesWithOptions: %v", err)
	}
	assertStringSet(t, diff.Added, []string{"claude-code/memory/added.md"})
	assertStringSet(t, diff.Removed, []string{"claude-code/memory/removed.md"})
	assertStringSet(t, diff.Changed, []string{"claude-code/memory/changed.md"})
	assertStringSet(t, diff.Unchanged, []string{"claude-code/memory/same.md"})
}

func TestDiffPrintArchiveDiffJSON(t *testing.T) {
	diff := archiveDiff{
		Added:     []string{"tool/added.txt"},
		Removed:   nil,
		Changed:   []string{"tool/changed.txt"},
		Unchanged: []string{"tool/same.txt"},
	}

	output := captureStdout(t, func() {
		if err := printArchiveDiff(diff, true); err != nil {
			t.Fatalf("printArchiveDiff: %v", err)
		}
	})

	var got archiveDiff
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json output did not decode: %v\n%s", err, output)
	}
	assertStringSet(t, got.Added, diff.Added)
	assertStringSet(t, got.Removed, []string{})
	assertStringSet(t, got.Changed, diff.Changed)
	assertStringSet(t, got.Unchanged, diff.Unchanged)
	if !strings.Contains(output, "\"Added\"") || strings.Index(output, "\"Added\"") > strings.Index(output, "\"Removed\"") {
		t.Fatalf("json output is not stable archiveDiff field order: %s", output)
	}
	if strings.Contains(output, "\"Removed\": null") {
		t.Fatalf("json output should use arrays for empty diff lists: %s", output)
	}
}

func createDiffArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	fixtures := map[string]diffFixtureFile{}
	for name, content := range files {
		fixtures[name] = diffFixtureFile{Content: content, Category: adapters.CategoryConfig}
	}
	return createDiffArchiveWithCategories(t, fixtures)
}

type diffFixtureFile struct {
	Content  string
	Category string
}

func createDiffArchiveWithCategories(t *testing.T, files map[string]diffFixtureFile) string {
	t.Helper()
	src := t.TempDir()
	var entries []adapters.FileEntry
	for name, file := range files {
		path := filepath.Join(src, strings.ReplaceAll(name, "/", "_"))
		writeFileForImportTest(t, path, file.Content)
		entries = append(entries, adapters.FileEntry{SourcePath: path, InArchive: name, Category: file.Category})
	}
	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	manifest := &backup.Manifest{Version: "1.0.0", Tools: map[string]backup.ToolManifest{"tool": {Included: []string{"config"}, FileCount: len(entries)}}}
	if err := backup.CreateArchive(archivePath, manifest, entries); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	return archivePath
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
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
