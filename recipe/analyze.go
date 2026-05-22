package recipe

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/yangyifan18/dotvibe/backup"
)

type AnalyzeOptions struct {
	IncludeRisks bool
	LintOptions  LintOptions
}

type RecipeInfo struct {
	Path        string           `json:"path"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Author      string           `json:"author,omitempty"`
	Schema      string           `json:"schema"`
	Created     time.Time        `json:"created"`
	Digest      string           `json:"digest"`
	TotalSize   int64            `json:"total_size"`
	Tools       []ToolSummary    `json:"tools"`
	Files       []RecipeFileInfo `json:"files"`
	Risks       []LintFinding    `json:"risks,omitempty"`
}

type ToolSummary struct {
	ID         string         `json:"id"`
	FileCount  int            `json:"file_count"`
	TotalSize  int64          `json:"total_size"`
	Categories map[string]int `json:"categories"`
}

type RecipeFileInfo struct {
	Path     string `json:"path"`
	ToolID   string `json:"tool_id"`
	Category string `json:"category"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
}

// Minimal declarations allow analysis to expose future lint fields before the lint core is implemented.
type LintOptions struct {
	ScanContent bool
	Strict      bool
}

type LintFinding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

func AnalyzeArchive(path string, opts AnalyzeOptions) (RecipeInfo, error) {
	ar, err := backup.ReadArchive(path)
	if err != nil {
		return RecipeInfo{}, fmt.Errorf("read recipe: %w", err)
	}
	defer ar.Close()
	if ar.Manifest.ArchiveKind != backup.ArchiveKindRecipe || ar.Manifest.Recipe == nil {
		return RecipeInfo{}, fmt.Errorf("archive is not a dotvibe recipe")
	}
	info, err := buildRecipeInfo(path, ar.Manifest)
	if err != nil {
		return RecipeInfo{}, err
	}
	if opts.IncludeRisks {
		result, err := LintArchive(path, opts.LintOptions)
		if err != nil {
			return RecipeInfo{}, err
		}
		info.Risks = result.Findings
	}
	return info, nil
}

func buildRecipeInfo(path string, manifest *backup.Manifest) (RecipeInfo, error) {
	digest, size, err := fileDigestAndSize(path)
	if err != nil {
		return RecipeInfo{}, err
	}
	files := make([]RecipeFileInfo, 0, len(manifest.Files))
	toolMap := map[string]*ToolSummary{}
	for _, file := range manifest.Files {
		toolID := file.ToolID
		if toolID == "" {
			toolID = toolIDFromRecipePath(file.Path)
		}
		files = append(files, RecipeFileInfo{Path: file.Path, ToolID: toolID, Category: file.Category, Size: file.Size, SHA256: file.SHA256})
		if toolMap[toolID] == nil {
			toolMap[toolID] = &ToolSummary{ID: toolID, Categories: map[string]int{}}
		}
		toolMap[toolID].FileCount++
		toolMap[toolID].TotalSize += file.Size
		toolMap[toolID].Categories[file.Category]++
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	tools := make([]ToolSummary, 0, len(toolMap))
	for _, tool := range toolMap {
		tools = append(tools, *tool)
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].ID < tools[j].ID })
	return RecipeInfo{
		Path:        path,
		Name:        manifest.Recipe.Name,
		Description: manifest.Recipe.Description,
		Author:      manifest.Recipe.Author,
		Schema:      manifest.Recipe.Schema,
		Created:     manifest.Created,
		Digest:      digest,
		TotalSize:   size,
		Tools:       tools,
		Files:       files,
	}, nil
}

func fileDigestAndSize(path string) (string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), int64(len(data)), nil
}

func toolIDFromRecipePath(path string) string {
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return ""
}
