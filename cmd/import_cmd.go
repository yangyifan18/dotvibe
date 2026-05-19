package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/young/dotvibe/adapters"
	"github.com/young/dotvibe/backup"
)

var (
	importYes     bool
	importOnly    string
	importProject string
	importForce   bool
)

var importCmd = &cobra.Command{
	Use:   "import <archive>",
	Short: "Restore from a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ar, err := backup.ReadArchive(args[0])
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}
		defer ar.Close()

		m := ar.Manifest
		files := ar.ListFiles()

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

		// Show what will be restored
		fmt.Println("Backup contents:")
		for toolID, entries := range toolFiles {
			tm := m.Tools[toolID]
			fmt.Printf("  %s: %s (%d files)\n", toolID, tm.Included, len(entries))
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

		if err := backup.ExtractArchive(args[0], tmpDir); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}

		// Restore each tool
		opts := adapters.RestoreOpts{
			Force:   importForce,
			Project: importProject,
		}

		for _, adapter := range adapters.AllAdapters() {
			entries, ok := toolFiles[adapter.ID()]
			if !ok {
				continue
			}

			fmt.Printf("Restoring %s... ", adapter.Name())
			if err := adapter.RestoreFiles(entries, tmpDir, opts); err != nil {
				fmt.Printf("ERROR: %v\n", err)
				continue
			}
			fmt.Printf("done (%d files)\n", len(entries))
		}

		return nil
	},
}

func init() {
	importCmd.Flags().BoolVarP(&importYes, "yes", "y", false, "skip confirmation")
	importCmd.Flags().StringVar(&importOnly, "only", "", "only restore specified tools")
	importCmd.Flags().StringVar(&importProject, "project", "", "restore specific project only")
	importCmd.Flags().BoolVar(&importForce, "force", false, "overwrite existing files")
	rootCmd.AddCommand(importCmd)
}
