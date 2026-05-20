package adapters

import (
	"os"
	"path/filepath"
	"strings"
)

type ClaudeAdapter struct {
	home string
}

func NewClaudeAdapter() *ClaudeAdapter {
	home, _ := os.UserHomeDir()
	return &ClaudeAdapter{home: home}
}

func (a *ClaudeAdapter) Name() string { return "Claude Code" }
func (a *ClaudeAdapter) ID() string   { return "claude-code" }

func (a *ClaudeAdapter) ensureHome() {
	if a.home == "" {
		a.home, _ = os.UserHomeDir()
	}
}

func (a *ClaudeAdapter) Detect() bool {
	a.ensureHome()
	_, err := os.Stat(filepath.Join(a.home, ".claude", "settings.json"))
	return err == nil
}

func (a *ClaudeAdapter) baseDir() string {
	a.ensureHome()
	return filepath.Join(a.home, ".claude")
}

func (a *ClaudeAdapter) ListFiles(opts ExportOpts) []FileEntry {
	var entries []FileEntry
	base := a.baseDir()

	// 1. Config files
	configFiles := []string{"settings.json"}
	for _, name := range configFiles {
		path := filepath.Join(base, name)
		if info, err := os.Stat(path); err == nil {
			entries = append(entries, FileEntry{
				SourcePath: path,
				InArchive:  "claude-code/config/" + name,
				Category:   CategoryConfig,
				Size:       info.Size(),
			})
		}
	}

	// 2. Project memory directories
	projectsDir := filepath.Join(base, "projects")
	if dirEntries, err := os.ReadDir(projectsDir); err == nil {
		for _, proj := range dirEntries {
			if !proj.IsDir() {
				continue
			}
			projectPrefix := "claude-code/projects/" + proj.Name()
			memDir := filepath.Join(projectsDir, proj.Name(), "memory")
			entries = append(entries, a.walkDir(memDir, projectPrefix+"/memory", CategoryMemory)...)

			claudeMd := filepath.Join(projectsDir, proj.Name(), "CLAUDE.md")
			if info, err := os.Stat(claudeMd); err == nil {
				entries = append(entries, FileEntry{
					SourcePath: claudeMd,
					InArchive:  projectPrefix + "/CLAUDE.md",
					Category:   CategoryMemory,
					Size:       info.Size(),
				})
			}
		}
	}

	// 3. Skills
	skillsDir := filepath.Join(base, "skills")
	entries = append(entries, a.walkDir(skillsDir, "claude-code/skills", CategorySkills)...)

	// 4. Plugins (config, not full install)
	pluginsDir := filepath.Join(base, "plugins")
	entries = append(entries, a.walkDir(pluginsDir, "claude-code/plugins", CategorySkills)...)

	// 5. History (optional)
	if opts.WithHistory {
		transcriptsDir := filepath.Join(base, "transcripts")
		entries = append(entries, a.walkDir(transcriptsDir, "claude-code/transcripts", CategoryHistory)...)

		historyFile := filepath.Join(base, "history.jsonl")
		if info, err := os.Stat(historyFile); err == nil {
			entries = append(entries, FileEntry{
				SourcePath: historyFile,
				InArchive:  "claude-code/history.jsonl",
				Category:   CategoryHistory,
				Size:       info.Size(),
			})
		}
	}

	return entries
}

func (a *ClaudeAdapter) walkDir(dir, archivePrefix, category string) []FileEntry {
	var entries []FileEntry

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(dir, path)
		archivePath := archivePrefix + "/" + filepath.ToSlash(rel)

		entries = append(entries, FileEntry{
			SourcePath: path,
			InArchive:  archivePath,
			Category:   category,
			Size:       info.Size(),
		})
		return nil
	})

	return entries
}

