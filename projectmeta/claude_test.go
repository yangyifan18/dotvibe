package projectmeta

import (
	"os"
	"testing"

	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/backup"
)

func TestClaudeProjectKeyToSourcePath(t *testing.T) {
	got := ClaudeProjectKeyToSourcePath("-Users-young-Softwares-dotvibe")
	if got != "/Users/young/Softwares/dotvibe" {
		t.Fatalf("path = %q", got)
	}
}

func TestCollectClaudeProjectsGroupsMemoryFilesAndHomeRelativePath(t *testing.T) {
	entries := []adapters.FileEntry{
		{InArchive: "claude-code/projects/-Users-young-Softwares-dotvibe/CLAUDE.md", Category: adapters.CategoryMemory},
		{InArchive: "claude-code/projects/-Users-young-Softwares-dotvibe/memory/MEMORY.md", Category: adapters.CategoryMemory},
		{InArchive: "claude-code/config/settings.json", Category: adapters.CategoryConfig},
	}
	projects := CollectClaudeProjects(entries, "/Users/young")
	if len(projects) != 1 {
		t.Fatalf("projects = %#v", projects)
	}
	p := projects[0]
	if p.ProjectKey != "-Users-young-Softwares-dotvibe" || p.SourcePath != "/Users/young/Softwares/dotvibe" || p.RelativeToHome != "Softwares/dotvibe" || p.PathScope != backup.ProjectPathScopeHome {
		t.Fatalf("project = %#v", p)
	}
	if len(p.MemoryFiles) != 2 {
		t.Fatalf("memory files = %#v", p.MemoryFiles)
	}
}

func TestReadGitMetadataSanitizesRemotes(t *testing.T) {
	repo := t.TempDir()
	runGitForProjectMetaTest(t, repo, "init")
	runGitForProjectMetaTest(t, repo, "remote", "add", "origin", "https://user:token@github.com/yangyifan18/dotvibe.git")
	git := ReadGitMetadata(repo)
	if !git.IsRepo || len(git.Remotes) != 1 {
		t.Fatalf("git = %#v", git)
	}
	if git.Remotes[0].URL != "https://github.com/yangyifan18/dotvibe.git" || !git.Remotes[0].CredentialsRedacted {
		t.Fatalf("remote = %#v", git.Remotes[0])
	}
}

func runGitForProjectMetaTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := gitCommand(dir, args...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
