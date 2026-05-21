package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

func TestDiffArchivesFiltersLegacyFilesByInferredCategory(t *testing.T) {
	left := createLegacyDiffArchiveNoFiles(t, map[string]string{
		"claude-code/memory/same.md":       "same",
		"claude-code/memory/changed.md":    "old",
		"claude-code/memory/removed.md":    "removed",
		"claude-code/config/settings.json": "old-config",
	})
	right := createLegacyDiffArchiveNoFiles(t, map[string]string{
		"claude-code/memory/same.md":       "same",
		"claude-code/memory/changed.md":    "new",
		"claude-code/memory/added.md":      "added",
		"claude-code/config/settings.json": "new-config",
	})

	diff, err := diffArchivesWithOptions(left, right, diffOptions{Category: adapters.CategoryMemory})
	if err != nil {
		t.Fatalf("diffArchivesWithOptions: %v", err)
	}
	assertStringSet(t, diff.Added, []string{"claude-code/memory/added.md"})
	assertStringSet(t, diff.Removed, []string{"claude-code/memory/removed.md"})
	assertStringSet(t, diff.Changed, []string{"claude-code/memory/changed.md"})
	assertStringSet(t, diff.Unchanged, []string{"claude-code/memory/same.md"})
}

func TestDiffArchivesInfersAdapterSpecificLegacyCategories(t *testing.T) {
	tests := []struct {
		name     string
		category string
		path     string
	}{
		{
			name:     "opencode xdg config",
			category: adapters.CategoryConfig,
			path:     "opencode/xdg-config/opencode.json",
		},
		{
			name:     "opencode home config",
			category: adapters.CategoryConfig,
			path:     "opencode/home-config/.opencode.json",
		},
		{
			name:     "codex agents",
			category: adapters.CategorySkills,
			path:     "codex-cli/agents/reviewer.md",
		},
		{
			name:     "claude project memory",
			category: adapters.CategoryMemory,
			path:     "claude-code/projects/example/CLAUDE.md",
		},
		{
			name:     "claude history",
			category: adapters.CategoryHistory,
			path:     "claude-code/history.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := createLegacyDiffArchiveNoFiles(t, map[string]string{
				"other-tool/misc/settings.json": "ignore",
			})
			right := createLegacyDiffArchiveNoFiles(t, map[string]string{
				"other-tool/misc/settings.json": "ignore-new",
				tt.path:                         "added",
			})

			diff, err := diffArchivesWithOptions(left, right, diffOptions{Category: tt.category})
			if err != nil {
				t.Fatalf("diffArchivesWithOptions: %v", err)
			}
			assertStringSet(t, diff.Added, []string{tt.path})
			assertStringSet(t, diff.Changed, []string{})
		})
	}
}

func TestDiffArchivesComparesLegacyFallbackFileChecksums(t *testing.T) {
	left := createLegacyDiffArchiveNoFiles(t, map[string]string{
		"tool/config/settings.json": "old",
	})
	right := createLegacyDiffArchiveNoFiles(t, map[string]string{
		"tool/config/settings.json": "new",
	})

	diff, err := diffArchivesWithOptions(left, right, diffOptions{})
	if err != nil {
		t.Fatalf("diffArchivesWithOptions: %v", err)
	}
	assertStringSet(t, diff.Changed, []string{"tool/config/settings.json"})
	assertStringSet(t, diff.Unchanged, []string{})
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

	var got struct {
		Added     []string `json:"added"`
		Removed   []string `json:"removed"`
		Changed   []string `json:"changed"`
		Unchanged []string `json:"unchanged"`
	}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json output did not decode: %v\n%s", err, output)
	}
	assertStringSet(t, got.Added, diff.Added)
	assertStringSet(t, got.Removed, []string{})
	assertStringSet(t, got.Changed, diff.Changed)
	assertStringSet(t, got.Unchanged, diff.Unchanged)
	if strings.Contains(output, "\"Added\"") || strings.Contains(output, "\"Removed\"") {
		t.Fatalf("json output should use lowercase field names: %s", output)
	}
	if !strings.Contains(output, "\"added\"") || strings.Index(output, "\"added\"") > strings.Index(output, "\"removed\"") {
		t.Fatalf("json output is not stable archiveDiff field order: %s", output)
	}
	if strings.Contains(output, "\"removed\": null") {
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

func createLegacyDiffArchiveNoFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "legacy.tar.gz")
	out, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create legacy archive: %v", err)
	}
	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)

	manifestData, err := json.Marshal(&backup.Manifest{
		Version: "1.0.0",
		Tools:   map[string]backup.ToolManifest{"tool": {Included: []string{"config", "memory"}, FileCount: len(files)}},
	})
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}
	writeTarBytesForDiffTest(t, tw, "manifest.json", manifestData)
	for name, content := range files {
		writeTarBytesForDiffTest(t, tw, name, []byte(content))
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("Close gzip: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("Close archive: %v", err)
	}
	return archivePath
}

func writeTarBytesForDiffTest(t *testing.T, tw *tar.Writer, name string, data []byte) {
	t.Helper()
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}); err != nil {
		t.Fatalf("WriteHeader %s: %v", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("Write %s: %v", name, err)
	}
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
