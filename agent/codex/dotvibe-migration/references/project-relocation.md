# Project Relocation Workflow

Use this when `dotvibe agent import-plan <archive> --json` reports `project_relocations`, or when the user mentions a changed username, changed home directory, missing project checkout, clone, repo mismatch, or loading old project memory.

## Rules

- Treat project metadata as private backup metadata.
- Use `project_relocations` from the JSON import plan as the source of truth.
- If per-project relocation safety conflicts with top-level `recommended_next_action`, choose the safer per-project action.
- Do not run `git clone` until the user confirms the sanitized remote and target path.
- Do not clone into an existing non-empty directory.
- Do not paste full private memory into chat by default.
- If `association_review_required=true`, dispatch a Project Association Review subagent before offering direct memory import.
- Direct memory import is allowed only for high-confidence `same_project` and only when local project memory is missing.
- Existing local memory always requires `dotvibe import --stage` and a final diff before writing.

## User Summary

Summarize relocations as choices, not raw JSON. Use this style:

```text
我发现 3 个项目记忆：

1. dotvibe：目标项目不存在，可从 sanitized remote clone 后导入
2. fastrelay：目标项目存在，但已有本地 memory，需要先 stage review
3. medicine：目标 repo remote 不一致，需要项目关联 review
```

For each project, include only concise fields:

- old source path
- planned target path
- target path status
- local memory status
- clone availability
- association review requirement

## Missing Project Flow

1. Summarize source path, target path, sanitized remote, and clone command.
2. Ask whether to clone now, choose another target path, or skip this project.
3. If the user confirms clone, run the command array exactly as shown by dotvibe.
4. Rerun `dotvibe agent import-plan <archive> --json` after clone.
5. Continue with memory load or stage flow.

Do not infer clone URLs when dotvibe did not provide a clone command. Ask the user for a repository URL or skip the project.

## Existing Project Flow

- If target memory is missing, ask whether to load old Mac memory.
- If target memory exists, run `dotvibe import <archive> --remap-home --stage --stage-dir <dir>` or use the explicit `--project-target` suggested by the plan.
- Summarize staged local and archive copies, then use `memory-merge.md` for review.
- Keep local memory by default when the association is uncertain.

## Project Association Review Subagent

When remotes mismatch or association is uncertain, dispatch a subagent with:

- Source path, source project key, sanitized source remotes, branch/head, and memory file names.
- Destination project path.
- Read-only destination facts: `git remote -v`, `git rev-parse --show-toplevel`, current branch, and lightweight identity files such as `go.mod`, `package.json`, `pyproject.toml`, or `README.md`.

Require this structured response:

```json
{
  "verdict": "same_project",
  "confidence": 0.93,
  "evidence": [
    "origin remote matches source origin",
    "repository basename is dotvibe",
    "go.mod module matches github.com/yangyifan18/dotvibe"
  ],
  "safe_action": "direct_import_allowed"
}
```

Allowed verdicts: `same_project`, `likely_fork`, `unrelated`, `uncertain`.
Allowed safe actions: `direct_import_allowed`, `stage_only`, `skip_recommended`.

Policy:

- `same_project` with high confidence may allow direct memory import when local memory is missing.
- `likely_fork` must use staging and user confirmation.
- `uncertain` must use staging or skip.
- `unrelated` defaults to skip.
- Show the user the verdict and concise evidence; do not ask the user to manually inspect remotes.
