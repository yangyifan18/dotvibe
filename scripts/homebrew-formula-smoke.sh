#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd -P)"
FORMULA=""
ALLOW_EXISTING_DOTVIBE=0
SKIP_ONLINE_AUDIT=0
KEEP=0
TMP_ROOT=""
LOG_DIR=""
DOTVIBE_WAS_INSTALLED=0
SCRIPT_INSTALLED_DOTVIBE=0

usage() {
  cat <<'USAGE'
Usage: scripts/homebrew-formula-smoke.sh --formula <path-or-name> [options]

Options:
  --formula <path-or-name>          Required. Formula file path, tap formula name, or user/tap/formula.
  --allow-existing-dotvibe          Allow test to proceed if dotvibe is already installed by brew.
  --skip-online-audit               Run audit without --online.
  --keep                            Keep logs and pass --keep to homebrew-preflight.sh.
  -h, --help                        Show this help.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --formula)
      [[ $# -ge 2 ]] || { echo "--formula requires a value" >&2; exit 2; }
      FORMULA="$2"
      shift 2
      ;;
    --allow-existing-dotvibe)
      ALLOW_EXISTING_DOTVIBE=1
      shift
      ;;
    --skip-online-audit)
      SKIP_ONLINE_AUDIT=1
      shift
      ;;
    --keep)
      KEEP=1
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

if [[ -z "$FORMULA" ]]; then
  echo "--formula is required" >&2
  usage >&2
  exit 2
fi

cleanup() {
  local status=$?
  if [[ "$SCRIPT_INSTALLED_DOTVIBE" == "1" && "$DOTVIBE_WAS_INSTALLED" == "0" && "$KEEP" == "0" ]]; then
    brew uninstall --force dotvibe > "$LOG_DIR/brew-uninstall.out" 2> "$LOG_DIR/brew-uninstall.err" || true
  fi
  if [[ -n "$TMP_ROOT" && -d "$TMP_ROOT" && "$status" != "0" ]]; then
    echo "Homebrew smoke failed. Logs: $LOG_DIR" >&2
  elif [[ "$KEEP" == "0" && -n "$TMP_ROOT" && -d "$TMP_ROOT" ]]; then
    rm -rf "$TMP_ROOT"
  elif [[ -n "$TMP_ROOT" ]]; then
    echo "Homebrew smoke logs: $TMP_ROOT"
  fi
  return "$status"
}
trap cleanup EXIT

command -v brew >/dev/null 2>&1 || { echo "brew not found" >&2; exit 1; }

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/dotvibe-brew-smoke.XXXXXX")"
TMP_ROOT="$(cd "$TMP_ROOT" && pwd -P)"
LOG_DIR="$TMP_ROOT/logs"
mkdir -p "$LOG_DIR"

brew --version > "$LOG_DIR/brew-version.out" 2> "$LOG_DIR/brew-version.err"
brew --prefix > "$LOG_DIR/brew-prefix.out" 2> "$LOG_DIR/brew-prefix.err"
brew --repository > "$LOG_DIR/brew-repository.out" 2> "$LOG_DIR/brew-repository.err"
brew config > "$LOG_DIR/brew-config.out" 2> "$LOG_DIR/brew-config.err"

if brew list --formula dotvibe > "$LOG_DIR/brew-list-dotvibe.out" 2> "$LOG_DIR/brew-list-dotvibe.err"; then
  DOTVIBE_WAS_INSTALLED=1
  if [[ "$ALLOW_EXISTING_DOTVIBE" != "1" ]]; then
    echo "dotvibe is already installed by brew; rerun with --allow-existing-dotvibe to proceed" >&2
    exit 1
  fi
fi

brew audit --strict --formula "$FORMULA" > "$LOG_DIR/brew-audit-strict.out" 2> "$LOG_DIR/brew-audit-strict.err"
if [[ "$SKIP_ONLINE_AUDIT" == "0" ]]; then
  brew audit --strict --online --formula "$FORMULA" > "$LOG_DIR/brew-audit-online.out" 2> "$LOG_DIR/brew-audit-online.err"
fi
brew install --build-from-source "$FORMULA" > "$LOG_DIR/brew-install.out" 2> "$LOG_DIR/brew-install.err"
SCRIPT_INSTALLED_DOTVIBE=1
brew test "$FORMULA" > "$LOG_DIR/brew-test.out" 2> "$LOG_DIR/brew-test.err"

BREWED_DOTVIBE="$(brew --prefix dotvibe)/bin/dotvibe"
if [[ ! -x "$BREWED_DOTVIBE" ]]; then
  echo "brewed dotvibe binary not found: $BREWED_DOTVIBE" >&2
  exit 1
fi

preflight_args=(--binary "$BREWED_DOTVIBE")
if [[ "$KEEP" == "1" ]]; then
  preflight_args+=(--keep)
fi
"$REPO_ROOT/scripts/homebrew-preflight.sh" "${preflight_args[@]}" > "$LOG_DIR/functional-preflight.out" 2> "$LOG_DIR/functional-preflight.err"

echo "Homebrew formula smoke passed."
echo "Formula: $FORMULA"
echo "Brewed binary: $BREWED_DOTVIBE"
if [[ "$KEEP" == "1" ]]; then
  echo "Logs: $LOG_DIR"
  echo "Cleanup: dotvibe install kept because --keep was set; uninstall manually if desired with: brew uninstall --force dotvibe"
else
  echo "Logs: removed after success; rerun with --keep to preserve them."
fi
