#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd -P)"
# shellcheck source=lib/preflight-fixtures.sh
source "$SCRIPT_DIR/lib/preflight-fixtures.sh"

DOTVIBE=""
KEEP=0
VERBOSE=0
SKIP_RECIPE=0
SKIP_IMPORT_WRITE=0
TMP_ROOT=""
LOG_DIR=""
REAL_HOME="${HOME:-}"
BACKUP=""
DELTA=""
RECIPE=""

usage() {
  cat <<'USAGE'
Usage: scripts/homebrew-preflight.sh [options]

Options:
  --binary <path>       Use an existing dotvibe binary instead of building one.
  --keep                Keep the temp root and print its path.
  --verbose             Print each dotvibe command before running.
  --skip-recipe         Skip recipe apply/rollback checks.
  --skip-import-write   Stop after dry-run/stage checks.
  -h, --help            Show this help.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary)
      [[ $# -ge 2 ]] || { echo "--binary requires a path" >&2; exit 2; }
      DOTVIBE="$2"
      shift 2
      ;;
    --keep)
      KEEP=1
      shift
      ;;
    --verbose)
      VERBOSE=1
      shift
      ;;
    --skip-recipe)
      SKIP_RECIPE=1
      shift
      ;;
    --skip-import-write)
      SKIP_IMPORT_WRITE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cleanup() {
  local status=$?
  if [[ "$KEEP" == "0" && -n "$TMP_ROOT" && -d "$TMP_ROOT" ]]; then
    rm -rf "$TMP_ROOT"
    if [[ "$status" == "0" ]]; then
      echo "Preflight passed; temp root removed."
    fi
  fi
}
trap cleanup EXIT

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/dotvibepreflight.XXXXXX")"
TMP_ROOT="$(cd "$TMP_ROOT" && pwd -P)"
LOG_DIR="$TMP_ROOT/logs"
mkdir -p "$LOG_DIR" "$TMP_ROOT/bin" "$TMP_ROOT/go-cache"

if [[ -n "$DOTVIBE" ]]; then
  DOTVIBE="$(cd "$(dirname "$DOTVIBE")" && pwd -P)/$(basename "$DOTVIBE")"
  [[ -x "$DOTVIBE" ]] || { echo "dotvibe binary is not executable: $DOTVIBE" >&2; exit 1; }
else
  DOTVIBE="$TMP_ROOT/bin/dotvibe"
  if [[ "$VERBOSE" == "1" ]]; then
    echo "+ GOCACHE=$TMP_ROOT/go-cache go build -o $DOTVIBE ."
  fi
  (cd "$REPO_ROOT" && GOCACHE="$TMP_ROOT/go-cache" go build -o "$DOTVIBE" .) > "$LOG_DIR/00-go-build.out" 2> "$LOG_DIR/00-go-build.err"
fi

SOURCE_HOME="$TMP_ROOT/Users/young"
TARGET_HOME="$TMP_ROOT/Users/youtopia"
seed_source_home "$SOURCE_HOME"
seed_target_home "$TARGET_HOME"
SOURCE_KEY_A="$(project_key_for_path "$SOURCE_HOME/Softwares/dotvibe-fixture-a")"
SOURCE_KEY_B="$(project_key_for_path "$SOURCE_HOME/Softwares/existing-fixture-b")"
PROJECT_TARGET_ARGS=(
  --project-target "$SOURCE_KEY_A=$TARGET_HOME/Softwares/dotvibe-fixture-a"
  --project-target "$SOURCE_KEY_B=$TARGET_HOME/Softwares/existing-fixture-b"
)

run_dotvibe() {
  local home="$1"
  shift
  local log_name="$1"
  shift
  mkdir -p "$home/.config"
  if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo "+ HOME=$home XDG_CONFIG_HOME=$home/.config $DOTVIBE $*"
  fi
  HOME="$home" XDG_CONFIG_HOME="$home/.config" "$DOTVIBE" "$@" > "$LOG_DIR/$log_name.out" 2> "$LOG_DIR/$log_name.err"
}

run_dotvibe_expect_failure() {
  local home="$1"
  shift
  local log_name="$1"
  shift
  mkdir -p "$home/.config"
  if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo "+ expect-fail HOME=$home XDG_CONFIG_HOME=$home/.config $DOTVIBE $*"
  fi
  if HOME="$home" XDG_CONFIG_HOME="$home/.config" "$DOTVIBE" "$@" > "$LOG_DIR/$log_name.out" 2> "$LOG_DIR/$log_name.err"; then
    echo "command unexpectedly succeeded: $DOTVIBE $*" >&2
    return 1
  fi
}

