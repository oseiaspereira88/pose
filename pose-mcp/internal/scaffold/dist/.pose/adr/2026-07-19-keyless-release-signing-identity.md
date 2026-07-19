# ADR: Keyless release signing identity

## Status
Accepted (2026-07-19) — spec `pose-release-signing`

## Context

Releases shipped SHA-256 checksums only: they detect corruption but cannot
establish who built the artifact, and the checksum travels over the same
channel as the archive it protects. The spec requires verifiable identity for
every archive and the checksum manifest, without long-lived signing keys in
repository secrets.

Alternatives considered:

1. **Long-lived GPG/cosign key in repository secrets** — key management,
   rotation and exfiltration risk; explicitly excluded by the security
   requirement.
2. **GitHub artifact attestations only** (`actions/attest-build-provenance`)
   — establishes provenance but verification requires `gh` and GitHub
   availability; provenance is owned by the next milestone
   (`pose-slsa-provenance`) and complements rather than replaces signatures.
3. **Keyless Sigstore signing with offline-verifiable bundles** — workload
   identity from the GitHub OIDC issuer, no stored keys, `cosign verify-blob
   --bundle` works offline against the bundled certificate, signature and
   transparency-log entry.

## Decision

Option 3 ([Sigstore](https://docs.sigstore.dev/)):

- **What is signed:** every GoReleaser artifact — archives, SBOMs and
  `checksums.txt` (`signs.artifacts: all`) — each with a Sigstore bundle
  published beside it as `<artifact>.sigstore.json`. Checksums remain
  published: verification layers compose, they do not replace each other.
- **Identity policy (R2):** the only accepted signer is this repository's
  release workflow via the GitHub OIDC issuer. Consumers pin
  `--certificate-oidc-issuer https://token.actions.githubusercontent.com`
  and `--certificate-identity-regexp
  '^https://github.com/oseiaspereira88/pose/\.github/workflows/release\.yml@refs/tags/v'`.
  Cryptographic validity without these constraints accepts the wrong signer
  and is documented as insufficient in `SECURITY.md`.
- **Enforcement (R3):** the release workflow verifies every bundle with the
  pinned identity (`tests/release/verify.sh`) after GoReleaser runs; a
  missing bundle or identity mismatch fails the run. Snapshot rehearsals
  (workflow_dispatch) sign and verify the same way without publishing.
- **Contract test:** `TestArtifactIdentityContract` pins the GoReleaser
  signing config, the workflow's `id-token: write` scope, the pinned
  installer actions and the documented verification command.

## Consequences

- Positive: adopters authenticate the builder identity, not just file
  integrity; no signing key exists to rotate or leak.
- Positive: verification is scriptable and cross-platform (single `cosign`
  binary) and works offline from the bundle.
- Trade-off: trust shifts to the GitHub OIDC issuer and Sigstore
  infrastructure; acceptable for a GitHub-released open-source product and
  strictly better than same-channel checksums.
- Residual: provenance (how it was built, from what) is the next milestone;
  wrong-identity and tampered-artifact negative paths run in CI via
  `verify.sh` on every release.
