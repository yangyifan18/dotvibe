package recipe

import (
	"path"
	"strings"

	"github.com/yangyifan18/dotvibe/adapters"
)

type RejectedEntry struct {
	Entry  adapters.FileEntry
	Reason string
}

func FilterShareableEntries(entries []adapters.FileEntry) ([]adapters.FileEntry, []RejectedEntry) {
	filtered := make([]adapters.FileEntry, 0, len(entries))
	var rejected []RejectedEntry
	for _, entry := range entries {
		if reason := rejectionReason(entry); reason != "" {
			rejected = append(rejected, RejectedEntry{Entry: entry, Reason: reason})
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, rejected
}

func rejectionReason(entry adapters.FileEntry) string {
	p := path.Clean(entry.InArchive)
	lower := strings.ToLower(p)
	if entry.Category == adapters.CategoryHistory || entry.Category == adapters.CategoryMemory {
		return "history or project memory is not shareable"
	}
	blockedPrefixes := []string{
		"claude-code/projects/",
		"claude-code/transcripts/",
		"codex-cli/sessions/",
	}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(p, prefix) {
			return "personal data path is not shareable"
		}
	}
	blockedNames := []string{"auth.json", "credentials.json", "token.json", ".env"}
	for _, name := range blockedNames {
		if strings.HasSuffix(lower, "/"+name) || lower == name {
			return "credential-like file is not shareable"
		}
	}
	blockedFragments := []string{"/cache/", "/telemetry/", "/node_modules/"}
	for _, fragment := range blockedFragments {
		if strings.Contains(lower, fragment) {
			return "cache or telemetry path is not shareable"
		}
	}
	return ""
}
