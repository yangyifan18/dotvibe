# Destination Import Workflow

Use this on the new Mac.

1. Run `dotvibe agent doctor --json`.
2. If required tools are missing, run `dotvibe setup <archive>` first without `--install`; ask before using `--install`.
3. Run `dotvibe agent import-plan <archive> --json`.
4. Summarize writes, identical files, conflicts, unsupported paths, and recommended next action.
5. If conflicts include project memory, recommend staging.
6. For staging, run `dotvibe import <archive> --stage --stage-dir <chosen-dir>`.
7. If the user chooses direct import, run `dotvibe import <archive> --dry-run` first.
8. Before a direct write, create or ask the user to create a fresh destination-side `dotvibe export` backup, unless the final action only writes new files into an empty target.
9. Ask before direct write. Use `--yes` only after explicit confirmation.
10. Avoid `--force` unless the user chooses overwrite after seeing conflicts and has a fresh backup or staged local copies.
11. After full-archive import, summarize changed files and note that full imports currently do not create rollback records; use the pre-import backup, staged/local copies, or manual recovery. For recipe apply flows, show `dotvibe rollback list` plus the relevant apply id.
