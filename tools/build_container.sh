#!/usr/bin/env bash
set -euo pipefail

# Builds the sharecrop backend container image following the repo standard:
#
#   manifest (multi-arch index) : <image>            e.g. sharecrop:1.4.0
#   per-arch image              : <image>-arm64      (primary, Graviton)
#   per-arch image              : <image>-amd64
#
# Both per-arch images are built and pushed, then a manifest list tagged with the
# bare <image> reference is assembled from them with `docker buildx imagetools`.
# arm64 is the primary target because the Fargate services run on arm64.
#
# Usage:
#   tools/build_container.sh <image:tag>
#
# Env:
#   PUSH=false      Build the arm64 image only and load it into the local docker
#                   for testing; skips the amd64 image and the manifest (buildx
#                   cannot load a multi-arch image, and imagetools needs a
#                   registry). Default: true (push all + create the manifest).
#   PLATFORMS_ARM   Override the arm64 platform (default linux/arm64).
#   PLATFORMS_AMD   Override the amd64 platform (default linux/amd64).

if [[ $# -ne 1 ]]; then
  echo "usage: tools/build_container.sh <image:tag>" >&2
  exit 2
fi

image="$1"
if [[ "$image" != *:* ]]; then
  echo "image reference must include a tag, e.g. sharecrop:1.4.0" >&2
  exit 2
fi

push="${PUSH:-true}"
platform_arm="${PLATFORMS_ARM:-linux/arm64}"
platform_amd="${PLATFORMS_AMD:-linux/amd64}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ "$push" != "true" ]]; then
  echo "PUSH=false: building ${image}-arm64 for local use only (no amd64, no manifest)"
  docker buildx build \
    --platform "$platform_arm" \
    --tag "${image}-arm64" \
    --load \
    "$repo_root"
  echo "loaded ${image}-arm64 into the local docker"
  exit 0
fi

build_and_push() {
  local platform="$1" tag="$2"
  echo "building and pushing ${tag} (${platform})"
  docker buildx build --platform "$platform" --tag "$tag" --push "$repo_root"
}

build_and_push "$platform_arm" "${image}-arm64"
build_and_push "$platform_amd" "${image}-amd64"

echo "assembling multi-arch manifest ${image} from the per-arch images"
docker buildx imagetools create \
  --tag "${image}" \
  "${image}-arm64" \
  "${image}-amd64"

echo "done:"
echo "  manifest : ${image}"
echo "  arm64    : ${image}-arm64"
echo "  amd64    : ${image}-amd64"
