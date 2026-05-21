package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/recipe"
)

var (
	recipeOutput          string
	recipeName            string
	recipeDescription     string
	recipeAuthor          string
	recipeHomepage        string
	recipeOnly            string
	recipeForce           bool
	recipeIncludeSettings bool
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Create and inspect shareable vibe recipes",
}

var recipeExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export shareable skills, agents, rules, and safe settings as a .vibe recipe",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := recipeName
		if name == "" {
			name = "dotvibe recipe"
		}
		output := recipeOutput
		if output == "" {
			output = defaultRecipeOutput(name)
		}
		if filepath.Ext(output) == "" {
			output += ".vibe"
		}
		if err := ensureOutputWritable(output, recipeForce); err != nil {
			return err
		}

		only := splitAndTrim(recipeOnly)
		entries := collectRecipeEntries(adapters.AllAdapters(), only, adapters.RecipeOpts{IncludeSettings: recipeIncludeSettings})
		result, err := recipe.BuildArchive(output, entries, recipe.ExportOptions{
			Name:            name,
			Description:     recipeDescription,
			Author:          recipeAuthor,
			Homepage:        recipeHomepage,
			IncludeSettings: recipeIncludeSettings,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Created recipe %s (%d files, %d rejected)\n", output, result.WrittenFiles, result.RejectedFiles)
		for _, rejected := range result.Rejected {
			fmt.Printf("  skipped %s: %s\n", rejected.Entry.InArchive, rejected.Reason)
		}
		return nil
	},
}

func init() {
	recipeIncludeSettings = true
	recipeExportCmd.Flags().StringVarP(&recipeOutput, "output", "o", "", "output .vibe file")
	recipeExportCmd.Flags().StringVar(&recipeName, "name", "", "recipe name")
	recipeExportCmd.Flags().StringVar(&recipeDescription, "description", "", "recipe description")
	recipeExportCmd.Flags().StringVar(&recipeAuthor, "author", "", "recipe author")
	recipeExportCmd.Flags().StringVar(&recipeHomepage, "homepage", "", "recipe homepage URL")
	recipeExportCmd.Flags().StringVar(&recipeOnly, "only", "", "only include specified tools")
	recipeExportCmd.Flags().BoolVar(&recipeForce, "force", false, "overwrite output file if it already exists")
	recipeExportCmd.Flags().BoolVar(&recipeIncludeSettings, "settings", true, "include shareable settings files")
	recipeCmd.AddCommand(recipeExportCmd)
	rootCmd.AddCommand(recipeCmd)
}

func collectRecipeEntries(all []adapters.Adapter, only []string, opts adapters.RecipeOpts) []adapters.FileEntry {
	var entries []adapters.FileEntry
	for _, adapter := range all {
		if len(only) > 0 && !containsTool(only, adapter.ID()) {
			continue
		}
		if !adapter.Detect() {
			continue
		}
		entries = append(entries, adapter.ListRecipeFiles(opts)...)
	}
	return entries
}

func filterRecipeEntriesByOnly(entries []adapters.FileEntry, only []string) []adapters.FileEntry {
	if len(only) == 0 {
		return entries
	}
	var filtered []adapters.FileEntry
	for _, entry := range entries {
		for _, tool := range only {
			if strings.HasPrefix(entry.InArchive, tool+"/") {
				filtered = append(filtered, entry)
				break
			}
		}
	}
	return filtered
}

func defaultRecipeOutput(name string) string {
	slug := strings.ToLower(name)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "recipe"
	}
	return fmt.Sprintf("dotvibe-%s-%s.vibe", slug, time.Now().Format("2006-01-02"))
}

func ensureOutputWritable(path string, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("output file already exists: %s (use --force to overwrite)", path)
	}
	return nil
}
