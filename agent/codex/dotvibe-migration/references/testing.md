# Skill Behavior Testing

Use this to validate that the dotvibe migration skill works from simple natural-language prompts.

## Manual Prompt Tests

Run these in a fresh Codex session with the skill installed.

### Source Export

Prompt:

```text
调用 dotvibe 导出记忆
```

Expected behavior:

- Skill triggers without the user naming `dotvibe-migration`.
- Agent says it will inspect without writing, uploading, or deleting.
- Agent runs `dotvibe agent doctor --json` and `dotvibe agent inventory --json`.
- Agent asks the three export choices: full private backup, project memory, team recipe.
- Agent does not write an archive until the user confirms the export plan.

### Destination Import

Prompt:

```text
调用 dotvibe 导入旧 Mac 的记忆，备份在 ~/Downloads/dotvibe.tar.gz
```

Expected behavior:

- Skill triggers without command instructions.
- Agent says it will inspect without writing tool directories.
- Agent runs `dotvibe agent doctor --json` and `dotvibe agent import-plan ~/Downloads/dotvibe.tar.gz --json`.
- Agent summarizes writes, identical files, conflicts, unsupported paths, and project relocations.
- Agent does not run direct import until the user confirms.

### Recipe Export

Prompt:

```text
帮我导出一个可以分享给团队的 vibe recipe
```

Expected behavior:

- Agent chooses recipe export profile.
- Agent explains that recipes exclude private project memory, transcripts, sessions, auth, and cache.
- Agent plans recipe export before writing.
- Agent verifies with `dotvibe recipe inspect` and `dotvibe recipe lint` after export.

### Project Relocation

Prompt:

```text
新 Mac 用户名变了，帮我导入旧项目 memory
```

Expected behavior:

- Agent asks for archive path if missing.
- Agent runs import plan.
- Agent reads `project-relocation.md` when relocations are present.
- Agent explains home remap, missing clone, local memory, and remote mismatch choices.

### Memory Merge

Prompt:

```text
帮我合并 dotvibe-stage-review 里的 project memory
```

Expected behavior:

- Agent reads the stage `import-plan.json` first.
- Agent compares staged archive files with local copies.
- Agent avoids pasting full private memory into chat.
- Agent presents keep local, use archive, write merged file, or skip choices.

## File-Level Checks

From the repo root, run:

```bash
rg -n "调用 dotvibe 导出记忆|调用 dotvibe 导入旧 Mac|references/intents.md|references/testing.md" agent/codex/dotvibe-migration/SKILL.md
rg -n "source_export|destination_import|recipe_export|project_relocation|memory_merge" agent/codex/dotvibe-migration/references/intents.md
rg -n "dotvibe agent doctor --json|dotvibe agent inventory --json|dotvibe agent import-plan <archive> --json" agent/codex/dotvibe-migration/references/*.md
```

Expected: all commands print matches and exit with status 0.

## Regression Checks

- Explicit prompt `use dotvibe-migration skill` still routes correctly.
- Command-centric prompt `help me run dotvibe agent import-plan` still receives command-level guidance.
- Safety gates still require confirmation for `--yes`, `--force`, `git clone`, `setup --install`, recipe apply, and `--allow-risk`.
