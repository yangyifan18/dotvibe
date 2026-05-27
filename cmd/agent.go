package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/agentapi"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/bootstrap"
	"github.com/yangyifan18/dotvibe/projectmeta"
)

type agentDoctorOptions struct {
	JSON    bool
	Version string
}

var agentDoctorOpts agentDoctorOptions

var agentCmd = &cobra.Command{Use: "agent", Short: "Agent-facing dotvibe automation helpers"}
var agentDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Report agent-facing migration capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := agentDoctorOpts
		if opts.Version == "" {
			opts.Version = version
		}
		return runAgentDoctor(opts, cmd.OutOrStdout())
	},
}

func init() {
	agentDoctorCmd.Flags().BoolVar(&agentDoctorOpts.JSON, "json", false, "print JSON")
	agentCmd.AddCommand(agentDoctorCmd)
	rootCmd.AddCommand(agentCmd)
}

func runAgentDoctor(opts agentDoctorOptions, w io.Writer) error {
	report := agentapi.BuildDoctorReport(agentapi.DoctorOptions{
		Version:    opts.Version,
		Adapters:   adapters.AllAdapters(),
		ToolChecks: bootstrap.DetectTools(bootstrap.DefaultToolSpecs()),
	})
	if opts.JSON {
		return writeAgentJSON(w, report)
	}
	fmt.Fprintln(w, "dotvibe agent doctor")
	fmt.Fprintf(w, "OK: %v\n", report.OK)
	fmt.Fprintln(w, "Capabilities: export, import, setup, recipe, rollback, agent plans, stage import")
	for _, tool := range report.Tools {
		fmt.Fprintf(w, "  %s: detected=%v installed=%v\n", tool.ID, tool.Detected, tool.Install.Installed)
	}
	return nil
}

func writeAgentJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

type agentInventoryOptions struct{ JSON bool }

var agentInventoryOpts agentInventoryOptions

var agentInventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Print agent-facing migration inventory",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAgentInventory(agentInventoryOpts, cmd.OutOrStdout())
	},
}

func init() {
	agentInventoryCmd.Flags().BoolVar(&agentInventoryOpts.JSON, "json", false, "print JSON")
	agentCmd.AddCommand(agentInventoryCmd)
}

func runAgentInventory(opts agentInventoryOptions, w io.Writer) error {
	report := agentapi.BuildInventory(agentapi.InventoryOptions{Adapters: adapters.AllAdapters()})
	if opts.JSON {
		return writeAgentJSON(w, report)
	}
	fmt.Fprintln(w, "dotvibe agent inventory")
	for _, tool := range report.Tools {
		fmt.Fprintf(w, "  %s: detected=%v categories=%d\n", tool.ID, tool.Detected, len(tool.Categories))
	}
	fmt.Fprintln(w, "Profiles: full, project-memory, recipe")
	return nil
}

type agentExportPlanOptions struct {
	JSON        bool
	Profile     string
	Output      string
	Name        string
	Author      string
	Only        string
	WithHistory bool
	Base        string
}

type agentImportPlanOptions struct {
	JSON           bool
	Only           string
	Project        string
	Force          bool
	Bases          []string
	NoRemapHome    bool
	ProjectTargets []string
}

var agentExportPlanOpts agentExportPlanOptions
var agentImportPlanOpts agentImportPlanOptions

var agentExportPlanCmd = &cobra.Command{
	Use:   "export-plan",
	Short: "Build an agent-facing export plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAgentExportPlan(agentExportPlanOpts, cmd.OutOrStdout())
	},
}
var agentImportPlanCmd = &cobra.Command{
	Use:   "import-plan <archive>",
	Short: "Build an agent-facing import plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAgentImportPlan(args[0], agentImportPlanOpts, cmd.OutOrStdout())
	},
}

