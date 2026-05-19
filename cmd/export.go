package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/young/dotvibe/adapters"
	"github.com/young/dotvibe/backup"
	"github.com/young/dotvibe/config"
)

var (
	exportOutput   string
	exportWithHist bool
	exportOnly     string
	exportExcludes []string
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
				// Skip directories
				info, err := os.Stat(f.SourcePath)
				if err != nil || info.IsDir() {
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

			// Count projects for claude-code
			if adapter.ID() == "claude-code" {
				s := adapter.Status()
				tm.ProjectCount = s.Projects
			}
			// Count agents for codex-cli
			if adapter.ID() == "codex-cli" {
				s := adapter.Status()
				tm.AgentCount = s.Agents
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

		fmt.Printf("Creating backup... ")
		if err := backup.CreateArchive(output, manifest, allEntries); err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}

		info, _ := os.Stat(output)
		fmt.Printf("done.\n  -> %s (%s)\n", output, formatSize(info.Size()))
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path")
	exportCmd.Flags().BoolVar(&exportWithHist, "with-history", false, "include session/transcript history")
	exportCmd.Flags().StringVar(&exportOnly, "only", "", "only backup specified tools (comma-separated)")
	exportCmd.Flags().StringSliceVar(&exportExcludes, "exclude", nil, "exclude matching paths (glob patterns)")
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
