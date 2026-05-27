# Project Memory Merge Workflow

Use this after `dotvibe import --stage` when local and archive project memory both exist.

## Inputs

- Archive copy: `<stage-dir>/files/<logical-path>`
- Local copy: `<stage-dir>/local/<logical-path>`
- Plan: `<stage-dir>/import-plan.json`

## Merge Rules

- If local file is missing, recommend using archive.
- If files are identical, skip.
- If both are Markdown and small, compare sections by heading.
- If sections are unrelated, propose appending missing sections.
- If the same rule or instruction changed on both sides, ask the user.
- If either side contains credential-like content, do not display it inline; summarize and ask.
- Show a final diff before writing.
- Keep local content by default when uncertain.

## Output

Produce one of these choices per file:

- keep local
- use archive
- write merged file
- skip for later manual review

Do not overwrite real tool directories until the user confirms the final diff.
