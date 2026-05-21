package adapters

import (
	"errors"
	"fmt"
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
	paths := []string{
		filepath.Join(a.home, ".codex", "config.toml"),
		filepath.Join(a.home, ".codex", "AGENTS.md"),
		filepath.Join(a.home, ".codex", "CODEX.md"),
		filepath.Join(a.home, ".codex", "agents"),
		filepath.Join(a.home, ".codex", "skills"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
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

func (a *CodexAdapter) ListRecipeFiles(opts RecipeOpts) []FileEntry {
	var entries []FileEntry
	base := a.baseDir()

	if opts.IncludeSettings {
		config := filepath.Join(base, "config.toml")
		if info, err := os.Stat(config); err == nil {
			entries = append(entries, FileEntry{
				SourcePath: config,
				InArchive:  "codex-cli/config/config.toml",
				Category:   CategorySettings,
				Size:       info.Size(),
			})
		}
	}

	for _, name := range []string{"AGENTS.md", "CODEX.md"} {
		path := filepath.Join(base, name)
		if info, err := os.Stat(path); err == nil {
			entries = append(entries, FileEntry{
				SourcePath: path,
				InArchive:  "codex-cli/rules/" + name,
				Category:   CategoryRules,
				Size:       info.Size(),
			})
		}
	}

	entries = append(entries, a.walkDir(filepath.Join(base, "agents"), "codex-cli/agents", CategoryAgents)...)
	entries = append(entries, a.walkDir(filepath.Join(base, "skills"), "codex-cli/skills", CategorySkills)...)
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

func (a *CodexAdapter) FilterRestoreEntries(entries []FileEntry, opts RestoreOpts) []FileEntry {
	return entries
}

func (a *CodexAdapter) PlanRestore(entries []FileEntry, opts RestoreOpts) ([]RestorePlanEntry, error) {
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

func (a *CodexAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) (RestoreSummary, error) {
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

func (a *CodexAdapter) adaptPath(archivePath string) (string, error) {
	if !strings.HasPrefix(archivePath, "codex-cli/") {
		return "", fmt.Errorf("unsupported Codex archive path: %s", archivePath)
	}
	rel := strings.TrimPrefix(archivePath, "codex-cli/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("unsupported Codex archive path: %s", archivePath)
	}

	switch parts[0] {
	case "config":
		return filepath.Join(a.baseDir(), parts[1]), nil
	case "rules":
		return filepath.Join(a.baseDir(), parts[1]), nil
	case "agents":
		return filepath.Join(a.baseDir(), "agents", parts[1]), nil
	case "skills":
		return filepath.Join(a.baseDir(), "skills", parts[1]), nil
	case "sessions":
		return filepath.Join(a.baseDir(), "sessions", parts[1]), nil
	default:
		return "", fmt.Errorf("unsupported Codex archive path: %s", archivePath)
	}
}
