package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/agentapi"
)

func TestRunAgentDoctorJSON(t *testing.T) {
	var out bytes.Buffer
	err := runAgentDoctor(agentDoctorOptions{JSON: true, Version: "test"}, &out)
	if err != nil {
		t.Fatalf("runAgentDoctor: %v", err)
	}
	var report agentapi.DoctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if report.DotvibeVersion != "test" || !report.Capabilities.AgentDoctor {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunAgentDoctorHuman(t *testing.T) {
	var out bytes.Buffer
	if err := runAgentDoctor(agentDoctorOptions{JSON: false, Version: "test"}, &out); err != nil {
		t.Fatalf("runAgentDoctor human: %v", err)
	}
	if !strings.Contains(out.String(), "dotvibe agent doctor") || !strings.Contains(out.String(), "Capabilities") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRunAgentInventoryJSON(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	writeFileForImportTest(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "# Reviewer\n")
	var out bytes.Buffer
	if err := runAgentInventory(agentInventoryOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentInventory: %v", err)
	}
	if !strings.Contains(out.String(), `"recommended_profiles"`) || !strings.Contains(out.String(), `"codex-cli"`) {
		t.Fatalf("unexpected inventory JSON: %s", out.String())
	}
}

func TestRunAgentExportPlanJSON(t *testing.T) {
	var out bytes.Buffer
	if err := runAgentExportPlan(agentExportPlanOptions{JSON: true, Profile: "recipe", Output: "team.vibe", Name: "Team", Author: "yyf"}, &out); err != nil {
		t.Fatalf("runAgentExportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"profile": "recipe"`) || !strings.Contains(out.String(), `"recipe"`) {
		t.Fatalf("unexpected export plan: %s", out.String())
	}
}

func TestRunAgentImportPlanJSON(t *testing.T) {
	home := t.TempDir()
	oldHome := testSetHome(t, home)
	defer oldHome()
	writeFileForImportTest(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "local\n")
	archive := createAgentPlanArchive(t, map[string]string{"codex-cli/agents/reviewer.md": "archive\n"})
	var out bytes.Buffer
	if err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out); err != nil {
		t.Fatalf("runAgentImportPlan: %v", err)
	}
	if !strings.Contains(out.String(), `"conflicts": 1`) || !strings.Contains(out.String(), `"recommended_next_action": "stage-or-choose-conflict-policy"`) {
		t.Fatalf("unexpected import plan: %s", out.String())
	}
}

func createAgentPlanArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	return createDiffArchive(t, files)
}

func TestRunAgentImportPlanValidatesRequiredBases(t *testing.T) {
	archive, _, expectedDigest := makeIncrementalImportArchivePair(t)
	var out bytes.Buffer
	err := runAgentImportPlan(archive, agentImportPlanOptions{JSON: true}, &out)
	if err == nil {
		t.Fatal("expected missing base archive to fail")
	}
	if !strings.Contains(err.Error(), expectedDigest) {
		t.Fatalf("error = %q, want base digest %s", err.Error(), expectedDigest)
	}
}
