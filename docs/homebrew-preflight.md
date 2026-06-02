# Homebrew Preflight

This runbook validates dotvibe before publishing the Homebrew tap.

## Why temp HOME instead of Docker?

The default macOS Homebrew release path should be tested on macOS. Docker can test Linux/Linuxbrew behavior, but it cannot faithfully test macOS Homebrew bottles, paths, permissions, or formula behavior. The default preflight therefore uses a temporary fake source HOME and target HOME. It behaves like a virtual machine for dotvibe data while still running the macOS binary locally.

For stronger isolation, run the same scripts inside a clean macOS VM before tagging the release.

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

Expected: `brew audit`, `brew install --build-from-source`, `brew test`, and the functional preflight against the brewed binary pass.

## Release gate checklist

- [ ] `go test ./... -count=1`
- [ ] `./scripts/homebrew-preflight.sh --keep`
- [ ] `VERSION=1.1 ./scripts/build-release.sh`
- [ ] GitHub Release has darwin/arm64, darwin/amd64, linux/amd64 artifacts and checksums.
- [ ] Formula `url` and `sha256` point to the release artifact or source tarball selected for the tap.
- [ ] `./scripts/homebrew-formula-smoke.sh --formula <formula>` passes.
- [ ] README Homebrew install line is updated only after the tap works.
