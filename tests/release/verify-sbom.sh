#!/usr/bin/env bash
# SBOM assertions for a release artifact set (specs pose-cyclonedx-sbom and
# pose-sbom-license-coverage-gate).
#
# Split out of verify.sh so the negative harness can reach them. Inside
# verify.sh these run after signature verification, which means a synthetic
# fixture — unsigned by construction — is rejected before any SBOM assertion is
# evaluated, and the rejection paths that matter here were unreachable. Every
# SBOM guarantee was therefore asserted and never demonstrated.
#
# Shell is the harness, never the POSE runtime.
#
# Usage: bash tests/release/verify-sbom.sh <artifact-dir>
# Env:   SBOM_MIN_LICENSE_PCT (default 75)
set -euo pipefail

dir="${1:?usage: verify-sbom.sh <artifact-dir>}"
min_pct="${SBOM_MIN_LICENSE_PCT:-75}"

fail=0
say() { echo "verify-sbom: $*"; }
err() { echo "verify-sbom: FAIL: $*" >&2; fail=1; }

shopt -s nullglob
archives=("$dir"/pose_*.tar.gz "$dir"/pose_*.zip)
if [[ ${#archives[@]} -eq 0 ]]; then
  err "no release archives found in $dir"
  exit 1
fi

# Direct production dependencies must be named by the inventory.
go_mod="$(dirname "$0")/../../pose-mcp/go.mod"
direct_deps=()
if [[ -f "$go_mod" ]]; then
  mapfile -t direct_deps < <(awk '/^require \(/{grab=1;next} /^\)/{grab=0} grab && $0 !~ /indirect/ {print $1} /^require [^(]/ && $0 !~ /indirect/ {print $2}' "$go_mod")
fi

for artifact in "${archives[@]}"; do
  sbom="$artifact.cdx.json"
  if [[ ! -f "$sbom" ]]; then
    err "missing SBOM: $sbom"
    continue
  fi
  if ! jq -e '.bomFormat == "CycloneDX" and (.specVersion | length > 0) and (.components | length > 0)' "$sbom" >/dev/null 2>&1; then
    err "SBOM schema check failed for $(basename "$sbom") (bomFormat/specVersion/components)"
    continue
  fi
  for dep in "${direct_deps[@]}"; do
    if ! grep -q "$dep" "$sbom"; then
      err "SBOM $(basename "$sbom") is missing direct production dependency $dep"
    fi
  done
  # License resolution depends on a syft setting and a populated module cache,
  # either of which can stop working without failing the build: before that
  # setting was added, releases shipped 1 component of 27 with a license while
  # the docs advertised an SBOM'd release. Measured coverage is 24/27 (88%); the
  # floor sits well below that so ordinary dependency churn does not trip it,
  # and far above the ~4% an outright collapse of the mechanism produces.
  read -r licensed total < <(jq -r '[.components[]] as $c
    | "\([$c[] | select(.licenses and (.licenses | length > 0))] | length) \($c | length)"' "$sbom")
  if [[ "$total" -eq 0 ]]; then
    err "SBOM $(basename "$sbom") has no components to measure license coverage against"
  elif (( licensed * 100 < min_pct * total )); then
    err "SBOM $(basename "$sbom") resolves licenses for $licensed/$total components, below the $min_pct% floor"
  else
    say "license coverage OK: $licensed/$total in $(basename "$sbom")"
  fi
  say "SBOM OK: $(basename "$sbom")"
done

exit "$fail"
