package projectmeta

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yangyifan18/dotvibe/backup"
)

const (
	RemapHomePrefix          = "home-prefix"
	RemapExplicitTarget      = "explicit-target"
	RemapNeedsUserTargetRoot = "needs-user-target-root"

	TargetPathMissing = "missing"
	TargetPathExists  = "exists"
	TargetPathUnknown = "unknown"

	TargetMemoryMissing = "missing"
	TargetMemoryExists  = "exists"

	ActionAskTargetPath               = "ask-target-path"
	ActionConfirmCloneThenStageMemory = "confirm-clone-then-stage-memory"
	ActionAskLoadMemory               = "ask-load-memory"
	ActionStageMemoryReview           = "stage-memory-review"
	ActionAssociationReviewRequired   = "association-review-required"
	ActionNoProjectMetadata           = "no-project-metadata"
)

type DestinationInfo struct {
	User string `json:"user"`
	Home string `json:"home"`
}

type RelocationOptions struct {
	Projects        []backup.ProjectManifest
	DestinationHome string
	DestinationUser string
	EnableHomeRemap bool
	ProjectTargets  map[string]string
}

type ProjectRelocation struct {
	ToolID                    string    `json:"tool_id"`
	SourceProjectKey          string    `json:"source_project_key"`
	TargetProjectKey          string    `json:"target_project_key,omitempty"`
	SourcePath                string    `json:"source_path,omitempty"`
	TargetPath                string    `json:"target_path,omitempty"`
	RemapStrategy             string    `json:"remap_strategy"`
	TargetPathStatus          string    `json:"target_path_status"`
	TargetMemoryStatus        string    `json:"target_memory_status"`
	RemoteStatus              string    `json:"remote_status,omitempty"`
	AssociationReviewRequired bool      `json:"association_review_required"`
	Clone                     ClonePlan `json:"clone,omitempty"`
	RecommendedNextAction     string    `json:"recommended_next_action"`
}

