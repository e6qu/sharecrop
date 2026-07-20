#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp)"
trap 'rm -f "$fixture"' EXIT

cat > "$fixture" <<'JSON'
[
  {"id":101,"created_at":"2026-07-20T03:00:00Z","metadata":{"container":{"tags":["aaaaaaaaaaaa"]}}},
  {"id":102,"created_at":"2026-07-20T02:59:00Z","metadata":{"container":{"tags":["aaaaaaaaaaaa-arm64"]}}},
  {"id":103,"created_at":"2026-07-20T02:59:00Z","metadata":{"container":{"tags":["aaaaaaaaaaaa-amd64"]}}},
  {"id":201,"created_at":"2026-07-20T02:00:00Z","metadata":{"container":{"tags":["bbbbbbbbbbbb"]}}},
  {"id":202,"created_at":"2026-07-20T01:59:00Z","metadata":{"container":{"tags":["bbbbbbbbbbbb-arm64"]}}},
  {"id":203,"created_at":"2026-07-20T01:59:00Z","metadata":{"container":{"tags":["bbbbbbbbbbbb-amd64"]}}},
  {"id":301,"created_at":"2026-07-20T01:00:00Z","metadata":{"container":{"tags":["cccccccccccc"]}}},
  {"id":302,"created_at":"2026-07-20T00:59:00Z","metadata":{"container":{"tags":["cccccccccccc-arm64"]}}},
  {"id":303,"created_at":"2026-07-20T00:59:00Z","metadata":{"container":{"tags":["cccccccccccc-amd64"]}}},
  {"id":401,"created_at":"2026-07-19T23:00:00Z","metadata":{"container":{"tags":["v0.3.5"]}}},
  {"id":402,"created_at":"2026-07-19T22:00:00Z","metadata":{"container":{"tags":["main"]}}},
  {"id":403,"created_at":"2026-07-19T21:00:00Z","metadata":{"container":{"tags":[]}}},
  {"id":404,"created_at":"2026-07-20T04:00:00Z","metadata":{"container":{"tags":["dddddddddddd"]}}},
  {"id":405,"created_at":"2026-07-20T03:30:00Z","metadata":{"container":{"tags":["aaaaaaaaaaaa","latest"]}}}
]
JSON

actual="$(jq -r --argjson keep 2 -f "$root/tools/prune_ghcr_versions_selection.jq" "$fixture" | sort -n)"
expected="$(printf '301\n302\n303\n401\n402\n403\n404\n405')"
if [[ "$actual" != "$expected" ]]; then
  echo "unexpected GitHub Container Registry retention selection" >&2
  diff -u <(printf '%s\n' "$expected") <(printf '%s\n' "$actual") >&2
  exit 1
fi

shellcheck \
  "$root/tools/build_container.sh" \
  "$root/tools/prune_ghcr_versions.sh" \
  "$root/tools/test_release_contract.sh" \
  "$root/tools/verify_container_shape.sh"

echo "release tag, shape-verifier, and retention contracts passed"
