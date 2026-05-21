package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

var (
	diffOnlyTool string
	diffCategory string
	diffJSON     bool
)

type archiveDiff struct {
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Changed   []string `json:"changed"`
	Unchanged []string `json:"unchanged"`
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

	leftFiles, err := manifestFileMapFromArchive(left)
	if err != nil {
		return archiveDiff{}, fmt.Errorf("read left archive metadata: %w", err)
	}
	rightFiles, err := manifestFileMapFromArchive(right)
	if err != nil {
		return archiveDiff{}, fmt.Errorf("read right archive metadata: %w", err)
	}
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
		file = normalizeDiffFileManifest(file.Path, file)
		files[file.Path] = file
	}
	if len(files) > 0 {
		return files
	}
	for _, path := range fallback {
		files[path] = normalizeDiffFileManifest(path, backup.FileManifest{Path: path})
	}
	return files
}

func manifestFileMapFromArchive(ar *backup.ArchiveReader) (map[string]backup.FileManifest, error) {
	files := manifestFileMap(ar.Manifest, ar.ListFiles())
	if ar.Manifest != nil && len(ar.Manifest.Files) > 0 {
		return files, nil
	}
	for path, file := range files {
		data, err := ar.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		file.Size = int64(len(data))
		file.SHA256 = fmt.Sprintf("%x", sum)
		files[path] = file
	}
	return files, nil
}

func normalizeDiffFileManifest(path string, file backup.FileManifest) backup.FileManifest {
	if file.Path == "" {
		file.Path = path
	}
	if file.ToolID == "" {
		file.ToolID = toolIDFromDiffArchivePath(path)
	}
	if file.Category == "" {
		file.Category = inferDiffCategoryFromArchivePath(path)
	}
	return file
}

func toolIDFromDiffArchivePath(path string) string {
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return ""
}

func inferDiffCategoryFromArchivePath(path string) string {
	segments := strings.Split(path, "/")
	if category := inferAdapterSpecificDiffCategory(segments); category != "" {
		return category
	}
	for _, segment := range segments {
		switch segment {
		case adapters.CategoryConfig:
			return adapters.CategoryConfig
		case adapters.CategoryMemory:
			return adapters.CategoryMemory
		case adapters.CategorySkills, "plugins":
			return adapters.CategorySkills
		case adapters.CategoryHistory, "sessions", "transcripts", "todos":
			return adapters.CategoryHistory
		}
	}
	return ""
}

func inferAdapterSpecificDiffCategory(segments []string) string {
	if len(segments) < 2 {
		return ""
	}
	switch segments[0] {
	case "opencode":
		switch segments[1] {
		case "xdg-config", "home-config":
			return adapters.CategoryConfig
		}
	case "codex-cli":
		if segments[1] == "agents" {
			return adapters.CategorySkills
		}
	case "claude-code":
		if segments[1] == "history.jsonl" {
			return adapters.CategoryHistory
		}
		if len(segments) >= 4 && segments[1] == "projects" && segments[len(segments)-1] == "CLAUDE.md" {
			return adapters.CategoryMemory
		}
	}
	return ""
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
