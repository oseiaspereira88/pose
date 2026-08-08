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

# --- SBOM assertions, reachable now that they are their own script ---
#
# Inside verify.sh these sit behind signature verification, so a synthetic
# fixture is rejected before any of them is evaluated. Every SBOM guarantee was
# asserted and never demonstrated — the same gap this file was written to close
# for the signature path (spec pose-sbom-negative-coverage).
sbom_verify="$repo_root/tests/release/verify-sbom.sh"

# expect_sbom_rejection <case-name> <artifact-dir> [env-assignment...]
expect_sbom_rejection() {
  local name="$1" dir="$2"
  shift 2
  if env "$@" bash "$sbom_verify" "$dir" >/dev/null 2>&1; then
    echo "verify-negative: FAIL: $name was accepted" >&2
    fail=1
  else
    echo "verify-negative: PASS: $name is rejected"
  fi
}

# A well-formed CycloneDX SBOM whose components carry no license at all — the
# state four releases actually shipped.
no_licenses="$work/no-licenses"
mkdir -p "$no_licenses"
echo "archive" > "$no_licenses/pose_9.9.9_linux_amd64.tar.gz"
python3 - "$no_licenses/pose_9.9.9_linux_amd64.tar.gz.cdx.json" <<'PYGEN'
import json, sys
comps = [{"name": f"mod{i}", "version": "1.0.0", "type": "library"} for i in range(20)]
json.dump({"bomFormat": "CycloneDX", "specVersion": "1.6", "components": comps}, open(sys.argv[1], "w"))
PYGEN
expect_sbom_rejection "an SBOM resolving no licenses" "$no_licenses" SBOM_MIN_LICENSE_PCT=75

# Coverage below the floor but not zero: the regression that degrades quietly.
thin_licenses="$work/thin-licenses"
mkdir -p "$thin_licenses"
echo "archive" > "$thin_licenses/pose_9.9.9_linux_amd64.tar.gz"
python3 - "$thin_licenses/pose_9.9.9_linux_amd64.tar.gz.cdx.json" <<'PYGEN'
import json, sys
comps = []
for i in range(20):
    c = {"name": f"mod{i}", "version": "1.0.0", "type": "library"}
    if i < 10:  # 50%, under the 75% floor
        c["licenses"] = [{"license": {"id": "MIT"}}]
    comps.append(c)
json.dump({"bomFormat": "CycloneDX", "specVersion": "1.6", "components": comps}, open(sys.argv[1], "w"))
PYGEN
expect_sbom_rejection "an SBOM below the license-coverage floor" "$thin_licenses" SBOM_MIN_LICENSE_PCT=75

# An SBOM with zero components must not divide by zero into a pass.
empty_components="$work/empty-components"
mkdir -p "$empty_components"
echo "archive" > "$empty_components/pose_9.9.9_linux_amd64.tar.gz"
echo '{"bomFormat":"CycloneDX","specVersion":"1.6","components":[]}' \
  > "$empty_components/pose_9.9.9_linux_amd64.tar.gz.cdx.json"
expect_sbom_rejection "an SBOM with no components" "$empty_components"

# The positive control: the same harness must accept a compliant inventory, or
# the rejections above prove nothing.
good_sbom="$work/good-sbom"
mkdir -p "$good_sbom"
echo "archive" > "$good_sbom/pose_9.9.9_linux_amd64.tar.gz"
# The inventory must also name every direct production dependency, so the
# control is built from the real go.mod rather than from invented module names.
python3 - "$good_sbom/pose_9.9.9_linux_amd64.tar.gz.cdx.json" "$repo_root/pose-mcp/go.mod" <<'PYGEN'
import json, re, sys
out, gomod = sys.argv[1], sys.argv[2]
deps, block = [], False
for line in open(gomod):
    if line.startswith("require ("):
        block = True; continue
    if block and line.startswith(")"):
        block = False; continue
    if "indirect" in line:
        continue
    m = re.match(r"\s*([\w.\-/]+)\s+v", line) if block else re.match(r"require\s+([\w.\-/]+)\s+v", line)
    if m:
        deps.append(m.group(1))
comps = [{"name": d, "version": "1.0.0", "type": "library",
          "licenses": [{"license": {"id": "MIT"}}]} for d in deps]
# Pad to 20 components, keeping coverage at 90% — above the 75% floor.
while len(comps) < 20:
    i = len(comps)
    c = {"name": f"pad{i}", "version": "1.0.0", "type": "library"}
    if len([x for x in comps if "licenses" in x]) * 100 < 90 * (len(comps) + 1):
        c["licenses"] = [{"license": {"id": "MIT"}}]
    comps.append(c)
json.dump({"bomFormat": "CycloneDX", "specVersion": "1.6", "components": comps}, open(out, "w"))
PYGEN
if SBOM_MIN_LICENSE_PCT=75 bash "$sbom_verify" "$good_sbom" >/dev/null 2>&1; then
  echo "verify-negative: PASS: a compliant SBOM is accepted"
else
  echo "verify-negative: FAIL: a compliant SBOM was rejected — the harness rejects everything" >&2
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "verify-negative: all rejection paths exercised"
else
  echo "verify-negative: the identity gate accepted something it must reject" >&2
fi
exit "$fail"
