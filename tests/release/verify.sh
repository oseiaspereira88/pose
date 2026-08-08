#!/usr/bin/env bash
# Artifact identity verification (specs pose-release-signing and
# pose-cyclonedx-sbom): every release archive and the checksum manifest must
# carry a Sigstore bundle that verifies against the pinned workflow identity,
# and every archive must ship a schema-valid CycloneDX SBOM naming the direct
# production dependencies. Used by release CI over dist-release/ and by
# consumers over downloaded assets. Shell is the harness, never the runtime.
#
# Usage: bash tests/release/verify.sh <artifact-dir> [identity-regexp]
set -euo pipefail

dir="${1:?usage: verify.sh <artifact-dir> [identity-regexp]}"
# Consumers should pin the tag form (see SECURITY.md); CI accepts any ref of
# the release workflow in this repository.
identity="${2:-^https://github.com/oseiaspereira88/pose/\.github/workflows/release\.yml@}"
issuer="https://token.actions.githubusercontent.com"

fail=0
say() { echo "verify: $*"; }
err() { echo "verify: FAIL: $*" >&2; fail=1; }

shopt -s nullglob
archives=("$dir"/pose_*.tar.gz "$dir"/pose_*.zip)
if [[ ${#archives[@]} -eq 0 ]]; then
  err "no release archives found in $dir"
fi

# R3 (signing): unsigned artifact or identity mismatch fails.
for artifact in "${archives[@]}" "$dir/checksums.txt"; do
  [[ -f "$artifact" ]] || { err "missing artifact: $artifact"; continue; }
  bundle="$artifact.sigstore.json"
  if [[ ! -f "$bundle" ]]; then
    err "missing signature bundle: $bundle"
    continue
  fi
  if ! cosign verify-blob \
    --bundle "$bundle" \
    --certificate-identity-regexp "$identity" \
    --certificate-oidc-issuer "$issuer" \
    "$artifact" >/dev/null 2>&1; then
    err "signature verification failed for $artifact (identity $identity)"
  else
    say "signature OK: $(basename "$artifact")"
  fi
done

# R1-R3 (SBOM): delegated to verify-sbom.sh so the same assertions can be
# exercised on their own by the negative harness — inside this script they sit
# behind signature verification, which a synthetic fixture can never pass.
if ! bash "$(dirname "$0")/verify-sbom.sh" "$dir"; then
  err "SBOM verification failed"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "verify: artifact identity verification FAILED" >&2
  exit 1
fi
say "all artifacts verified (signatures + SBOMs)"
