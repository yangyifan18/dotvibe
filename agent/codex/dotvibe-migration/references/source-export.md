# Source Export Workflow

Use this on the old Mac when the user says something like `调用 dotvibe 导出记忆`, `备份这台 Mac 的 vibe coding 记忆`, or `export my coding agent memories`.

## Opening

Say:

```text
我会先检查这台 Mac 上有哪些可迁移的工具和记忆；这一步不会写入、上传或删除任何东西。
```

Then run:

```bash
dotvibe agent doctor --json
dotvibe agent inventory --json
```

Stop if `doctor.ok=false`; summarize the issue and give the install/fix command if available.

## User Choices

After inventory, ask this choice question:

```text
你想导出哪一种？

1. 完整私有备份：Claude Code + Codex + OpenCode 配置和记忆
2. 项目 memory 备份：主要用于迁移 Claude Code 项目记忆
3. 团队 recipe：只分享 skills / agents / rules / safe settings，不含私人项目记忆
```

Defaults:

- If the user says `默认` or is unsure, choose option 1.
- Exclude history/sessions by default unless the user explicitly asks to include them.
- Include all detected tools by default.
- If the user chooses recipe, explain that recipes are intended to be shareable only after lint passes.

## Plan Before Write

Map choices to profiles:

| Choice | Export plan command |
|---|---|
| Full private backup | `dotvibe agent export-plan --profile full --json` |
| Project memory backup | `dotvibe agent export-plan --profile project-memory --json` |
| Team recipe | `dotvibe agent export-plan --profile recipe --json` |

If the user selects specific tools, add `--only <comma-separated-tools>` to the export-plan command.

Run the export-plan command and summarize:

- profile
- privacy risk
- planned archive path
- exact command that will write
- whether history/sessions are included

Ask for confirmation before running the write command from the plan. Do not write an archive until the user confirms.

## Execute Export

After confirmation, run the command array from the export plan exactly. Do not invent extra flags.

After a private archive export, verify with:

```bash
dotvibe list <archive>
```

After a recipe export, verify with:

```bash
dotvibe recipe inspect <recipe>
dotvibe recipe lint <recipe>
```

If recipe lint reports errors, tell the user the recipe should not be shared unless they inspect and explicitly accept the risk.

## Transfer Handoff

End with:

- archive or recipe path
- size if available
- private vs shareable classification
- included tools or profile
- destination prompt the user can paste, for example:

```text
调用 dotvibe 导入旧 Mac 的记忆，备份在 <archive-path>
```
