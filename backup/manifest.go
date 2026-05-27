package backup

import (
	"encoding/json"
	"os"
	"time"
)

const (
	ArchiveKindFull        = "full"
	ArchiveKindIncremental = "incremental"
	ArchiveKindRecipe      = "recipe"

	FileStorageInline = "inline"
	FileStorageBase   = "base"

	RecipeSchemaV1 = "dotvibe.recipe.v1"
)

type Manifest struct {
	Version       string                  `json:"version"`
	FormatVersion int                     `json:"format_version,omitempty"`
	ArchiveKind   string                  `json:"archive_kind,omitempty"`
	Created       time.Time               `json:"created"`
	Hostname      string                  `json:"hostname"`
	SourceHome    string                  `json:"source_home,omitempty"`
	SourceUser    string                  `json:"source_user,omitempty"`
	Base          *BaseArchiveRef         `json:"base,omitempty"`
	Recipe        *RecipeMetadata         `json:"recipe,omitempty"`
	Tools         map[string]ToolManifest `json:"tools"`
	Files         []FileManifest          `json:"files,omitempty"`
	Projects      []ProjectManifest       `json:"projects,omitempty"`
}

const (
	ProjectPathScopeHome        = "home"
	ProjectPathScopeOutsideHome = "outside_home"
)

type ProjectManifest struct {
	ToolID         string             `json:"tool_id"`
	ProjectKey     string             `json:"project_key"`
	SourcePath     string             `json:"source_path,omitempty"`
	SourceHome     string             `json:"source_home,omitempty"`
	RelativeToHome string             `json:"relative_to_home,omitempty"`
	PathScope      string             `json:"path_scope,omitempty"`
	MemoryFiles    []string           `json:"memory_files,omitempty"`
	Git            ProjectGitMetadata `json:"git,omitempty"`
}

type ProjectGitMetadata struct {
	IsRepo       bool               `json:"is_repo"`
	WorktreeRoot string             `json:"worktree_root,omitempty"`
	Branch       string             `json:"branch,omitempty"`
	Head         string             `json:"head,omitempty"`
	Dirty        bool               `json:"dirty,omitempty"`
	Remotes      []ProjectGitRemote `json:"remotes,omitempty"`
}

type ProjectGitRemote struct {
	Name                string `json:"name"`
	URL                 string `json:"url"`
	Sanitized           bool   `json:"sanitized"`
	Cloneable           bool   `json:"cloneable"`
	CredentialsRedacted bool   `json:"credentials_redacted,omitempty"`
	Reason              string `json:"reason,omitempty"`
}

type RecipeMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Schema      string   `json:"schema"`
	SharePolicy string   `json:"share_policy"`
	SourceTools []string `json:"source_tools,omitempty"`
}

type BaseArchiveRef struct {
	FileName       string    `json:"file_name"`
	Created        time.Time `json:"created"`
	ManifestSHA256 string    `json:"manifest_sha256"`
}

type ToolManifest struct {
	Version      string   `json:"version,omitempty"`
	Included     []string `json:"included"`
	ProjectCount int      `json:"project_count,omitempty"`
	FileCount    int      `json:"file_count,omitempty"`
	AgentCount   int      `json:"agent_count,omitempty"`
}

type FileManifest struct {
	Path       string `json:"path"`
	ToolID     string `json:"tool_id,omitempty"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
	Category   string `json:"category,omitempty"`
	Storage    string `json:"storage,omitempty"`
	StoredPath string `json:"stored_path,omitempty"`
}

func (m *Manifest) Normalize() {
	if m.FormatVersion == 0 {
		m.FormatVersion = 1
	}
	if m.ArchiveKind == "" {
		m.ArchiveKind = ArchiveKindFull
	}
	for i := range m.Files {
		if m.Files[i].Storage == "" {
			m.Files[i].Storage = FileStorageInline
		}
		if m.Files[i].StoredPath == "" && m.Files[i].Storage == FileStorageInline {
			m.Files[i].StoredPath = m.Files[i].Path
		}
		if m.Files[i].ToolID == "" {
			m.Files[i].ToolID = toolIDFromArchivePath(m.Files[i].Path)
		}
	}
}

func toolIDFromArchivePath(path string) string {
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return ""
}

func WriteManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	m.Normalize()
	return &m, nil
}