func (a *ClaudeAdapter) Status() ToolStatus {
	base := a.baseDir()
	s := ToolStatus{
		Name: "Claude Code",
		Path: base,
	}

	projectsDir := filepath.Join(base, "projects")
	if entries, err := os.ReadDir(projectsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				s.Projects++
			}
		}
	}

	skillsDir := filepath.Join(base, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		s.Skills = len(entries)
	}

	transcriptsDir := filepath.Join(base, "transcripts")
	if entries, err := os.ReadDir(transcriptsDir); err == nil {
		s.Sessions = len(entries)
	}

	s.ConfigFile = filepath.Join(base, "settings.json")

	var totalSize int64
	filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	s.Size = totalSize

	return s
}

func (a *ClaudeAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error {
	entries = FilterProjectEntries(entries, opts.Project)
	for _, entry := range entries {
		destPath := a.adaptPath(entry.InArchive)

		if _, err := os.Stat(destPath); err == nil && !opts.Force {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		srcData, err := os.ReadFile(filepath.Join(archiveDir, entry.InArchive))
		if err != nil {
			return err
		}

		if err := os.WriteFile(destPath, srcData, 0644); err != nil {
			return err
		}
	}
	return nil
}

func (a *ClaudeAdapter) adaptPath(archivePath string) string {
	rel := strings.TrimPrefix(archivePath, "claude-code/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) < 2 {
		return filepath.Join(a.baseDir(), rel)
	}

	switch parts[0] {
	case "config":
		return filepath.Join(a.baseDir(), parts[1])
	case "projects":
		subparts := strings.SplitN(parts[1], "/", 2)
		if len(subparts) < 2 {
			return filepath.Join(a.baseDir(), "projects", subparts[0])
		}
		return filepath.Join(a.baseDir(), "projects", subparts[0], subparts[1])
	case "memory":
		subparts := strings.SplitN(parts[1], "/", 2)
		if len(subparts) < 2 {
			return filepath.Join(a.baseDir(), "projects", subparts[0])
		}
		if subparts[1] == "CLAUDE.md" {
			return filepath.Join(a.baseDir(), "projects", subparts[0], "CLAUDE.md")
		}
		return filepath.Join(a.baseDir(), "projects", subparts[0], "memory", subparts[1])
	case "skills":
		return filepath.Join(a.baseDir(), "skills", parts[1])
	case "plugins":
		return filepath.Join(a.baseDir(), "plugins", parts[1])
	case "transcripts":
		return filepath.Join(a.baseDir(), "transcripts", parts[1])
	default:
		return filepath.Join(a.baseDir(), rel)
	}
}

func FilterProjectEntries(entries []FileEntry, project string) []FileEntry {
	if project == "" {
		return entries
	}
	key := ClaudeProjectKey(project)
	var filtered []FileEntry
	for _, entry := range entries {
		projectKey, ok := claudeArchiveProject(entry.InArchive)
		if ok && projectKey == key {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func ClaudeProjectKey(project string) string {
	project = strings.TrimSpace(project)
	if project == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			project = home
		}
	} else if strings.HasPrefix(project, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			project = filepath.Join(home, strings.TrimPrefix(project, "~/"))
		}
	}
	project = strings.Trim(project, string(os.PathSeparator))
	if strings.HasPrefix(project, "-") {
		return project
	}
	if project == "" {
		return project
	}
	parts := strings.FieldsFunc(project, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	return "-" + strings.Join(parts, "-")
}

func claudeArchiveProject(archivePath string) (string, bool) {
	if strings.HasPrefix(archivePath, "claude-code/projects/") {
		rel := strings.TrimPrefix(archivePath, "claude-code/projects/")
		parts := strings.SplitN(rel, "/", 2)
		return parts[0], len(parts) == 2
	}
	if strings.HasPrefix(archivePath, "claude-code/memory/") {
		rel := strings.TrimPrefix(archivePath, "claude-code/memory/")
		parts := strings.SplitN(rel, "/", 2)
		return parts[0], len(parts) == 2
	}
	return "", false
}
