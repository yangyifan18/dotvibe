package recipe

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestLintArchiveFlagsSecretPatterns(t *testing.T) {
	archivePath := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/leaky.md":  {content: "OPENAI_API_KEY=sk-proj-abcdefghijklmnopqrstuvwxyz123456\n", category: adapters.CategoryAgents},
		"claude-code/skills/key.pem": {content: "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----\n", category: adapters.CategorySkills},
	}, ExportOptions{Name: "Leaky"})

	result, err := LintArchive(archivePath, LintOptions{ScanContent: true})
	if err != nil {
		t.Fatalf("LintArchive: %v", err)
	}
	assertFinding(t, result.Findings, SeverityError, "openai_key")
	assertFinding(t, result.Findings, SeverityError, "pem_private_key")
	if !result.HasErrors() {
		t.Fatalf("expected HasErrors for %#v", result.Findings)
	}
}

func TestLintArchiveAWSAccessKeyIsWarningButSecretIsError(t *testing.T) {
	archivePath := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/aws.md": {content: "id=AKIA1234567890ABCDEF\naws_secret_access_key = verysecret\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "AWS"})
	result, err := LintArchive(archivePath, LintOptions{ScanContent: true})
	if err != nil {
		t.Fatalf("LintArchive: %v", err)
	}
	assertFinding(t, result.Findings, SeverityWarning, "aws_access_key")
	assertFinding(t, result.Findings, SeverityError, "aws_secret_key")
}

func TestLintArchiveNoContentSkipsSecretContent(t *testing.T) {
	archivePath := buildTestRecipeArchive(t, map[string]testRecipeFile{
		"codex-cli/agents/leaky.md": {content: "sk-proj-abcdefghijklmnopqrstuvwxyz123456\n", category: adapters.CategoryAgents},
	}, ExportOptions{Name: "No Content"})
	result, err := LintArchive(archivePath, LintOptions{ScanContent: false})
	if err != nil {
		t.Fatalf("LintArchive: %v", err)
	}
	if hasFinding(result.Findings, "openai_key") {
		t.Fatalf("content finding should be absent when ScanContent=false: %#v", result.Findings)
	}
}

func TestLintArchiveFlagsUnsafeManifestPaths(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "unsafe.vibe")
	if err := createRawRecipeArchiveForLintTest(archivePath, "../secret.txt", "secret"); err != nil {
		t.Fatalf("create raw recipe: %v", err)
	}
	_, err := LintArchive(archivePath, LintOptions{ScanContent: true})
	if err == nil {
		t.Fatal("expected archive reader to reject unsafe path before lint succeeds")
	}
}

func createRawRecipeArchiveForLintTest(archivePath, logicalPath, content string) error {
	manifest := &backup.Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   backup.ArchiveKindRecipe,
		Created:       time.Now().UTC(),
		Recipe:        &backup.RecipeMetadata{Name: "Unsafe", Schema: backup.RecipeSchemaV1, SharePolicy: "shareable-only"},
		Tools:         map[string]backup.ToolManifest{"codex-cli": {Included: []string{adapters.CategoryAgents}, FileCount: 1}},
		Files: []backup.FileManifest{{
			Path:       logicalPath,
			ToolID:     "codex-cli",
			Category:   adapters.CategoryAgents,
			Size:       int64(len(content)),
			SHA256:     strings.Repeat("a", 64),
			Storage:    backup.FileStorageInline,
			StoredPath: logicalPath,
		}},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0644, Size: int64(len(data))}); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: logicalPath, Mode: 0644, Size: int64(len(content))}); err != nil {
		return err
	}
	_, err = tw.Write([]byte(content))
	return err
}

func assertFinding(t *testing.T, findings []LintFinding, severity string, code string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Severity == severity && finding.Code == code {
			return
		}
	}
	t.Fatalf("finding %s/%s not found in %#v", severity, code, findings)
}

func hasFinding(findings []LintFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