func init() {
	agentExportPlanCmd.Flags().BoolVar(&agentExportPlanOpts.JSON, "json", false, "print JSON")
	agentExportPlanCmd.Flags().StringVar(&agentExportPlanOpts.Profile, "profile", "full", "export profile: full, project-memory, recipe")
	agentExportPlanCmd.Flags().StringVarP(&agentExportPlanOpts.Output, "output", "o", "", "planned output archive")
	agentExportPlanCmd.Flags().StringVar(&agentExportPlanOpts.Name, "name", "", "recipe name")
	agentExportPlanCmd.Flags().StringVar(&agentExportPlanOpts.Author, "author", "", "recipe author")
	agentExportPlanCmd.Flags().StringVar(&agentExportPlanOpts.Only, "only", "", "comma-separated tools")
	agentExportPlanCmd.Flags().BoolVar(&agentExportPlanOpts.WithHistory, "with-history", false, "include history in full export plan")
	agentExportPlanCmd.Flags().StringVar(&agentExportPlanOpts.Base, "base", "", "base archive for incremental export plan")
	agentImportPlanCmd.Flags().BoolVar(&agentImportPlanOpts.JSON, "json", false, "print JSON")
	agentImportPlanCmd.Flags().StringVar(&agentImportPlanOpts.Only, "only", "", "comma-separated tools")
	agentImportPlanCmd.Flags().StringVar(&agentImportPlanOpts.Project, "project", "", "Claude project filter")
	agentImportPlanCmd.Flags().BoolVar(&agentImportPlanOpts.Force, "force", false, "plan overwrite conflicts")
	agentImportPlanCmd.Flags().StringSliceVar(&agentImportPlanOpts.Bases, "base", nil, "base archives for incremental import plan")
	agentImportPlanCmd.Flags().BoolVar(&agentImportPlanOpts.NoRemapHome, "no-remap-home", false, "disable default home-prefix project remap in agent plans")
	agentImportPlanCmd.Flags().StringSliceVar(&agentImportPlanOpts.ProjectTargets, "project-target", nil, "project target override source-key=target-path")
	agentCmd.AddCommand(agentExportPlanCmd, agentImportPlanCmd)
}

func runAgentExportPlan(opts agentExportPlanOptions, w io.Writer) error {
	plan, err := agentapi.BuildExportPlan(agentapi.ExportPlanOptions{Profile: opts.Profile, Output: opts.Output, Name: opts.Name, Author: opts.Author, OnlyTools: splitAndTrim(opts.Only), IncludeHistory: opts.WithHistory, BaseArchive: opts.Base})
	if err != nil {
		return err
	}
	if opts.JSON {
		return writeAgentJSON(w, plan)
	}
	fmt.Fprintf(w, "Profile: %s\nCommand: %s\nRisk: %s\n", plan.Profile, strings.Join(plan.Command, " "), plan.Risk)
	return nil
}

func runAgentImportPlan(path string, opts agentImportPlanOptions, w io.Writer) error {
	ar, toolFiles, selectedFiles, err := readImportPlanArchive(path, opts)
	if err != nil {
		return err
	}
	defer ar.Close()
	set, err := backup.OpenArchiveSetForFiles(path, opts.Bases, selectedFiles)
	var issues []agentapi.AgentIssue
	if err != nil {
		issues = append(issues, agentapi.AgentIssue{Severity: "error", Code: importPlanArchiveSetIssueCode(err), Message: err.Error()})
		if !opts.JSON {
			return fmt.Errorf("failed to read archive: %w", err)
		}
	} else {
		defer set.Close()
	}
	destHome, destUser := currentDestinationIdentity()
	projectTargets, err := parseProjectTargets(opts.ProjectTargets)
	if err != nil {
		return err
	}
	keyRemaps := projectKeyRemapsForManifest(ar.Manifest, destHome, destUser, !opts.NoRemapHome, projectTargets)
	preview := buildAgentRestorePreview(toolFiles, adapters.RestoreOpts{Force: opts.Force, Project: opts.Project, ProjectKeyRemaps: keyRemaps})
	plan, err := agentapi.BuildImportPlan(agentapi.ImportPlanOptions{
		ArchivePath:      path,
		Manifest:         ar.Manifest,
		RestorePlan:      preview,
		Bases:            opts.Bases,
		SelectedFiles:    selectedFiles,
		ArchiveSet:       set,
		Issues:           issues,
		DestinationHome:  destHome,
		DestinationUser:  destUser,
		ProjectTargets:   projectTargets,
		EnableHomeRemap:  !opts.NoRemapHome,
		ProjectKeyRemaps: keyRemaps,
	})
	if err != nil {
		return err
	}
	if opts.JSON {
		return writeAgentJSON(w, plan)
	}
	fmt.Fprintf(w, "Import plan: total=%d writes=%d conflicts=%d overwrites=%d unsupported=%d action=%s\n", plan.Summary.Total, plan.Summary.Writes, plan.Summary.Conflicts, plan.Summary.Overwrites, plan.Summary.Unsupported, plan.RecommendedNextAction)
	return nil
}

