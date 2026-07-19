#!/usr/bin/env bash
# Release compatibility gate (spec pose-release-compatibility-matrix): proves
# that the candidate binary, repository schema, scaffold, MCP metadata and
# public documentation are mutually compatible, and exercises every supported
# prior-version upgrade declared in compatibility.json with pinned,
# authenticated artifacts. Shell is the test harness, never the POSE runtime.
#
# Usage: bash tests/release/compat.sh [vX.Y.Z]
# The optional tag must match compatibility.json engine_version.
set -euo pipefail

tag="${1:-}"
repo_root="$(git rev-parse --show-toplevel)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
matrix="$repo_root/compatibility.json"
report="${COMPAT_REPORT:-$repo_root/compatibility-report.md}"

engine_version="$(jq -r .engine_version "$matrix")"
schema_version="$(jq -r .schema_version "$matrix")"

if [[ -n "$tag" && "$tag" != "v$engine_version" ]]; then
  echo "compat: tag $tag does not match compatibility.json engine_version $engine_version" >&2
  exit 1
fi

sha_cmd="sha256sum"
command -v "$sha_cmd" >/dev/null 2>&1 || sha_cmd="shasum -a 256"

{
  echo "# POSE release compatibility report"
  echo
  echo "- candidate: ${tag:-"(untagged candidate)"} / engine_version $engine_version"
  echo "- schema_version: $schema_version"
  echo "- commit: $(git -C "$repo_root" rev-parse HEAD)"
  echo
} > "$report"

overall=0
gate() {
  local title="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo "- PASS: $title" >> "$report"
  else
    echo "- FAIL: $title" >> "$report"
    overall=1
  fi
}

# Candidate binary stamped exactly like the release pipeline stamps it.
candidate="$work/pose"
(cd "$repo_root/pose-mcp" && go build -ldflags "-s -w -X github.com/harne8/pose-mcp/internal/version.Version=$engine_version" -o "$candidate" ./cmd/pose)
got="$("$candidate" version | awk 'NR==1{print $2}')"
if [[ "$got" != "$engine_version" ]]; then
  echo "compat: candidate reports $got, want $engine_version" >&2
  exit 1
fi
echo "- PASS: candidate binary reports $engine_version on every surface (stamped build)" >> "$report"

echo >> "$report"
echo "## Contract gates (same candidate tree)" >> "$report"
gate "version contract (CLI, MCP, registry, release pipeline)" \
  go -C "$repo_root/pose-mcp" test ./internal/version/... -count=1
gate "MCP catalog conformance (golden, docs, registry, schemas)" \
  go -C "$repo_root/pose-mcp" test ./internal/mcpserver -run 'Catalog|Initialize|ToolsList' -count=1
gate "compatibility matrix + schema upgrade fixtures" \
  go -C "$repo_root/pose-mcp" test ./internal/cli -run 'Compat|Version|Install|Doctor' -count=1
gate "embedded scaffold parity" \
  go -C "$repo_root/pose-mcp" test ./internal/scaffold -count=1
gate "installer E2E (fresh install, verified download, doctor, strict gate)" \
  bash "$repo_root/tests/install/run.sh"

echo >> "$report"
echo "## Supported prior-version upgrades" >> "$report"
count="$(jq '.supported_upgrades | length' "$matrix")"
if [[ "$count" -eq 0 ]]; then
  echo "- none declared: support window starts at $engine_version (see support_policy.window)" >> "$report"
else
  os="$(go env GOOS)"
  arch="$(go env GOARCH)"
  for i in $(seq 0 $((count - 1))); do
    from="$(jq -r ".supported_upgrades[$i].from" "$matrix")"
    pin="$(jq -r ".supported_upgrades[$i].checksums_sha256" "$matrix")"
    base_url="https://github.com/oseiaspereira88/pose/releases/download/v$from"
    prior_dir="$work/prior-$from"
    mkdir -p "$prior_dir"
    (
      cd "$prior_dir"
      curl -fsSLO "$base_url/checksums.txt"
      echo "$pin  checksums.txt" | $sha_cmd --check - >/dev/null
      asset="pose_${from}_${os}_${arch}.tar.gz"
      curl -fsSLO "$base_url/$asset"
      $sha_cmd --check --ignore-missing checksums.txt >/dev/null
      tar -xzf "$asset" pose
    )
    fixture="$work/fixture-$from"
    mkdir -p "$fixture"
    git -C "$fixture" init -q
    if "$prior_dir/pose" install "$fixture" --skip-mcp >/dev/null \
      && (cd "$fixture" && "$candidate" upgrade >/dev/null && "$candidate" check --strict >/dev/null); then
      echo "- PASS: $from → $engine_version (verified artifact, install → upgrade → strict gate)" >> "$report"
    else
      echo "- FAIL: $from → $engine_version" >> "$report"
      overall=1
    fi
  done
fi

echo >> "$report"
if [[ "$overall" -eq 0 ]]; then
  echo "Result: COMPATIBLE — release gate passed." >> "$report"
else
  echo "Result: INCOMPATIBLE — do not release this candidate." >> "$report"
fi
cat "$report"
exit "$overall"
