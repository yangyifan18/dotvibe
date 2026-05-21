package recipe

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yangyifan18/dotvibe/backup"
)

const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

type LintResult struct {
	Findings []LintFinding `json:"findings"`
}

func (r LintResult) HasErrors() bool {
	for _, finding := range r.Findings {
		if finding.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r LintResult) HasWarnings() bool {
	for _, finding := range r.Findings {
		if finding.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

func (r LintResult) ExitCode(strict bool) int {
	if r.HasErrors() || (strict && r.HasWarnings()) {
		return 1
	}
	return 0
}

func LintArchive(path string, opts LintOptions) (LintResult, error) {
	ar, err := backup.ReadArchive(path)
	if err != nil {
		return LintResult{}, err
	}
	defer ar.Close()
	var result LintResult
	if ar.Manifest.ArchiveKind != backup.ArchiveKindRecipe || ar.Manifest.Recipe == nil {
		result.add(SeverityError, "not_recipe", "", "archive is not a dotvibe recipe")
		return result, nil
	}
	if ar.Manifest.Recipe.Schema != backup.RecipeSchemaV1 {
		result.add(SeverityError, "schema_mismatch", "", fmt.Sprintf("unsupported recipe schema %q", ar.Manifest.Recipe.Schema))
	}
	if ar.Manifest.Recipe.Author == "" {
		result.add(SeverityInfo, "missing_author", "", "recipe has no author")
	}
	if ar.Manifest.Recipe.Description == "" {
		result.add(SeverityInfo, "missing_description", "", "recipe has no description")
	}
	for _, file := range ar.Manifest.Files {
		lintPath(file.Path, &result)
		if opts.ScanContent {
			data, readErr := ar.ReadFile(file.Path)
			if readErr != nil {
				result.add(SeverityError, "read_failed", file.Path, readErr.Error())
				continue
			}
			lintContent(file.Path, data, &result)
		}
	}
	return result, nil
}

func (r *LintResult) add(severity, code, path, message string) {
	r.Findings = append(r.Findings, LintFinding{Severity: severity, Code: code, Path: path, Message: message})
}

var secretRules = []struct {
	code     string
	severity string
	re       *regexp.Regexp
	message  string
}{
	{"pem_private_key", SeverityError, regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`), "private key marker found"},
	{"openai_key", SeverityError, regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{20,}|sk-proj-[A-Za-z0-9_-]+|sk-svcacct-[A-Za-z0-9_-]+)\b`), "OpenAI-compatible API key found"},
	{"anthropic_key", SeverityError, regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}\b`), "Anthropic API key found"},
	{"github_token", SeverityError, regexp.MustCompile(`\b(ghp_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]+)\b`), "GitHub token found"},
	{"aws_access_key", SeverityWarning, regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`), "AWS access key ID found"},
	{"aws_secret_key", SeverityError, regexp.MustCompile(`(?i)aws_secret_access_key\s*=\s*[^\s]+`), "AWS secret access key found"},
	{"email", SeverityWarning, regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`), "email address found"},
	{"local_user_path", SeverityWarning, regexp.MustCompile(`/Users/[A-Za-z0-9._\-]+/`), "local user path found"},
}

func lintPath(path string, result *LintResult) {
	lower := strings.ToLower(path)
	blockedPrefixes := map[string]string{
		"claude-code/projects/":    "project_memory_path",
		"claude-code/transcripts/": "transcripts_path",
		"codex-cli/sessions/":      "sessions_path",
	}
	for prefix, code := range blockedPrefixes {
		if strings.HasPrefix(lower, prefix) {
			result.add(SeverityError, code, path, "personal history or project memory path is not allowed in recipes")
		}
	}
	blockedFragments := map[string]string{
		"/cache/":     "cache_path",
		"/telemetry/": "telemetry_path",
	}
	for fragment, code := range blockedFragments {
		if strings.Contains(lower, fragment) {
			result.add(SeverityError, code, path, "cache or telemetry path is not allowed in recipes")
		}
	}
	blockedNames := []string{"auth.json", "credentials.json", "token.json", ".env", "id_rsa"}
	for _, name := range blockedNames {
		if strings.HasSuffix(lower, "/"+name) || lower == name {
			result.add(SeverityError, "credential_filename", path, "credential-like filename is not allowed in recipes")
		}
	}
}

func lintContent(path string, data []byte, result *LintResult) {
	kind := ClassifyText(data, len(data))
	if kind == TextKindBinary {
		result.add(SeverityWarning, "binary_file", path, "binary file cannot be content-scanned")
		return
	}
	if kind == TextKindLarge {
		result.add(SeverityWarning, "large_file", path, "large file was not fully content-scanned")
		return
	}
	content := string(data)
	for _, rule := range secretRules {
		if rule.re.MatchString(content) {
			result.add(rule.severity, rule.code, path, rule.message)
		}
	}
}
