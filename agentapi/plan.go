package agentapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

type ExportPlanOptions struct {
	Profile        string
	Output         string
	Name           string
	Author         string
	OnlyTools      []string
	IncludeHistory bool
	BaseArchive    string
}

type ExportPlan struct {
	Profile       string       `json:"profile"`
	Command       []string     `json:"command"`
	SafeToRun     bool         `json:"safe_to_run"`
	WritesArchive bool         `json:"writes_archive"`
	Risk          string       `json:"risk"`
	Warnings      []AgentIssue `json:"warnings"`
	Notes         []string     `json:"notes"`
}

type ImportPlanOptions struct {
	ArchivePath string
	Manifest    *backup.Manifest
	RestorePlan []adapters.RestorePlanEntry
	Bases       []string
}

type ImportPlan struct {
	ArchivePath           string            `json:"archive_path"`
	ArchiveKind           string            `json:"archive_kind"`
	Summary               ImportPlanSummary `json:"summary"`
	Entries               []ImportPlanEntry `json:"entries"`
	Issues                []AgentIssue      `json:"issues"`
	RecommendedNextAction string            `json:"recommended_next_action"`
	GeneratedAt           time.Time         `json:"generated_at"`
}

type ImportPlanSummary struct {
	Total       int `json:"total"`
	Writes      int `json:"writes"`
	Identical   int `json:"identical"`
	Conflicts   int `json:"conflicts"`
	Unsupported int `json:"unsupported"`
}

type ImportPlanEntry struct {
	Path        string `json:"path"`
	ToolID      string `json:"tool_id"`
	Category    string `json:"category,omitempty"`
	TargetPath  string `json:"target_path,omitempty"`
	Action      string `json:"action"`
	Reason      string `json:"reason,omitempty"`
	NeedsReview bool   `json:"needs_review"`
}

func BuildExportPlan(opts ExportPlanOptions) (ExportPlan, error) {
	profile := opts.Profile
	if profile == "" {
		profile = "full"
	}
	out := opts.Output
	if out == "" {
		out = defaultAgentExportOutput(profile)
	}
	plan := ExportPlan{Profile: profile, SafeToRun: true, WritesArchive: true, Warnings: []AgentIssue{}, Notes: []string{}}
	switch profile {
	case "full":
		plan.Risk = "private"
		plan.Command = []string{"dotvibe", "export", "-o", out}
		if opts.IncludeHistory {
			plan.Command = append(plan.Command, "--with-history")
		}
		if opts.BaseArchive != "" {
			plan.Command = append(plan.Command, "--base", opts.BaseArchive)
		}
	case "project-memory":
		plan.Risk = "private"
		plan.Command = []string{"dotvibe", "export", "--only", "claude-code", "-o", out}
		plan.Notes = append(plan.Notes, "Project selection is applied during destination import with --project.")
	case "recipe":
		plan.Risk = "shareable"
		plan.Command = []string{"dotvibe", "recipe", "export", "-o", out}
		if opts.Name != "" {
			plan.Command = append(plan.Command, "--name", opts.Name)
		}
		if opts.Author != "" {
			plan.Command = append(plan.Command, "--author", opts.Author)
		}
	default:
		return ExportPlan{}, fmt.Errorf("unsupported export profile %q", opts.Profile)
	}
	if len(opts.OnlyTools) > 0 {
		plan.Command = append(plan.Command, "--only", strings.Join(opts.OnlyTools, ","))
	}
	return plan, nil
}

func defaultAgentExportOutput(profile string) string {
	stamp := time.Now().Format("2006-01-02")
	if profile == "recipe" {
		return "dotvibe-recipe-" + stamp + ".vibe"
	}
	return "dotvibe-" + stamp + ".tar.gz"
}

func BuildImportPlan(opts ImportPlanOptions) (ImportPlan, error) {
	if opts.Manifest == nil {
		return ImportPlan{}, fmt.Errorf("manifest is required")
	}
	plan := ImportPlan{ArchivePath: opts.ArchivePath, ArchiveKind: opts.Manifest.ArchiveKind, Entries: []ImportPlanEntry{}, Issues: []AgentIssue{}, GeneratedAt: time.Now().UTC()}
	if plan.ArchiveKind == "" {
		plan.ArchiveKind = backup.ArchiveKindFull
	}
	for _, restore := range opts.RestorePlan {
		entry := ImportPlanEntry{Path: restore.InArchive, ToolID: archiveToolIDForAgent(restore.InArchive), Category: restore.Category, TargetPath: restore.TargetPath, Action: importPlanAction(restore), Reason: restore.Reason}
		entry.NeedsReview = entry.Action == "conflict"
		plan.Entries = append(plan.Entries, entry)
		plan.Summary.Total++
		switch entry.Action {
		case "write":
			plan.Summary.Writes++
		case "same":
			plan.Summary.Identical++
		case "conflict":
			plan.Summary.Conflicts++
		case "unsupported":
			plan.Summary.Unsupported++
		}
	}
	plan.RecommendedNextAction = "safe-to-import"
	if plan.Summary.Conflicts > 0 {
		plan.RecommendedNextAction = "stage-or-choose-conflict-policy"
	}
	if plan.Summary.Unsupported > 0 {
		plan.RecommendedNextAction = "fix-unsupported-paths"
	}
	return plan, nil
}

func importPlanAction(entry adapters.RestorePlanEntry) string {
	switch entry.Action {
	case adapters.RestoreWrite:
		return "write"
	case adapters.RestoreOverwrite:
		return "conflict"
	case adapters.RestoreSkip:
		if entry.Reason == "target content matches archive" {
			return "same"
		}
		return "conflict"
	default:
		return "unsupported"
	}
}

func archiveToolIDForAgent(path string) string {
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return ""
}
