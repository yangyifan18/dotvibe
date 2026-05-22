package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

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
	for _, entry := range groupRecipeDiffEntries(entries) {
		fmt.Fprintf(w, "  [%s/%s] %s", entry.ToolID, entry.Category, entry.Path)
		if entry.LeftSHA256 != "" || entry.RightSHA256 != "" {
			fmt.Fprintf(w, " (%s -> %s)", shortDiffSHA(entry.LeftSHA256), shortDiffSHA(entry.RightSHA256))
		}
		fmt.Fprintln(w)
		if entry.ContentDiff != "" {
			fmt.Fprintln(w, entry.ContentDiff)
		}
	}
}

func groupRecipeDiffEntries(entries []recipe.DiffEntry) []recipe.DiffEntry {
	grouped := append([]recipe.DiffEntry(nil), entries...)
	sort.SliceStable(grouped, func(i, j int) bool {
		if grouped[i].ToolID != grouped[j].ToolID {
			return grouped[i].ToolID < grouped[j].ToolID
		}
		if grouped[i].Category != grouped[j].Category {
			return grouped[i].Category < grouped[j].Category
		}
		return grouped[i].Path < grouped[j].Path
	})
	return grouped
}

func shortDiffSHA(sum string) string {
	if sum == "" {
		return "-"
	}
	if len(sum) <= 12 {
		return sum
	}
	return sum[:12]
}
