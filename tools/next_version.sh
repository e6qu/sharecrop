#!/usr/bin/env bash
set -euo pipefail

# Computes the next release version from conventional commits since the latest
# vMAJOR.MINOR.PATCH tag and prints it as "vX.Y.Z" on stdout, or prints nothing
# when no commit since that tag warrants a release. Bump rules (conventional
# commits, https://www.conventionalcommits.org):
#
#   - a "!" after the type/scope (e.g. feat!:) or a "BREAKING CHANGE" footer -> major
#   - feat            -> minor
#   - fix | perf      -> patch
#   - anything else (chore, docs, refactor, test, ci, build, style, ...) -> no release
#
# Because merges squash to the PR title (see AGENTS.md), PR titles must follow the
# conventional-commit format for this to bump correctly.

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
re_fix='^(fix|perf)(\(.+\))?:'

bump="none"
while IFS= read -r line; do
  if [[ "$line" =~ $re_breaking ]] || [[ "$line" == *"BREAKING CHANGE"* ]]; then
    bump="major"
    break
  elif [[ "$line" =~ $re_feat ]]; then
    [[ "$bump" != "major" ]] && bump="minor"
  elif [[ "$line" =~ $re_fix ]]; then
    [[ "$bump" == "none" ]] && bump="patch"
  fi
done < <(git log "$range" --format='%s%n%b')

case "$bump" in
  major) echo "v$((major + 1)).0.0" ;;
  minor) echo "v${major}.$((minor + 1)).0" ;;
  patch) echo "v${major}.${minor}.$((patch + 1))" ;;
  none) : ;;
esac
