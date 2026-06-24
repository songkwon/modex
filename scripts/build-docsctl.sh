#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Build docsctl executables for common platforms.

Optional:
  VERSION      docsctl version string embedded into the binary
               (default: git describe --tags --always --dirty, or dev)
  OUTPUT_DIR   output directory (default: dist/docsctl)
  PLATFORMS    whitespace-separated GOOS/GOARCH targets
               (default: linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64)
  CGO_ENABLED  Go CGO setting (default: 0)
  GOPROXY      Go module proxy, if needed by your environment

Examples:
  ./scripts/build-docsctl.sh
  VERSION=v0.2.0 OUTPUT_DIR=release/docsctl ./scripts/build-docsctl.sh
  PLATFORMS="linux/amd64 darwin/arm64 windows/amd64" ./scripts/build-docsctl.sh
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
docsctl_dir="${repo_root}/tools/docsctl"
output_dir="${OUTPUT_DIR:-${repo_root}/dist/docsctl}"
platforms="${PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"

require_cmd go

if [[ -z "${VERSION:-}" ]]; then
  if command -v git >/dev/null 2>&1; then
    version="$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null || true)"
  fi
  version="${version:-dev}"
else
  version="$VERSION"
fi

mkdir -p "$output_dir"

echo "Building docsctl ${version}"
echo "Output directory: ${output_dir}"
echo

for platform in $platforms; do
  goos="${platform%%/*}"
  goarch="${platform##*/}"

  if [[ -z "$goos" || -z "$goarch" || "$goos" == "$goarch" ]]; then
    echo "Invalid platform: ${platform}. Expected GOOS/GOARCH, for example linux/amd64." >&2
    exit 1
  fi

  binary="docsctl-${goos}-${goarch}"
  if [[ "$goos" == "windows" ]]; then
    binary="${binary}.exe"
  fi

  echo "Building ${binary}"
  (
    cd "$docsctl_dir"
    CGO_ENABLED="${CGO_ENABLED:-0}" GOOS="$goos" GOARCH="$goarch" go build \
      -trimpath \
      -ldflags="-s -w -X main.docsctlVersion=${version}" \
      -o "${output_dir}/${binary}" \
      ./cmd/docsctl
  )
done

if command -v shasum >/dev/null 2>&1; then
  (
    cd "$output_dir"
    shasum -a 256 docsctl-* > checksums.txt
  )
  echo
  echo "Wrote checksums: ${output_dir}/checksums.txt"
fi

echo
echo "Done."
