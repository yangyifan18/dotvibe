package recipe

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

type ExportOptions struct {
	Name            string
	Description     string
	Author          string
	Homepage        string
	IncludeSettings bool
}

type ExportResult struct {
	WrittenFiles  int
	RejectedFiles int
	Rejected      []RejectedEntry
}

func BuildArchive(dst string, entries []adapters.FileEntry, opts ExportOptions) (ExportResult, error) {
	filtered, rejected := FilterShareableEntries(entries)
	if len(filtered) == 0 {
		return ExportResult{RejectedFiles: len(rejected), Rejected: rejected}, fmt.Errorf("recipe has no shareable files")
	}
	if opts.Name == "" {
		opts.Name = "dotvibe recipe"
	}
	toolManifests := buildToolManifests(filtered)
	hostname, _ := os.Hostname()
	manifest := &backup.Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   backup.ArchiveKindRecipe,
		Created:       time.Now().UTC(),
		Hostname:      hostname,
		Recipe: &backup.RecipeMetadata{
			Name:        opts.Name,
			Description: opts.Description,
			Author:      opts.Author,
			Homepage:    opts.Homepage,
			Schema:      backup.RecipeSchemaV1,
			SharePolicy: "shareable-only",
			SourceTools: sourceTools(toolManifests),
		},
		Tools: toolManifests,
	}
	plan, err := backup.BuildRecipeArchivePlan(manifest, filtered)
	if err != nil {
		return ExportResult{}, err
	}
	if err := backup.CreateArchiveWithStoredEntries(dst, plan.Manifest, plan.StoredEntries); err != nil {
		return ExportResult{}, err
	}
	return ExportResult{WrittenFiles: len(filtered), RejectedFiles: len(rejected), Rejected: rejected}, nil
}

func buildToolManifests(entries []adapters.FileEntry) map[string]backup.ToolManifest {
	byTool := map[string]map[string]bool{}
	counts := map[string]int{}
	for _, entry := range entries {
		tool := toolID(entry.InArchive)
		if tool == "" {
			continue
		}
		if byTool[tool] == nil {
			byTool[tool] = map[string]bool{}
		}
		byTool[tool][entry.Category] = true
		counts[tool]++
	}
	out := map[string]backup.ToolManifest{}
	for tool, categories := range byTool {
		included := make([]string, 0, len(categories))
		for category := range categories {
			included = append(included, category)
		}
		sort.Strings(included)
		out[tool] = backup.ToolManifest{Included: included, FileCount: counts[tool]}
	}
	return out
}

func sourceTools(tools map[string]backup.ToolManifest) []string {
	out := make([]string, 0, len(tools))
	for tool := range tools {
		out = append(out, tool)
	}
	sort.Strings(out)
	return out
}

func toolID(path string) string {
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return ""
}
