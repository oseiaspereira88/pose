#!/usr/bin/env bash
# Negative proof for the artifact-identity gate (spec pose-release-signing, R3).
#
# verify.sh has always contained the rejection logic, but no release run ever
# exercised it: every run verified artifacts that were correctly signed, so the
# gate's failure path was asserted and never demonstrated. Its R3 was waived for
# exactly that reason when the spec's trace was retrofitted.
#
# This builds deliberately broken artifact sets and requires verify.sh to reject
# each one. Shell is the harness, never the POSE runtime.
#
# Usage: bash tests/release/verify-negative.sh
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

verify="$repo_root/tests/release/verify.sh"
fail=0

# expect_rejection <case-name> <artifact-dir>
expect_rejection() {
  local name="$1" dir="$2"
  if bash "$verify" "$dir" >/dev/null 2>&1; then
    echo "verify-negative: FAIL: $name was accepted" >&2
    fail=1
  else
    echo "verify-negative: PASS: $name is rejected"
  fi
}

# A well-formed archive name with no signature bundle beside it.
missing_bundle="$work/missing-bundle"
mkdir -p "$missing_bundle"
echo "not a real archive" > "$missing_bundle/pose_9.9.9_linux_amd64.tar.gz"
: > "$missing_bundle/checksums.txt"
expect_rejection "an archive without a signature bundle" "$missing_bundle"

# Bundle present, SBOM absent: the SBOM requirement is part of the same gate.
missing_sbom="$work/missing-sbom"
mkdir -p "$missing_sbom"
echo "not a real archive" > "$missing_sbom/pose_9.9.9_linux_amd64.tar.gz"
echo "{}" > "$missing_sbom/pose_9.9.9_linux_amd64.tar.gz.sigstore.json"
: > "$missing_sbom/checksums.txt"
echo "{}" > "$missing_sbom/checksums.txt.sigstore.json"
expect_rejection "an archive without a CycloneDX SBOM" "$missing_sbom"

# No archives at all: an empty directory must not pass as "nothing to verify".
empty="$work/empty"
mkdir -p "$empty"
expect_rejection "an artifact set with no archives" "$empty"

# A malformed SBOM: present, but not a CycloneDX document.
bad_sbom="$work/bad-sbom"
mkdir -p "$bad_sbom"
echo "not a real archive" > "$bad_sbom/pose_9.9.9_linux_amd64.tar.gz"
echo "{}" > "$bad_sbom/pose_9.9.9_linux_amd64.tar.gz.sigstore.json"
echo '{"not":"cyclonedx"}' > "$bad_sbom/pose_9.9.9_linux_amd64.tar.gz.cdx.json"
: > "$bad_sbom/checksums.txt"
echo "{}" > "$bad_sbom/checksums.txt.sigstore.json"
expect_rejection "an archive with a malformed SBOM" "$bad_sbom"

if [[ "$fail" -eq 0 ]]; then
  echo "verify-negative: all rejection paths exercised"
else
  echo "verify-negative: the identity gate accepted something it must reject" >&2
fi
exit "$fail"
