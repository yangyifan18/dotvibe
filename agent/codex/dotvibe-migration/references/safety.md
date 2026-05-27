# Safety Policy

Use this before installer commands, writes, overwrites, clone operations, recipe apply, lint overrides, or any action that changes tool directories.

## Privacy Classification

- Treat full archives as private.
- Treat project-memory archives as private.
- Treat recipes as shareable only after `dotvibe recipe lint` passes.
- Do not paste secrets, tokens, private transcripts, or full private memories into chat.

## Mandatory Confirmation Gates

Ask for explicit confirmation before:

- writing an export archive
- running `dotvibe import` without `--dry-run` or `--stage`
- adding `--yes`
- adding `--force`
- running `git clone`
- running `dotvibe setup --install`
- applying a recipe
- using `--allow-risk`

Confirmation must name the action, target path or archive path, and risk.

## Required Preview Gates

Before writes:

- Use `dotvibe agent export-plan --json` before export.
- Use `dotvibe agent import-plan <archive> --json` before import.
- Use `dotvibe import <archive> --dry-run` before direct import when practical.
- Use `dotvibe import <archive> --stage --stage-dir <dir>` when conflicts or existing project memory are present.
- Use `dotvibe recipe inspect`, `dotvibe recipe lint`, and `dotvibe recipe diff` before recipe apply.

## Stop Conditions

Stop and do not override:

- checksum failure
- path traversal or unsafe archive path error
- missing base archive
- unsupported archive paths in an import plan
- recipe lint errors, unless the user explicitly chooses the documented `--allow-risk` override after seeing the lint report
- project association verdict `unrelated`, unless the user asks for manual staging only

## Recovery Guidance

After writes, explain recovery:

- Recipe apply has rollback records; show `dotvibe rollback list` and the relevant apply id when available.
- Full archive import currently relies on a fresh pre-import backup, staged local copies, or manual recovery.
- For direct imports, recommend creating a destination-side backup first unless the plan only writes new files into empty targets.