type ClonePlan struct {
	Recommended bool     `json:"recommended"`
	Remote      string   `json:"remote,omitempty"`
	TargetPath  string   `json:"target_path,omitempty"`
	Command     []string `json:"command,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

func BuildRelocationPlans(opts RelocationOptions) []ProjectRelocation {
	plans := make([]ProjectRelocation, 0, len(opts.Projects))
	for _, project := range opts.Projects {
		plan := ProjectRelocation{
			ToolID:             project.ToolID,
			SourceProjectKey:   project.ProjectKey,
			SourcePath:         project.SourcePath,
			TargetPathStatus:   TargetPathUnknown,
			TargetMemoryStatus: TargetMemoryMissing,
		}
		plan.TargetPath, plan.RemapStrategy = targetPathForProject(project, opts)
		if plan.TargetPath == "" {
			plan.RecommendedNextAction = ActionAskTargetPath
			plans = append(plans, plan)
			continue
		}
		plan.TargetProjectKey = ClaudePathToProjectKey(plan.TargetPath)
		plan.TargetPathStatus = pathStatus(plan.TargetPath)
		plan.TargetMemoryStatus = claudeTargetMemoryStatus(opts.DestinationHome, plan.TargetProjectKey)
		plan.RemoteStatus = compareRemotes(project, plan.TargetPath)
		plan.AssociationReviewRequired = plan.RemoteStatus == "mismatch"
		plan.Clone = clonePlanForProject(project, plan.TargetPath, plan.TargetPathStatus)
		plan.RecommendedNextAction = recommendedRelocationAction(plan)
		plans = append(plans, plan)
	}
	return plans
}

func targetPathForProject(project backup.ProjectManifest, opts RelocationOptions) (string, string) {
	if target, ok := opts.ProjectTargets[project.ProjectKey]; ok && target != "" {
		return target, RemapExplicitTarget
	}
	if opts.EnableHomeRemap && project.PathScope == backup.ProjectPathScopeHome && project.RelativeToHome != "" && opts.DestinationHome != "" {
		return filepath.Join(opts.DestinationHome, filepath.FromSlash(project.RelativeToHome)), RemapHomePrefix
	}
	return "", RemapNeedsUserTargetRoot
}

func ClaudePathToProjectKey(path string) string {
	return "-" + strings.Join(pathSegments(path), "-")
}

func pathSegments(path string) []string {
	trimmed := strings.Trim(path, string(os.PathSeparator))
	if trimmed == "" {
		return nil
	}
	return strings.FieldsFunc(trimmed, func(r rune) bool { return r == '/' || r == '\\' })
}

func pathStatus(path string) string {
	if _, err := os.Stat(path); err == nil {
		return TargetPathExists
	}
	return TargetPathMissing
}

func claudeTargetMemoryStatus(destHome, targetKey string) string {
	if destHome == "" || targetKey == "" {
		return TargetMemoryMissing
	}
	base := filepath.Join(destHome, ".claude", "projects", targetKey)
	if _, err := os.Stat(filepath.Join(base, "CLAUDE.md")); err == nil {
		return TargetMemoryExists
	}
	if _, err := os.Stat(filepath.Join(base, "MEMORY.md")); err == nil {
		return TargetMemoryExists
	}
	if _, err := os.Stat(filepath.Join(base, "memory")); err == nil {
		return TargetMemoryExists
	}
	return TargetMemoryMissing
}

func clonePlanForProject(project backup.ProjectManifest, targetPath, targetStatus string) ClonePlan {
	remote := firstCloneableRemote(project.Git.Remotes)
	if targetStatus != TargetPathMissing {
		return ClonePlan{Recommended: false, Reason: "target-exists"}
	}
	if remote.URL == "" {
		return ClonePlan{Recommended: false, Reason: "no-cloneable-remote"}
	}
	return ClonePlan{Recommended: true, Remote: remote.URL, TargetPath: targetPath, Command: []string{"git", "clone", remote.URL, targetPath}}
}

func firstCloneableRemote(remotes []backup.ProjectGitRemote) backup.ProjectGitRemote {
	for _, remote := range remotes {
		if remote.Cloneable && remote.URL != "" {
			return remote
		}
	}
	return backup.ProjectGitRemote{}
}

func compareRemotes(project backup.ProjectManifest, targetPath string) string {
	if pathStatus(targetPath) != TargetPathExists || !project.Git.IsRepo {
		return "unknown"
	}
	targetGit := ReadGitMetadata(targetPath)
	if !targetGit.IsRepo {
		return "not-a-git-repo"
	}
	sourceURLs := map[string]struct{}{}
	for _, remote := range project.Git.Remotes {
		if remote.Cloneable && remote.URL != "" {
			sourceURLs[remote.URL] = struct{}{}
		}
	}
	for _, remote := range targetGit.Remotes {
		if _, ok := sourceURLs[remote.URL]; ok {
			return "match"
		}
	}
	if len(sourceURLs) > 0 && len(targetGit.Remotes) > 0 {
		return "mismatch"
	}
	return "unknown"
}

func recommendedRelocationAction(plan ProjectRelocation) string {
	if plan.AssociationReviewRequired {
		return ActionAssociationReviewRequired
	}
	if plan.TargetPath == "" {
		return ActionAskTargetPath
	}
	if plan.Clone.Recommended {
		return ActionConfirmCloneThenStageMemory
	}
	if plan.TargetMemoryStatus == TargetMemoryExists {
		return ActionStageMemoryReview
	}
	return ActionAskLoadMemory
}

func BuildProjectKeyRemaps(plans []ProjectRelocation) map[string]string {
	out := map[string]string{}
	for _, plan := range plans {
		if plan.SourceProjectKey != "" && plan.TargetProjectKey != "" && plan.SourceProjectKey != plan.TargetProjectKey {
			out[plan.SourceProjectKey] = plan.TargetProjectKey
		}
	}
	return out
}

func RemapClaudeArchivePath(path string, keyRemaps map[string]string) string {
	if len(keyRemaps) == 0 || !strings.HasPrefix(path, "claude-code/projects/") {
		return path
	}
	rel := strings.TrimPrefix(path, "claude-code/projects/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) != 2 {
		return path
	}
	if targetKey, ok := keyRemaps[parts[0]]; ok && targetKey != "" {
		return "claude-code/projects/" + targetKey + "/" + parts[1]
	}
	return path
}
