package agentapi

import (
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/bootstrap"
)

type fakeAdapter struct {
	id       string
	name     string
	detected bool
	status   adapters.ToolStatus
}

func (a fakeAdapter) Name() string { return a.name }
func (a fakeAdapter) ID() string   { return a.id }
func (a fakeAdapter) Detect() bool { return a.detected }
func (a fakeAdapter) ListFiles(opts adapters.ExportOpts) []adapters.FileEntry {
	return nil
}
func (a fakeAdapter) ListRecipeFiles(opts adapters.RecipeOpts) []adapters.FileEntry {
	return nil
}
func (a fakeAdapter) Status() adapters.ToolStatus { return a.status }
func (a fakeAdapter) FilterRestoreEntries(entries []adapters.FileEntry, opts adapters.RestoreOpts) []adapters.FileEntry {
	return entries
}
func (a fakeAdapter) PlanRestore(entries []adapters.FileEntry, opts adapters.RestoreOpts) ([]adapters.RestorePlanEntry, error) {
	return nil, nil
}
func (a fakeAdapter) RestoreFiles(entries []adapters.FileEntry, archiveDir string, opts adapters.RestoreOpts) (adapters.RestoreSummary, error) {
	return adapters.RestoreSummary{}, nil
}

func TestBuildDoctorReportIncludesCapabilitiesAndTools(t *testing.T) {
	report := BuildDoctorReport(DoctorOptions{
		Version:    "test-version",
		Adapters:   []adapters.Adapter{fakeAdapter{id: "codex-cli", name: "Codex CLI", detected: true, status: adapters.ToolStatus{Path: "/tmp/home/.codex"}}},
		ToolChecks: []bootstrap.ToolCheckResult{{ID: "codex-cli", Name: "Codex CLI", Installed: true, FoundBinary: "/usr/local/bin/codex"}},
	})
	if !report.OK || report.DotvibeVersion != "test-version" {
		t.Fatalf("report = %#v", report)
	}
	if !report.Capabilities.Export || !report.Capabilities.Import || !report.Capabilities.AgentImportPlan {
		t.Fatalf("capabilities = %#v", report.Capabilities)
	}
	if len(report.Tools) != 1 || report.Tools[0].ID != "codex-cli" || !report.Tools[0].Detected || report.Tools[0].Install.Installed != true {
		t.Fatalf("tools = %#v", report.Tools)
	}
}
