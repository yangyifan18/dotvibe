# Safety Policy

- Treat full archives as private.
- Treat recipes as shareable only after `dotvibe recipe lint` passes.
- Do not paste secrets or large private memories into chat.
- Use JSON plans and dry-runs before writes.
- Ask before installer commands.
- Ask before `--yes` and `--force`.
- Prefer staging when conflicts exist.
- After writes, tell the user how to inspect rollback records.
- Stop if dotvibe reports checksum, path traversal, missing base archive, or lint errors unless the user explicitly chooses a documented override.
