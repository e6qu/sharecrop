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
remaining_versions_file="$(mktemp)"
trap 'rm -f "$versions_file" "$remaining_versions_file"' EXIT
gh api --paginate "${base}?per_page=100" | jq -s 'add' > "$versions_file"

# Select the ids to delete: keep the newest $keep complete commit-SHA roots and
# their architecture siblings, then emit every untagged, incomplete, mixed-tag,
# unrecognized, or older package version.
# See tools/prune_ghcr_versions_selection.jq for the standalone, tested filter.
ids="$(jq -r --argjson keep "$keep" -f "$(dirname "${BASH_SOURCE[0]}")/prune_ghcr_versions_selection.jq" "$versions_file")"

count=0
for id in $ids; do
  if gh api -X DELETE "${base}/${id}" >/dev/null; then
    count=$((count + 1))
  fi
done

gh api --paginate "${base}?per_page=100" | jq -s 'add' > "$remaining_versions_file"

remaining_releases="$(jq '[.[].metadata.container.tags[]? | select(test("^[0-9a-f]{12}$"))] | unique | length' "$remaining_versions_file")"
if ((remaining_releases > keep)); then
  echo "retained ${remaining_releases} releases; expected at most ${keep}" >&2
  exit 1
fi

remaining_unrecognized="$(jq '[.[] | select(
  (.metadata.container.tags | length) == 0
  or any(.metadata.container.tags[]; test("^[0-9a-f]{12}(-(amd64|arm64))?$"; "i") | not)
)] | length' "$remaining_versions_file")"
if ((remaining_unrecognized > 0)); then
  echo "retained ${remaining_unrecognized} untagged or non-release package version(s)" >&2
  exit 1
fi

remaining_incomplete="$(jq '[.[].metadata.container.tags[]?] as $tags
  | [ $tags[]
      | select(test("^[0-9a-f]{12}$"))
      | . as $tag
      | select(
          ($tags | index($tag + "-amd64")) == null
          or ($tags | index($tag + "-arm64")) == null
        )
    ] | unique | length' "$remaining_versions_file")"
if ((remaining_incomplete > 0)); then
  echo "retained ${remaining_incomplete} incomplete immutable release(s)" >&2
  exit 1
fi

remaining_versions="$(jq 'length' "$remaining_versions_file")"
maximum_versions=$((keep * 3))
if ((remaining_versions > maximum_versions)); then
  echo "retained ${remaining_versions} package versions; expected at most ${maximum_versions}" >&2
  exit 1
fi

echo "pruned ${count} image version(s); retained ${remaining_releases} immutable release(s) across ${remaining_versions} package version(s)"
