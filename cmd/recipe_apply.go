package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	plan, err := buildRecipeApplyPlanFromArchive(toolFiles, tmpDir, adapters.RestoreOpts{Force: opts.Force})
	if err != nil {
		return err
	}
	resolved := plan.Entries
	if opts.Yes {
		resolved = recipe.ResolveNonInteractiveConflicts(plan.Entries, recipe.ConflictOptions{Yes: opts.Yes, Force: opts.Force})
	} else {
		resolved, err = resolveInteractiveConflicts(plan.Entries, os.Stdin, w)
		if err != nil {
			return err
		}
	}
	record.Entries = rollbackEntriesFromPlan(resolved)
	if err := store.Save(record); err != nil {
		return err
	}
	summary, err := executeRecipeApplyPlan(resolved, store, &record, tmpDir, opts.StateRoot)
	if saveErr := store.Save(record); saveErr != nil && err == nil {
		err = saveErr
	}
	printRecipeApplySummary(w, summary)
	fmt.Fprintf(w, "Apply ID: %s\n", applyID)
	return err
}

func printLintSummary(w io.Writer, result recipe.LintResult) {
	fmt.Fprintf(w, "Lint: errors=%d warnings=%d info=%d\n", countLintSeverity(result.Findings, recipe.SeverityError), countLintSeverity(result.Findings, recipe.SeverityWarning), countLintSeverity(result.Findings, recipe.SeverityInfo))
}

func shortRandomID() string {
	return fmt.Sprintf("%06x", time.Now().UnixNano()&0xffffff)
}

func buildRecipeApplyPlanFromArchive(toolFiles map[string][]adapters.FileEntry, archiveDir string, opts adapters.RestoreOpts) (recipe.ApplyPlan, error) {
	var inputs []recipe.ApplyInput
	for _, adapter := range adapters.AllAdapters() {
		entries, ok := toolFiles[adapter.ID()]
		if !ok {
			continue
		}
		plans, err := adapter.PlanRestore(entries, opts)
		if err != nil {
			return recipe.ApplyPlan{}, err
		}
		for _, plan := range plans {
			data, err := os.ReadFile(filepath.Join(archiveDir, plan.InArchive))
			if err != nil {
				return recipe.ApplyPlan{}, err
			}
			inputs = append(inputs, recipe.ApplyInput{Entry: plan.FileEntry, TargetPath: plan.TargetPath, RecipeContent: data})
		}
	}
	return recipe.BuildApplyPlan(inputs)
}

func rollbackEntriesFromPlan(entries []recipe.ApplyPlanEntry) []rollback.RollbackEntry {
	out := make([]rollback.RollbackEntry, 0, len(entries))
	for _, entry := range entries {
		action := rollbackAction(entry.ResolvedAction)
		if action == "" {
			continue
		}
		beforeState := rollback.BeforeMissing
		if entry.TargetSHA256 != "" {
			beforeState = rollback.BeforeFile
		}
		out = append(out, rollback.RollbackEntry{LogicalPath: entry.Entry.InArchive, TargetPath: entry.TargetPath, Action: action, Status: rollback.StatusPending, BeforeState: beforeState, BeforeSHA256: entry.TargetSHA256, AfterSHA256: entry.RecipeSHA256})
	}
	return out
}

func rollbackAction(action string) string {
	switch action {
	case recipe.ApplyActionWrite:
		return rollback.ActionWrite
	case recipe.ApplyActionOverwrite:
		return rollback.ActionOverwrite
	case recipe.ApplyActionSave:
		return rollback.ActionSave
	default:
		return ""
	}
}

type recipeApplySummary struct {
	Written     int
	Overwritten int
	Saved       int
	Skipped     int
	Failed      int
}

func executeRecipeApplyPlan(entries []recipe.ApplyPlanEntry, store rollback.Store, record *rollback.RollbackRecord, archiveDir string, stateRoot string) (recipeApplySummary, error) {
	var summary recipeApplySummary
	var errs []error
	for _, entry := range entries {
		switch entry.ResolvedAction {
		case recipe.ApplyActionWrite, recipe.ApplyActionOverwrite:
			if err := applyWriteEntry(entry, store, record); err != nil {
				summary.Failed++
				errs = append(errs, err)
				continue
			}
			if entry.ResolvedAction == recipe.ApplyActionOverwrite {
				summary.Overwritten++
			} else {
				summary.Written++
			}
		case recipe.ApplyActionSave:
			if err := applySaveEntry(entry, stateRoot, record); err != nil {
				summary.Failed++
				errs = append(errs, err)
				continue
			}
			summary.Saved++
		case recipe.ApplyActionSame, recipe.ApplyActionSkip, recipe.ApplyActionConflict:
			summary.Skipped++
		}
	}
	return summary, errors.Join(errs...)
}

