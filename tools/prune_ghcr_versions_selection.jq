# Given a GitHub Container Registry package-versions array and $keep, emit the ids
# of the versions to delete: everything belonging to a release older than the
# newest $keep releases. A release is a version tagged vMAJOR.MINOR.PATCH; its
# per-arch images are tagged <release>-arm64 / <release>-amd64.
def is_release_tag: test("^v[0-9]+\\.[0-9]+\\.[0-9]+$");
def release_tags: [.metadata.container.tags[]? | select(is_release_tag)];

(map(select((release_tags | length) > 0)) | sort_by(.created_at) | reverse) as $releases
| ($releases[$keep:] | map(release_tags[])) as $old_releases
| ($old_releases | map(., . + "-arm64", . + "-amd64") | unique) as $doomed_tags
| map(select(any(.metadata.container.tags[]?; . as $t | $doomed_tags | index($t))))
| .[].id
