package projectmeta

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

var gitCommand = func(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}

func CollectClaudeProjects(entries []adapters.FileEntry, sourceHome string) []backup.ProjectManifest {
	byKey := map[string][]string{}
	for _, entry := range entries {
		key, ok := ClaudeProjectKeyFromArchivePath(entry.InArchive)
		if !ok || entry.Category != adapters.CategoryMemory {
			continue
		}
		byKey[key] = append(byKey[key], entry.InArchive)
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	projects := make([]backup.ProjectManifest, 0, len(keys))
	for _, key := range keys {
		sourcePath := ClaudeProjectKeyToSourcePath(key)
		rel, scope := relativeToHome(sourcePath, sourceHome)
		files := append([]string{}, byKey[key]...)
		sort.Strings(files)
		projects = append(projects, backup.ProjectManifest{
			ToolID:         "claude-code",
			ProjectKey:     key,
			SourcePath:     sourcePath,
			SourceHome:     sourceHome,
			RelativeToHome: rel,
			PathScope:      scope,
			MemoryFiles:    files,
			Git:            ReadGitMetadata(sourcePath),
		})
	}
	return projects
}

func ClaudeProjectKeyFromArchivePath(path string) (string, bool) {
	if strings.HasPrefix(path, "claude-code/projects/") {
		rel := strings.TrimPrefix(path, "claude-code/projects/")
		parts := strings.SplitN(rel, "/", 2)
		return parts[0], len(parts) == 2
	}
	if strings.HasPrefix(path, "claude-code/memory/") {
		rel := strings.TrimPrefix(path, "claude-code/memory/")
		parts := strings.SplitN(rel, "/", 2)
		return parts[0], len(parts) == 2
	}
	return "", false
}

func ClaudeProjectKeyToSourcePath(key string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(key), "-")
	if trimmed == "" {
		return ""
	}
	return string(os.PathSeparator) + strings.ReplaceAll(trimmed, "-", string(os.PathSeparator))
}

func relativeToHome(sourcePath, sourceHome string) (string, string) {
	if sourcePath == "" || sourceHome == "" {
		return "", backup.ProjectPathScopeOutsideHome
	}
	rel, err := filepath.Rel(sourceHome, sourcePath)
	if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." {
		return filepath.ToSlash(rel), backup.ProjectPathScopeHome
	}
	return "", backup.ProjectPathScopeOutsideHome
}

func ReadGitMetadata(projectPath string) backup.ProjectGitMetadata {
	git := backup.ProjectGitMetadata{}
	if projectPath == "" {
		return git
	}
	root, ok := runGitOutput(projectPath, "rev-parse", "--show-toplevel")
	if !ok {
		return git
	}
	git.IsRepo = true
	git.WorktreeRoot = root
	if branch, ok := runGitOutput(projectPath, "branch", "--show-current"); ok {
		git.Branch = branch
	}
	if head, ok := runGitOutput(projectPath, "rev-parse", "--short", "HEAD"); ok {
		git.Head = head
	}
	if status, ok := runGitOutput(projectPath, "status", "--porcelain"); ok {
		git.Dirty = status != ""
	}
	if remotes, ok := runGitOutput(projectPath, "remote", "-v"); ok {
		git.Remotes = parseGitRemotes(remotes)
	}
	return git
}

func runGitOutput(dir string, args ...string) (string, bool) {
	cmd := gitCommand(dir, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func parseGitRemotes(output string) []backup.ProjectGitRemote {
	seen := map[string]struct{}{}
	var remotes []backup.ProjectGitRemote
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.Contains(line, "(fetch)") {
			continue
		}
		key := fields[0] + "\x00" + fields[1]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		remotes = append(remotes, SanitizeRemote(fields[0], fields[1]))
	}
	sort.Slice(remotes, func(i, j int) bool {
		cmp := bytes.Compare([]byte(remotes[i].Name), []byte(remotes[j].Name))
		if cmp == 0 {
			return remotes[i].URL < remotes[j].URL
		}
		return cmp < 0
	})
	return remotes
}
