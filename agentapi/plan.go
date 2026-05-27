package agentapi

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/projectmeta"
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
	ArchivePath      string
	Manifest         *backup.Manifest
	RestorePlan      []adapters.RestorePlanEntry
	Bases            []string
	SelectedFiles    []string
	ArchiveSet       *backup.ArchiveSet
	Issues           []AgentIssue
	DestinationHome  string
	DestinationUser  string
	ProjectTargets   map[string]string
	EnableHomeRemap  bool
	ProjectKeyRemaps map[string]string
}

type ImportPlan struct {
	ArchivePath           string                          `json:"archive_path"`
	ArchiveKind           string                          `json:"archive_kind"`
	BaseArchives          []ImportPlanBase                `json:"base_archives,omitempty"`
	Summary               ImportPlanSummary               `json:"summary"`
	Destination           projectmeta.DestinationInfo     `json:"destination,omitempty"`
	ProjectRelocations    []projectmeta.ProjectRelocation `json:"project_relocations,omitempty"`
	Entries               []ImportPlanEntry               `json:"entries"`
	Issues                []AgentIssue                    `json:"issues"`
	RecommendedNextAction string                          `json:"recommended_next_action"`
	GeneratedAt           time.Time                       `json:"generated_at"`
}

type ImportPlanBase struct {
	FileName       string   `json:"file_name"`
	ManifestSHA256 string   `json:"manifest_sha256"`
	ProvidedPaths  []string `json:"provided_paths,omitempty"`
	Resolved       bool     `json:"resolved"`
	Error          string   `json:"error,omitempty"`
}

type ImportPlanSummary struct {
	Total       int `json:"total"`
	Writes      int `json:"writes"`
	Identical   int `json:"identical"`
	Conflicts   int `json:"conflicts"`
	Overwrites  int `json:"overwrites"`
	Unsupported int `json:"unsupported"`
}

