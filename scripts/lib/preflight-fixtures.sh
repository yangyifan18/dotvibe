#!/usr/bin/env bash
set -euo pipefail

write_file() {
  local path="$1"
  local content="$2"
  mkdir -p "$(dirname "$path")"
  printf '%s\n' "$content" > "$path"
}

project_key_for_path() {
  local path="$1"
  path="${path#/}"
  printf -- '-%s\n' "${path//\//-}"
}

init_fixture_repo() {
  local repo="$1"
  local remote="$2"
  mkdir -p "$repo"
  git -C "$repo" init -q
  git -C "$repo" config user.email "dotvibe-preflight@example.invalid"
  git -C "$repo" config user.name "dotvibe preflight"
  write_file "$repo/README.md" "# $(basename "$repo")"
  git -C "$repo" add README.md
  git -C "$repo" commit -q -m "init fixture repo"
  git -C "$repo" remote add origin "$remote"
}

seed_source_home() {
  local source_home="$1"
  local project_a="$source_home/Softwares/dotvibe-fixture-a"
  local project_b="$source_home/Softwares/existing-fixture-b"
  local reconstructed_project_a="$source_home/Softwares/dotvibe/fixture/a"
  local reconstructed_project_b="$source_home/Softwares/existing/fixture/b"

  write_file "$source_home/.claude/settings.json" '{"theme":"preflight"}'
  write_file "$source_home/.claude/CLAUDE.md" '# Global Claude rules for preflight'
  write_file "$source_home/.claude/skills/reviewer/SKILL.md" '# Reviewer skill'
  write_file "$source_home/.claude/agents/planner.md" '# Planner agent'
  write_file "$source_home/.claude/commands/ship.md" '/ship preflight'
  write_file "$source_home/.claude/auth.json" '{"token":"sk-preflight-should-not-export"}'
  write_file "$source_home/.claude/transcripts/session.jsonl" '{"token":"transcript-should-not-export"}'

  write_file "$source_home/.codex/config.toml" 'model = "gpt-preflight"'
  write_file "$source_home/.codex/AGENTS.md" '# Codex project rules'
  write_file "$source_home/.codex/agents/reviewer.md" '# Codex reviewer agent'
  write_file "$source_home/.codex/skills/migration/SKILL.md" '# Migration skill'
  write_file "$source_home/.codex/sessions/session.jsonl" '{"token":"codex-session-should-not-export"}'

  write_file "$source_home/.config/opencode/opencode.json" '{"provider":"preflight"}'
  write_file "$source_home/.opencode/oh-my-openagent.json" '{"agent":"preflight"}'

  write_file "$project_a/README.md" "# dotvibe-fixture-a"
  write_file "$project_b/README.md" "# existing-fixture-b"

  # Claude project keys are lossy for path segments containing hyphens. These
  # mirror repos let the current metadata reader recover git remotes while the
  # public fixture paths above keep the release scenario user-friendly.
  init_fixture_repo "$reconstructed_project_a" "https://github.com/example/dotvibe-fixture-a.git"
  init_fixture_repo "$reconstructed_project_b" "https://github.com/example/existing-fixture-b.git"

  local key_a
  key_a="$(project_key_for_path "$project_a")"
  write_file "$source_home/.claude/projects/$key_a/CLAUDE.md" '# Claude memory for fixture A'
  write_file "$source_home/.claude/projects/$key_a/memory/MEMORY.md" 'fixture A private project memory'

  local key_b
  key_b="$(project_key_for_path "$project_b")"
  write_file "$source_home/.claude/projects/$key_b/CLAUDE.md" '# Claude memory for fixture B'
  write_file "$source_home/.claude/projects/$key_b/MEMORY.md" 'fixture B root memory'
}

seed_target_home() {
  local target_home="$1"
  local project_b="$target_home/Softwares/existing-fixture-b"

  write_file "$target_home/.claude/settings.json" '{"theme":"existing-target"}'
  write_file "$target_home/.codex/config.toml" 'model = "existing-target"'
  init_fixture_repo "$project_b" "https://github.com/example/not-the-same-repo.git"

  local target_key_b
  target_key_b="$(project_key_for_path "$project_b")"
  write_file "$target_home/.claude/projects/$target_key_b/CLAUDE.md" '# Existing target memory for fixture B'
}

assert_file_exists() { test -f "$1" || { echo "missing file: $1" >&2; return 1; }; }
assert_file_not_exists() { test ! -e "$1" || { echo "unexpected path exists: $1" >&2; return 1; }; }
assert_contains() { grep -Fq -- "$2" "$1" || { echo "missing text '$2' in $1" >&2; return 1; }; }
assert_not_contains() { ! grep -Fq -- "$2" "$1" || { echo "unexpected text '$2' in $1" >&2; return 1; }; }
