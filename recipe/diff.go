package recipe

import (
	"fmt"
	"sort"

	"github.com/yangyifan18/dotvibe/backup"
)

const (
	ContentDiffNotRequested = "not_requested"
	ContentDiffText         = "text"
	ContentDiffBinary       = "binary"
	ContentDiffLarge        = "large"
)

type DiffOptions struct {
	IncludeContent bool
}

type RecipeDiff struct {
	Added     []DiffEntry `json:"added"`
	Removed   []DiffEntry `json:"removed"`
	Changed   []DiffEntry `json:"changed"`
	SameCount int         `json:"same_count"`
}

type DiffEntry struct {
	Path              string `json:"path"`
	ToolID            string `json:"tool_id"`
	Category          string `json:"category"`
	LeftSHA256        string `json:"left_sha256,omitempty"`
	RightSHA256       string `json:"right_sha256,omitempty"`
	ContentDiffStatus string `json:"content_diff_status"`
	ContentDiff       string `json:"content_diff,omitempty"`
}

func DiffArchives(leftPath, rightPath string, opts DiffOptions) (RecipeDiff, error) {
	left, err := backup.ReadArchive(leftPath)
	if err != nil {
		return RecipeDiff{}, fmt.Errorf("read left recipe: %w", err)
	}
	defer left.Close()
	right, err := backup.ReadArchive(rightPath)
	if err != nil {
		return RecipeDiff{}, fmt.Errorf("read right recipe: %w", err)
	}
	defer right.Close()
	if left.Manifest.ArchiveKind != backup.ArchiveKindRecipe || right.Manifest.ArchiveKind != backup.ArchiveKindRecipe {
		return RecipeDiff{}, fmt.Errorf("both archives must be dotvibe recipes")
	}
	return diffRecipeReaders(left, right, opts)
}

func diffRecipeReaders(left, right *backup.ArchiveReader, opts DiffOptions) (RecipeDiff, error) {
	leftFiles := fileManifestMap(left.Manifest.Files)
	rightFiles := fileManifestMap(right.Manifest.Files)
	paths := map[string]bool{}
	for path := range leftFiles {
		paths[path] = true
	}
	for path := range rightFiles {
		paths[path] = true
	}
	var diff RecipeDiff
	for path := range paths {
		l, inLeft := leftFiles[path]
		r, inRight := rightFiles[path]
		switch {
		case !inLeft:
			diff.Added = append(diff.Added, diffEntry(path, backup.FileManifest{}, r))
		case !inRight:
			diff.Removed = append(diff.Removed, diffEntry(path, l, backup.FileManifest{}))
		case l.SHA256 == r.SHA256:
			diff.SameCount++
		default:
			entry := diffEntry(path, l, r)
			entry.ContentDiffStatus = ContentDiffNotRequested
			if opts.IncludeContent {
				contentStatus, contentDiff, err := recipeContentDiff(left, right, path, l.Size, r.Size)
				if err != nil {
					return RecipeDiff{}, err
				}
				entry.ContentDiffStatus = contentStatus
				entry.ContentDiff = contentDiff
			}
			diff.Changed = append(diff.Changed, entry)
		}
	}
	sortDiffEntries(diff.Added)
	sortDiffEntries(diff.Removed)
	sortDiffEntries(diff.Changed)
	return diff, nil
}

func fileManifestMap(files []backup.FileManifest) map[string]backup.FileManifest {
	out := map[string]backup.FileManifest{}
	for _, file := range files {
		out[file.Path] = file
	}
	return out
}

func diffEntry(path string, left, right backup.FileManifest) DiffEntry {
	file := right
	if file.Path == "" {
		file = left
	}
	return DiffEntry{Path: path, ToolID: file.ToolID, Category: file.Category, LeftSHA256: left.SHA256, RightSHA256: right.SHA256, ContentDiffStatus: ContentDiffNotRequested}
}

func sortDiffEntries(entries []DiffEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
}

func recipeContentDiff(left, right *backup.ArchiveReader, path string, leftSize, rightSize int64) (string, string, error) {
	leftData, err := left.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	rightData, err := right.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	leftKind := ClassifyText(leftData, int(leftSize))
	rightKind := ClassifyText(rightData, int(rightSize))
	if leftKind == TextKindLarge || rightKind == TextKindLarge {
		return ContentDiffLarge, "", nil
	}
	if leftKind == TextKindBinary || rightKind == TextKindBinary {
		return ContentDiffBinary, "", nil
	}
	return ContentDiffText, UnifiedTextDiff(path+" (left)", path+" (right)", leftData, rightData), nil
}
