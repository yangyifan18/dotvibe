package recipe

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yangyifan18/dotvibe/adapters"
)

const (
	ApplyActionWrite     = "write"
	ApplyActionSame      = "same"
	ApplyActionConflict  = "conflict"
	ApplyActionOverwrite = "overwrite"
	ApplyActionSave      = "save"
	ApplyActionSkip      = "skip"

	ConflictChoiceKeep    = "k"
	ConflictChoiceUse     = "r"
	ConflictChoiceSave    = "s"
	ConflictChoiceDiff    = "d"
	ConflictChoiceKeepAll = "ka"
	ConflictChoiceUseAll  = "ra"
	ConflictChoiceSaveAll = "sa"
)

type ApplyInput struct {
	Entry         adapters.FileEntry
	TargetPath    string
	RecipeContent []byte
}

type ApplyPlan struct {
	Entries []ApplyPlanEntry
}

type ApplyPlanEntry struct {
	Entry          adapters.FileEntry
	TargetPath     string
	RecipeContent  []byte
	Action         string
	ResolvedAction string
	TargetSHA256   string
	RecipeSHA256   string
}

type ConflictOptions struct {
	Yes   bool
	Force bool
}

func BuildApplyPlan(inputs []ApplyInput) (ApplyPlan, error) {
	entries := make([]ApplyPlanEntry, 0, len(inputs))
	for _, input := range inputs {
		recipeSHA := bytesSHA256(input.RecipeContent)
		entry := ApplyPlanEntry{Entry: input.Entry, TargetPath: input.TargetPath, RecipeContent: input.RecipeContent, RecipeSHA256: recipeSHA}
		data, err := os.ReadFile(input.TargetPath)
		if os.IsNotExist(err) {
			entry.Action = ApplyActionWrite
			entry.ResolvedAction = ApplyActionWrite
			entries = append(entries, entry)
			continue
		}
		if err != nil {
			return ApplyPlan{}, err
		}
		entry.TargetSHA256 = bytesSHA256(data)
		if entry.TargetSHA256 == recipeSHA {
			entry.Action = ApplyActionSame
			entry.ResolvedAction = ApplyActionSame
		} else {
			entry.Action = ApplyActionConflict
			entry.ResolvedAction = ApplyActionConflict
		}
		entries = append(entries, entry)
	}
	return ApplyPlan{Entries: entries}, nil
}

func ResolveNonInteractiveConflicts(entries []ApplyPlanEntry, opts ConflictOptions) []ApplyPlanEntry {
	resolved := make([]ApplyPlanEntry, len(entries))
	copy(resolved, entries)
	for i := range resolved {
		if resolved[i].Action != ApplyActionConflict {
			continue
		}
		if opts.Yes && opts.Force {
			resolved[i].ResolvedAction = ApplyActionOverwrite
		} else {
			resolved[i].ResolvedAction = ApplyActionSkip
		}
	}
	return resolved
}

func ApplyConflictChoice(entries []ApplyPlanEntry, choice string) []ApplyPlanEntry {
	resolved := make([]ApplyPlanEntry, len(entries))
	copy(resolved, entries)
	for i := range resolved {
		if resolved[i].Action != ApplyActionConflict {
			continue
		}
		switch choice {
		case ConflictChoiceKeep, ConflictChoiceKeepAll:
			resolved[i].ResolvedAction = ApplyActionSkip
		case ConflictChoiceUse, ConflictChoiceUseAll:
			resolved[i].ResolvedAction = ApplyActionOverwrite
		case ConflictChoiceSave, ConflictChoiceSaveAll:
			resolved[i].ResolvedAction = ApplyActionSave
		}
		if choice == ConflictChoiceKeep || choice == ConflictChoiceUse || choice == ConflictChoiceSave {
			break
		}
	}
	return resolved
}

func IncomingPath(stateRoot, applyID, logicalPath string) string {
	parts := append([]string{stateRoot, "incoming", applyID}, splitSlashPath(logicalPath)...)
	return filepath.Join(parts...)
}

func bytesSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func splitSlashPath(path string) []string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		if part != "" && part != "." && part != ".." {
			parts = append(parts, part)
		}
	}
	return parts
}
