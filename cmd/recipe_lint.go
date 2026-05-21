package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/recipe"
)

type recipeLintOptions struct {
	JSON        bool
	Strict      bool
	ScanContent bool
}

var recipeLintOpts = recipeLintOptions{ScanContent: true}
var recipeLintNoContent bool

var recipeLintCmd = &cobra.Command{
	Use:   "lint <recipe.vibe>",
	Short: "Lint a .vibe recipe for privacy and portability risks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := recipeLintOpts
		if recipeLintNoContent {
			opts.ScanContent = false
		}
		return runRecipeLint(args[0], opts, cmd.OutOrStdout())
	},
}

func init() {
	recipeLintCmd.Flags().BoolVar(&recipeLintOpts.JSON, "json", false, "print JSON")
	recipeLintCmd.Flags().BoolVar(&recipeLintOpts.Strict, "strict", false, "treat warnings as failures")
	recipeLintCmd.Flags().BoolVar(&recipeLintOpts.ScanContent, "content", true, "scan file contents")
	recipeLintCmd.Flags().BoolVar(&recipeLintNoContent, "no-content", false, "disable content scanning")
	recipeCmd.AddCommand(recipeLintCmd)
}

func runRecipeLint(path string, opts recipeLintOptions, w io.Writer) error {
	result, err := recipe.LintArchive(path, recipe.LintOptions{ScanContent: opts.ScanContent, Strict: opts.Strict})
	if err != nil {
		return err
	}
	if opts.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
	} else {
		for _, finding := range result.Findings {
			fmt.Fprintf(w, "%s %-24s %s %s\n", finding.Severity, finding.Code, finding.Path, finding.Message)
		}
		fmt.Fprintf(w, "Summary: errors=%d warnings=%d info=%d\n", countLintSeverity(result.Findings, recipe.SeverityError), countLintSeverity(result.Findings, recipe.SeverityWarning), countLintSeverity(result.Findings, recipe.SeverityInfo))
	}
	if result.ExitCode(opts.Strict) != 0 {
		return fmt.Errorf("recipe lint failed")
	}
	return nil
}

func countLintSeverity(findings []recipe.LintFinding, severity string) int {
	count := 0
	for _, finding := range findings {
		if finding.Severity == severity {
			count++
		}
	}
	return count
}
