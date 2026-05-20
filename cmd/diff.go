package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/backup"
)

type archiveDiff struct {
	Added     []string
	Removed   []string
	Changed   []string
	Unchanged []string
}

var diffCmd = &cobra.Command{
	Use:   "diff <archive-a> <archive-b>",
	Short: "Compare two backup archives",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		diff, err := diffArchives(args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Printf("Added: %d\n", len(diff.Added))
		printDiffPaths(diff.Added)
		fmt.Printf("Removed: %d\n", len(diff.Removed))
		printDiffPaths(diff.Removed)
		fmt.Printf("Changed: %d\n", len(diff.Changed))
		printDiffPaths(diff.Changed)
		fmt.Printf("Unchanged: %d\n", len(diff.Unchanged))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func diffArchives(leftPath, rightPath string) (archiveDiff, error) {
	left, err := backup.ReadArchive(leftPath)
	if err != nil {
		return archiveDiff{}, fmt.Errorf("read left archive: %w", err)
	}
	defer left.Close()

	right, err := backup.ReadArchive(rightPath)
	if err != nil {
		return archiveDiff{}, fmt.Errorf("read right archive: %w", err)
	}
	defer right.Close()

	leftFiles := manifestFileMap(left.Manifest, left.ListFiles())
	rightFiles := manifestFileMap(right.Manifest, right.ListFiles())

	paths := map[string]bool{}
	for path := range leftFiles {
		paths[path] = true
	}
	for path := range rightFiles {
		paths[path] = true
	}

	var diff archiveDiff
	for path := range paths {
		leftFile, inLeft := leftFiles[path]
		rightFile, inRight := rightFiles[path]
		switch {
		case !inLeft:
			diff.Added = append(diff.Added, path)
		case !inRight:
			diff.Removed = append(diff.Removed, path)
		case leftFile.SHA256 != "" && rightFile.SHA256 != "" && leftFile.SHA256 != rightFile.SHA256:
			diff.Changed = append(diff.Changed, path)
		case leftFile.SHA256 != "" && rightFile.SHA256 != "" && leftFile.SHA256 == rightFile.SHA256:
			diff.Unchanged = append(diff.Unchanged, path)
		default:
			diff.Unchanged = append(diff.Unchanged, path)
		}
	}
	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Changed)
	sort.Strings(diff.Unchanged)
	return diff, nil
}

func manifestFileMap(manifest *backup.Manifest, fallback []string) map[string]backup.FileManifest {
	files := map[string]backup.FileManifest{}
	for _, file := range manifest.Files {
		files[file.Path] = file
	}
	if len(files) > 0 {
		return files
	}
	for _, path := range fallback {
		files[path] = backup.FileManifest{Path: path}
	}
	return files
}

func printDiffPaths(paths []string) {
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}
}
