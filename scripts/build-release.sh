#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-dev}"
MODULE="github.com/yangyifan18/dotvibe"
LDFLAGS="-s -w -X ${MODULE}/cmd.version=${VERSION}"
DIST_DIR="${DIST_DIR:-dist}"

mkdir -p "${DIST_DIR}"

targets=(
  "darwin arm64"
  "darwin amd64"
  "linux amd64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"${target}"
  out="${DIST_DIR}/dotvibe_${VERSION}_${goos}_${goarch}"
  if [[ "${goos}" == "windows" ]]; then
    out="${out}.exe"
  fi
  echo "building ${out}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags "${LDFLAGS}" -o "${out}" .
done
