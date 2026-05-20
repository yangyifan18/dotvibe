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
	"strings"
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
		objectPathForSHA256(mustFileSHA256ForTest(t, src+"/config/settings.json")),
		objectPathForSHA256(mustFileSHA256ForTest(t, src+"/memory/project/MEMORY.md")),
		objectPathForSHA256(mustFileSHA256ForTest(t, src+"/skills/custom/skill.md")),
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

func TestReadArchiveRejectsEmbeddedPathTraversal(t *testing.T) {
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json": `{"version":"1.0.0","tools":{}}`,
		"claude-code/config/../../.ssh/authorized_keys": "owned",
	})

	_, err := ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected embedded path traversal to be rejected")
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

func TestReadArchiveRejectsDuplicateLogicalManifestPath(t *testing.T) {
	content := "{}"
	sum := sha256.Sum256([]byte(content))
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 2}},
		Files: []FileManifest{
			{
				Path:       "tool/config.json",
				ToolID:     "tool",
				Size:       int64(len(content)),
				SHA256:     fmt.Sprintf("%x", sum),
				Category:   "config",
				Storage:    FileStorageInline,
				StoredPath: "objects/sha256/first",
			},
			{
				Path:       "tool/config.json",
				ToolID:     "tool",
				Size:       int64(len(content)),
				SHA256:     fmt.Sprintf("%x", sum),
				Category:   "config",
				Storage:    FileStorageInline,
				StoredPath: "objects/sha256/second",
			},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}
	archivePath := createRawArchive(t, map[string]string{
		"manifest.json":         string(data),
		"objects/sha256/first":  content,
		"objects/sha256/second": content,
	})

	_, err = ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected duplicate logical manifest path to fail")
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

func TestReadArchiveRejectsUnsupportedFileStorage(t *testing.T) {
	content := "{}"
	sum := sha256.Sum256([]byte(content))
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 1}},
		Files: []FileManifest{
			{
				Path:       "tool/config.json",
				ToolID:     "tool",
				Size:       int64(len(content)),
				SHA256:     fmt.Sprintf("%x", sum),
				Category:   "config",
				Storage:    "remote",
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

	_, err = ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected unsupported file storage to fail")
	}
}

func TestReadArchiveRejectsExtraPayloadWhenManifestListsFiles(t *testing.T) {
	content := "{}"
	sum := sha256.Sum256([]byte(content))
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 1}},
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
		"manifest.json":            string(data),
		"objects/sha256/payload":   content,
		"objects/sha256/unchecked": "extra",
	})

	_, err = ReadArchive(archivePath)
	if err == nil {
		t.Fatal("expected extra unchecked payload to fail")
	}
}

func TestReadArchiveListFilesUsesLogicalManifestPaths(t *testing.T) {
	archivePath := createStoredPathArchive(t, "{}")
	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	files := ra.ListFiles()
	if len(files) != 1 || files[0] != "tool/config.json" {
		t.Fatalf("ListFiles = %#v, want logical path", files)
	}
}

func TestReadArchiveListFilesSortsLogicalManifestPaths(t *testing.T) {
	src := t.TempDir()
	first := filepath.Join(src, "first.txt")
	second := filepath.Join(src, "second.txt")
	writeFile(t, first, "first")
	writeFile(t, second, "second")
	manifest := &Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 2}},
		Files: []FileManifest{
			{Path: "tool/z.txt", ToolID: "tool", Size: 5, SHA256: mustFileSHA256ForTest(t, first), Category: "config", Storage: FileStorageInline, StoredPath: "objects/first"},
			{Path: "tool/a.txt", ToolID: "tool", Size: 6, SHA256: mustFileSHA256ForTest(t, second), Category: "config", Storage: FileStorageInline, StoredPath: "objects/second"},
		},
	}
	archivePath := filepath.Join(t.TempDir(), "stored.tar.gz")
	if err := CreateArchiveWithStoredEntries(archivePath, manifest, []StoredEntry{
		{SourcePath: first, StoredPath: "objects/first"},
		{SourcePath: second, StoredPath: "objects/second"},
	}); err != nil {
		t.Fatalf("CreateArchiveWithStoredEntries: %v", err)
	}

	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	assertStringSetBackup(t, ra.ListFiles(), []string{"tool/a.txt", "tool/z.txt"})
	if got := ra.ListFiles(); got[0] != "tool/a.txt" || got[1] != "tool/z.txt" {
		t.Fatalf("ListFiles order = %#v, want sorted logical paths", got)
	}
}

