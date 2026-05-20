package backup

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

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
	path := filepath.Join(t.TempDir(), "raw.tar.gz")
	f, err := os.Create(path)
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
	return path
}
