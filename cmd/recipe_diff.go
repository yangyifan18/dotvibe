package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/recipe"
)

type recipeDiffOptions struct {
	JSON    bool
	Content bool
}

var recipeDiffOpts recipeDiffOptions

var recipeDiffCmd = &cobra.Command{
	Use:   "diff <left.vibe> <right.vibe>",
	Short: "Compare two .vibe recipes",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRecipeDiff(args[0], args[1], recipeDiffOpts, cmd.OutOrStdout())
	},
}

func init() {
	recipeDiffCmd.Flags().BoolVar(&recipeDiffOpts.JSON, "json", false, "print JSON")
	recipeDiffCmd.Flags().BoolVar(&recipeDiffOpts.Content, "content", false, "include unified diffs for text files")
	recipeCmd.AddCommand(recipeDiffCmd)
}

func runRecipeDiff(left, right string, opts recipeDiffOptions, w io.Writer) error {
	diff, err := recipe.DiffArchives(left, right, recipe.DiffOptions{IncludeContent: opts.Content})
	if err != nil {
		return err
	}
	if opts.JSON {
		data, err := json.MarshalIndent(diff, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
		return nil
	}
	fmt.Fprintf(w, "Added: %d\n", len(diff.Added))
	printRecipeDiffEntries(w, diff.Added)
	fmt.Fprintf(w, "Removed: %d\n", len(diff.Removed))
	printRecipeDiffEntries(w, diff.Removed)
	fmt.Fprintf(w, "Changed: %d\n", len(diff.Changed))
	printRecipeDiffEntries(w, diff.Changed)
	fmt.Fprintf(w, "Same: %d\n", diff.SameCount)
	return nil
}

func printRecipeDiffEntries(w io.Writer, entries []recipe.DiffEntry) {
	for _, entry := range entries {
		fmt.Fprintf(w, "  %s\n", entry.Path)
		if entry.ContentDiff != "" {
			fmt.Fprintln(w, entry.ContentDiff)
		}
	}
}
