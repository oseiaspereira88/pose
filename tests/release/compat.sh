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
  go -C "$repo_root/pose-mcp" test ./internal/cli -run 'Compat|Version|Install|Doctor|Upgrade' -count=1
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

  # Upgrade compatibility lab (spec pose-upgrade-compatibility-lab): each
  # N-minus pair is exercised against a *populated* instance — pt-BR locale
  # content, a user-modified managed file, a real spec and knowledge note —
  # not a bare fresh install, then proves the candidate's upgrade is
  # idempotent and preserves everything untouched. Mirrors the depth of the
  # network-free Go fixtures in internal/cli/upgrade_test.go.
  check_upgrade_pair() {
    local prior_bin="$1" fixture="$2"
    mkdir -p "$fixture"
    git -C "$fixture" init -q
    "$prior_bin" install "$fixture" --locale pt-BR --skip-mcp >/dev/null || return 1
    (
      cd "$fixture"
      "$prior_bin" new-spec upgrade-lab-fixture >/dev/null
      "$prior_bin" new-knowledge handoff upgrade-lab-fixture --owner @pose-maintainers >/dev/null
      printf '\n<!-- upgrade-lab: user customization preserved across upgrade -->\n' >> AGENTS.md
    ) || return 1
    local before after
    before="$($sha_cmd "$fixture/AGENTS.md" | awk '{print $1}')"
    (cd "$fixture" && "$candidate" upgrade >/dev/null) || return 1
    (cd "$fixture" && "$candidate" check --strict >/dev/null) || return 1
    local reapply
    reapply="$(cd "$fixture" && "$candidate" upgrade)"
    [[ "$reapply" == *"already at schema"* ]] || return 1
    after="$($sha_cmd "$fixture/AGENTS.md" | awk '{print $1}')"
    [[ "$before" == "$after" ]] || return 1
    [[ -f "$fixture/.pose/specs/upgrade-lab-fixture/spec.md" ]] || return 1
    compgen -G "$fixture/.pose/knowledge/*upgrade-lab-fixture*.md" >/dev/null || return 1
  }

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
    if check_upgrade_pair "$prior_dir/pose" "$fixture"; then
      echo "- PASS: $from → $engine_version (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)" >> "$report"
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
