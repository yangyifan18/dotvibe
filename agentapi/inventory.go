package agentapi

import (
	"sort"

	"github.com/yangyifan18/dotvibe/adapters"
)

type InventoryOptions struct {
	Adapters []adapters.Adapter
}

type InventoryReport struct {
	Tools               []InventoryTool    `json:"tools"`
	RecommendedProfiles []MigrationProfile `json:"recommended_profiles"`
}

type InventoryTool struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Detected   bool                `json:"detected"`
	Path       string              `json:"path,omitempty"`
	SizeBytes  int64               `json:"size_bytes"`
	Categories []InventoryCategory `json:"categories"`
}

type InventoryCategory struct {
	ID        string `json:"id"`
	FileCount int    `json:"file_count"`
	SizeBytes int64  `json:"size_bytes"`
	Shareable bool   `json:"shareable"`
	Private   bool   `json:"private"`
}

type MigrationProfile struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Risk        string   `json:"risk"`
	CommandKind string   `json:"command_kind"`
	Notes       []string `json:"notes"`
}

func BuildInventory(opts InventoryOptions) InventoryReport {
	report := InventoryReport{Tools: []InventoryTool{}, RecommendedProfiles: defaultMigrationProfiles()}
	for _, adapter := range opts.Adapters {
		detected := adapter.Detect()
		status := adapters.ToolStatus{}
		files := []adapters.FileEntry{}
		if detected {
			status = adapter.Status()
			files = adapter.ListFiles(adapters.ExportOpts{WithHistory: true})
		}
		report.Tools = append(report.Tools, InventoryTool{
			ID:         adapter.ID(),
			Name:       adapter.Name(),
			Detected:   detected,
			Path:       status.Path,
			SizeBytes:  status.Size,
			Categories: summarizeInventoryCategories(files),
		})
	}
	sort.Slice(report.Tools, func(i, j int) bool { return report.Tools[i].ID < report.Tools[j].ID })
	return report
}

func summarizeInventoryCategories(files []adapters.FileEntry) []InventoryCategory {
	byCategory := map[string]*InventoryCategory{}
	for _, file := range files {
		cat := file.Category
		if cat == "" {
			cat = adapters.CategoryConfig
		}
		if byCategory[cat] == nil {
			byCategory[cat] = &InventoryCategory{ID: cat, Shareable: isShareableCategory(cat), Private: isPrivateCategory(cat)}
		}
		byCategory[cat].FileCount++
		byCategory[cat].SizeBytes += file.Size
	}
	out := make([]InventoryCategory, 0, len(byCategory))
	for _, category := range byCategory {
		out = append(out, *category)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func isShareableCategory(category string) bool {
	switch category {
	case adapters.CategorySkills, adapters.CategoryAgents, adapters.CategoryRules, adapters.CategoryCommands, adapters.CategorySettings:
		return true
	default:
		return false
	}
}

func isPrivateCategory(category string) bool {
	switch category {
	case adapters.CategoryMemory, adapters.CategoryHistory, adapters.CategoryConfig:
		return true
	default:
		return false
	}
}

func defaultMigrationProfiles() []MigrationProfile {
	return []MigrationProfile{
		{ID: "full", Label: "Full migration", Description: "Back up config, memory, skills, agents, and safe defaults", Risk: "private", CommandKind: "export", Notes: []string{"History is excluded unless include_history is selected."}},
		{ID: "project-memory", Label: "Project memories", Description: "Create a Claude Code archive; destination can restore selected projects with --project", Risk: "private", CommandKind: "export", Notes: []string{"Current export is Claude Code archive-level and may include non-memory Claude files; project filtering is applied during import."}},
		{ID: "recipe", Label: "Shareable recipe", Description: "Export shareable skills, agents, rules, and safe settings", Risk: "shareable", CommandKind: "recipe-export", Notes: []string{"Personal memory, sessions, transcripts, auth, telemetry, and cache are stripped."}},
	}
}
