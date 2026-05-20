package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/backup"
)

var (
	diffOnlyTool string
	diffCategory string
	diffJSON     bool
)

type archiveDiff struct {
	Added     []string
	Removed   []string
	Changed   []string
	Unchanged []string
}

type diffOptions struct {
	OnlyTool string
	Category string
	JSON     bool
}

var diffCmd = &cobra.Command{
	Use:   "diff <archive-a> <archive-b>",
	Short: "Compare two backup archives",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		diff, err := diffArchivesWithOptions(args[0], args[1], diffOptions{
			OnlyTool: diffOnlyTool,
			Category: diffCategory,
			JSON:     diffJSON,
		})
		if err != nil {
			return err
		}
		return printArchiveDiff(diff, diffJSON)
	},
}

func init() {
	diffCmd.Flags().StringVar(&diffOnlyTool, "only", "", "Only compare files for the given tool ID")
	diffCmd.Flags().StringVar(&diffCategory, "category", "", "Only compare files in the given category")
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "Print diff as stable JSON")
	rootCmd.AddCommand(diffCmd)
}

func diffArchives(leftPath, rightPath string) (archiveDiff, error) {
	return diffArchivesWithOptions(leftPath, rightPath, diffOptions{})
}

func diffArchivesWithOptions(leftPath, rightPath string, opts diffOptions) (archiveDiff, error) {
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
	leftFiles = filterManifestFileMap(leftFiles, opts)
	rightFiles = filterManifestFileMap(rightFiles, opts)

	return compareManifestFileMaps(leftFiles, rightFiles), nil
}

func compareManifestFileMaps(leftFiles, rightFiles map[string]backup.FileManifest) archiveDiff {
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
	return diff
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

func filterManifestFileMap(files map[string]backup.FileManifest, opts diffOptions) map[string]backup.FileManifest {
	if opts.OnlyTool == "" && opts.Category == "" {
		return files
	}
	filtered := map[string]backup.FileManifest{}
	for path, file := range files {
		if opts.OnlyTool != "" && !matchesDiffTool(path, file, opts.OnlyTool) {
			continue
		}
		if opts.Category != "" && file.Category != opts.Category {
			continue
		}
		filtered[path] = file
	}
	return filtered
}

func matchesDiffTool(path string, file backup.FileManifest, tool string) bool {
	return file.ToolID == tool || strings.HasPrefix(path, tool+"/")
}

func printArchiveDiff(diff archiveDiff, asJSON bool) error {
	if asJSON {
		diff = archiveDiffWithNonNilSlices(diff)
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(diff)
	}
	fmt.Printf("Added: %d\n", len(diff.Added))
	printDiffPaths(diff.Added)
	fmt.Printf("Removed: %d\n", len(diff.Removed))
	printDiffPaths(diff.Removed)
	fmt.Printf("Changed: %d\n", len(diff.Changed))
	printDiffPaths(diff.Changed)
	fmt.Printf("Unchanged: %d\n", len(diff.Unchanged))
	return nil
}

func archiveDiffWithNonNilSlices(diff archiveDiff) archiveDiff {
	if diff.Added == nil {
		diff.Added = []string{}
	}
	if diff.Removed == nil {
		diff.Removed = []string{}
	}
	if diff.Changed == nil {
		diff.Changed = []string{}
	}
	if diff.Unchanged == nil {
		diff.Unchanged = []string{}
	}
	return diff
}

func printDiffPaths(paths []string) {
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}
}