# 1. Source detection and agent inventory.
run_dotvibe "$SOURCE_HOME" "01-status" status
run_dotvibe "$SOURCE_HOME" "02-agent-doctor" agent doctor --json
run_dotvibe "$SOURCE_HOME" "03-agent-inventory" agent inventory --json
assert_contains "$LOG_DIR/01-status.out" "Claude Code"
assert_contains "$LOG_DIR/01-status.out" "Codex CLI"
assert_contains "$LOG_DIR/01-status.out" "OpenCode"
assert_contains "$LOG_DIR/02-agent-doctor.out" '"agent_doctor": true'
assert_contains "$LOG_DIR/03-agent-inventory.out" '"claude-code"'
assert_contains "$LOG_DIR/03-agent-inventory.out" '"codex-cli"'
assert_contains "$LOG_DIR/03-agent-inventory.out" '"opencode"'

# 2. Full export/list and default privacy boundaries.
BACKUP="$TMP_ROOT/dotvibe-full.tar.gz"
run_dotvibe "$SOURCE_HOME" "04-export-full" export -o "$BACKUP"
run_dotvibe "$SOURCE_HOME" "05-list-full" list "$BACKUP"
tar -tzf "$BACKUP" > "$LOG_DIR/05-tar-list.txt"
assert_file_exists "$BACKUP"
assert_contains "$LOG_DIR/05-list-full.out" "claude-code"
assert_contains "$LOG_DIR/05-list-full.out" "codex-cli"
assert_contains "$LOG_DIR/05-list-full.out" "opencode"
assert_not_contains "$LOG_DIR/05-tar-list.txt" "auth.json"
assert_not_contains "$LOG_DIR/05-tar-list.txt" "transcripts/session.jsonl"
assert_not_contains "$LOG_DIR/05-tar-list.txt" "codex-cli/sessions/session.jsonl"

# 3. Export force behavior.
run_dotvibe_expect_failure "$SOURCE_HOME" "06-export-existing" export -o "$BACKUP"
assert_contains "$LOG_DIR/06-export-existing.err" "already exists"
run_dotvibe "$SOURCE_HOME" "07-export-force" export -o "$BACKUP" --force

# 4. Destination import dry-run and no-write behavior.
run_dotvibe "$TARGET_HOME" "08-import-dry-run" import "$BACKUP" --dry-run
assert_contains "$LOG_DIR/08-import-dry-run.out" "Dry run"
assert_file_not_exists "$TARGET_HOME/.claude/skills/reviewer/SKILL.md"
assert_file_not_exists "$TARGET_HOME/.codex/agents/reviewer.md"

# 5. Agent import-plan with default HOME remap and project review signals.
run_dotvibe "$TARGET_HOME" "09-agent-import-plan" agent import-plan "$BACKUP" --json "${PROJECT_TARGET_ARGS[@]}"
assert_contains "$LOG_DIR/09-agent-import-plan.out" '"project_relocations"'
assert_contains "$LOG_DIR/09-agent-import-plan.out" '"confirm-clone-then-stage-memory"'
assert_contains "$LOG_DIR/09-agent-import-plan.out" '"association_review_required": true'
assert_contains "$LOG_DIR/09-agent-import-plan.out" "$TARGET_HOME/Softwares/dotvibe-fixture-a"

# 6. Stage workspace for project memory review.
STAGE_DIR="$TMP_ROOT/stage-review"
run_dotvibe "$TARGET_HOME" "10-import-stage" import "$BACKUP" --stage --stage-dir "$STAGE_DIR" --remap-home "${PROJECT_TARGET_ARGS[@]}"
find "$STAGE_DIR" -type f | sort > "$LOG_DIR/10-stage-files.txt"
assert_contains "$LOG_DIR/10-import-stage.out" "Staged"
assert_contains "$LOG_DIR/10-stage-files.txt" "CLAUDE.md"
assert_contains "$LOG_DIR/10-stage-files.txt" "MEMORY.md"