func importPlanArchiveSetIssueCode(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "base archive") || strings.Contains(msg, "base-backed") {
		return "missing_base_archive"
	}
	return "archive_set_error"
}

func buildAgentRestorePreview(toolFiles map[string][]adapters.FileEntry, opts adapters.RestoreOpts) []adapters.RestorePlanEntry {
	var preview []adapters.RestorePlanEntry
	handled := map[string]struct{}{}
	for _, adapter := range adapters.AllAdapters() {
		entries, ok := toolFiles[adapter.ID()]
		if !ok {
			continue
		}
		handled[adapter.ID()] = struct{}{}
		for _, entry := range entries {
			plan, err := adapter.PlanRestore([]adapters.FileEntry{entry}, opts)
			if err != nil {
				preview = append(preview, unsupportedRestorePlanEntry(entry, err.Error()))
				continue
			}
			preview = append(preview, plan...)
		}
	}
	var unknownTools []string
	for toolID := range toolFiles {
		if _, ok := handled[toolID]; !ok {
			unknownTools = append(unknownTools, toolID)
		}
	}
	sort.Strings(unknownTools)
	for _, toolID := range unknownTools {
		for _, entry := range toolFiles[toolID] {
			preview = append(preview, unsupportedRestorePlanEntry(entry, "unsupported tool: "+toolID))
		}
	}
	return preview
}

func unsupportedRestorePlanEntry(entry adapters.FileEntry, reason string) adapters.RestorePlanEntry {
	return adapters.RestorePlanEntry{
		FileEntry:  entry,
		Action:     adapters.RestoreUnsupported,
		Reason:     reason,
		TargetPath: "",
	}
}

func readImportPlanArchive(path string, opts agentImportPlanOptions) (*backup.ArchiveReader, map[string][]adapters.FileEntry, []string, error) {
	ar, err := backup.ReadArchive(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read archive: %w", err)
	}
	toolFiles := groupRecipeManifestFiles(ar.Manifest.Files)
	if len(ar.Manifest.Files) == 0 {
		toolFiles = groupRecipeFilesByTool(ar.ListFiles())
	}
	if opts.Only != "" {
		toolFiles = filterApplyTools(toolFiles, splitAndTrim(opts.Only))
	}
	if opts.Project != "" {
		toolFiles = filterImportEntriesByProject(toolFiles, opts.Project)
	}
	return ar, toolFiles, flattenImportFiles(toolFiles), nil
}

func currentDestinationIdentity() (string, string) {
	home, _ := os.UserHomeDir()
	name := ""
	if current, err := user.Current(); err == nil {
		name = current.Username
		if slash := strings.LastIndex(name, string(os.PathSeparator)); slash >= 0 {
			name = name[slash+1:]
		}
	}
	return home, name
}

func parseProjectTargets(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, value := range values {
		key, target, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		target = strings.TrimSpace(target)
		if !ok || key == "" || target == "" {
			return nil, fmt.Errorf("project target must be source-key=target-path: %s", value)
		}
		out[key] = target
	}
	return out, nil
}

func projectKeyRemapsForManifest(m *backup.Manifest, destHome, destUser string, remapHome bool, projectTargets map[string]string) map[string]string {
	if m == nil || len(m.Projects) == 0 {
		return nil
	}
	plans := projectmeta.BuildRelocationPlans(projectmeta.RelocationOptions{
		Projects:        m.Projects,
		DestinationHome: destHome,
		DestinationUser: destUser,
		EnableHomeRemap: remapHome,
		ProjectTargets:  projectTargets,
	})
	return projectmeta.BuildProjectKeyRemaps(plans)
}
