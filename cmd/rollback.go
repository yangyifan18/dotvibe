package cmd

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/rollback"
)

type rollbackRunOptions struct {
	StateRoot string
	Path      string
	Yes       bool
	Force     bool
}

var rollbackOpts rollbackRunOptions
var rollbackPruneKeep int
var rollbackPruneOlderThan string
var rollbackPruneDryRun bool

var rollbackCmd = &cobra.Command{
	Use:   "rollback [apply-id]",
	Short: "List, run, or prune rollback transactions",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runRollback(args[0], rollbackOpts, cmd.OutOrStdout())
	},
}
var rollbackListCmd = &cobra.Command{Use: "list", Short: "List rollback transactions", RunE: func(cmd *cobra.Command, args []string) error {
	return runRollbackList(rollbackOpts.StateRoot, cmd.OutOrStdout())
}}
var rollbackRunCmd = &cobra.Command{Use: "run <apply-id>", Short: "Rollback an apply transaction", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	return runRollback(args[0], rollbackOpts, cmd.OutOrStdout())
}}
var rollbackPruneCmd = &cobra.Command{Use: "prune", Short: "Prune rollback transactions", RunE: func(cmd *cobra.Command, args []string) error {
	return runRollbackPrune(rollbackOpts.StateRoot, rollbackPruneKeep, rollbackPruneOlderThan, rollbackPruneDryRun, cmd.OutOrStdout())
}}

func init() {
	addRollbackRunFlags(rollbackCmd)
	addRollbackRunFlags(rollbackRunCmd)
	rollbackPruneCmd.Flags().IntVar(&rollbackPruneKeep, "keep", 0, "keep newest N rollback records")
	rollbackPruneCmd.Flags().StringVar(&rollbackPruneOlderThan, "older-than", "", "prune records older than duration, for example 30d")
	rollbackPruneCmd.Flags().BoolVar(&rollbackPruneDryRun, "dry-run", false, "preview prune")
	rollbackCmd.AddCommand(rollbackListCmd, rollbackRunCmd, rollbackPruneCmd)
	rootCmd.AddCommand(rollbackCmd)
}

func addRollbackRunFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&rollbackOpts.Path, "path", "", "rollback only a file or directory")
	cmd.Flags().BoolVarP(&rollbackOpts.Yes, "yes", "y", false, "skip confirmation")
	cmd.Flags().BoolVar(&rollbackOpts.Force, "force", false, "rollback even if target changed after apply")
}

func runRollback(id string, opts rollbackRunOptions, w io.Writer) error {
	store := rollback.NewStore(stateRootOrDefault(opts.StateRoot))
	record, err := store.Load(id)
	if err != nil {
		return err
	}
	entries := record.FilterEntries(opts.Path)
	if len(entries) == 0 {
		return fmt.Errorf("no rollback entries match path %q", opts.Path)
	}
	fmt.Fprintf(w, "Rollback %s: %d entries selected\n", record.ID, len(entries))
	for _, entry := range entries {
		fmt.Fprintf(w, "- %s -> %s\n", entry.Action, entry.TargetPath)
	}
	if !opts.Yes {
		ok, err := confirmRollback(os.Stdin, w)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("rollback cancelled")
		}
	}
	var errs []error
	for _, entry := range entries {
		if entry.Status != rollback.StatusApplied {
			continue
		}
		if err := rollbackOne(store, record, entry, opts.Force); err != nil {
			errs = append(errs, err)
			fmt.Fprintf(w, "refused %s: %v\n", entry.TargetPath, err)
		} else {
			fmt.Fprintf(w, "rolled back %s\n", entry.TargetPath)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("rollback completed with %d error(s)", len(errs))
	}
	return nil
}

