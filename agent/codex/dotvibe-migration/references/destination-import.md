# Destination Import Workflow

Use this on the new Mac when the user says something like `调用 dotvibe 导入旧 Mac 的记忆，备份在 <path>`, `恢复旧 Mac 的 dotvibe 备份`, or `import old Mac memories`.

## Opening

If the user did not provide an archive path, ask one concise question:

```text
备份文件在哪里？请给我本机路径，例如 `~/Downloads/dotvibe-2026-05-27.tar.gz`。
```

If the user provided a path, say:

```text
我会先检查这台新 Mac 的工具状态和备份内容；不会直接写入任何 agent 目录。
```

Then run:

```bash
dotvibe agent doctor --json
dotvibe agent import-plan <archive> --json
```

If required tools appear missing, run setup planning without install:

```bash
dotvibe setup <archive>
```

Ask before using `dotvibe setup --install` or any install command.

## Plan Summary

Summarize the JSON import plan in user terms:

- archive kind
- included tools
- files that would write
- identical files that will be skipped
- conflicts needing review
- unsupported paths that block direct import
- missing base archives, if any
- project relocations, if any
- safest next action

Do not ask the user to choose raw flags. Convert choices to commands internally.

## Choice Translation

| User-facing choice | Internal command |
|---|---|
| Just preview | `dotvibe import <archive> --dry-run` |
| Stage everything needing review | `dotvibe import <archive> --stage --stage-dir <dir>` |
| Use new Mac username/home remap | add `--remap-home` |
| Use a custom target path for one project | add `--project-target <source-key>=<target-path>` |
| Import one Claude project | add `--project <source-project>` |
| Direct import after confirmation | add `--yes` |
| Overwrite after backup or stage review | add `--force --yes` |

## Standard Import Decisions

- If there are unsupported paths, do not write. Explain the unsupported entries.
- If there are missing base archives, stop and ask for the required base archive path.
- If there are conflicts, prefer `dotvibe import <archive> --stage --stage-dir <dir>`.
- If project relocations exist, read `project-relocation.md` before asking import choices.
- If target project memory already exists, stage before writing.
- If the user asks for direct import, run `dotvibe import <archive> --dry-run` first.
- Before direct full-archive import, require a fresh destination-side backup or staged local copies unless the plan only writes new files into empty targets.

## Safe Write Flow

Only after explicit confirmation, run the selected write command.

Examples:

```bash
dotvibe import <archive> --stage --stage-dir dotvibe-stage-review
```

```bash
dotvibe import <archive> --remap-home --yes
```

```bash
dotvibe import <archive> --remap-home --project <source-project> --stage --stage-dir dotvibe-stage-project
```

Avoid `--force` unless the user has seen the conflict summary and has a fresh backup or staged local copies.

## After Import

Summarize:

- what was written or staged
- what was skipped
- what still needs review
- recovery instructions

For full archive imports, state that dotvibe currently relies on a pre-import backup, staged local copies, or manual recovery. For recipe apply flows, show `dotvibe rollback list` and the relevant apply id when available.
