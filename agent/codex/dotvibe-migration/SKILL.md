---
name: dotvibe-migration
description: Use when the user wants to migrate, export, import, restore, review, or merge vibe coding memories/configs with dotvibe across Claude Code, Codex CLI, or OpenCode, especially when moving to a new Mac or applying a .vibe recipe with guided choices.
---

# dotvibe Migration

Guide users through dotvibe migrations as a choice-driven assistant. Use dotvibe's JSON and dry-run surfaces for decisions; do not parse human output when a JSON command exists.

## Start Here

1. Run `dotvibe agent doctor --json`.
2. If the user is on the source Mac, read `references/source-export.md`.
3. If the user is on the destination Mac, read `references/destination-import.md`.
4. If `import-plan` reports `project_relocations`, or the user mentions a changed username/home path, missing project checkout, clone, repo mismatch, or loading old project memory, read `references/project-relocation.md`.
5. If project memory conflicts exist or the user asks to review/merge memory, read `references/memory-merge.md`.
6. For risky actions, read `references/safety.md`.

## Rules

- Ask choice-style questions; avoid open-ended command prompts.
- Always run a plan or dry-run before writing.
- Before direct full-archive imports or `--force`, require either a fresh destination-side backup or a staging workspace the user has reviewed.
- Never use `--force`, `--yes`, setup installs, or direct import writes without explicit user confirmation.
- Prefer `dotvibe import --stage` for existing project memory.
- Do not paste secrets or full private memory into chat unless the user explicitly asks after a warning.
- After any write, summarize the result and show recovery instructions. Recipe apply creates rollback records; full archive import currently requires the pre-import backup, staged local copies, or manual recovery.

## Common Entrypoints

- Source Mac export: `dotvibe agent inventory --json`, then `dotvibe agent export-plan --json`.
- Destination Mac import: `dotvibe agent import-plan <archive> --json`, then `dotvibe import <archive> --stage --stage-dir <dir>` when review is needed.
- Shareable setup: use `dotvibe recipe inspect/lint/diff/apply` and rollback guidance.