func confirmRollback(r io.Reader, w io.Writer) (bool, error) {
	fmt.Fprint(w, "Continue? [y/N] ")
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func rollbackOne(store rollback.Store, record rollback.RollbackRecord, entry rollback.RollbackEntry, force bool) error {
	if entry.Action == rollback.ActionSave {
		if entry.SavedCopy != "" {
			return os.Remove(entry.SavedCopy)
		}
		return nil
	}
	if !force {
		current, err := os.ReadFile(entry.TargetPath)
		if entry.BeforeState == rollback.BeforeMissing {
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if bytesSHAForCmd(current) != entry.AfterSHA256 {
				return fmt.Errorf("target changed after apply")
			}
		} else {
			if os.IsNotExist(err) {
				return fmt.Errorf("target missing after apply")
			}
			if err != nil {
				return err
			}
			if bytesSHAForCmd(current) != entry.AfterSHA256 {
				return fmt.Errorf("target changed after apply")
			}
		}
	}
	if entry.BeforeState == rollback.BeforeMissing {
		err := os.Remove(entry.TargetPath)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	blobPath := filepath.Join(store.RecordDir(record.ID), entry.BeforeBlob)
	data, err := os.ReadFile(blobPath)
	if err != nil {
		return err
	}
	if got := bytesSHAForCmd(data); got != entry.BeforeSHA256 {
		return fmt.Errorf("before blob checksum mismatch")
	}
	if err := os.MkdirAll(filepath.Dir(entry.TargetPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(entry.TargetPath, data, 0644)
}

func bytesSHAForCmd(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func runRollbackList(stateRoot string, w io.Writer) error {
	store := rollback.NewStore(stateRootOrDefault(stateRoot))
	records, err := store.List()
	if err != nil {
		return err
	}
	for _, record := range records {
		fmt.Fprintf(w, "%s %s %s digest=%s entries=%d actions=%s statuses=%s\n", record.ID, record.Created.Format(time.RFC3339), record.RecipeName, shortDigestForList(record.RecipeDigest), len(record.Entries), rollbackActionCounts(record.Entries), rollbackStatusCounts(record.Entries))
	}
	return nil
}

func shortDigestForList(digest string) string {
	if digest == "" {
		return "unknown"
	}
	if len(digest) <= 12 {
		return digest
	}
	return digest[:12]
}

func rollbackActionCounts(entries []rollback.RollbackEntry) string {
	return orderedRollbackCounts(entries, []string{rollback.ActionWrite, rollback.ActionOverwrite, rollback.ActionSave}, func(entry rollback.RollbackEntry) string {
		return entry.Action
	})
}

func rollbackStatusCounts(entries []rollback.RollbackEntry) string {
	return orderedRollbackCounts(entries, []string{rollback.StatusPending, rollback.StatusApplied, rollback.StatusFailed}, func(entry rollback.RollbackEntry) string {
		return entry.Status
	})
}

func orderedRollbackCounts(entries []rollback.RollbackEntry, order []string, value func(rollback.RollbackEntry) string) string {
	counts := map[string]int{}
	for _, entry := range entries {
		if v := value(entry); v != "" {
			counts[v]++
		}
	}
	var parts []string
	for _, key := range order {
		if counts[key] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
			delete(counts, key)
		}
	}
	for key, count := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", key, count))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

func runRollbackPrune(stateRoot string, keep int, olderThan string, dryRun bool, w io.Writer) error {
	var duration time.Duration
	var err error
	if olderThan != "" {
		duration, err = parsePruneDuration(olderThan)
		if err != nil {
			return err
		}
	}
	store := rollback.NewStore(stateRootOrDefault(stateRoot))
	plan, err := store.Prune(rollback.PruneOptions{Keep: keep, OlderThan: duration, DryRun: dryRun})
	if err != nil {
		return err
	}
	for _, id := range plan.DeletedIDs {
		fmt.Fprintf(w, "%s %s\n", pruneVerb(dryRun), id)
	}
	return nil
}

func parsePruneDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func stateRootOrDefault(root string) string {
	if root != "" {
		return root
	}
	return rollback.DefaultStateRoot()
}

func pruneVerb(dryRun bool) string {
	if dryRun {
		return "would delete"
	}
	return "deleted"
}