# 7. Direct import writes only into fake target HOME.
if [[ "$SKIP_IMPORT_WRITE" == "0" ]]; then
  run_dotvibe "$TARGET_HOME" "11-import-write" import "$BACKUP" --yes --remap-home "${PROJECT_TARGET_ARGS[@]}"
  assert_file_exists "$TARGET_HOME/.claude/skills/reviewer/SKILL.md"
  assert_file_exists "$TARGET_HOME/.codex/agents/reviewer.md"
  assert_file_exists "$TARGET_HOME/.config/opencode/opencode.json"
  assert_contains "$LOG_DIR/11-import-write.out" "$TARGET_HOME"
  if [[ -n "$REAL_HOME" ]]; then
    assert_not_contains "$LOG_DIR/11-import-write.out" "$REAL_HOME"
  fi
fi

# 8. Incremental backup and diff.
key_a="$(project_key_for_path "$SOURCE_HOME/Softwares/dotvibe-fixture-a")"
write_file "$SOURCE_HOME/.claude/projects/$key_a/memory/MEMORY.md" "fixture A changed memory for incremental diff"
DELTA="$TMP_ROOT/dotvibe-delta.tar.gz"
run_dotvibe "$SOURCE_HOME" "12-export-delta" export --base "$BACKUP" -o "$DELTA"
run_dotvibe "$SOURCE_HOME" "13-diff-memory" diff --only claude-code --category memory "$BACKUP" "$DELTA"
run_dotvibe "$TARGET_HOME" "14-import-delta-dry-run" import "$DELTA" --base "$BACKUP" --dry-run
assert_file_exists "$DELTA"
assert_contains "$LOG_DIR/13-diff-memory.out" "changed"
assert_contains "$LOG_DIR/14-import-delta-dry-run.out" "Dry run"

# 9. Recipe export/lint/apply/rollback in fake target HOME.
if [[ "$SKIP_RECIPE" == "0" ]]; then
  RECIPE="$TMP_ROOT/team-recipe.vibe"
  run_dotvibe "$SOURCE_HOME" "15-recipe-export" recipe export --name "Preflight Vibe Recipe" --author "dotvibe-preflight" -o "$RECIPE"
  run_dotvibe "$SOURCE_HOME" "16-recipe-inspect" recipe inspect "$RECIPE"
  run_dotvibe "$SOURCE_HOME" "17-recipe-lint" recipe lint "$RECIPE" --strict
  run_dotvibe "$TARGET_HOME" "18-recipe-apply-dry-run" recipe apply "$RECIPE" --dry-run
  run_dotvibe "$TARGET_HOME" "19-recipe-apply" recipe apply "$RECIPE" --yes
  run_dotvibe "$TARGET_HOME" "20-rollback-list" rollback list
  assert_file_exists "$RECIPE"
  assert_not_contains "$LOG_DIR/16-recipe-inspect.out" "transcripts"
  assert_not_contains "$LOG_DIR/16-recipe-inspect.out" "projects/"
  assert_contains "$LOG_DIR/18-recipe-apply-dry-run.out" "Dry run"
  assert_contains "$LOG_DIR/20-rollback-list.out" "Preflight Vibe Recipe"

  ROLLBACK_ID="$(awk 'NR==1 {print $1}' "$LOG_DIR/20-rollback-list.out")"
  if [[ -z "$ROLLBACK_ID" ]]; then
    echo "rollback list did not return an id" >&2
    exit 1
  fi
  run_dotvibe "$TARGET_HOME" "21-recipe-rollback" rollback "$ROLLBACK_ID" --yes
  assert_contains "$LOG_DIR/21-recipe-rollback.out" "rolled back"
fi

# 10. Setup dry-run stays non-destructive.
run_dotvibe "$TARGET_HOME" "22-setup-dry-run" setup "$BACKUP"
assert_contains "$LOG_DIR/22-setup-dry-run.out" "dotvibe setup plan"
assert_not_contains "$LOG_DIR/22-setup-dry-run.out" "Running"

if [[ "$KEEP" == "1" ]]; then
  echo "Preflight passed."
  echo "Temp root: $TMP_ROOT"
  echo "Backup: $BACKUP"
  echo "Delta: $DELTA"
  if [[ "$SKIP_RECIPE" == "0" ]]; then
    echo "Recipe: $RECIPE"
  else
    echo "Recipe: skipped"
  fi
  echo "Logs: $LOG_DIR"
fi
