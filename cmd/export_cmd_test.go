package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestExportRefusesExistingOutputWithoutForce(t *testing.T) {
	oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase
	defer func() {
		exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase = oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase
	}()
	path := filepath.Join(t.TempDir(), "existing.tar.gz")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	exportOutput = path
	exportForce = false
	exportOnly = ""
	exportWithHist = false
	exportExcludes = nil
	exportBase = ""

	if err := exportCmd.RunE(exportCmd, nil); err == nil {
		t.Fatal("expected export to refuse existing output without --force")
	}
}

func TestExportAllowsExistingOutputWithForce(t *testing.T) {
	oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase
	defer func() {
		exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase = oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase
	}()
	path := filepath.Join(t.TempDir(), "existing.tar.gz")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	exportOutput = path
	exportForce = true
	exportOnly = ""
	exportWithHist = false
	exportExcludes = nil
	exportBase = ""

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("expected forced export to overwrite existing output: %v", err)
	}
}

func TestIsExportableFileSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if !isExportableFile(target) {
		t.Fatal("regular file should be exportable")
	}
	if isExportableFile(link) {
		t.Fatal("symlink should not be exportable")
	}
}

func TestExportBaseFlagRequiresReadableArchive(t *testing.T) {
	oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase
	defer func() {
		exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase = oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase
	}()
	exportOutput = filepath.Join(t.TempDir(), "delta.tar.gz")
	exportForce = true
	exportOnly = ""
	exportWithHist = false
	exportExcludes = nil
	exportBase = filepath.Join(t.TempDir(), "missing.tar.gz")

	if err := exportCmd.RunE(exportCmd, nil); err == nil {
		t.Fatal("expected missing base archive to fail")
	}
}

func TestExportBaseFlagRejectsOutputOverwriteOfBaseArchive(t *testing.T) {
	tests := []struct {
		name       string
		outputPath func(t *testing.T, baseArchive string) string
	}{
		{
			name: "exact same path",
			outputPath: func(t *testing.T, baseArchive string) string {
				return baseArchive
			},
		},
		{
			name: "output symlink to base",
			outputPath: func(t *testing.T, baseArchive string) string {
				link := filepath.Join(t.TempDir(), "output-link.tar.gz")
				if err := os.Symlink(baseArchive, link); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
				return link
			},
		},
		{
			name: "output hardlink to base",
			outputPath: func(t *testing.T, baseArchive string) string {
				link := filepath.Join(t.TempDir(), "output-hardlink.tar.gz")
				if err := os.Link(baseArchive, link); err != nil {
					t.Skipf("hardlinks unavailable: %v", err)
				}
				return link
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase
			defer func() {
				exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase = oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase
			}()
			baseArchive := createDiffArchive(t, map[string]string{"tool/config/base.txt": "same"})

			exportOutput = tt.outputPath(t, baseArchive)
			exportForce = true
			exportOnly = "missing-tool"
			exportWithHist = false
			exportExcludes = nil
			exportBase = baseArchive

			if err := exportCmd.RunE(exportCmd, nil); err == nil {
				t.Fatal("expected export to reject using the base archive as output")
			}
			baseReader, err := backup.ReadArchive(baseArchive)
			if err != nil {
				t.Fatalf("base archive should remain readable: %v", err)
			}
			defer baseReader.Close()
		})
	}
}

func TestCreateArchiveFromExportPlanWritesIncrementalMetadata(t *testing.T) {
	src := t.TempDir()
	baseFile := filepath.Join(src, "base.txt")
	changedFile := filepath.Join(src, "changed.txt")
	writeFileForImportTest(t, baseFile, "same")
	writeFileForImportTest(t, changedFile, "new")
	baseArchive := createDiffArchive(t, map[string]string{"tool/config/base.txt": "same"})

	baseReader, err := backup.ReadArchive(baseArchive)
	if err != nil {
		t.Fatalf("ReadArchive base: %v", err)
	}
	defer baseReader.Close()

	manifest := &backup.Manifest{Version: "1.0.0", Tools: map[string]backup.ToolManifest{"tool": {Included: []string{"config"}, FileCount: 2}}}
	entries := []adapters.FileEntry{
		{SourcePath: baseFile, InArchive: "tool/config/base.txt", Category: adapters.CategoryConfig},
		{SourcePath: changedFile, InArchive: "tool/config/changed.txt", Category: adapters.CategoryConfig},
	}
	baseDigest := baseReader.ManifestDigest()
	if baseDigest == "" {
		t.Fatal("expected base archive manifest digest")
	}
	plan, err := backup.BuildIncrementalArchivePlan(manifest, entries, baseReader.Manifest, backup.BaseArchiveRef{FileName: filepath.Base(baseArchive), Created: baseReader.Manifest.Created, ManifestSHA256: baseDigest})
	if err != nil {
		t.Fatalf("BuildIncrementalArchivePlan: %v", err)
	}
	out := filepath.Join(t.TempDir(), "delta.tar.gz")
	if err := backup.CreateArchiveWithStoredEntries(out, plan.Manifest, plan.StoredEntries); err != nil {
		t.Fatalf("CreateArchiveWithStoredEntries: %v", err)
	}
	delta, err := backup.ReadArchive(out)
	if err != nil {
		t.Fatalf("ReadArchive delta: %v", err)
	}
	defer delta.Close()
	if delta.Manifest.ArchiveKind != backup.ArchiveKindIncremental || delta.Manifest.Base == nil {
		t.Fatalf("missing incremental metadata: %#v", delta.Manifest)
	}
	if delta.Manifest.Base.ManifestSHA256 != baseDigest {
		t.Fatalf("base digest = %q, want %q", delta.Manifest.Base.ManifestSHA256, baseDigest)
	}
}

func TestExportIncludesClaudeProjectMetadata(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	projectPath := filepath.Join(home, "Softwares", "dotvibe")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatal(err)
	}
	runGitForExportTest(t, projectPath, "init")
	runGitForExportTest(t, projectPath, "remote", "add", "origin", "https://token@github.com/yangyifan18/dotvibe.git")
	projectKey := adapters.ClaudeProjectKey(projectPath)
	writeFileForImportTest(t, filepath.Join(home, ".claude", "projects", projectKey, "CLAUDE.md"), "# memory\n")

	oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase
	defer func() {
		exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes, exportBase = oldOutput, oldForce, oldOnly, oldHist, oldExcludes, oldBase
	}()
	archive := filepath.Join(t.TempDir(), "backup.tar.gz")
	exportOutput = archive
	exportForce = false
	exportOnly = "claude-code"
	exportWithHist = false
	exportExcludes = nil
	exportBase = ""

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("export: %v", err)
	}
	ar, err := backup.ReadArchive(archive)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	defer ar.Close()
	if ar.Manifest.SourceHome != home {
		t.Fatalf("source home = %q, want %q", ar.Manifest.SourceHome, home)
	}
	if len(ar.Manifest.Projects) != 1 {
		t.Fatalf("projects = %#v", ar.Manifest.Projects)
	}
	project := ar.Manifest.Projects[0]
	if project.ProjectKey != projectKey || project.RelativeToHome != "Softwares/dotvibe" {
		t.Fatalf("project = %#v", project)
	}
	if len(project.Git.Remotes) != 1 || project.Git.Remotes[0].URL != "https://github.com/yangyifan18/dotvibe.git" || !project.Git.Remotes[0].CredentialsRedacted {
		t.Fatalf("remote = %#v", project.Git.Remotes)
	}
}

func runGitForExportTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
