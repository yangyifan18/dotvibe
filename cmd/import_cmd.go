package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

var (
	importYes     bool
	importOnly    string
	importProject string
	importForce   bool
	importDryRun  bool
	importBases   []string
)

var importCmd = &cobra.Command{
	Use:   "import <archive>",
	Short: "Restore from a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		head, err := backup.ReadArchive(args[0])
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		m := head.Manifest
		files := head.ListFiles()
		if err := head.Close(); err != nil {
			return err
		}

		// Group files by tool
		toolFiles := map[string][]adapters.FileEntry{}
		for _, f := range files {
			toolID := ""
			for i, c := range f {
				if c == '/' {
					toolID = f[:i]
					break
				}
			}
			toolFiles[toolID] = append(toolFiles[toolID], adapters.FileEntry{
				InArchive: f,
			})
		}

		// Filter by --only
		if importOnly != "" {
			filtered := map[string][]adapters.FileEntry{}
			for _, tool := range splitAndTrim(importOnly) {
				if entries, ok := toolFiles[tool]; ok {
					filtered[tool] = entries
				}
			}
			toolFiles = filtered
		}
		if importProject != "" {
			toolFiles = filterImportEntriesByProject(toolFiles, importProject)
		}
		if countImportEntries(toolFiles) == 0 {
			return fmt.Errorf("no files match the selected import filters")
		}
		selectedFiles := flattenImportFiles(toolFiles)
		set, err := backup.OpenArchiveSetForFiles(args[0], importBases, selectedFiles)
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}
		defer set.Close()

		// Show what will be restored
		fmt.Println("Backup contents:")
		for toolID, entries := range toolFiles {
			tm := m.Tools[toolID]
			fmt.Printf("  %s: %s (%d files)\n", toolID, tm.Included, len(entries))
		}
		if importProject != "" {
			fmt.Printf("Project filter: %s\n", adapters.ClaudeProjectKey(importProject))
		}
		opts := adapters.RestoreOpts{
			Force:   importForce,
			Project: importProject,
		}
		preview, err := buildRestorePreview(toolFiles, opts)
		if err != nil {
			return err
		}
		printRestorePreview(preview)
		if importDryRun {
			fmt.Println("Dry run: no files restored.")
			return nil
		}

		if !importYes {
			fmt.Print("\nProceed? [Y/n] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "" && answer != "y" && answer != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Extract archive to temp dir
		tmpDir, err := os.MkdirTemp("", "dotvibe-import-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		if err := backup.ExtractArchiveSetFiles(args[0], importBases, tmpDir, selectedFiles); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}

		// Restore each tool
		var total adapters.RestoreSummary
		var errs []error
		for _, adapter := range adapters.AllAdapters() {
			entries, ok := toolFiles[adapter.ID()]
			if !ok {
				continue
			}

			fmt.Printf("Restoring %s... ", adapter.Name())
			summary, err := adapter.RestoreFiles(entries, tmpDir, opts)
			total.Written += summary.Written
			total.Skipped += summary.Skipped
			total.Overwritten += summary.Overwritten
			total.Failed += summary.Failed
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				errs = append(errs, err)
				continue
			}
			fmt.Printf("done (written=%d skipped=%d overwritten=%d)\n", summary.Written, summary.Skipped, summary.Overwritten)
		}

		fmt.Printf("Summary: written=%d skipped=%d overwritten=%d failed=%d\n", total.Written, total.Skipped, total.Overwritten, total.Failed)
		if total.Failed > 0 || len(errs) > 0 {
			return fmt.Errorf("restore failed: %w", errors.Join(errs...))
		}
		return nil
	},
}

func init() {
	importCmd.Flags().BoolVarP(&importYes, "yes", "y", false, "skip confirmation")
	importCmd.Flags().StringVar(&importOnly, "only", "", "only restore specified tools")
	importCmd.Flags().StringVar(&importProject, "project", "", "restore specific project only")
	importCmd.Flags().BoolVar(&importForce, "force", false, "overwrite existing files")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "preview restore without writing files")
	importCmd.Flags().StringSliceVar(&importBases, "base", nil, "base archive for incremental restore; repeat or comma-separate for a chain")
	rootCmd.AddCommand(importCmd)
}

func filterImportEntriesByProject(toolFiles map[string][]adapters.FileEntry, project string) map[string][]adapters.FileEntry {
	filtered := map[string][]adapters.FileEntry{}
	for toolID, entries := range toolFiles {
		for _, adapter := range adapters.AllAdapters() {
			if adapter.ID() == toolID {
				entries = adapter.FilterRestoreEntries(entries, adapters.RestoreOpts{Project: project})
				break
			}
		}
		if len(entries) > 0 {
			filtered[toolID] = entries
		}
	}
	return filtered
}

func countImportEntries(toolFiles map[string][]adapters.FileEntry) int {
	total := 0
	for _, entries := range toolFiles {
		total += len(entries)
	}
	return total
}

func flattenImportFiles(toolFiles map[string][]adapters.FileEntry) []string {
	var files []string
	for _, entries := range toolFiles {
		for _, entry := range entries {
			files = append(files, entry.InArchive)
		}
	}
	return files
}

func buildRestorePreview(toolFiles map[string][]adapters.FileEntry, opts adapters.RestoreOpts) ([]adapters.RestorePlanEntry, error) {
	var preview []adapters.RestorePlanEntry
	for _, adapter := range adapters.AllAdapters() {
		entries, ok := toolFiles[adapter.ID()]
		if !ok {
			continue
		}
		plan, err := adapter.PlanRestore(entries, opts)
		if err != nil {
			return nil, err
		}
		preview = append(preview, plan...)
	}
	return preview, nil
}

func printRestorePreview(preview []adapters.RestorePlanEntry) {
	fmt.Println("Restore preview:")
	for _, entry := range preview {
		fmt.Printf("  %s -> %s [%s: %s]\n", entry.InArchive, entry.TargetPath, entry.Action, entry.Reason)
	}
}
