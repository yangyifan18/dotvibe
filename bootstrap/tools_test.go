package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectToolsFindsBinariesOnPath(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "claude"))
	t.Setenv("PATH", dir)

	results := DetectTools(DefaultToolSpecs())
	claude := resultByID(results, "claude-code")
	if !claude.Installed {
		t.Fatalf("claude-code should be installed: %#v", claude)
	}
	if claude.FoundBinary == "" {
		t.Fatalf("claude-code should report found binary: %#v", claude)
	}
	codex := resultByID(results, "codex-cli")
	if codex.Installed {
		t.Fatalf("codex-cli should be missing in isolated PATH: %#v", codex)
	}
}

func TestMissingToolIncludesInstallCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	results := DetectTools(DefaultToolSpecs())
	codex := resultByID(results, "codex-cli")
	if codex.Installed {
		t.Fatal("codex should be missing")
	}
	if len(codex.InstallCommands) == 0 || codex.InstallCommands[0].Command == "" {
		t.Fatalf("missing install command: %#v", codex)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func resultByID(results []ToolCheckResult, id string) ToolCheckResult {
	for _, result := range results {
		if result.ID == id {
			return result
		}
	}
	return ToolCheckResult{}
}
