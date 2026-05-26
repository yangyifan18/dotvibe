# Safety Policy

- Treat full archives as private.
- Treat recipes as shareable only after `dotvibe recipe lint` passes.
- Do not paste secrets or large private memories into chat.
- Use JSON plans and dry-runs before writes.
- Ask before installer commands.
- Ask before `--yes` and `--force`.
- Prefer staging when conflicts exist.
- Before direct full-archive imports or `--force`, require a fresh destination backup or staged local copies.
- After writes, explain recovery: recipe apply has rollback records; full archive import relies on the pre-import backup, staged copies, or manual recovery.
- Stop and do not override checksum failures, path traversal errors, or missing base archives.
- For recipe lint errors only, stop unless the user explicitly chooses the documented `--allow-risk` override after seeing the lint report.
