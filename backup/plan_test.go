package backup

import (
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
)

func TestBuildFullArchivePlanDeduplicatesPayloads(t *testing.T) {
	src := t.TempDir()
	first := filepath.Join(src, "first.txt")
	second := filepath.Join(src, "second.txt")
	writeFile(t, first, "same")
	writeFile(t, second, "same")
	manifest := &Manifest{Version: "1.0.0", Tools: map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 2}}}
	entries := []adapters.FileEntry{
		{SourcePath: first, InArchive: "tool/config/first.txt", Category: adapters.CategoryConfig},
		{SourcePath: second, InArchive: "tool/config/second.txt", Category: adapters.CategoryConfig},
	}

	plan, err := BuildFullArchivePlan(manifest, entries)
	if err != nil {
		t.Fatalf("BuildFullArchivePlan: %v", err)
	}
	if len(plan.StoredEntries) != 1 {
		t.Fatalf("stored entries = %d, want 1", len(plan.StoredEntries))
	}
	if len(plan.Manifest.Files) != 2 {
		t.Fatalf("manifest files = %d, want 2", len(plan.Manifest.Files))
	}
	if plan.Manifest.Files[0].StoredPath != plan.Manifest.Files[1].StoredPath {
		t.Fatalf("duplicate files should share stored path: %#v", plan.Manifest.Files)
	}
	if plan.Manifest.ArchiveKind != ArchiveKindFull || plan.Manifest.FormatVersion != 2 {
		t.Fatalf("full metadata missing: %#v", plan.Manifest)
	}
	if plan.Manifest.Files[0].StoredPath != objectPathForSHA256(plan.Manifest.Files[0].SHA256) {
		t.Fatalf("stored path = %q, want object path for sha", plan.Manifest.Files[0].StoredPath)
	}
}

func TestBuildIncrementalArchivePlanReusesBaseUnchangedFiles(t *testing.T) {
	src := t.TempDir()
	unchanged := filepath.Join(src, "unchanged.txt")
	changed := filepath.Join(src, "changed.txt")
	newFile := filepath.Join(src, "new.txt")
	writeFile(t, unchanged, "same")
	writeFile(t, changed, "new")
	writeFile(t, newFile, "brand new")
	unchangedSHA := mustFileSHA256ForTest(t, unchanged)
	base := &Manifest{
		Version:       "1.0.0",
		FormatVersion: 2,
		ArchiveKind:   ArchiveKindFull,
		Files: []FileManifest{
			{Path: "tool/config/unchanged.txt", Size: 4, SHA256: unchangedSHA, Category: adapters.CategoryConfig, Storage: FileStorageInline, StoredPath: "tool/config/unchanged.txt"},
			{Path: "tool/config/changed.txt", Size: 3, SHA256: "old-sha", Category: adapters.CategoryConfig, Storage: FileStorageInline, StoredPath: "tool/config/changed.txt"},
			{Path: "tool/config/removed.txt", Size: 7, SHA256: "removed", Category: adapters.CategoryConfig, Storage: FileStorageInline, StoredPath: "tool/config/removed.txt"},
		},
	}
	manifest := &Manifest{Version: "1.0.0", Tools: map[string]ToolManifest{"tool": {Included: []string{"config"}, FileCount: 3}}}
	entries := []adapters.FileEntry{
		{SourcePath: unchanged, InArchive: "tool/config/unchanged.txt", Category: adapters.CategoryConfig},
		{SourcePath: changed, InArchive: "tool/config/changed.txt", Category: adapters.CategoryConfig},
		{SourcePath: newFile, InArchive: "tool/config/new.txt", Category: adapters.CategoryConfig},
	}

	plan, err := BuildIncrementalArchivePlan(manifest, entries, base, BaseArchiveRef{FileName: "base.tar.gz", ManifestSHA256: "basehash"})
	if err != nil {
		t.Fatalf("BuildIncrementalArchivePlan: %v", err)
	}
	if plan.Manifest.ArchiveKind != ArchiveKindIncremental || plan.Manifest.Base == nil {
		t.Fatalf("incremental metadata missing: %#v", plan.Manifest)
	}
	if plan.Manifest.Base.FileName != "base.tar.gz" || plan.Manifest.Base.ManifestSHA256 != "basehash" {
		t.Fatalf("base ref not preserved: %#v", plan.Manifest.Base)
	}
	if len(plan.StoredEntries) != 2 {
		t.Fatalf("stored entries = %d, want changed and new payloads", len(plan.StoredEntries))
	}
	if plan.Added != 1 || plan.Changed != 1 || plan.Reused != 1 {
		t.Fatalf("counts = added %d changed %d reused %d, want 1/1/1", plan.Added, plan.Changed, plan.Reused)
	}
	files := fileManifestMapForTest(plan.Manifest.Files)
	if files["tool/config/unchanged.txt"].Storage != FileStorageBase {
		t.Fatalf("unchanged file should use base storage: %#v", files["tool/config/unchanged.txt"])
	}
	if files["tool/config/changed.txt"].Storage != FileStorageInline {
		t.Fatalf("changed file should use inline storage: %#v", files["tool/config/changed.txt"])
	}
	if files["tool/config/new.txt"].Storage != FileStorageInline {
		t.Fatalf("new file should use inline storage: %#v", files["tool/config/new.txt"])
	}
	if _, ok := files["tool/config/removed.txt"]; ok {
		t.Fatalf("removed file should not be in current manifest: %#v", files)
	}
}

func fileManifestMapForTest(files []FileManifest) map[string]FileManifest {
	out := map[string]FileManifest{}
	for _, f := range files {
		out[f.Path] = f
	}
	return out
}
