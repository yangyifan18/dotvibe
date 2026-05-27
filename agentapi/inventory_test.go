package agentapi

import (
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

type inventoryAdapter struct {
	fakeAdapter
	files []adapters.FileEntry
}

func (a inventoryAdapter) ListFiles(opts adapters.ExportOpts) []adapters.FileEntry {
	return a.files
}
func (a inventoryAdapter) ListRecipeFiles(opts adapters.RecipeOpts) []adapters.FileEntry {
	return a.files
}

func TestBuildInventoryGroupsCategoriesAndProfiles(t *testing.T) {
	inv := BuildInventory(InventoryOptions{Adapters: []adapters.Adapter{inventoryAdapter{
		fakeAdapter: fakeAdapter{id: "codex-cli", name: "Codex CLI", detected: true, status: adapters.ToolStatus{Size: 30}},
		files: []adapters.FileEntry{
			{InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents, Size: 10},
			{InArchive: "codex-cli/sessions/one.jsonl", Category: adapters.CategoryHistory, Size: 20},
		},
	}}})
	if len(inv.Tools) != 1 || inv.Tools[0].ID != "codex-cli" || inv.Tools[0].SizeBytes != 30 {
		t.Fatalf("inventory tools = %#v", inv.Tools)
	}
	if got := inv.Tools[0].Categories[0].ID; got != adapters.CategoryAgents {
		t.Fatalf("first category = %q", got)
	}
	if len(inv.RecommendedProfiles) != 3 || inv.RecommendedProfiles[2].ID != "recipe" {
		t.Fatalf("profiles = %#v", inv.RecommendedProfiles)
	}
}
