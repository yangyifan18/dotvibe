package bootstrap

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultToolSpecsIncludeStructuredInstallCommands(t *testing.T) {
	specs := DefaultToolSpecs()
	expectedOrder := []string{"claude-code", "codex-cli", "opencode"}
	if len(specs) != len(expectedOrder) {
		t.Fatalf("expected %d tool specs, got %d: %#v", len(expectedOrder), len(specs), specs)
	}
	for i, expectedID := range expectedOrder {
		if specs[i].ID != expectedID {
			t.Fatalf("spec %d ID = %q, want %q", i, specs[i].ID, expectedID)
		}
	}

	tests := []struct {
		id         string
		manager    string
		command    string
		executable string
		args       []string
	}{
		{
			id:         "claude-code",
			manager:    "npm",
			command:    "npm install -g @anthropic-ai/claude-code",
			executable: "npm",
			args:       []string{"install", "-g", "@anthropic-ai/claude-code"},
		},
		{
			id:         "codex-cli",
			manager:    "npm",
			command:    "npm i -g @openai/codex",
			executable: "npm",
			args:       []string{"i", "-g", "@openai/codex"},
		},
		{
			id:         "opencode",
			manager:    "brew",
			command:    "brew install anomalyco/tap/opencode",
			executable: "brew",
			args:       []string{"install", "anomalyco/tap/opencode"},
		},
		{
			id:         "opencode",
			manager:    "npm",
			command:    "npm i -g opencode-ai",
			executable: "npm",
			args:       []string{"i", "-g", "opencode-ai"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"/"+tt.manager, func(t *testing.T) {
			cmd := installCommandByManager(t, resultSpecByID(t, specs, tt.id), tt.manager)
			if cmd.Command != tt.command {
				t.Fatalf("display command = %q, want %q", cmd.Command, tt.command)
			}
			if !cmd.SafeRun {
				t.Fatalf("%s command should be safe to run: %#v", tt.manager, cmd)
			}
			if cmd.ManualOnly || cmd.UsesShell {
				t.Fatalf("%s command should be structured for direct exec: %#v", tt.manager, cmd)
			}
			if cmd.Executable != tt.executable {
				t.Fatalf("executable = %q, want %q", cmd.Executable, tt.executable)
			}
			if !reflect.DeepEqual(cmd.Args, tt.args) {
				t.Fatalf("args = %#v, want %#v", cmd.Args, tt.args)
			}
		})
	}
}

func TestDefaultToolSpecsKeepCurlInstallerManualOnly(t *testing.T) {
	opencode := resultSpecByID(t, DefaultToolSpecs(), "opencode")
	curl := installCommandByManager(t, opencode, "curl")
	if curl.Command != "curl -fsSL https://opencode.ai/install | bash" {
		t.Fatalf("curl display command = %q", curl.Command)
	}
	if curl.SafeRun {
		t.Fatalf("curl pipe installer must not be marked safe: %#v", curl)
	}
	if !curl.ManualOnly || !curl.UsesShell {
		t.Fatalf("curl pipe installer should be manual shell command: %#v", curl)
	}
	if curl.Executable != "" || len(curl.Args) != 0 {
		t.Fatalf("curl pipe installer should not expose structured auto command: %#v", curl)
	}
}

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

func resultSpecByID(t *testing.T, specs []ToolSpec, id string) ToolSpec {
	t.Helper()
	for _, spec := range specs {
		if spec.ID == id {
			return spec
		}
	}
	t.Fatalf("missing tool spec %q", id)
	return ToolSpec{}
}

func installCommandByManager(t *testing.T, spec ToolSpec, manager string) InstallCommand {
	t.Helper()
	for _, cmd := range spec.InstallCommands {
		if cmd.Manager == manager {
			return cmd
		}
	}
	t.Fatalf("missing %q install command in %#v", manager, spec)
	return InstallCommand{}
}
