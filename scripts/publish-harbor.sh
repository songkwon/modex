#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Build backend, frontend, and mcp Linux images, then push them to Harbor.

Required:
  HARBOR_REGISTRY   Harbor registry host, for example harbor.example.com

Optional:
  HARBOR_PROJECT    Harbor project / namespace (default: modex)
  TAG               Image tag (default: latest)
  PLATFORM          Target platform (default: linux/amd64)
  BUILDER           docker buildx builder name (default: modex-harbor-builder)
  GOPROXY           Go module proxy for backend and mcp builds
                    (default: https://goproxy.cn,direct)
  CACHE_TO          Export BuildKit cache to Harbor (default: true)
  CACHE_FROM        Import BuildKit cache from Harbor (default: true)

Examples:
  docker login harbor.example.com
  HARBOR_REGISTRY=harbor.example.com HARBOR_PROJECT=modex ./scripts/publish-harbor.sh

  PLATFORM=linux/arm64 HARBOR_REGISTRY=harbor.example.com ./scripts/publish-harbor.sh
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

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: $name" >&2
    echo >&2
    usage >&2
    exit 1
  fi
}

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

require_cmd docker
require_env HARBOR_REGISTRY

harbor_registry="${HARBOR_REGISTRY%/}"
harbor_project="${HARBOR_PROJECT:-modex}"
tag="${TAG:-latest}"
platform="${PLATFORM:-linux/amd64}"
builder="${BUILDER:-modex-harbor-builder}"
goproxy="${GOPROXY:-https://goproxy.cn,direct}"
cache_to="${CACHE_TO:-true}"
cache_from="${CACHE_FROM:-true}"

image_prefix="${harbor_registry}/${harbor_project}"

if ! docker buildx inspect "$builder" >/dev/null 2>&1; then
  echo "Creating docker buildx builder: $builder"
  docker buildx create --name "$builder" --use >/dev/null
else
  docker buildx use "$builder" >/dev/null
fi

docker buildx inspect --bootstrap >/dev/null

build_backend() {
  local image="${image_prefix}/backend:${tag}"
  local cache_ref="${image_prefix}/build-cache:backend-${platform//\//-}"
  local -a cache_args=()
  [[ "$cache_from" == "true" ]] && cache_args+=("--cache-from=type=registry,ref=${cache_ref}")
  [[ "$cache_to" == "true" ]] && cache_args+=("--cache-to=type=registry,ref=${cache_ref},mode=max")
  echo "Building and pushing ${image} (${platform})"
  docker buildx build \
    --platform "$platform" \
    --provenance=false \
    --sbom=false \
    --build-arg "GOPROXY=${goproxy}" \
    "${cache_args[@]}" \
    -t "$image" \
    --push \
    "$repo_root/backend"
}

build_frontend() {
  local image="${image_prefix}/frontend:${tag}"
  echo "Building and pushing ${image} (${platform})"
  docker buildx build \
    --platform "$platform" \
    --provenance=false \
    --sbom=false \
    -t "$image" \
    --push \
    "$repo_root/frontend"
}

build_mcp() {
  local image="${image_prefix}/mcp:${tag}"
  local cache_ref="${image_prefix}/build-cache:mcp-${platform//\//-}"
  local -a cache_args=()
  [[ "$cache_from" == "true" ]] && cache_args+=("--cache-from=type=registry,ref=${cache_ref}")
  [[ "$cache_to" == "true" ]] && cache_args+=("--cache-to=type=registry,ref=${cache_ref},mode=max")
  echo "Building and pushing ${image} (${platform})"
  docker buildx build \
    --platform "$platform" \
    --provenance=false \
    --sbom=false \
    --build-arg "GOPROXY=${goproxy}" \
    "${cache_args[@]}" \
    -t "$image" \
    --push \
    "$repo_root/mcp"
}

echo "Publishing images to Harbor:"
echo "  ${image_prefix}/backend:${tag}"
echo "  ${image_prefix}/frontend:${tag}"
echo "  ${image_prefix}/mcp:${tag}"
echo "Target platform: ${platform}"
echo

build_backend
build_frontend
build_mcp

echo
echo "Done."