func TestCreateArchiveWithStoredEntriesRejectsInvalidManifestBeforeWriting(t *testing.T) {
	src := t.TempDir()
	payload := filepath.Join(src, "payload.txt")
	writeFile(t, payload, "payload")
	dst := filepath.Join(t.TempDir(), "stored.tar.gz")

	tests := []struct {
		name     string
		manifest *Manifest
		entries  []StoredEntry
	}{
		{
			name: "duplicate logical paths",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageInline, StoredPath: "objects/payload"},
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageInline, StoredPath: "objects/payload2"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
		{
			name: "unsupported storage value",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: "remote", StoredPath: "objects/payload"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
		{
			name: "missing inline payload",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageInline, StoredPath: "objects/missing"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
		{
			name: "unsafe manifest stored path",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageInline, StoredPath: "../payload"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
		{
			name: "unsafe entry stored path",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageInline, StoredPath: "objects/payload"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "../payload"}},
		},
		{
			name: "size mismatch",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 99, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageInline, StoredPath: "objects/payload"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
		{
			name: "checksum mismatch",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: strings.Repeat("0", 64), Storage: FileStorageInline, StoredPath: "objects/payload"},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
		{
			name: "extra stored entry",
			manifest: &Manifest{
				Version: "1.0.0",
				Tools:   map[string]ToolManifest{},
				Files: []FileManifest{
					{Path: "tool/config.json", Size: 7, SHA256: mustFileSHA256ForTest(t, payload), Storage: FileStorageBase},
				},
			},
			entries: []StoredEntry{{SourcePath: payload, StoredPath: "objects/payload"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateArchiveWithStoredEntries(dst, tt.manifest, tt.entries); err == nil {
				t.Fatal("expected invalid manifest/entries to fail")
			}
			if _, err := os.Stat(dst); !os.IsNotExist(err) {
				t.Fatalf("archive should not be created before validation succeeds: %v", err)
			}
		})
	}
}

func TestReadArchiveRejectsUnknownLogicalPathWhenManifestListsFiles(t *testing.T) {
	archivePath := createStoredPathArchive(t, "{}")
	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	if _, err := ra.ReadFile("objects/sha256/payload"); err == nil {
		t.Fatal("expected physical payload lookup to fail when manifest lists logical files")
	}
	if _, err := ra.ReadFile("tool/missing.json"); err == nil {
		t.Fatal("expected unknown logical path to fail")
	}
}

func TestReadArchiveRejectsBaseStoredReadFile(t *testing.T) {
	archivePath := createBaseStoredArchive(t)
	ra, err := ReadArchive(archivePath)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ra.Close()

	_, err = ra.ReadFile("tool/config.json")
	if err == nil {
		t.Fatal("expected base-backed logical read to fail")
	}
}

func TestExtractArchiveMaterializesStoredPathAsLogicalPath(t *testing.T) {
	content := "{}"
	archivePath := createStoredPathArchive(t, content)
	dest := t.TempDir()

	if err := ExtractArchive(archivePath, dest); err != nil {
		t.Fatalf("ExtractArchive: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "tool/config.json"))
	if err != nil {
		t.Fatalf("read logical extracted file: %v", err)
	}
	if string(got) != content {
		t.Fatalf("extracted logical content = %q, want %q", string(got), content)
	}
	if _, err := os.Stat(filepath.Join(dest, "objects/sha256/payload")); !os.IsNotExist(err) {
		t.Fatalf("stored payload path should not be materialized directly: %v", err)
	}
}

func TestExtractArchiveFailsOnBaseStoredManifestFile(t *testing.T) {
	err := ExtractArchive(createBaseStoredArchive(t), t.TempDir())
	if err == nil {
		t.Fatal("expected base-backed extract to fail")
	}
}

func createStoredPathArchive(t *testing.T, content string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(content))
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 1}},
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
	return createRawArchive(t, map[string]string{
		"manifest.json":          string(data),
		"objects/sha256/payload": content,
	})
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

func createBaseStoredArchive(t *testing.T) string {
	t.Helper()
	manifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindIncremental,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 1}},
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
	return createRawArchive(t, map[string]string{"manifest.json": string(data)})
}

func mustFileSHA256ForTest(t *testing.T, path string) string {
	t.Helper()
	sum, err := sourceFileSHA256(path)
	if err != nil {
		t.Fatalf("sourceFileSHA256(%s): %v", path, err)
	}
	return sum
}

func assertStringSetBackup(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got %#v want %#v", len(got), len(want), got, want)
	}
	counts := map[string]int{}
	for _, item := range got {
		counts[item]++
	}
	for _, item := range want {
		counts[item]--
	}
	for item, count := range counts {
		if count != 0 {
			t.Fatalf("set mismatch for %q: got %#v want %#v", item, got, want)
		}
	}
}

