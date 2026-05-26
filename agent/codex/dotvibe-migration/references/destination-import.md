# Destination Import Workflow

Use this on the new Mac.

1. Run `dotvibe agent doctor --json`.
2. If required tools are missing, run `dotvibe setup <archive>` first without `--install`; ask before using `--install`.
3. Run `dotvibe agent import-plan <archive> --json`.
4. Summarize writes, identical files, conflicts, unsupported paths, and recommended next action.
5. If conflicts include project memory, recommend staging.
6. For staging, run `dotvibe import <archive> --stage --stage-dir <chosen-dir>`.
7. If the user chooses direct import, run `dotvibe import <archive> --dry-run` first.
8. Ask before direct write. Use `--yes` only after explicit confirmation.
9. Avoid `--force` unless the user chooses overwrite after seeing conflicts.
10. After full-archive import, summarize changed files and note that full imports currently do not create rollback records; use staged/local copies or a fresh backup for manual recovery. For recipe apply flows, show `dotvibe rollback list` plus the relevant apply id.
