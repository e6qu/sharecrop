#!/usr/bin/env bash
set -euo pipefail

# Computes the next release version from conventional commits since the latest
# vMAJOR.MINOR.PATCH tag and prints it as "vX.Y.Z" on stdout. Every merge to main
# builds, so this always bumps at least a patch. Bump rules (conventional commits,
# https://www.conventionalcommits.org):
#
#   - a "!" after the type/scope (e.g. feat!:) or a "BREAKING CHANGE" footer -> major
#   - feat                                        -> minor
#   - anything else (fix, perf, chore, docs, ...) -> patch (the default)
#
# Because merges squash to the PR title (see AGENTS.md), PR titles should follow
# the conventional-commit format so feat/breaking changes bump the right part.

latest="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1 || true)"
if [[ -z "$latest" ]]; then
  base="v0.0.0"
  range="HEAD"
else
  base="$latest"
  range="${latest}..HEAD"
fi

version="${base#v}"
major="${version%%.*}"
rest="${version#*.}"
minor="${rest%%.*}"
patch="${rest#*.}"

# Regexes in variables so bash's [[ =~ ]] treats the parens/backslashes as ERE
# rather than the shell processing them.
re_breaking='^[a-z]+(\(.+\))?!:'
re_feat='^feat(\(.+\))?:'

# Default to a patch bump so every merge builds; feat and breaking changes raise
# it to minor/major.
bump="patch"
while IFS= read -r line; do
  if [[ "$line" =~ $re_breaking ]] || [[ "$line" == *"BREAKING CHANGE"* ]]; then
    bump="major"
    break
  elif [[ "$line" =~ $re_feat ]]; then
    [[ "$bump" != "major" ]] && bump="minor"
  fi
done < <(git log "$range" --format='%s%n%b')

case "$bump" in
  major) echo "v$((major + 1)).0.0" ;;
  minor) echo "v${major}.$((minor + 1)).0" ;;
  patch) echo "v${major}.${minor}.$((patch + 1))" ;;
esac
