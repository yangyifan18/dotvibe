# Homebrew Preflight

This runbook validates dotvibe Homebrew releases and formula changes. The v1.1 tap is live, so use this before changing the formula, cutting future releases, or investigating Homebrew install regressions.

## Why temp HOME instead of Docker?

The default macOS Homebrew release path should be tested on macOS. Docker can test Linux/Linuxbrew behavior, but it cannot faithfully test macOS Homebrew bottles, paths, permissions, or formula behavior. The default preflight therefore uses a temporary fake source HOME and target HOME. It behaves like a virtual machine for dotvibe data while still running the macOS binary locally.

For stronger isolation, run the same scripts inside a clean macOS VM before tagging a release.

## v1.1 validation snapshot

Checked on 2026-06-03:

- `brew install yangyifan18/tap/dotvibe` passed.
- `dotvibe --version` printed `dotvibe version 1.1`.
- `brew test yangyifan18/tap/dotvibe` passed.
- The Homebrew-installed binary passed the full fake-HOME preflight:

```bash
./scripts/homebrew-preflight.sh --binary "$(brew --prefix dotvibe)/bin/dotvibe" --keep
```

- Covered workflows: `status`, `agent doctor/inventory/import-plan`, `export/list`, default privacy exclusions, `import --dry-run`, `import --stage`, direct fake-HOME import, incremental diff, recipe export/lint/apply/rollback, and `setup` dry-run.
- Tap CI run `26806025769` passed on `macos-26`, `macos-15-intel`, and `ubuntu`.

Known acceptable limitations for v1.1:

- Homebrew currently installs from source; bottles are not published yet.
- Local Homebrew may warn about Xcode/CLT versions or unrelated third-party taps; these are not dotvibe failures when build/test still pass.
- Tap CI may show a Node 20 deprecation annotation from GitHub Actions dependencies; treat it as follow-up noise unless it becomes a failing check.

## Functional preflight

```bash
./scripts/homebrew-preflight.sh --keep --verbose
```

Expected: status/export/list/diff/import/stage/recipe/setup checks pass and all artifacts are under the printed temp root.

## Homebrew formula smoke

The formula smoke is opt-in because `brew install --build-from-source` writes to the real Homebrew prefix. The dotvibe data checks still run inside fake `HOME` directories, but the Homebrew install/test layer is not part of the temp-HOME sandbox.

```bash
./scripts/homebrew-formula-smoke.sh --formula ./Formula/dotvibe.rb --keep
```

When `--formula` is a file path, the wrapper converts it to the formula name because Homebrew audits formula tokens, not direct paths. Run the script from a registered tap checkout such as `$(brew --repository yangyifan18/tap)` so `dotvibe` resolves to the local formula.

Expected: `brew audit`, `brew install --build-from-source`, `brew test`, and the functional preflight against the brewed binary pass.

## Release gate checklist

- [ ] `go test ./... -count=1`
- [ ] `./scripts/homebrew-preflight.sh --keep`
- [ ] `VERSION=1.1 ./scripts/build-release.sh`
- [ ] GitHub Release has darwin/arm64, darwin/amd64, linux/amd64 artifacts and checksums.
- [ ] Formula `url` and `sha256` point to the release artifact or source tarball selected for the tap.
- [ ] `./scripts/homebrew-formula-smoke.sh --formula <formula>` passes.
- [ ] README Homebrew install line remains live only after the tap and formula smoke pass.
