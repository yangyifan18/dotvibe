package adapters

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OpenCodeAdapter struct {
	home string
}

func NewOpenCodeAdapter() *OpenCodeAdapter {
	home, _ := os.UserHomeDir()
	return &OpenCodeAdapter{home: home}
}

func (a *OpenCodeAdapter) ensureHome() {
	if a.home == "" {
		a.home, _ = os.UserHomeDir()
	}
}

func (a *OpenCodeAdapter) Name() string { return "OpenCode" }
func (a *OpenCodeAdapter) ID() string   { return "opencode" }

func (a *OpenCodeAdapter) Detect() bool {
	a.ensureHome()
	paths := []string{
		filepath.Join(a.home, ".config", "opencode", "opencode.json"),
		filepath.Join(a.home, ".opencode", "oh-my-openagent.json"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func (a *OpenCodeAdapter) configDirs() []string {
	a.ensureHome()
	var dirs []string
	d1 := filepath.Join(a.home, ".config", "opencode")
	if _, err := os.Stat(d1); err == nil {
		dirs = append(dirs, d1)
	}
	d2 := filepath.Join(a.home, ".opencode")
	if _, err := os.Stat(d2); err == nil {
		dirs = append(dirs, d2)
	}
	return dirs
}

func (a *OpenCodeAdapter) ListFiles(opts ExportOpts) []FileEntry {
	var entries []FileEntry

	for _, dir := range a.configDirs() {
		rootName := "xdg-config"
		if filepath.Base(dir) == ".opencode" {
			rootName = "home-config"
		}
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			if strings.Contains(rel, "node_modules") || strings.Contains(rel, "cache") {
				return nil
			}

			archivePath := "opencode/" + rootName + "/" + filepath.ToSlash(rel)
			entries = append(entries, FileEntry{
				SourcePath: path,
				InArchive:  archivePath,
				Category:   CategoryConfig,
				Size:       info.Size(),
			})
			return nil
		})
	}

	return entries
}

func (a *OpenCodeAdapter) ListRecipeFiles(opts RecipeOpts) []FileEntry {
	if !opts.IncludeSettings {
		return nil
	}
	return a.ListFiles(ExportOpts{})
}

func (a *OpenCodeAdapter) Status() ToolStatus {
	s := ToolStatus{
		Name: "OpenCode",
	}

	dirs := a.configDirs()
	if len(dirs) > 0 {
		s.Path = dirs[0]
	}

	var totalSize int64
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
	}
	s.Size = totalSize

	s.ConfigFile = filepath.Join(a.home, ".config", "opencode", "opencode.json")
	if _, err := os.Stat(s.ConfigFile); err != nil {
		s.ConfigFile = filepath.Join(a.home, ".opencode", "oh-my-openagent.json")
	}

	return s
}

func (a *OpenCodeAdapter) FilterRestoreEntries(entries []FileEntry, opts RestoreOpts) []FileEntry {
	return entries
}

func (a *OpenCodeAdapter) PlanRestore(entries []FileEntry, opts RestoreOpts) ([]RestorePlanEntry, error) {
	a.ensureHome()
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

func (a *OpenCodeAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) (RestoreSummary, error) {
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

func (a *OpenCodeAdapter) adaptPath(archivePath string) (string, error) {
	switch {
	case strings.HasPrefix(archivePath, "opencode/xdg-config/"):
		rel := strings.TrimPrefix(archivePath, "opencode/xdg-config/")
		return filepath.Join(a.home, ".config", "opencode", rel), nil
	case strings.HasPrefix(archivePath, "opencode/home-config/"):
		rel := strings.TrimPrefix(archivePath, "opencode/home-config/")
		return filepath.Join(a.home, ".opencode", rel), nil
	case strings.HasPrefix(archivePath, "opencode/config/"):
		rel := strings.TrimPrefix(archivePath, "opencode/config/")
		return filepath.Join(a.home, ".config", "opencode", rel), nil
	default:
		return "", fmt.Errorf("unsupported OpenCode archive path: %s", archivePath)
	}
}
