#!/usr/bin/env bash
set -euo pipefail

# Prunes old container image versions from the GitHub Container Registry, keeping
# only the most recent N release versions (vMAJOR.MINOR.PATCH) and their per-arch
# (-arm64 / -amd64) images. Best-effort: it needs `gh` authenticated with a token
# that has packages write/delete on the package, and it is safe to run repeatedly.
#
# Usage: tools/prune_ghcr_versions.sh <owner> <package> [keep]
#   keep defaults to 25.

owner="${1:?usage: prune_ghcr_versions.sh <owner> <package> [keep]}"
package="${2:?usage: prune_ghcr_versions.sh <owner> <package> [keep]}"
keep="${3:-25}"

owner_type="$(gh api "/users/${owner}" --jq .type)"
case "$owner_type" in
  Organization) base="/orgs/${owner}/packages/container/${package}/versions" ;;
  User) base="/users/${owner}/packages/container/${package}/versions" ;;
  *)
    echo "unknown owner type: ${owner_type}" >&2
    exit 1
    ;;
esac

versions="$(gh api --paginate "$base")"

# Select the ids to delete: sort the release-tagged versions newest-first, drop
# the newest $keep, then take the release tags of the rest plus their
# -arm64/-amd64 siblings, and emit every version id carrying one of those tags.
# See tools/prune_ghcr_versions_selection.jq for the standalone, tested filter.
ids="$(printf '%s' "$versions" | jq -r --argjson keep "$keep" -f "$(dirname "${BASH_SOURCE[0]}")/prune_ghcr_versions_selection.jq")"

count=0
for id in $ids; do
  if gh api -X DELETE "${base}/${id}" >/dev/null; then
    count=$((count + 1))
  fi
done
echo "pruned ${count} image version(s); kept the newest ${keep} release(s)"
