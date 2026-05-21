package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
	"github.com/yangyifan18/dotvibe/recipe"
	"github.com/yangyifan18/dotvibe/rollback"
)

type recipeApplyOptions struct {
	Yes         bool
	Force       bool
	DryRun      bool
	Only        string
	AllowRisk   bool
	ScanContent bool
	StateRoot   string
}

var recipeApplyOpts = recipeApplyOptions{ScanContent: true}

var recipeApplyCmd = &cobra.Command{
	Use:   "apply <recipe.vibe>",
	Short: "Apply a shareable .vibe recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRecipeApply(args[0], recipeApplyOpts, cmd.OutOrStdout())
	},
}

func init() {
	recipeApplyCmd.Flags().BoolVarP(&recipeApplyOpts.Yes, "yes", "y", false, "skip confirmation")
	recipeApplyCmd.Flags().BoolVar(&recipeApplyOpts.Force, "force", false, "overwrite conflicts")
	recipeApplyCmd.Flags().BoolVar(&recipeApplyOpts.DryRun, "dry-run", false, "preview apply without writing files")
	recipeApplyCmd.Flags().StringVar(&recipeApplyOpts.Only, "only", "", "only apply specified tools")
	recipeApplyCmd.Flags().BoolVar(&recipeApplyOpts.AllowRisk, "allow-risk", false, "allow applying recipes with lint errors")
	recipeCmd.AddCommand(recipeApplyCmd)
}

func runRecipeApply(path string, opts recipeApplyOptions, w io.Writer) error {
	if opts.StateRoot == "" {
		opts.StateRoot = rollback.DefaultStateRoot()
	}
	lintResult, err := recipe.LintArchive(path, recipe.LintOptions{ScanContent: opts.ScanContent})
	if err != nil {
		return err
	}
	printLintSummary(w, lintResult)
	if lintResult.HasErrors() && !opts.AllowRisk {
		return fmt.Errorf("recipe has lint errors; use --allow-risk to apply anyway")
	}
	ar, err := readRecipeArchive(path)
	if err != nil {
		return err
	}
	defer ar.Close()
	toolFiles := groupRecipeManifestFiles(ar.Manifest.Files)
	if opts.Only != "" {
		toolFiles = filterApplyTools(toolFiles, splitAndTrim(opts.Only))
	}
	if countImportEntries(toolFiles) == 0 {
		return fmt.Errorf("no recipe files match selected filters")
	}
	preview, err := buildApplyPreview(toolFiles, adapters.RestoreOpts{Force: opts.Force})
	if err != nil {
		return err
	}
	printRecipeSummary(ar.Manifest)
	printRestorePreview(preview)
	if opts.DryRun {
		fmt.Fprintln(w, "Dry run: recipe not applied.")
		return nil
	}
	applyID := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), shortRandomID())
	tmpDir, err := os.MkdirTemp("", "dotvibe-apply-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	selectedFiles := flattenImportFiles(toolFiles)
	if err := backup.ExtractArchiveSetFiles(path, nil, tmpDir, selectedFiles); err != nil {
		return fmt.Errorf("failed to extract recipe: %w", err)
	}
	record := rollback.RollbackRecord{ID: applyID, Operation: rollback.OperationRecipeApply, Created: time.Now(), RecipePath: path, RecipeName: ar.Manifest.Recipe.Name}
	store := rollback.NewStore(opts.StateRoot)
	if err := store.Save(record); err != nil {
		return err
	}
	if err := restoreGroupedFiles(toolFiles, tmpDir, adapters.RestoreOpts{Force: opts.Force}); err != nil {
		return err
	}
	fmt.Fprintf(w, "Apply ID: %s\n", applyID)
	return nil
}

func printLintSummary(w io.Writer, result recipe.LintResult) {
	fmt.Fprintf(w, "Lint: errors=%d warnings=%d info=%d\n", countLintSeverity(result.Findings, recipe.SeverityError), countLintSeverity(result.Findings, recipe.SeverityWarning), countLintSeverity(result.Findings, recipe.SeverityInfo))
}

func shortRandomID() string {
	return fmt.Sprintf("%06x", time.Now().UnixNano()&0xffffff)
}
