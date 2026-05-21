package adapters

import (
	"errors"
	"fmt"
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
	paths := []string{
		filepath.Join(a.home, ".claude", "settings.json"),
		filepath.Join(a.home, ".claude", "CLAUDE.md"),
		filepath.Join(a.home, ".claude", "skills"),
		filepath.Join(a.home, ".claude", "agents"),
		filepath.Join(a.home, ".claude", "commands"),
		filepath.Join(a.home, ".claude", "plugins"),
		filepath.Join(a.home, ".claude", "projects"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
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

			rootMemory := filepath.Join(projectsDir, proj.Name(), "MEMORY.md")
			if info, err := os.Stat(rootMemory); err == nil {
				entries = append(entries, FileEntry{
					SourcePath: rootMemory,
					InArchive:  projectPrefix + "/MEMORY.md",
					Category:   CategoryMemory,
					Size:       info.Size(),
				})
			}

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

func (a *ClaudeAdapter) ListRecipeFiles(opts RecipeOpts) []FileEntry {
	var entries []FileEntry
	base := a.baseDir()

	if opts.IncludeSettings {
		settings := filepath.Join(base, "settings.json")
		if info, err := os.Stat(settings); err == nil {
			entries = append(entries, FileEntry{
				SourcePath: settings,
				InArchive:  "claude-code/config/settings.json",
				Category:   CategorySettings,
				Size:       info.Size(),
			})
		}
	}

	globalRule := filepath.Join(base, "CLAUDE.md")
	if info, err := os.Stat(globalRule); err == nil {
		entries = append(entries, FileEntry{
			SourcePath: globalRule,
			InArchive:  "claude-code/rules/CLAUDE.md",
			Category:   CategoryRules,
			Size:       info.Size(),
		})
	}

	entries = append(entries, a.walkDir(filepath.Join(base, "skills"), "claude-code/skills", CategorySkills)...)
	entries = append(entries, a.walkDir(filepath.Join(base, "agents"), "claude-code/agents", CategoryAgents)...)
	entries = append(entries, a.walkDir(filepath.Join(base, "commands"), "claude-code/commands", CategoryCommands)...)
	entries = append(entries, a.walkDir(filepath.Join(base, "plugins"), "claude-code/plugins", CategorySkills)...)
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

func (a *ClaudeAdapter) FilterRestoreEntries(entries []FileEntry, opts RestoreOpts) []FileEntry {
	return FilterProjectEntries(entries, opts.Project)
}

func (a *ClaudeAdapter) PlanRestore(entries []FileEntry, opts RestoreOpts) ([]RestorePlanEntry, error) {
	entries = a.FilterRestoreEntries(entries, opts)
	plans := make([]RestorePlanEntry, 0, len(entries))
	for _, entry := range entries {
		destPath, err := a.adaptPath(entry.InArchive)
		if err != nil {
			return nil, err
		}
		plan := RestorePlanEntry{FileEntry: entry, TargetPath: destPath, Action: RestoreWrite, Reason: "new file"}
		if _, err := os.Stat(destPath); err == nil {
			if opts.Force {
				plan.Action = RestoreOverwrite
				plan.Reason = "target exists and --force is set"
			} else {
				plan.Action = RestoreSkip
				plan.Reason = "target exists; use --force to overwrite"
			}
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func (a *ClaudeAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) (RestoreSummary, error) {
	plans, err := a.PlanRestore(entries, opts)
	if err != nil {
		return RestoreSummary{}, err
	}

	var summary RestoreSummary
	var errs []error
	for _, plan := range plans {
		if plan.Action == RestoreSkip {
			summary.Skipped++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(plan.TargetPath), 0755); err != nil {
			summary.Failed++
			errs = append(errs, err)
			continue
		}

		srcData, err := os.ReadFile(filepath.Join(archiveDir, plan.InArchive))
		if err != nil {
			summary.Failed++
			errs = append(errs, err)
			continue
		}

		if err := os.WriteFile(plan.TargetPath, srcData, 0644); err != nil {
			summary.Failed++
			errs = append(errs, err)
			continue
		}
		if plan.Action == RestoreOverwrite {
			summary.Overwritten++
		} else {
			summary.Written++
		}
	}
	return summary, errors.Join(errs...)
}

func (a *ClaudeAdapter) adaptPath(archivePath string) (string, error) {
	if !strings.HasPrefix(archivePath, "claude-code/") {
		return "", fmt.Errorf("unsupported Claude archive path: %s", archivePath)
	}
	rel := strings.TrimPrefix(archivePath, "claude-code/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("unsupported Claude archive path: %s", archivePath)
	}

	switch parts[0] {
	case "config":
		return filepath.Join(a.baseDir(), parts[1]), nil
	case "projects":
		subparts := strings.SplitN(parts[1], "/", 2)
		if len(subparts) < 2 {
			return "", fmt.Errorf("unsupported Claude project archive path: %s", archivePath)
		}
		return filepath.Join(a.baseDir(), "projects", subparts[0], subparts[1]), nil
	case "memory":
		subparts := strings.SplitN(parts[1], "/", 2)
		if len(subparts) < 2 {
			return "", fmt.Errorf("unsupported legacy Claude memory archive path: %s", archivePath)
		}
		if subparts[1] == "CLAUDE.md" {
			return filepath.Join(a.baseDir(), "projects", subparts[0], "CLAUDE.md"), nil
		}
		return filepath.Join(a.baseDir(), "projects", subparts[0], "memory", subparts[1]), nil
	case "skills":
		return filepath.Join(a.baseDir(), "skills", parts[1]), nil
	case "plugins":
		return filepath.Join(a.baseDir(), "plugins", parts[1]), nil
	case "transcripts":
		return filepath.Join(a.baseDir(), "transcripts", parts[1]), nil
	default:
		return "", fmt.Errorf("unsupported Claude archive path: %s", archivePath)
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
