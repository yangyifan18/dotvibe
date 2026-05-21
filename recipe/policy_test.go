package recipe

import (
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestFilterShareableEntriesStripsPersonalData(t *testing.T) {
	entries := []adapters.FileEntry{
		{InArchive: "claude-code/skills/reviewer/SKILL.md", Category: adapters.CategorySkills},
		{InArchive: "claude-code/projects/private/MEMORY.md", Category: adapters.CategoryMemory},
		{InArchive: "claude-code/transcripts/session.jsonl", Category: adapters.CategoryHistory},
		{InArchive: "codex-cli/agents/reviewer.md", Category: adapters.CategoryAgents},
		{InArchive: "codex-cli/sessions/2026/01/private.jsonl", Category: adapters.CategoryHistory},
		{InArchive: "codex-cli/config/auth.json", Category: adapters.CategoryConfig},
	}

	filtered, rejected := FilterShareableEntries(entries)
	paths := archivePaths(filtered)
	if len(rejected) != 4 {
		t.Fatalf("rejected = %d, want 4 (%#v)", len(rejected), rejected)
	}
	assertPolicyContains(t, paths, "claude-code/skills/reviewer/SKILL.md")
	assertPolicyContains(t, paths, "codex-cli/agents/reviewer.md")
	assertPolicyNotContainsPrefix(t, paths, "claude-code/projects/")
	assertPolicyNotContainsPrefix(t, paths, "codex-cli/sessions/")
}

func TestFilterShareableEntriesKeepsRulesAndSettings(t *testing.T) {
	entries := []adapters.FileEntry{
		{InArchive: "claude-code/rules/CLAUDE.md", Category: adapters.CategoryRules},
		{InArchive: "claude-code/config/settings.json", Category: adapters.CategorySettings},
		{InArchive: "codex-cli/rules/AGENTS.md", Category: adapters.CategoryRules},
		{InArchive: "codex-cli/config/config.toml", Category: adapters.CategorySettings},
	}
	filtered, rejected := FilterShareableEntries(entries)
	if len(rejected) != 0 {
		t.Fatalf("unexpected rejected entries: %#v", rejected)
	}
	if len(filtered) != 4 {
		t.Fatalf("filtered = %d, want 4", len(filtered))
	}
}

func archivePaths(entries []adapters.FileEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.InArchive)
	}
	return paths
}

func assertPolicyContains(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("%q not found in %#v", want, items)
}

func assertPolicyNotContainsPrefix(t *testing.T, items []string, prefix string) {
	t.Helper()
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			t.Fatalf("unexpected path with prefix %q in %#v", prefix, items)
		}
	}
}
