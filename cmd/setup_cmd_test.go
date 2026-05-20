package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/bootstrap"
)

func TestFormatSetupPlanShowsMissingInstallCommands(t *testing.T) {
	results := []bootstrap.ToolCheckResult{
		{ID: "claude-code", Name: "Claude Code", Installed: true, FoundBinary: "/usr/local/bin/claude"},
		{
			ID:        "codex-cli",
			Name:      "Codex CLI",
			Installed: false,
			InstallCommands: []bootstrap.InstallCommand{{
				Manager:    "npm",
				Command:    "npm i -g @openai/codex",
				SafeRun:    true,
				Executable: "npm",
				Args:       []string{"i", "-g", "@openai/codex"},
			}},
		},
		{
			ID:        "opencode",
			Name:      "OpenCode",
			Installed: false,
			InstallCommands: []bootstrap.InstallCommand{{
				Manager:    "curl",
				Command:    "curl -fsSL https://opencode.ai/install | bash",
				SafeRun:    false,
				ManualOnly: true,
				UsesShell:  true,
			}},
		},
	}
	var buf bytes.Buffer
	printSetupPlan(&buf, results, "backup.tar.gz")
	out := buf.String()
	if !strings.Contains(out, "Claude Code: installed") {
		t.Fatalf("installed tool missing from output: %s", out)
	}
	if !strings.Contains(out, "npm i -g @openai/codex") {
		t.Fatalf("install command missing from output: %s", out)
	}
	if !strings.Contains(out, "manual-review") {
		t.Fatalf("manual-review marker missing from output: %s", out)
	}
	if !strings.Contains(out, "Restore after setup: backup.tar.gz") {
		t.Fatalf("restore archive missing from output: %s", out)
	}
}

func TestBuildInstallCommandPlanSkipsUnsafeByDefault(t *testing.T) {
	results := []bootstrap.ToolCheckResult{
		{
			ID:        "claude-code",
			Name:      "Claude Code",
			Installed: true,
			InstallCommands: []bootstrap.InstallCommand{{
				Manager:    "npm",
				Command:    "npm install -g @anthropic-ai/claude-code",
				SafeRun:    true,
				Executable: "npm",
				Args:       []string{"install", "-g", "@anthropic-ai/claude-code"},
			}},
		},
		{
			ID:        "opencode",
			Name:      "OpenCode",
			Installed: false,
			InstallCommands: []bootstrap.InstallCommand{
				{Manager: "curl", Command: "curl -fsSL https://opencode.ai/install | bash", SafeRun: true, UsesShell: true},
				{Manager: "npm", Command: "npm i -g opencode-ai", SafeRun: true},
				{Manager: "brew", Command: "brew install anomalyco/tap/opencode", SafeRun: true, Executable: "brew", Args: []string{"install", "anomalyco/tap/opencode"}},
			},
		},
	}
	commands := buildInstallCommandPlan(results)
	if len(commands) != 1 || commands[0].Manager != "brew" {
		t.Fatalf("expected safe structured brew command, got %#v", commands)
	}
}
