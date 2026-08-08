#!/usr/bin/env bash
# Clean-tree assertion for the release pipeline (spec pose-release-clean-tree-attribution).
#
# goreleaser refuses to build from a dirty worktree, but it only finds out after
# every expensive gate has run, and its message names the files rather than the
# step that wrote them. Calling this after each step that could write turns a
# late, anonymous failure into an immediate, attributed one.
#
# Untracked build outputs the release itself produces are expected and passed in
# as allowed paths.
#
# Usage: bash tests/release/assert-clean-tree.sh "<step name>" [allowed-path...]
set -euo pipefail

step="${1:?usage: assert-clean-tree.sh \"<step name>\" [allowed-path...]}"
shift || true

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

dirty="$(git status --porcelain)"

# Drop the paths this step is allowed to create.
for allowed in "$@"; do
  dirty="$(printf '%s\n' "$dirty" | grep -v -- " $allowed\$" || true)"
done

dirty="$(printf '%s\n' "$dirty" | sed '/^[[:space:]]*$/d')"

if [ -z "$dirty" ]; then
  echo "clean-tree: OK after: $step"
  exit 0
fi

echo "clean-tree: FAIL: \"$step\" left the worktree dirty" >&2
echo "" >&2
printf '%s\n' "$dirty" >&2
echo "" >&2
echo "goreleaser refuses to build from a dirty tree. Failing here names the step" >&2
echo "instead of letting it surface later with only the file list." >&2
echo "Either stop the step writing into the worktree, or pass the path as an" >&2
echo "allowed argument to this assertion if the release is meant to produce it." >&2
exit 1
