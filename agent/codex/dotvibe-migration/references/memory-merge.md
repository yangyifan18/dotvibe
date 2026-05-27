# Project Memory Merge Workflow

Use this after `dotvibe import --stage` when local and archive project memory both exist, or when the user says `帮我合并 project memory`, `review staged memory`, or `merge dotvibe-stage-review`.

## Inputs

- Archive copy: `<stage-dir>/files/<logical-path>`
- Local copy: `<stage-dir>/local/<logical-path>`
- Plan: `<stage-dir>/import-plan.json`

Read `import-plan.json` first so you know which files were staged, which targets were remapped, and which files need review.

## Merge Rules

- If local file is missing, recommend using archive.
- If files are identical, skip.
- If both are Markdown and small, compare sections by heading.
- If sections are unrelated, propose appending missing sections.
- If the same rule or instruction changed on both sides, ask the user.
- If either side contains credential-like content, do not display it inline; summarize and ask.
- Show a final diff before writing.
- Keep local content by default when uncertain.

## User-Facing Choices

Produce one of these choices per file:

1. keep local
2. use archive
3. write merged file
4. skip for later manual review

Do not overwrite real tool directories until the user confirms the final diff.

## Safe Write Guidance

If the user chooses a merged result, write it only to the staged review workspace first unless they explicitly confirm a real import/write command. For real tool directories, prefer a dotvibe import command with the same remap/project flags used to create the stage workspace.
