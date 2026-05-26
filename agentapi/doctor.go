package agentapi

import (
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/bootstrap"
)

type DoctorOptions struct {
	Version    string
	Adapters   []adapters.Adapter
	ToolChecks []bootstrap.ToolCheckResult
}

type DoctorReport struct {
	DotvibeVersion string            `json:"dotvibe_version"`
	OK             bool              `json:"ok"`
	Issues         []AgentIssue      `json:"issues"`
	Tools          []AgentToolStatus `json:"tools"`
	Capabilities   AgentCapabilities `json:"capabilities"`
}

type AgentIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type AgentToolStatus struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Detected bool              `json:"detected"`
	Path     string            `json:"path,omitempty"`
	Install  AgentInstallState `json:"install"`
}

type AgentInstallState struct {
	Installed   bool   `json:"installed"`
	FoundBinary string `json:"found_binary,omitempty"`
}

type AgentCapabilities struct {
	Export            bool `json:"export"`
	IncrementalExport bool `json:"incremental_export"`
	Import            bool `json:"import"`
	Setup             bool `json:"setup"`
	Recipe            bool `json:"recipe"`
	Rollback          bool `json:"rollback"`
	AgentDoctor       bool `json:"agent_doctor"`
	AgentInventory    bool `json:"agent_inventory"`
	AgentExportPlan   bool `json:"agent_export_plan"`
	AgentImportPlan   bool `json:"agent_import_plan"`
	StageImport       bool `json:"stage_import"`
}

func BuildDoctorReport(opts DoctorOptions) DoctorReport {
	checksByID := map[string]bootstrap.ToolCheckResult{}
	for _, check := range opts.ToolChecks {
		checksByID[check.ID] = check
	}
	report := DoctorReport{
		DotvibeVersion: opts.Version,
		OK:             true,
		Issues:         []AgentIssue{},
		Tools:          []AgentToolStatus{},
		Capabilities: AgentCapabilities{
			Export: true, IncrementalExport: true, Import: true, Setup: true, Recipe: true, Rollback: true,
			AgentDoctor: true, AgentInventory: true, AgentExportPlan: true, AgentImportPlan: true, StageImport: true,
		},
	}
	for _, adapter := range opts.Adapters {
		detected := adapter.Detect()
		status := adapters.ToolStatus{}
		if detected {
			status = adapter.Status()
		}
		check := checksByID[adapter.ID()]
		report.Tools = append(report.Tools, AgentToolStatus{
			ID:       adapter.ID(),
			Name:     adapter.Name(),
			Detected: detected,
			Path:     status.Path,
			Install:  AgentInstallState{Installed: check.Installed, FoundBinary: check.FoundBinary},
		})
	}
	return report
}
