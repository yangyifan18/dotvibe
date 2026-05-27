package projectmeta

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangyifan18/dotvibe/backup"
)

func TestBuildRelocationPlansMissingHomeScopedProjectHasClonePlan(t *testing.T) {
	destHome := filepath.Join(t.TempDir(), "youtopia")
	projects := []backup.ProjectManifest{{
		ToolID:         "claude-code",
		ProjectKey:     "-Users-young-Softwares-dotvibe",
		SourcePath:     "/Users/young/Softwares/dotvibe",
		SourceHome:     "/Users/young",
		RelativeToHome: "Softwares/dotvibe",
		PathScope:      backup.ProjectPathScopeHome,
		Git: backup.ProjectGitMetadata{IsRepo: true, Remotes: []backup.ProjectGitRemote{{
			Name: "origin", URL: "git@github.com:yangyifan18/dotvibe.git", Sanitized: true, Cloneable: true,
		}}},
	}}
	plans := BuildRelocationPlans(RelocationOptions{Projects: projects, DestinationHome: destHome, DestinationUser: "youtopia", EnableHomeRemap: true})
	if len(plans) != 1 {
		t.Fatalf("plans = %#v", plans)
	}
	plan := plans[0]
	wantTarget := filepath.Join(destHome, "Softwares", "dotvibe")
	if plan.TargetPath != wantTarget || plan.TargetProjectKey != ClaudePathToProjectKey(wantTarget) || plan.TargetPathStatus != TargetPathMissing {
		t.Fatalf("plan = %#v", plan)
	}
	if !plan.Clone.Recommended || len(plan.Clone.Command) != 4 || plan.Clone.Command[0] != "git" {
		t.Fatalf("clone = %#v", plan.Clone)
	}
}

func TestBuildRelocationPlansOutsideHomeNeedsUserTargetRoot(t *testing.T) {
	plans := BuildRelocationPlans(RelocationOptions{
		Projects:        []backup.ProjectManifest{{ToolID: "claude-code", ProjectKey: "-Volumes-Work-dotvibe", SourcePath: "/Volumes/Work/dotvibe", PathScope: backup.ProjectPathScopeOutsideHome}},
		DestinationHome: "/Users/youtopia",
		EnableHomeRemap: true,
	})
	if len(plans) != 1 || plans[0].RemapStrategy != RemapNeedsUserTargetRoot || plans[0].RecommendedNextAction != ActionAskTargetPath {
		t.Fatalf("plans = %#v", plans)
	}
}

func TestBuildRelocationPlansExistingTargetMemoryConflict(t *testing.T) {
	destHome := t.TempDir()
	targetPath := filepath.Join(destHome, "Softwares", "dotvibe")
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatal(err)
	}
	targetKey := ClaudePathToProjectKey(targetPath)
	memory := filepath.Join(destHome, ".claude", "projects", targetKey, "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(memory), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memory, []byte("local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	plans := BuildRelocationPlans(RelocationOptions{
		Projects:        []backup.ProjectManifest{{ToolID: "claude-code", ProjectKey: "-Users-young-Softwares-dotvibe", RelativeToHome: "Softwares/dotvibe", PathScope: backup.ProjectPathScopeHome}},
		DestinationHome: destHome,
		EnableHomeRemap: true,
	})
	if plans[0].TargetMemoryStatus != TargetMemoryExists || plans[0].RecommendedNextAction != ActionStageMemoryReview {
		t.Fatalf("plan = %#v", plans[0])
	}
}

func TestBuildRelocationPlansFlagsRemoteMismatchForExistingRepo(t *testing.T) {
	destHome := t.TempDir()
	targetPath := filepath.Join(destHome, "Softwares", "dotvibe")
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatal(err)
	}
	runGitForRelocationTest(t, targetPath, "init")
	runGitForRelocationTest(t, targetPath, "remote", "add", "origin", "git@github.com:someone/fork.git")
	plans := BuildRelocationPlans(RelocationOptions{
		Projects: []backup.ProjectManifest{{
			ToolID:         "claude-code",
			ProjectKey:     "-Users-young-Softwares-dotvibe",
			RelativeToHome: "Softwares/dotvibe",
			PathScope:      backup.ProjectPathScopeHome,
			Git: backup.ProjectGitMetadata{IsRepo: true, Remotes: []backup.ProjectGitRemote{{
				Name: "origin", URL: "git@github.com:yangyifan18/dotvibe.git", Sanitized: true, Cloneable: true,
			}}},
		}},
		DestinationHome: destHome,
		EnableHomeRemap: true,
	})
	if len(plans) != 1 || !plans[0].AssociationReviewRequired || plans[0].RecommendedNextAction != ActionAssociationReviewRequired {
		t.Fatalf("plans = %#v", plans)
	}
}

func runGitForRelocationTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := gitCommand(dir, args...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
