package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/config"
)

var (
	exportOutput   string
	exportWithHist bool
	exportOnly     string
	exportExcludes []string
	exportForce    bool
	exportBase     string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Create a backup archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := adapters.ExportOpts{
			WithHistory:     exportWithHist,
			ExcludePatterns: exportExcludes,
		}
		if exportOnly != "" {
			for _, s := range splitAndTrim(exportOnly) {
				opts.OnlyTools = append(opts.OnlyTools, s)
			}
		}

		excluder := config.NewExcluder(opts.ExcludePatterns)

		output := exportOutput
		if output == "" {
			output = fmt.Sprintf("dotvibe-%s.tar.gz", time.Now().Format("2006-01-02"))
		}
		if _, err := os.Stat(output); err == nil && !exportForce {
			return fmt.Errorf("output file already exists: %s (use --force to overwrite)", output)
		}

		var baseManifest *backup.Manifest
		var baseRef backup.BaseArchiveRef
		if exportBase != "" {
			baseArchive, err := backup.ReadArchive(exportBase)
			if err != nil {
				return fmt.Errorf("failed to read base archive: %w", err)
			}
			defer baseArchive.Close()
			baseManifest = baseArchive.Manifest
			baseRef = backup.BaseArchiveRef{
				FileName:       filepath.Base(exportBase),
				Created:        baseManifest.Created,
				ManifestSHA256: baseArchive.ManifestDigest(),
			}
		}

		var allEntries []adapters.FileEntry
		toolManifests := make(map[string]backup.ToolManifest)

		for _, adapter := range adapters.AllAdapters() {
			if !adapter.Detect() {
				continue
			}

			if len(opts.OnlyTools) > 0 && !containsTool(opts.OnlyTools, adapter.ID()) {
				continue
			}

			fmt.Printf("Scanning %s... ", adapter.Name())
			files := adapter.ListFiles(opts)

			var filtered []adapters.FileEntry
			for _, f := range files {
				if !isExportableFile(f.SourcePath) {
					continue
				}
				if !excluder.IsExcluded(f.InArchive) {
					filtered = append(filtered, f)
				}
			}

			fmt.Printf("%d files\n", len(filtered))
			allEntries = append(allEntries, filtered...)

			categories := map[string]bool{}
			for _, f := range filtered {
				categories[f.Category] = true
			}
			var included []string
			for cat := range categories {
				included = append(included, cat)
			}

			tm := backup.ToolManifest{
				Included:  included,
				FileCount: len(filtered),
			}

			if adapter.ID() == "claude-code" {
				tm.ProjectCount = countClaudeProjects(filtered)
			}
			if adapter.ID() == "codex-cli" {
				tm.AgentCount = countCodexAgents(filtered)
			}

			toolManifests[adapter.ID()] = tm
		}

		hostname, _ := os.Hostname()
		manifest := &backup.Manifest{
			Version:  "1.0.0",
			Created:  time.Now().UTC(),
			Hostname: hostname,
			Tools:    toolManifests,
		}

		var plan backup.ArchivePlan
		var err error
		if exportBase != "" {
			fmt.Printf("Creating incremental backup against %s (%d files)... ", exportBase, len(allEntries))
			plan, err = backup.BuildIncrementalArchivePlan(manifest, allEntries, baseManifest, baseRef)
		} else {
			fmt.Printf("Creating backup (%d files)... ", len(allEntries))
			plan, err = backup.BuildFullArchivePlan(manifest, allEntries)
		}
		if err != nil {
			return fmt.Errorf("failed to plan archive: %w", err)
		}
		if err := backup.CreateArchiveWithStoredEntries(output, plan.Manifest, plan.StoredEntries); err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}

		info, _ := os.Stat(output)
		if exportBase != "" {
			fmt.Printf("done. added=%d changed=%d reused=%d\n", plan.Added, plan.Changed, plan.Reused)
		} else {
			fmt.Printf("done.\n")
		}
		fmt.Printf("  -> %s (%s)\n", output, formatSize(info.Size()))
		return nil
	},
}

func isExportableFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path")
	exportCmd.Flags().BoolVar(&exportWithHist, "with-history", false, "include session/transcript history")
	exportCmd.Flags().StringVar(&exportOnly, "only", "", "only backup specified tools (comma-separated)")
	exportCmd.Flags().StringSliceVar(&exportExcludes, "exclude", nil, "exclude matching paths (glob patterns)")
	exportCmd.Flags().BoolVar(&exportForce, "force", false, "overwrite output file if it already exists")
	exportCmd.Flags().StringVar(&exportBase, "base", "", "base archive for incremental export")
	rootCmd.AddCommand(exportCmd)
}

func splitAndTrim(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func containsTool(tools []string, id string) bool {
	for _, t := range tools {
		if t == id {
			return true
		}
	}
	return false
}

func countClaudeProjects(entries []adapters.FileEntry) int {
	projects := map[string]bool{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.InArchive, "claude-code/projects/") {
			rel := strings.TrimPrefix(entry.InArchive, "claude-code/projects/")
			if project, _, ok := strings.Cut(rel, "/"); ok {
				projects[project] = true
			}
		}
	}
	return len(projects)
}

func countCodexAgents(entries []adapters.FileEntry) int {
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.InArchive, "codex-cli/agents/") {
			count++
		}
	}
	return count
}
