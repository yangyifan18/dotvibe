package cmd

import (
	"encoding/json"
	"fmt"
	"io"

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
	fmt.Fprintf(w, "Author: %s\n", info.Author)
	fmt.Fprintf(w, "Schema: %s\n", info.Schema)
	fmt.Fprintf(w, "Digest: %.12s\n", info.Digest)
	fmt.Fprintf(w, "Files: %d (%s)\n", len(info.Files), formatSize(info.TotalSize))
	for _, tool := range info.Tools {
		fmt.Fprintf(w, "  %s: %d files\n", tool.ID, tool.FileCount)
	}
	if len(info.Risks) > 0 {
		fmt.Fprintf(w, "Risks: %d findings\n", len(info.Risks))
	}
	return nil
}
