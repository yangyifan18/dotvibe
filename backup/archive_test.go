package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestCreateArchive(t *testing.T) {
	// Create test files
	src := t.TempDir()
	writeFile(t, src+"/config/settings.json", `{"theme":"dark"}`)
	writeFile(t, src+"/memory/project/MEMORY.md", "# Memory\n")
	writeFile(t, src+"/skills/custom/skill.md", "# Skill\n")

	entries := []adapters.FileEntry{
		{SourcePath: src + "/config/settings.json", InArchive: "test-tool/config/settings.json", Category: adapters.CategoryConfig},
		{SourcePath: src + "/memory/project/MEMORY.md", InArchive: "test-tool/memory/project/MEMORY.md", Category: adapters.CategoryMemory},
		{SourcePath: src + "/skills/custom/skill.md", InArchive: "test-tool/skills/custom/skill.md", Category: adapters.CategorySkills},
	}

	manifest := &Manifest{
		Version:  "1.0.0",
		Hostname: "test",
		Tools: map[string]ToolManifest{
			"test-tool": {Included: []string{"config", "memory", "skills"}, FileCount: 3},
		},
	}

	dst := t.TempDir() + "/test.tar.gz"
	err := CreateArchive(dst, manifest, entries)
	if err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}

	// Verify the archive can be read
	f, err := os.Open(dst)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	files := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		files[hdr.Name] = true
	}

	expected := []string{
		"manifest.json",
		"test-tool/config/settings.json",
		"test-tool/memory/project/MEMORY.md",
		"test-tool/skills/custom/skill.md",
	}
	for _, name := range expected {
		if !files[name] {
			t.Errorf("archive missing file: %s", name)
		}
	}
}

func TestCreateArchive_Empty(t *testing.T) {
	manifest := &Manifest{
		Version: "1.0.0",
		Tools:   map[string]ToolManifest{},
	}

	dst := t.TempDir() + "/empty.tar.gz"
	err := CreateArchive(dst, manifest, nil)
	if err != nil {
		t.Fatalf("CreateArchive with no entries: %v", err)
	}

	// Should still be a valid gzip with at least manifest.json
	f, _ := os.Open(dst)
	defer f.Close()
	gz, _ := gzip.NewReader(f)
	defer gz.Close()

	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("expected at least manifest.json, got error: %v", err)
	}
	if hdr.Name != "manifest.json" {
		t.Errorf("first file = %q, want manifest.json", hdr.Name)
	}
}

func TestReadArchive(t *testing.T) {
	// Create a test archive first
	src := t.TempDir()
	writeFile(t, src+"/config/settings.json", `{"theme":"dark"}`)
	writeFile(t, src+"/memory/MEMORY.md", "# Memory\n")

	entries := []adapters.FileEntry{
		{SourcePath: src + "/config/settings.json", InArchive: "claude-code/config/settings.json", Category: adapters.CategoryConfig},
		{SourcePath: src + "/memory/MEMORY.md", InArchive: "claude-code/memory/MEMORY.md", Category: adapters.CategoryMemory},
	}

	manifest := &Manifest{
		Version:  "1.0.0",
		Hostname: "old-mac",
		Tools: map[string]ToolManifest{
			"claude-code": {Included: []string{"config", "memory"}, FileCount: 2},
		},
	}

	archivePath := t.TempDir() + "/test.tar.gz"
	if err := CreateArchive(archivePath, manifest, entries); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}

	// Now read it back
	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	if ra.Manifest.Hostname != "old-mac" {
		t.Errorf("Hostname = %q, want %q", ra.Manifest.Hostname, "old-mac")
	}

	files := ra.ListFiles()
	if len(files) != 2 {
		t.Errorf("ListFiles count = %d, want 2", len(files))
	}

	// Extract a file
	data, err := ra.ReadFile("claude-code/config/settings.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != `{"theme":"dark"}` {
		t.Errorf("ReadFile content = %q, want %q", string(data), `{"theme":"dark"}`)
	}
}

func TestReadArchive_NonExistent(t *testing.T) {
	_, err := ReadArchive("/nonexistent/archive.tar.gz")
	if err == nil {
		t.Error("expected error for missing archive")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestExtractArchiveRejectsPathTraversal(t *testing.T) {
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json": `{"version":"1.0.0","tools":{}}`,
		"../evil.txt":   "owned",
	})
	dest := t.TempDir()

	err := ExtractArchive(archivePath, dest)
	if err == nil {
		t.Fatal("expected path traversal archive to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(dest, "..", "evil.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("path traversal wrote outside destination: %v", statErr)
	}
}

func TestReadArchiveRejectsUnsafeNames(t *testing.T) {
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json": `{"version":"1.0.0","tools":{}}`,
		"/tmp/evil":     "owned",
	})

	_, err := ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected unsafe absolute path to be rejected")
	}
}

func TestReadArchiveRequiresManifest(t *testing.T) {
	archivePath := createRawArchive(t, map[string]string{
		"tool/config.json": "{}",
	})

	_, err := ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected archive without manifest to fail")
	}
}

func createRawArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	entries := make([]rawArchiveEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, rawArchiveEntry{Name: name, Content: content})
	}
	return createRawArchiveEntries(t, entries)
}

type rawArchiveEntry struct {
	Name    string
	Content string
}

func createRawArchiveEntries(t *testing.T, entries []rawArchiveEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "raw.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for _, entry := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: entry.Name, Mode: 0644, Size: int64(len(entry.Content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(entry.Content)); err != nil {
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
	return path
}