func TestCreateArchiveRejectsDuplicateArchivePath(t *testing.T) {
	src := t.TempDir()
	first := filepath.Join(src, "first.txt")
	second := filepath.Join(src, "second.txt")
	writeFile(t, first, "one")
	writeFile(t, second, "two")
	manifest := &Manifest{Version: "1.0.0", Tools: map[string]ToolManifest{}}

	err := CreateArchive(filepath.Join(t.TempDir(), "backup.tar.gz"), manifest, []adapters.FileEntry{
		{SourcePath: first, InArchive: "tool/config.json", Category: adapters.CategoryConfig},
		{SourcePath: second, InArchive: "tool/config.json", Category: adapters.CategoryConfig},
	})
	if err == nil {
		t.Fatal("expected duplicate archive path to fail")
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

func TestOpenArchiveSetReadsBaseStoredFileFromMatchingBase(t *testing.T) {
	dir := t.TempDir()
	basePayload := filepath.Join(dir, "base-memory.md")
	writeFile(t, basePayload, "# base memory\n")
	basePath := filepath.Join(dir, "base.tar.gz")
	baseManifest := &Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"memory"}, FileCount: 1}},
	}
	if err := CreateArchive(basePath, baseManifest, []adapters.FileEntry{{
		SourcePath: basePayload,
		InArchive:  "tool/memory.md",
		Category:   "memory",
	}}); err != nil {
		t.Fatalf("CreateArchive base: %v", err)
	}
	baseReader, err := ReadArchive(basePath)
	if err != nil {
		t.Fatalf("ReadArchive base: %v", err)
	}
	baseDigest := baseReader.ManifestDigest()
	if err := baseReader.Close(); err != nil {
		t.Fatalf("Close base: %v", err)
	}

	headPath := filepath.Join(dir, "delta.tar.gz")
	headManifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindIncremental,
		Base:          &BaseArchiveRef{FileName: "base.tar.gz", ManifestSHA256: baseDigest},
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"memory"}, FileCount: 1}},
		Files: []FileManifest{{
			Path:     "tool/memory.md",
			ToolID:   "tool",
			Size:     int64(len("# base memory\n")),
			SHA256:   mustFileSHA256ForTest(t, basePayload),
			Category: "memory",
			Storage:  FileStorageBase,
		}},
	}
	writeManifestOnlyArchive(t, headPath, headManifest)

	set, err := OpenArchiveSet(headPath, []string{basePath})
	if err != nil {
		t.Fatalf("OpenArchiveSet: %v", err)
	}
	defer set.Close()

	got, err := set.ReadFile("tool/memory.md")
	if err != nil {
		t.Fatalf("ReadFile from base: %v", err)
	}
	if string(got) != "# base memory\n" {
		t.Fatalf("content = %q, want base content", string(got))
	}
}

func TestOpenArchiveSetFailsWhenRequiredBaseMissing(t *testing.T) {
	headPath := filepath.Join(t.TempDir(), "delta.tar.gz")
	wantDigest := strings.Repeat("a", 64)
	headManifest := Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindIncremental,
		Base:          &BaseArchiveRef{FileName: "base.tar.gz", ManifestSHA256: wantDigest},
		Tools:         map[string]ToolManifest{"tool": {Included: []string{"memory"}, FileCount: 1}},
		Files: []FileManifest{{
			Path:     "tool/memory.md",
			ToolID:   "tool",
			Size:     13,
			SHA256:   strings.Repeat("b", 64),
			Category: "memory",
			Storage:  FileStorageBase,
		}},
	}
	writeManifestOnlyArchive(t, headPath, headManifest)

	_, err := OpenArchiveSet(headPath, nil)
	if err == nil {
		t.Fatal("expected missing base to fail")
	}
	if !strings.Contains(err.Error(), wantDigest) {
		t.Fatalf("error = %q, want expected base fingerprint %s", err.Error(), wantDigest)
	}
}

func writeManifestOnlyArchive(t *testing.T, archivePath string, manifest Manifest) {
	t.Helper()
	manifest.Normalize()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal manifest: %v", err)
	}
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
}
