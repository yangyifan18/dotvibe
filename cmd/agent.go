package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/agentapi"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/bootstrap"
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
	JSON    bool
	Only    string
	Project string
	Force   bool
	Bases   []string
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
	ar, toolFiles, err := readImportPlanArchive(path, opts)
	if err != nil {
		return err
	}
	defer ar.Close()
	preview, err := buildRestorePreview(toolFiles, adapters.RestoreOpts{Force: opts.Force, Project: opts.Project})
	if err != nil {
		return err
	}
	plan, err := agentapi.BuildImportPlan(agentapi.ImportPlanOptions{ArchivePath: path, Manifest: ar.Manifest, RestorePlan: preview, Bases: opts.Bases})
	if err != nil {
		return err
	}
	if opts.JSON {
		return writeAgentJSON(w, plan)
	}
	fmt.Fprintf(w, "Import plan: total=%d writes=%d conflicts=%d action=%s\n", plan.Summary.Total, plan.Summary.Writes, plan.Summary.Conflicts, plan.RecommendedNextAction)
	return nil
}

func readImportPlanArchive(path string, opts agentImportPlanOptions) (*backup.ArchiveReader, map[string][]adapters.FileEntry, error) {
	ar, err := backup.ReadArchive(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read archive: %w", err)
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
	return ar, toolFiles, nil
}
