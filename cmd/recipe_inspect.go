package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/recipe"
)

var recipeInspectJSON bool

var recipeInspectCmd = &cobra.Command{
	Use:   "inspect <recipe.vibe>",
	Short: "Inspect a .vibe recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRecipeInspect(args[0], recipeInspectJSON, cmd.OutOrStdout())
	},
}

func init() {
	recipeInspectCmd.Flags().BoolVar(&recipeInspectJSON, "json", false, "print JSON")
	recipeCmd.AddCommand(recipeInspectCmd)
}

func runRecipeInspect(path string, asJSON bool, w io.Writer) error {
	info, err := recipe.AnalyzeArchive(path, recipe.AnalyzeOptions{IncludeRisks: true, LintOptions: recipe.LintOptions{ScanContent: true}})
	if err != nil {
		return err
	}
	if asJSON {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
		return nil
	}
	fmt.Fprintf(w, "Recipe: %s\n", info.Name)
	if info.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", info.Description)
	}
	if info.Author != "" {
		fmt.Fprintf(w, "Author: %s\n", info.Author)
	}
	fmt.Fprintf(w, "Schema: %s\n", info.Schema)
	if !info.Created.IsZero() {
		fmt.Fprintf(w, "Created: %s\n", info.Created.Format(time.RFC3339))
	}
	fmt.Fprintf(w, "Digest: %.12s\n", info.Digest)
	fmt.Fprintf(w, "Files: %d (%s)\n", len(info.Files), formatSize(info.TotalSize))
	fmt.Fprintf(w, "Risks: errors=%d warnings=%d info=%d\n", countLintSeverity(info.Risks, recipe.SeverityError), countLintSeverity(info.Risks, recipe.SeverityWarning), countLintSeverity(info.Risks, recipe.SeverityInfo))
	fmt.Fprintln(w, "Tools:")
	for _, tool := range info.Tools {
		fmt.Fprintf(w, "  %s: %d files, %s, categories=%s\n", tool.ID, tool.FileCount, formatSize(tool.TotalSize), formatCategoryCounts(tool.Categories))
	}
	if len(info.Files) > 0 {
		fmt.Fprintln(w, "Files:")
		for _, file := range info.Files {
			fmt.Fprintf(w, "  [%s/%s] %s (%s)\n", file.ToolID, file.Category, file.Path, formatSize(file.Size))
		}
	}
	return nil
}

func formatCategoryCounts(categories map[string]int) string {
	if len(categories) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(categories))
	for category := range categories {
		keys = append(keys, category)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, category := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", category, categories[category]))
	}
	return strings.Join(parts, ",")
}
