package recipe

import (
	"errors"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/yangyifan18/dotvibe/adapters"
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
	result := newLintResult()
	if err := validateRecipeManifest(ar.Manifest); err != nil {
		var manifestErr recipeManifestError
		if errors.As(err, &manifestErr) {
			result.add(SeverityError, manifestErr.code, "", manifestErr.message)
		} else {
			result.add(SeverityError, "invalid_manifest", "", err.Error())
		}
		return result, nil
	}
	if ar.Manifest.Recipe.Author == "" {
		result.add(SeverityInfo, "missing_author", "", "recipe has no author")
	}
	if ar.Manifest.Recipe.Description == "" {
		result.add(SeverityInfo, "missing_description", "", "recipe has no description")
	}
	for _, file := range ar.Manifest.Files {
		lintFileManifest(file, &result)
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

func newLintResult() LintResult {
	return LintResult{Findings: []LintFinding{}}
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

func lintFileManifest(file backup.FileManifest, result *LintResult) {
	lintCategory(file.Path, file.Category, result)
	lintPath(file.Path, result)
}

func lintCategory(filePath, category string, result *LintResult) {
	switch strings.ToLower(category) {
	case adapters.CategoryHistory, adapters.CategoryMemory:
		result.add(SeverityError, "sensitive_category", filePath, "memory or history category is not allowed in recipes")
	}
}

func lintPath(filePath string, result *LintResult) {
	lower := strings.ToLower(path.Clean(filePath))
	reported := map[string]bool{}
	addPathFinding := func(code, message string) {
		if reported[code] {
			return
		}
		reported[code] = true
		result.add(SeverityError, code, filePath, message)
	}
	blockedPrefixes := map[string]string{
		"claude-code/projects/":    "project_memory_path",
		"claude-code/transcripts/": "transcripts_path",
		"codex-cli/sessions/":      "sessions_path",
	}
	for prefix, code := range blockedPrefixes {
		if strings.HasPrefix(lower, prefix) {
			addPathFinding(code, "personal history or project memory path is not allowed in recipes")
		}
	}

	segmentCodes := map[string]string{
		"history":        "history_path",
		"history.jsonl":  "history_path",
		"project":        "project_path",
		"projects":       "project_path",
		"transcript":     "transcripts_path",
		"transcripts":    "transcripts_path",
		"session":        "sessions_path",
		"sessions":       "sessions_path",
		"cache":          "cache_path",
		".cache":         "cache_path",
		"cache.json":     "cache_path",
		"telemetry":      "telemetry_path",
		"telemetry.json": "telemetry_path",
	}
	for _, segment := range strings.Split(lower, "/") {
		if code, ok := segmentCodes[segment]; ok {
			addPathFinding(code, "sensitive path segment is not allowed in recipes")
		}
	}

	if isCredentialLikeFilename(path.Base(lower)) {
		result.add(SeverityError, "credential_filename", filePath, "credential-like filename is not allowed in recipes")
	}
}

func isCredentialLikeFilename(name string) bool {
	switch name {
	case "auth.json", "credentials.json", "token.json", "tokens.json", ".env", "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa":
		return true
	}
	return strings.HasSuffix(name, ".pem")
}

func lintContent(path string, data []byte, result *LintResult) {
	if len(data) > RecipeTextDiffMaxBytes {
		if looksBinary(data) {
			result.add(SeverityWarning, "binary_file", path, "binary file cannot be content-scanned")
			return
		}
		lintSecretRules(path, data[:RecipeTextDiffMaxBytes], result)
		result.add(SeverityWarning, "large_file", path, "large file was not fully content-scanned")
		return
	}
	kind := ClassifyText(data, len(data))
	if kind == TextKindBinary {
		result.add(SeverityWarning, "binary_file", path, "binary file cannot be content-scanned")
		return
	}
	if kind == TextKindLarge {
		result.add(SeverityWarning, "large_file", path, "large file was not fully content-scanned")
		return
	}
	lintSecretRules(path, data, result)
}

func lintSecretRules(path string, data []byte, result *LintResult) {
	content := string(data)
	for _, rule := range secretRules {
		if rule.re.MatchString(content) {
			result.add(rule.severity, rule.code, path, rule.message)
		}
	}
}

func looksBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return !utf8.Valid(data)
}
