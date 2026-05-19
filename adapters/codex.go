package adapters

import (
	"os"
	"path/filepath"
	"strings"
)

type CodexAdapter struct {
	home string
}

func NewCodexAdapter() *CodexAdapter {
	home, _ := os.UserHomeDir()
	return &CodexAdapter{home: home}
}

func (a *CodexAdapter) ensureHome() {
	if a.home == "" {
		a.home, _ = os.UserHomeDir()
	}
}

func (a *CodexAdapter) Name() string { return "Codex CLI" }
func (a *CodexAdapter) ID() string   { return "codex-cli" }

func (a *CodexAdapter) Detect() bool {
	a.ensureHome()
	_, err := os.Stat(filepath.Join(a.home, ".codex", "config.toml"))
	return err == nil
}

func (a *CodexAdapter) baseDir() string {
	a.ensureHome()
	return filepath.Join(a.home, ".codex")
}

func (a *CodexAdapter) ListFiles(opts ExportOpts) []FileEntry {
	var entries []FileEntry
	base := a.baseDir()

	// 1. Config files
	configFiles := []string{"config.toml", "AGENTS.md", "CODEX.md"}
	for _, name := range configFiles {
		path := filepath.Join(base, name)
		if info, err := os.Stat(path); err == nil {
			entries = append(entries, FileEntry{
				SourcePath: path,
				InArchive:  "codex-cli/config/" + name,
				Category:   CategoryConfig,
				Size:       info.Size(),
			})
		}
	}

	// 2. Custom agents
	agentsDir := filepath.Join(base, "agents")
	if dirEntries, err := os.ReadDir(agentsDir); err == nil {
		for _, de := range dirEntries {
			if de.IsDir() {
				continue
			}
			path := filepath.Join(agentsDir, de.Name())
			if info, err := os.Stat(path); err == nil {
				entries = append(entries, FileEntry{
					SourcePath: path,
					InArchive:  "codex-cli/agents/" + de.Name(),
					Category:   CategorySkills,
					Size:       info.Size(),
				})
			}
		}
	}

	// 3. Skills
	skillsDir := filepath.Join(base, "skills")
	entries = append(entries, a.walkDir(skillsDir, "codex-cli/skills", CategorySkills)...)

	// 4. Sessions (optional)
	if opts.WithHistory {
		sessionsDir := filepath.Join(base, "sessions")
		entries = append(entries, a.walkDir(sessionsDir, "codex-cli/sessions", CategoryHistory)...)
	}

	return entries
}

func (a *CodexAdapter) walkDir(dir, archivePrefix, category string) []FileEntry {
	var entries []FileEntry
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
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

func (a *CodexAdapter) Status() ToolStatus {
	base := a.baseDir()
	s := ToolStatus{
		Name: "Codex CLI",
		Path: base,
	}

	agentsDir := filepath.Join(base, "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				s.Agents++
			}
		}
	}

	skillsDir := filepath.Join(base, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		s.Skills = len(entries)
	}

	sessionsDir := filepath.Join(base, "sessions")
	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			s.Sessions++
		}
		return nil
	})

	s.ConfigFile = filepath.Join(base, "config.toml")

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

func (a *CodexAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error {
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

func (a *CodexAdapter) adaptPath(archivePath string) string {
	rel := strings.TrimPrefix(archivePath, "codex-cli/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) < 2 {
		return filepath.Join(a.baseDir(), rel)
	}

	switch parts[0] {
	case "config":
		return filepath.Join(a.baseDir(), parts[1])
	case "agents":
		return filepath.Join(a.baseDir(), "agents", parts[1])
	case "skills":
		return filepath.Join(a.baseDir(), "skills", parts[1])
	case "sessions":
		return filepath.Join(a.baseDir(), "sessions", parts[1])
	default:
		return filepath.Join(a.baseDir(), rel)
	}
}
