# Source Export Workflow

Use this on the old Mac.

1. Run `dotvibe agent doctor --json` and stop if `ok=false`.
2. Run `dotvibe agent inventory --json`.
3. Present choices:
   - Full migration: private archive for the same user on a new Mac.
   - Project memories: private archive focused on Claude Code memories; project filtering happens during import.
   - Shareable recipe: skills, agents, rules, and safe settings only.
4. Ask whether to include history/sessions. Default is no.
5. Ask selected tools if the user wants a smaller archive.
6. Run `dotvibe agent export-plan --profile <profile> --json` and show the planned command.
7. Ask for confirmation before running the export command.
8. After export, run `dotvibe list <archive>` for full archives or `dotvibe recipe inspect <recipe>` and `dotvibe recipe lint <recipe>` for recipes.
9. Give transfer handoff: path, size, checksum if available, and destination instructions.
