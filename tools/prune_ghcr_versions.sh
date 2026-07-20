#!/usr/bin/env bash
set -euo pipefail

# Prunes old container image versions from the GitHub Container Registry, keeping
# only the most recent N immutable 12-character commit-SHA releases and their
# per-architecture (-arm64 / -amd64) images. It needs `gh` authenticated with a
# token that has packages write/delete on the package.
#
# Usage: tools/prune_ghcr_versions.sh <owner> <package> [keep]
#   keep defaults to 20.

owner="${1:?usage: prune_ghcr_versions.sh <owner> <package> [keep]}"
package="${2:?usage: prune_ghcr_versions.sh <owner> <package> [keep]}"
keep="${3:-20}"

if [[ ! "$keep" =~ ^[1-9][0-9]*$ ]]; then
  echo "keep must be a positive integer (got ${keep})" >&2
  exit 2
fi

owner_type="$(gh api "/users/${owner}" --jq .type)"
case "$owner_type" in
  Organization) base="/orgs/${owner}/packages/container/${package}/versions" ;;
  User) base="/users/${owner}/packages/container/${package}/versions" ;;
  *)
    echo "unknown owner type: ${owner_type}" >&2
    exit 1
    ;;
esac

versions_file="$(mktemp)"
trap 'rm -f "$versions_file"' EXIT
gh api --paginate "${base}?per_page=100" | jq -s 'add' > "$versions_file"

# Select the ids to delete: keep the newest $keep commit-SHA roots and their
# architecture siblings, then emit every other tagged package version. Untagged
# child manifests are left for GitHub Container Registry garbage collection.
# See tools/prune_ghcr_versions_selection.jq for the standalone, tested filter.
ids="$(jq -r --argjson keep "$keep" -f "$(dirname "${BASH_SOURCE[0]}")/prune_ghcr_versions_selection.jq" "$versions_file")"

count=0
for id in $ids; do
  if gh api -X DELETE "${base}/${id}" >/dev/null; then
    count=$((count + 1))
  fi
done
echo "pruned ${count} image version(s); kept the newest ${keep} release(s)"
