#!/usr/bin/env bash
set -euo pipefail

# Builds the sharecrop backend container image following the repo standard:
#
#   manifest (multi-arch index) : <image>            e.g. ghcr.io/e6qu/sharecrop:0123456789ab
#   per-arch image              : <image>-arm64      (primary, Graviton)
#   per-arch image              : <image>-amd64
#
# The image bakes a CPU-arch-specific wazero cache by running the freshly built
# binary at build time, so each arch must be built where that binary can execute:
# natively on a runner of that arch (fast), or under emulation (slow). Build the
# per-arch images first, then assemble the manifest from them.
#
# Usage:
#   tools/build_container.sh <image:tag> [arch]   # build+push <image:tag>-<arch>
#                                                 # (arch defaults to the host arch)
#   tools/build_container.sh <image:tag> manifest # assemble <image:tag> from the
#                                                 # per-arch images (both must exist)
#
# Env:
#   PUSH=false   For a per-arch build, load the image into the local docker
#                instead of pushing (single arch, host arch only) - for testing.

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: tools/build_container.sh <image:tag> [arch|manifest]" >&2
  exit 2
fi

image="$1"
if [[ "$image" != *:* ]]; then
  echo "image reference must include a tag, e.g. ghcr.io/e6qu/sharecrop:0123456789ab" >&2
  exit 2
fi
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

host_arch() {
  case "$(uname -m)" in
    arm64 | aarch64) echo "arm64" ;;
    x86_64 | amd64) echo "amd64" ;;
    *) echo "unsupported host arch $(uname -m)" >&2; exit 2 ;;
  esac
}

if [[ "${2:-}" == "manifest" ]]; then
  echo "assembling multi-arch manifest ${image} from the per-arch images"
  docker buildx imagetools create --tag "${image}" "${image}-arm64" "${image}-amd64"
  "$repo_root/tools/verify_container_shape.sh" "${image}" manifest
  echo "done: manifest ${image} -> ${image}-arm64, ${image}-amd64"
  exit 0
fi

arch="${2:-$(host_arch)}"
if [[ "$arch" != "arm64" && "$arch" != "amd64" ]]; then
  echo "arch must be arm64 or amd64 (got ${arch})" >&2
  exit 2
fi
if [[ "$arch" != "$(host_arch)" ]]; then
  echo "warning: building ${arch} on a $(host_arch) host runs the whole build under emulation (slow)" >&2
fi

output=(--output "type=registry,oci-mediatypes=true")
if [[ "${PUSH:-true}" != "true" ]]; then
  output=(--load)
  echo "PUSH=false: loading ${image}-${arch} into the local docker (not pushing)"
fi

echo "building ${image}-${arch} (linux/${arch})"
docker buildx build \
  --platform "linux/${arch}" \
  --tag "${image}-${arch}" \
  --provenance=false \
  --sbom=false \
  "${output[@]}" \
  "$repo_root"
if [[ "${PUSH:-true}" == "true" ]]; then
  "$repo_root/tools/verify_container_shape.sh" "${image}" "$arch"
else
  actual_platform="$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "${image}-${arch}")"
  if [[ "$actual_platform" != "linux/${arch}" ]]; then
    echo "loaded ${image}-${arch} has platform ${actual_platform}, expected linux/${arch}" >&2
    exit 1
  fi
fi
echo "done: ${image}-${arch}"