type ImportPlanEntry struct {
	Path           string `json:"path"`
	ToolID         string `json:"tool_id"`
	Category       string `json:"category,omitempty"`
	TargetPath     string `json:"target_path,omitempty"`
	Action         string `json:"action"`
	Reason         string `json:"reason,omitempty"`
	NeedsReview    bool   `json:"needs_review"`
	SizeBytes      int64  `json:"size_bytes,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	Storage        string `json:"storage,omitempty"`
	LocalStagePath string `json:"local_stage_path,omitempty"`
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
		if hasNonClaudeOnlyTool(opts.OnlyTools) {
			return ExportPlan{}, fmt.Errorf("project-memory profile only supports claude-code")
		}
		plan.Risk = "private"
		plan.Command = []string{"dotvibe", "export", "--only", "claude-code", "-o", out}
		plan.Notes = append(plan.Notes, "This creates a Claude Code archive; project selection is applied during destination import with --project.")
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
	if len(opts.OnlyTools) > 0 && profile != "project-memory" {
		plan.Command = append(plan.Command, "--only", strings.Join(opts.OnlyTools, ","))
	}
	return plan, nil
}

func hasNonClaudeOnlyTool(tools []string) bool {
	for _, tool := range tools {
		if tool != "" && tool != "claude-code" {
			return true
		}
	}
	return false
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
	opts.Manifest.Normalize()
	plan := ImportPlan{ArchivePath: opts.ArchivePath, ArchiveKind: opts.Manifest.ArchiveKind, Entries: []ImportPlanEntry{}, Issues: append([]AgentIssue{}, opts.Issues...), GeneratedAt: time.Now().UTC()}
	if plan.ArchiveKind == "" {
		plan.ArchiveKind = backup.ArchiveKindFull
	}
	if opts.DestinationHome != "" || opts.DestinationUser != "" {
		plan.Destination = projectmeta.DestinationInfo{Home: opts.DestinationHome, User: opts.DestinationUser}
	}
	if len(opts.Manifest.Projects) > 0 {
		plan.ProjectRelocations = projectmeta.BuildRelocationPlans(projectmeta.RelocationOptions{
			Projects:        opts.Manifest.Projects,
			DestinationHome: opts.DestinationHome,
			DestinationUser: opts.DestinationUser,
			EnableHomeRemap: opts.EnableHomeRemap,
			ProjectTargets:  opts.ProjectTargets,
		})
	}
	plan.BaseArchives = importPlanBases(opts.Manifest, opts.Bases, opts.Issues)
	filesByPath := importPlanFilesByPath(opts.Manifest)
	seen := map[string]struct{}{}
	for _, restore := range opts.RestorePlan {
		entry := ImportPlanEntry{Path: restore.InArchive, ToolID: archiveToolIDForAgent(restore.InArchive), Category: restore.Category, TargetPath: restore.TargetPath, Action: importPlanAction(restore), Reason: restore.Reason}
		entry.LocalStagePath = projectmeta.RemapClaudeArchivePath(entry.Path, opts.ProjectKeyRemaps)
		if entry.LocalStagePath == entry.Path {
			entry.LocalStagePath = ""
		}
		if file, ok := filesByPath[restore.InArchive]; ok {
			enrichImportPlanEntry(&entry, file)
		}
		if same, err := importPlanEntryMatchesTarget(opts.ArchiveSet, restore); err != nil {
			plan.Issues = append(plan.Issues, AgentIssue{Severity: "warning", Code: "compare_failed", Message: err.Error()})
		} else if same {
			entry.Action = "same"
			entry.Reason = "target content matches archive"
		}
		entry.NeedsReview = entry.Action == "conflict"
		plan.Entries = append(plan.Entries, entry)
		seen[entry.Path] = struct{}{}
		addImportPlanSummary(&plan.Summary, entry.Action)
	}
	for _, selected := range opts.SelectedFiles {
		if _, ok := seen[selected]; ok {
			continue
		}
		entry := unsupportedImportPlanEntry(selected, filesByPath[selected], "unsupported tool or archive path")
		plan.Entries = append(plan.Entries, entry)
		addImportPlanSummary(&plan.Summary, entry.Action)
	}
	plan.RecommendedNextAction = recommendedImportPlanAction(plan)
	return plan, nil
}

func importPlanBases(m *backup.Manifest, provided []string, issues []AgentIssue) []ImportPlanBase {
	if m == nil || m.Base == nil {
		return nil
	}
	base := ImportPlanBase{
		FileName:       m.Base.FileName,
		ManifestSHA256: m.Base.ManifestSHA256,
		ProvidedPaths:  append([]string{}, provided...),
		Resolved:       !hasImportPlanIssue(issues, "missing_base_archive"),
	}
	if !base.Resolved {
		base.Error = "required base archive is missing or incomplete"
	}
	return []ImportPlanBase{base}
}

func hasImportPlanIssue(issues []AgentIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func importPlanFilesByPath(m *backup.Manifest) map[string]backup.FileManifest {
	out := map[string]backup.FileManifest{}
	if m == nil {
		return out
	}
	for _, file := range m.Files {
		out[file.Path] = file
	}
	return out
}

func enrichImportPlanEntry(entry *ImportPlanEntry, file backup.FileManifest) {
	if file.ToolID != "" {
		entry.ToolID = file.ToolID
	}
	if file.Category != "" {
		entry.Category = file.Category
	}
	entry.SizeBytes = file.Size
	entry.SHA256 = file.SHA256
	entry.Storage = file.Storage
}

func importPlanAction(entry adapters.RestorePlanEntry) string {
	switch entry.Action {
	case adapters.RestoreWrite:
		return "write"
	case adapters.RestoreOverwrite:
		return "overwrite"
	case adapters.RestoreSkip:
		if entry.Reason == "target content matches archive" {
			return "same"
		}
		return "conflict"
	case adapters.RestoreUnsupported:
		return "unsupported"
	default:
		return "unsupported"
	}
}

func importPlanEntryMatchesTarget(set *backup.ArchiveSet, entry adapters.RestorePlanEntry) (bool, error) {
	if set == nil || entry.TargetPath == "" {
		return false, nil
	}
	targetData, err := os.ReadFile(entry.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read target %s: %w", entry.TargetPath, err)
	}
	archiveData, err := set.ReadFile(entry.InArchive)
	if err != nil {
		return false, fmt.Errorf("failed to read archive file %s: %w", entry.InArchive, err)
	}
	return bytes.Equal(targetData, archiveData), nil
}

func unsupportedImportPlanEntry(path string, file backup.FileManifest, reason string) ImportPlanEntry {
	entry := ImportPlanEntry{Path: path, ToolID: archiveToolIDForAgent(path), Category: file.Category, Action: "unsupported", Reason: reason}
	enrichImportPlanEntry(&entry, file)
	if entry.ToolID == "" {
		entry.ToolID = archiveToolIDForAgent(path)
	}
	return entry
}

func addImportPlanSummary(summary *ImportPlanSummary, action string) {
	summary.Total++
	switch action {
	case "write":
		summary.Writes++
	case "same":
		summary.Identical++
	case "conflict":
		summary.Conflicts++
	case "overwrite":
		summary.Overwrites++
	case "unsupported":
		summary.Unsupported++
	}
}

func recommendedImportPlanAction(plan ImportPlan) string {
	for _, issue := range plan.Issues {
		if issue.Severity != "error" {
			continue
		}
		if issue.Code == "missing_base_archive" {
			return "provide-base-archives"
		}
		return "resolve-plan-issues"
	}
	if plan.Summary.Unsupported > 0 {
		return "fix-unsupported-paths"
	}
	if plan.Summary.Conflicts > 0 {
		return "stage-or-choose-conflict-policy"
	}
	if plan.Summary.Overwrites > 0 {
		return "confirm-overwrite"
	}
	return "safe-to-import"
}

func archiveToolIDForAgent(path string) string {
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return ""
}
