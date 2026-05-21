package backup

import (
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/yangyifan18/dotvibe/adapters"
)

// ArchivePlan describes the logical manifest and physical payloads to write.
type ArchivePlan struct {
	Manifest      *Manifest
	StoredEntries []StoredEntry
	Added         int
	Changed       int
	Reused        int
}

func BuildFullArchivePlan(manifest *Manifest, entries []adapters.FileEntry) (ArchivePlan, error) {
	if manifest == nil {
		return ArchivePlan{}, fmt.Errorf("manifest is nil")
	}
	manifest.FormatVersion = 2
	manifest.ArchiveKind = ArchiveKindFull
	manifest.Base = nil
	return buildArchivePlan(manifest, entries, nil)
}

func BuildIncrementalArchivePlan(manifest *Manifest, entries []adapters.FileEntry, base *Manifest, baseRef BaseArchiveRef) (ArchivePlan, error) {
	if manifest == nil {
		return ArchivePlan{}, fmt.Errorf("manifest is nil")
	}
	if base == nil {
		return ArchivePlan{}, fmt.Errorf("base manifest is required for incremental export")
	}
	manifest.FormatVersion = 2
	manifest.ArchiveKind = ArchiveKindIncremental
	manifest.Base = &baseRef
	return buildArchivePlan(manifest, entries, base)
}

func buildArchivePlan(manifest *Manifest, entries []adapters.FileEntry, base *Manifest) (ArchivePlan, error) {
	baseByPath := map[string]FileManifest{}
	if base != nil {
		base.Normalize()
		for _, file := range base.Files {
			baseByPath[file.Path] = file
		}
	}

	plan := ArchivePlan{Manifest: manifest}
	files := make([]FileManifest, 0, len(entries))
	stored := []StoredEntry{}
	seenLogical := map[string]struct{}{}
	seenStored := map[string]struct{}{}
	for _, entry := range entries {
		if err := validateArchivePath(entry.InArchive); err != nil {
			return ArchivePlan{}, err
		}
		if _, ok := seenLogical[entry.InArchive]; ok {
			return ArchivePlan{}, fmt.Errorf("duplicate archive path: %s", entry.InArchive)
		}
		seenLogical[entry.InArchive] = struct{}{}

		size, sum, err := sourceFileInfo(entry.SourcePath)
		if err != nil {
			return ArchivePlan{}, err
		}

		fm := FileManifest{
			Path:     entry.InArchive,
			ToolID:   toolIDFromArchivePath(entry.InArchive),
			Size:     size,
			SHA256:   sum,
			Category: entry.Category,
		}
		if baseFile, ok := baseByPath[entry.InArchive]; ok && baseFile.SHA256 == sum {
			fm.Storage = FileStorageBase
			plan.Reused++
		} else {
			fm.Storage = FileStorageInline
			fm.StoredPath = objectPathForSHA256(sum)
			if _, ok := seenStored[fm.StoredPath]; !ok {
				stored = append(stored, StoredEntry{SourcePath: entry.SourcePath, StoredPath: fm.StoredPath})
				seenStored[fm.StoredPath] = struct{}{}
			}
			if _, existed := baseByPath[entry.InArchive]; existed {
				plan.Changed++
			} else {
				plan.Added++
			}
		}
		files = append(files, fm)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	sort.Slice(stored, func(i, j int) bool { return stored[i].StoredPath < stored[j].StoredPath })
	manifest.Files = files
	manifest.Normalize()
	plan.Manifest = manifest
	plan.StoredEntries = stored
	return plan, nil
}

func sourceFileInfo(filePath string) (int64, string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, "", err
	}
	sum, err := sourceFileSHA256(filePath)
	if err != nil {
		return 0, "", err
	}
	return info.Size(), sum, nil
}

func objectPathForSHA256(sum string) string {
	if len(sum) < 4 {
		return path.Join("objects", "sha256", sum)
	}
	return path.Join("objects", "sha256", sum[:2], sum[2:])
}
