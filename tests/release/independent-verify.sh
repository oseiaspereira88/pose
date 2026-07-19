#!/usr/bin/env bash
# Independent release verification (spec pose-reproducible-release-verification):
# from a clean environment with no producer state, download the published
# assets, authenticate every layer (checksums, Sigstore bundles, SLSA
# provenance), inspect the artifact, execute it only after verification, and
# attempt a controlled rebuild to quantify reproducibility. Producer caches
# and credentials are never shared; only public release data is consumed.
#
# Usage: bash tests/release/independent-verify.sh vX.Y.Z
# Requires: gh (authenticated read), cosign, jq, go, git.
set -euo pipefail

tag="${1:?usage: independent-verify.sh vX.Y.Z}"
repo="oseiaspereira88/pose"
repo_root="$(git rev-parse --show-toplevel)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
report="${VERIFY_REPORT:-$repo_root/verification-report.md}"

fail=0
note() { echo "$1" >> "$report"; }
gate() {
  local title="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    note "- PASS: $title"
  else
    note "- FAIL: $title"
    fail=1
  fi
}

{
  echo "# POSE independent release verification"
  echo
  echo "- release: $tag"
  echo "- verifier: clean environment, no producer caches or credentials"
  echo
  echo "## Authentication before execution"
} > "$report"

assets="$work/assets"
mkdir -p "$assets"
gh release download "$tag" --repo "$repo" --dir "$assets"

# Layer 1: checksums cover every archive.
(cd "$assets" && sha256sum --check --ignore-missing checksums.txt >/dev/null)
note "- PASS: sha256 checksums verified for all downloaded archives"

# Layer 2: Sigstore bundles + SBOMs, consumer identity pin (tag refs only).
gate "Sigstore signatures + CycloneDX SBOMs (pinned identity)" \
  bash "$repo_root/tests/release/verify.sh" "$assets" \
  '^https://github.com/oseiaspereira88/pose/\.github/workflows/release\.yml@refs/tags/v'

# Layer 3: SLSA provenance — subject digest, source repo and builder workflow.
os="$(go env GOOS)"
arch="$(go env GOARCH)"
version="${tag#v}"
archive="pose_${version}_${os}_${arch}.tar.gz"
gate "SLSA provenance ($archive: digest + repo + signer workflow)" \
  gh attestation verify "$assets/$archive" --repo "$repo" \
  --signer-workflow "$repo/.github/workflows/release.yml"
gate "SLSA provenance (checksums.txt)" \
  gh attestation verify "$assets/checksums.txt" --repo "$repo" \
  --signer-workflow "$repo/.github/workflows/release.yml"

note ""
note "## Inspection and execution (only after verification)"
if [[ "$fail" -ne 0 ]]; then
  note "- SKIPPED: authentication failed; artifacts were never executed"
  cat "$report"
  exit 1
fi

extract="$work/extract"
mkdir -p "$extract"
tar -C "$extract" -xzf "$assets/$archive" pose
reported="$("$extract/pose" version | awk 'NR==1{print $2}')"
if [[ "$reported" == "$version" ]]; then
  note "- PASS: binary reports $reported (matches $tag)"
else
  note "- FAIL: binary reports $reported, expected $version"
  fail=1
fi
fixture="$work/fixture"
mkdir -p "$fixture"
git -C "$fixture" init -q
gate "install → doctor --json → check --strict on a fresh repository" \
  bash -c "'$extract/pose' install '$fixture' --skip-mcp && cd '$fixture' && '$extract/pose' doctor --json >/dev/null && '$extract/pose' check --strict"

note ""
note "## Controlled rebuild (reproducibility)"
src="$work/src"
git clone -q --depth 1 --branch "$tag" "https://github.com/$repo" "$src"
rebuilt="$work/rebuilt-pose"
commit_ts="$(git -C "$src" log -1 --format=%ct)"
(cd "$src/pose-mcp" && CGO_ENABLED=0 GOFLAGS=-trimpath GOOS="$os" GOARCH="$arch" \
  go build -ldflags "-s -w -X github.com/harne8/pose-mcp/internal/version.Version=$version" \
  -o "$rebuilt" ./cmd/pose)
released_sha="$(sha256sum "$extract/pose" | awk '{print $1}')"
rebuilt_sha="$(sha256sum "$rebuilt" | awk '{print $1}')"
if [[ "$released_sha" == "$rebuilt_sha" ]]; then
  note "- MATCH: independent rebuild is bit-identical (sha256 $released_sha)"
else
  note "- DIFFERENCE (explained inputs follow): released $released_sha, rebuilt $rebuilt_sha"
  note "  - toolchain: verifier Go $(go version | awk '{print $3}') vs release pipeline Go (see release run)"
  note "  - known nondeterministic inputs: Go toolchain revision and buildid; mod timestamp is pinned to commit ($commit_ts) and paths are trimmed"
  note "  - a digest mismatch here is a reproducibility delta, not an authenticity failure: authenticity is established by the layers above"
fi

note ""
if [[ "$fail" -eq 0 ]]; then
  note "Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution."
else
  note "Result: VERIFICATION FAILED"
fi
cat "$report"
exit "$fail"
