# Natural Language Intent Router

Use this first when the user asks for dotvibe help in natural language. The user does not need to name this skill or list dotvibe commands.

## Core Rule

Classify the user's intent, then enter the matching standard workflow. Run JSON planning commands internally and expose only choices, summaries, and confirmation gates to the user.

## Intent Map

| Intent | User phrases that should match | Next reference |
|---|---|---|
| `source_export` | `调用 dotvibe 导出记忆`, `备份这台 Mac 的 vibe coding 记忆`, `帮我备份 Claude Code / Codex / OpenCode`, `export my coding agent memories`, `use dotvibe to export memories` | `source-export.md` |
| `destination_import` | `调用 dotvibe 导入旧 Mac 的记忆`, `导入旧 Mac 备份`, `恢复 dotvibe 备份`, `import old Mac memories`, `restore this dotvibe backup on my new Mac` | `destination-import.md` |
| `recipe_export` | `导出一个可以分享的 vibe recipe`, `把我的 agents/skills 打包分享给同事`, `create a shareable .vibe recipe` | `source-export.md` with recipe profile |
| `recipe_apply` | `应用这个 .vibe recipe`, `安装同事分享的 recipe`, `apply this vibe recipe` | `safety.md`, then recipe inspect/lint/diff/apply flow |
| `project_relocation` | `新 Mac 用户名变了`, `项目还没 clone`, `把旧项目 memory 放到新路径`, `load old project memory`, `changed home path` | `project-relocation.md` after `dotvibe agent import-plan <archive> --json` |
| `memory_merge` | `帮我合并 project memory`, `review staged memory`, `merge dotvibe-stage-review` | `memory-merge.md` |
| `status_help` | `dotvibe 能迁移什么`, `看看这台机器有什么可备份`, `what can dotvibe migrate` | Run `dotvibe agent doctor --json` and `dotvibe agent inventory --json`, then summarize |

## Precedence

When multiple intents match, choose the most specific safe workflow:

1. If a `.vibe` file is mentioned with apply/install language, use `recipe_apply`.
2. If an archive path is mentioned with import/restore language, use `destination_import`.
3. If changed username, changed home, missing clone, or project memory appears during import, use `project_relocation` after import planning.
4. If staged files or a stage directory are mentioned, use `memory_merge`.
5. If export/backup/share language appears, use `source_export` or `recipe_export`.
6. If the intent is still unclear, run `dotvibe agent doctor --json`, then ask one choice question: export, import, recipe, or status.

## Response Style

Do not start with command instructions. Start with a brief reassurance and then inspect state.

Source export opening:

```text
我会先检查这台 Mac 上有哪些可迁移的工具和记忆；这一步不会写入、上传或删除任何东西。
```

Destination import opening:

```text
我会先检查这台新 Mac 的工具状态和备份内容；不会直接写入任何 agent 目录。
```

## Internal Command Policy

Use these commands for state and decisions:

```bash
dotvibe agent doctor --json
dotvibe agent inventory --json
dotvibe agent export-plan --profile <profile> --json
dotvibe agent import-plan <archive> --json
```

Use human-output commands only for final user-facing verification when no JSON alternative exists, such as `dotvibe list <archive>` after a private backup export or `dotvibe recipe inspect <recipe>` after a recipe export.

## User Choice Policy

Ask users choices, not raw flags. Translate choices into commands internally:

| User choice | Internal command shape |
|---|---|
| Preview import only | `dotvibe import <archive> --dry-run` |
| Stage import review | `dotvibe import <archive> --stage --stage-dir <dir>` |
| New Mac username/home remap | add `--remap-home` |
| Explicit project destination | add `--project-target <source-key>=<target-path>` |
| Selected Claude project | add `--project <source-project>` |
| Confirmed direct import | add `--yes` |
| Confirmed overwrite after backup/stage | add `--force --yes` |

Never ask the user to remember or choose flags directly unless they ask for command-level detail.
