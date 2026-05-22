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

func TestLintArchiveScansLargeTextPrefixForSecrets(t *testing.T) {
	meta := validRecipeMetadata("Large Leaky")
	meta.Author = "yangyifan"
	meta.Description = "large file"
	archivePath := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/large.md": {
			content:  "sk-proj-" + strings.Repeat("a", 64) + "\n" + strings.Repeat("x", RecipeTextDiffMaxBytes+1),
			category: adapters.CategoryAgents,
		},
	}, meta)

	result, err := LintArchive(archivePath, LintOptions{ScanContent: true})
	if err != nil {
		t.Fatalf("LintArchive: %v", err)
	}
	assertFinding(t, result.Findings, SeverityError, "openai_key")
	assertFinding(t, result.Findings, SeverityWarning, "large_file")
}

func TestLintArchiveFlagsSensitiveCategoriesPathsAndCredentialNames(t *testing.T) {
	meta := validRecipeMetadata("Sensitive")
	meta.Author = "yangyifan"
	meta.Description = "sensitive samples"
	archivePath := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/shared.md":     {content: "memory\n", category: adapters.CategoryMemory},
		"opencode/history/session.jsonl": {content: "history\n", category: adapters.CategoryAgents},
		"custom-tool/project/notes.md":   {content: "project\n", category: adapters.CategoryAgents},
		"codex-cli/config/tokens.json":   {content: "{}\n", category: adapters.CategoryConfig},
		"codex-cli/config/id_ed25519":    {content: "not-secret\n", category: adapters.CategoryConfig},
		"claude-code/config/client.pem":  {content: "not-secret\n", category: adapters.CategoryConfig},
		"claude-code/history.jsonl":      {content: "history\n", category: adapters.CategoryConfig},
		"codex-cli/.cache/index.json":    {content: "{}\n", category: adapters.CategoryConfig},
		"opencode/telemetry.json":        {content: "{}\n", category: adapters.CategoryConfig},
	}, meta)

	result, err := LintArchive(archivePath, LintOptions{ScanContent: false})
	if err != nil {
		t.Fatalf("LintArchive: %v", err)
	}
	assertFinding(t, result.Findings, SeverityError, "sensitive_category")
	assertFinding(t, result.Findings, SeverityError, "history_path")
	assertFinding(t, result.Findings, SeverityError, "project_path")
	assertFinding(t, result.Findings, SeverityError, "cache_path")
	assertFinding(t, result.Findings, SeverityError, "telemetry_path")
	assertFinding(t, result.Findings, SeverityError, "credential_filename")
}

func TestLintArchiveJSONUsesEmptyFindingsArray(t *testing.T) {
	meta := validRecipeMetadata("Clean")
	meta.Author = "yangyifan"
	meta.Description = "clean recipe"
	archivePath := buildRecipeArchiveWithMetadata(t, map[string]testRecipeFile{
		"codex-cli/agents/safe.md": {content: "safe\n", category: adapters.CategoryAgents},
	}, meta)

	result, err := LintArchive(archivePath, LintOptions{ScanContent: true})
	if err != nil {
		t.Fatalf("LintArchive: %v", err)
	}
	if len(result.Findings) != 0 || result.Findings == nil {
		t.Fatalf("findings = %#v, want non-nil empty slice", result.Findings)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal LintResult: %v", err)
	}
	if !strings.Contains(string(data), `"findings":[]`) {
		t.Fatalf("findings should marshal as [], got %s", string(data))
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
