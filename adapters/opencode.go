package adapters

import (
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
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			if strings.Contains(rel, "node_modules") || strings.Contains(rel, "cache") {
				return nil
			}

			archivePath := "opencode/config/" + filepath.ToSlash(rel)
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

func (a *OpenCodeAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error {
	a.ensureHome()
	for _, entry := range entries {
		rel := strings.TrimPrefix(entry.InArchive, "opencode/config/")
		destPath := filepath.Join(a.home, ".config", "opencode", rel)

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