func applyWriteEntry(entry recipe.ApplyPlanEntry, store rollback.Store, record *rollback.RollbackRecord) error {
	idx := findRollbackEntry(record, entry.Entry.InArchive)
	if idx >= 0 {
		record.Entries[idx].Status = rollback.StatusPending
		if record.Entries[idx].BeforeState == rollback.BeforeFile {
			oldData, err := os.ReadFile(entry.TargetPath)
			if err != nil {
				record.Entries[idx].Status = rollback.StatusFailed
				record.Entries[idx].Error = err.Error()
				return err
			}
			_, blob, err := rollback.WriteBlob(store.RecordDir(record.ID), oldData)
			if err != nil {
				record.Entries[idx].Status = rollback.StatusFailed
				record.Entries[idx].Error = err.Error()
				return err
			}
			record.Entries[idx].BeforeBlob = blob
		}
	}
	if err := os.MkdirAll(filepath.Dir(entry.TargetPath), 0755); err != nil {
		markRollbackFailed(record, idx, err)
		return err
	}
	if err := os.WriteFile(entry.TargetPath, entry.RecipeContent, 0644); err != nil {
		markRollbackFailed(record, idx, err)
		return err
	}
	if idx >= 0 {
		record.Entries[idx].Status = rollback.StatusApplied
	}
	return nil
}

func applySaveEntry(entry recipe.ApplyPlanEntry, stateRoot string, record *rollback.RollbackRecord) error {
	savedPath := recipe.IncomingPath(stateRoot, record.ID, entry.Entry.InArchive)
	if err := os.MkdirAll(filepath.Dir(savedPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(savedPath, entry.RecipeContent, 0644); err != nil {
		return err
	}
	idx := findRollbackEntry(record, entry.Entry.InArchive)
	if idx >= 0 {
		record.Entries[idx].SavedCopy = savedPath
		record.Entries[idx].Status = rollback.StatusApplied
	}
	return nil
}

func findRollbackEntry(record *rollback.RollbackRecord, logicalPath string) int {
	for i := range record.Entries {
		if record.Entries[i].LogicalPath == logicalPath {
			return i
		}
	}
	return -1
}

func markRollbackFailed(record *rollback.RollbackRecord, idx int, err error) {
	if idx >= 0 {
		record.Entries[idx].Status = rollback.StatusFailed
		record.Entries[idx].Error = err.Error()
	}
}

func printRecipeApplySummary(w io.Writer, summary recipeApplySummary) {
	fmt.Fprintf(w, "Summary: written=%d overwritten=%d saved=%d skipped=%d failed=%d\n", summary.Written, summary.Overwritten, summary.Saved, summary.Skipped, summary.Failed)
}

func resolveInteractiveConflicts(entries []recipe.ApplyPlanEntry, r io.Reader, w io.Writer) ([]recipe.ApplyPlanEntry, error) {
	resolved := make([]recipe.ApplyPlanEntry, len(entries))
	copy(resolved, entries)
	scanner := bufio.NewScanner(r)
	for i := range resolved {
		if resolved[i].Action != recipe.ApplyActionConflict {
			continue
		}
		for {
			fmt.Fprintf(w, "Conflict: %s\nTarget: %s\n[k] keep local [r] use recipe [s] save but not replace [d] show diff [ka] keep all [ra] use all [sa] save all\nChoice: ", resolved[i].Entry.InArchive, resolved[i].TargetPath)
			if !scanner.Scan() {
				resolved[i].ResolvedAction = recipe.ApplyActionSkip
				break
			}
			choice := strings.TrimSpace(scanner.Text())
			switch choice {
			case recipe.ConflictChoiceDiff:
				current, _ := os.ReadFile(resolved[i].TargetPath)
				fmt.Fprintln(w, recipe.UnifiedTextDiff(resolved[i].TargetPath, resolved[i].Entry.InArchive, current, resolved[i].RecipeContent))
				continue
			case recipe.ConflictChoiceKeep:
				resolved[i].ResolvedAction = recipe.ApplyActionSkip
			case recipe.ConflictChoiceUse:
				resolved[i].ResolvedAction = recipe.ApplyActionOverwrite
			case recipe.ConflictChoiceSave:
				resolved[i].ResolvedAction = recipe.ApplyActionSave
			case recipe.ConflictChoiceKeepAll, recipe.ConflictChoiceUseAll, recipe.ConflictChoiceSaveAll:
				return recipe.ApplyConflictChoice(resolved, choice), nil
			default:
				fmt.Fprintln(w, "Unknown choice")
				continue
			}
			break
		}
	}
	return resolved, scanner.Err()
}
