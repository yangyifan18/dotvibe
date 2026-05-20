package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

func TestSetupConfirmationEOFDeniesActions(t *testing.T) {
	var out bytes.Buffer
	ok := confirmSetupActions(strings.NewReader(""), &out, []bootstrap.InstallCommand{{
		Manager: "npm",
		Command: "npm i -g @openai/codex",
	}}, "backup.tar.gz")
	if ok {
		t.Fatal("EOF confirmation should deny setup actions")
	}
	if !strings.Contains(out.String(), "Restore backup: backup.tar.gz") {
		t.Fatalf("combined restore action missing from prompt: %s", out.String())
	}
}

func TestSetupConfirmationReadErrorDeniesActions(t *testing.T) {
	var out bytes.Buffer
	ok := confirmSetupActions(errReader{}, &out, nil, "backup.tar.gz")
	if ok {
		t.Fatal("read error confirmation should deny setup actions")
	}
}

func TestSetupInstallArchiveRequiresConfirmationBeforeRestoreWhenNoCommandsSelected(t *testing.T) {
	restoreCalled := false
	withSetupTestHooks(t, setupTestHooks{
		detect: func() []bootstrap.ToolCheckResult {
			return []bootstrap.ToolCheckResult{{ID: "codex-cli", Name: "Codex CLI", Installed: true, FoundBinary: "/bin/codex"}}
		},
		restore: func(string) error {
			restoreCalled = true
			return errors.New("restore should not run")
		},
	})
	setupInstall = true
	setupYes = false
	setupOnly = ""
	setupBases = nil

	cmd, _ := setupTestCommand()
	cmd.SetIn(strings.NewReader(""))
	if err := runSetup(cmd, []string{"backup.tar.gz"}); err != nil {
		t.Fatalf("setup should cancel without trying restore: %v", err)
	}
	if restoreCalled {
		t.Fatal("restore ran without explicit confirmation")
	}
}

func TestSetupYesSkipsConfirmation(t *testing.T) {
	restoreCalled := false
	withSetupTestHooks(t, setupTestHooks{
		detect: func() []bootstrap.ToolCheckResult {
			return []bootstrap.ToolCheckResult{{ID: "codex-cli", Name: "Codex CLI", Installed: true, FoundBinary: "/bin/codex"}}
		},
		restore: func(archive string) error {
			restoreCalled = archive == "backup.tar.gz"
			return nil
		},
	})
	setupInstall = true
	setupYes = true
	setupOnly = ""
	setupBases = nil

	cmd, _ := setupTestCommand()
	cmd.SetIn(strings.NewReader(""))
	if err := runSetup(cmd, []string{"backup.tar.gz"}); err != nil {
		t.Fatalf("setup --yes failed: %v", err)
	}
	if !restoreCalled {
		t.Fatal("setup --yes should run restore without prompting")
	}
}

type setupTestHooks struct {
	detect  func() []bootstrap.ToolCheckResult
	restore func(string) error
}

func withSetupTestHooks(t *testing.T, hooks setupTestHooks) {
	t.Helper()
	oldDetect, oldRestore := detectSetupTools, setupRestore
	oldInstall, oldYes, oldOnly, oldBases := setupInstall, setupYes, setupOnly, setupBases
	if hooks.detect != nil {
		detectSetupTools = hooks.detect
	}
	if hooks.restore != nil {
		setupRestore = hooks.restore
	}
	t.Cleanup(func() {
		detectSetupTools = oldDetect
		setupRestore = oldRestore
		setupInstall = oldInstall
		setupYes = oldYes
		setupOnly = oldOnly
		setupBases = oldBases
	})
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func setupTestCommand() (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	return cmd, &out
}
