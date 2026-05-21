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
	applyYes    bool
	applyForce  bool
	applyDryRun bool
	applyOnly   string
)

var applyCmd = &cobra.Command{
	Use:   "apply <recipe.vibe>",
	Short: "Apply a shareable .vibe recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ar, err := readRecipeArchive(args[0])
		if err != nil {
			return err
		}
		defer ar.Close()

		toolFiles := groupRecipeManifestFiles(ar.Manifest.Files)
		if applyOnly != "" {
			toolFiles = filterApplyTools(toolFiles, splitAndTrim(applyOnly))
		}
		if countImportEntries(toolFiles) == 0 {
			return fmt.Errorf("no recipe files match selected filters")
		}

		opts := adapters.RestoreOpts{Force: applyForce}
		preview, err := buildApplyPreview(toolFiles, opts)
		if err != nil {
			return err
		}
		printRecipeSummary(ar.Manifest)
		printRestorePreview(preview)
		if applyDryRun {
			fmt.Println("Dry run: recipe not applied.")
			return nil
		}

		if !applyYes {
			fmt.Print("\nApply recipe? [Y/n] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "" && answer != "y" && answer != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		tmpDir, err := os.MkdirTemp("", "dotvibe-apply-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		selectedFiles := flattenImportFiles(toolFiles)
		if err := backup.ExtractArchiveSetFiles(args[0], nil, tmpDir, selectedFiles); err != nil {
			return fmt.Errorf("failed to extract recipe: %w", err)
		}
		return restoreGroupedFiles(toolFiles, tmpDir, opts)
	},
}

func init() {
	applyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "skip confirmation")
	applyCmd.Flags().BoolVar(&applyForce, "force", false, "overwrite existing files")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "preview apply without writing files")
	applyCmd.Flags().StringVar(&applyOnly, "only", "", "only apply specified tools")
	rootCmd.AddCommand(applyCmd)
}

func readRecipeArchive(path string) (*backup.ArchiveReader, error) {
	ar, err := backup.ReadArchive(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipe: %w", err)
	}
	if ar.Manifest.ArchiveKind != backup.ArchiveKindRecipe || ar.Manifest.Recipe == nil {
		ar.Close()
		return nil, fmt.Errorf("archive is not a dotvibe recipe")
	}
	return ar, nil
}

func groupRecipeFilesByTool(files []string) map[string][]adapters.FileEntry {
	toolFiles := map[string][]adapters.FileEntry{}
	for _, file := range files {
		toolID := archiveToolID(file)
		toolFiles[toolID] = append(toolFiles[toolID], adapters.FileEntry{InArchive: file})
	}
	return toolFiles
}

func groupRecipeManifestFiles(files []backup.FileManifest) map[string][]adapters.FileEntry {
	toolFiles := map[string][]adapters.FileEntry{}
	for _, file := range files {
		toolID := file.ToolID
		if toolID == "" {
			toolID = archiveToolID(file.Path)
		}
		toolFiles[toolID] = append(toolFiles[toolID], adapters.FileEntry{
			InArchive: file.Path,
			Category:  file.Category,
			Size:      file.Size,
		})
	}
	return toolFiles
}

func archiveToolID(file string) string {
	for i, c := range file {
		if c == '/' {
			return file[:i]
		}
	}
	return ""
}

func filterApplyTools(toolFiles map[string][]adapters.FileEntry, only []string) map[string][]adapters.FileEntry {
	filtered := map[string][]adapters.FileEntry{}
	for _, tool := range only {
		if entries, ok := toolFiles[tool]; ok {
			filtered[tool] = entries
		}
	}
	return filtered
}

func buildApplyPreview(toolFiles map[string][]adapters.FileEntry, opts adapters.RestoreOpts) ([]adapters.RestorePlanEntry, error) {
	return buildRestorePreview(toolFiles, opts)
}

func printRecipeSummary(m *backup.Manifest) {
	fmt.Printf("Recipe: %s\n", m.Recipe.Name)
	if m.Recipe.Description != "" {
		fmt.Printf("Description: %s\n", m.Recipe.Description)
	}
	if m.Recipe.Author != "" {
		fmt.Printf("Author: %s\n", m.Recipe.Author)
	}
}

func restoreGroupedFiles(toolFiles map[string][]adapters.FileEntry, archiveDir string, opts adapters.RestoreOpts) error {
	var total adapters.RestoreSummary
	var errs []error
	for _, adapter := range adapters.AllAdapters() {
		entries, ok := toolFiles[adapter.ID()]
		if !ok {
			continue
		}

		fmt.Printf("Applying %s... ", adapter.Name())
		summary, err := adapter.RestoreFiles(entries, archiveDir, opts)
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
		return fmt.Errorf("apply failed: %w", errors.Join(errs...))
	}
	return nil
}