func TestCreateArchiveWritesFileManifestWithChecksums(t *testing.T) {
	src := t.TempDir()
	filePath := filepath.Join(src, "config.json")
	writeFile(t, filePath, `{"ok":true}`)
	manifest := &Manifest{Version: "1.0.0", Tools: map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 1}}}
	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")

	if err := CreateArchive(archivePath, manifest, []adapters.FileEntry{{SourcePath: filePath, InArchive: "tool/config.json", Category: adapters.CategoryConfig}}); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()
	if len(ra.Manifest.Files) != 1 {
		t.Fatalf("manifest files = %d, want 1", len(ra.Manifest.Files))
	}
	fm := ra.Manifest.Files[0]
	if fm.Path != "tool/config.json" || fm.Category != adapters.CategoryConfig || fm.SHA256 == "" || fm.Size == 0 {
		t.Fatalf("unexpected file manifest: %#v", fm)
	}
}

func TestReadArchiveRejectsChecksumMismatch(t *testing.T) {
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json":    `{"version":"1.0.0","tools":{},"files":[{"path":"tool/config.json","size":2,"sha256":"0000000000000000000000000000000000000000000000000000000000000000","category":"config"}]}`,
		"tool/config.json": "{}",
	})
	_, err := ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected checksum mismatch to fail")
	}
}

func TestReadArchiveRejectsDuplicateArchivePath(t *testing.T) {
	content := "{}"
	sum := sha256.Sum256([]byte(content))
	archivePath := createRawArchiveEntries(t, []rawArchiveEntry{
		{Name: "manifest.json", Content: fmt.Sprintf(`{"version":"1.0.0","tools":{},"files":[{"path":"tool/config.json","size":2,"sha256":"%x","category":"config"}]}`, sum)},
		{Name: "tool/config.json", Content: content},
		{Name: "tool/config.json", Content: content},
	})

	_, err := ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected duplicate archive path to fail")
	}
}

func TestReadArchiveUsesStoredPathForInlineManifestFile(t *testing.T) {
	content := "{}"
	sum := sha256.Sum256([]byte(content))
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools: map[string]ToolManifest{
			"tool": {Included: []string{"config"}, FileCount: 1},
		},
		Files: []FileManifest{
			{
				Path:       "tool/config.json",
				ToolID:     "tool",
				Size:       int64(len(content)),
				SHA256:     fmt.Sprintf("%x", sum),
				Category:   "config",
				Storage:    FileStorageInline,
				StoredPath: "objects/sha256/payload",
			},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json":          string(data),
		"objects/sha256/payload": content,
	})

	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	got, err := ra.ReadFile("tool/config.json")
	if err != nil {
		t.Fatalf("ReadFile logical path: %v", err)
	}
	if string(got) != content {
		t.Fatalf("ReadFile logical content = %q, want %q", string(got), content)
	}
}

func TestCreateArchiveRejectsUnsafeArchivePath(t *testing.T) {
	src := t.TempDir()
	filePath := filepath.Join(src, "evil.txt")
	writeFile(t, filePath, "owned")
	manifest := &Manifest{Version: "1.0.0", Tools: map[string]ToolManifest{}}

	err := CreateArchive(filepath.Join(t.TempDir(), "backup.tar.gz"), manifest, []adapters.FileEntry{
		{SourcePath: filePath, InArchive: "../evil.txt", Category: adapters.CategoryConfig},
	})
	if err == nil {
		t.Fatal("expected unsafe archive path to fail")
	}
}

func TestReadArchiveNormalizesLegacyEmbeddedManifest(t *testing.T) {
	content := "{}"
	sum := sha256.Sum256([]byte(content))
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json":    fmt.Sprintf(`{"version":"1.0.0","tools":{},"files":[{"path":"tool/config.json","size":2,"sha256":"%x","category":"config"}]}`, sum),
		"tool/config.json": content,
	})

	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	if ra.Manifest.FormatVersion != 1 || ra.Manifest.ArchiveKind != ArchiveKindFull {
		t.Fatalf("legacy defaults = (%d,%q), want (1,%q)", ra.Manifest.FormatVersion, ra.Manifest.ArchiveKind, ArchiveKindFull)
	}
	got := ra.Manifest.Files[0]
	if got.ToolID != "tool" || got.Storage != FileStorageInline || got.StoredPath != got.Path {
		t.Fatalf("legacy file metadata = %#v", got)
	}
}

func TestReadArchiveAllowsBaseStoredManifestFileWithoutPayload(t *testing.T) {
	created := time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC)
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindIncremental,
		Created:       created,
		Hostname:      "new-mac",
		Base: &BaseArchiveRef{
			FileName:       "dotvibe-2026-05-19.tar.gz",
			Created:        created.Add(-24 * time.Hour),
			ManifestSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Tools: map[string]ToolManifest{
			"tool": {Included: []string{"config"}, FileCount: 1},
		},
		Files: []FileManifest{
			{
				Path:     "tool/config.json",
				ToolID:   "tool",
				Size:     2,
				SHA256:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Category: "config",
				Storage:  FileStorageBase,
			},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json": string(data),
	})

	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	if ra.Manifest.Files[0].Storage != FileStorageBase {
		t.Fatalf("file storage = %q, want %q", ra.Manifest.Files[0].Storage, FileStorageBase)
	}
}
