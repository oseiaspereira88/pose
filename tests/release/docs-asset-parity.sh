#!/usr/bin/env bash
# Documented-install parity (spec pose-docs-asset-parity).
#
# Every `releases/download/<version>/<asset>` URL the documentation hands a
# user must resolve to an asset the release actually publishes. `pose.rb` was
# documented as a copyable brew command for four releases while never being
# uploaded, so that URL 404'd; nothing noticed, because nothing compared the
# two sides.
#
# Extracts the asset names the docs promise, expands the version placeholders,
# and requires each to be present in the published release. Shell is the
# harness, never the POSE runtime.
#
# Usage: bash tests/release/docs-asset-parity.sh vX.Y.Z
# Requires: gh (authenticated read).
set -euo pipefail

tag="${1:?usage: docs-asset-parity.sh vX.Y.Z}"
version="${tag#v}"
repo="${DOCS_PARITY_REPO:-oseiaspereira88/pose}"
repo_root="$(git rev-parse --show-toplevel)"

# Documentation sources that hand a user a download URL.
sources=(
  "$repo_root/README.md"
  "$repo_root/docs-site/docs/package-channels.md"
  "$repo_root/docs-site/docs/ci.md"
)

# Collect the asset names the docs reference, with placeholders expanded.
# Both `vX.Y.Z` (literal placeholder) and `v${V}`/`v$V` (shell-substituted in a
# copyable snippet) stand for the release version.
documented="$(
  for src in "${sources[@]}"; do
    [ -f "$src" ] || continue
    grep -oE 'releases/download/[^ )`"]+' "$src" || true
  done \
    | sed -e "s|releases/download/[^/]*/||" \
    | sed -e "s|\${V}|$version|g" -e "s|\$V|$version|g" \
    | sort -u
)"

if [ -z "$documented" ]; then
  echo "docs-asset-parity: no documented download URLs found — the extraction is broken, not the docs" >&2
  exit 1
fi

published="$(gh release view "$tag" --repo "$repo" --json assets --jq '.assets[].name' | sort -u)"

if [ -z "$published" ]; then
  echo "docs-asset-parity: release $tag publishes no assets" >&2
  exit 1
fi

fail=0
while IFS= read -r asset; do
  [ -n "$asset" ] || continue
  if printf '%s\n' "$published" | grep -Fxq "$asset"; then
    echo "docs-asset-parity: PASS: $asset is published"
  else
    echo "docs-asset-parity: FAIL: the docs link to $asset, which $tag does not publish" >&2
    fail=1
  fi
done <<< "$documented"

if [ "$fail" -eq 0 ]; then
  echo "docs-asset-parity: every documented download resolves"
else
  echo "docs-asset-parity: the documentation promises assets the release does not publish" >&2
fi
exit "$fail"
