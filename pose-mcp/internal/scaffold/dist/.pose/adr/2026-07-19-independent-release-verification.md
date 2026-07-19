# ADR: Independent release verification

## Status
Accepted (2026-07-19) — spec `pose-reproducible-release-verification`

## Context

All release gates so far run in the producer workflow: they share its
checkout, caches and credentials. A packaging, provenance or compatibility
failure that also affects the producer environment is invisible there —
circular evidence. The spec requires a clean-environment verifier that
authenticates artifacts before execution and quantifies reproducibility
limits ([SLSA 1.2](https://slsa.dev/spec/v1.2/),
[TUF](https://theupdateframework.io/)).

Alternatives considered for the rebuild comparison: promising bit-identical
reproducibility across arbitrary environments (rejected — Go binaries vary
with toolchain revision and buildid, and the non-goal forbids that claim);
skipping rebuilds entirely (rejected — packaging drift would go unmeasured).

## Decision

- **Separate workflow** (`.github/workflows/verify-release.yml`), triggered
  by `release: published` and manually per tag. It runs with `contents:
  read` only, `cache: false`, no producer secrets and no producer build
  state (R1). It consumes exclusively public release data via
  `gh release download`.
- **Authenticate before executing (R2, security):**
  `tests/release/independent-verify.sh` verifies, in order: SHA-256
  checksums; Sigstore bundles and CycloneDX SBOMs with the consumer-pinned
  tag identity (`tests/release/verify.sh`); SLSA provenance for the platform
  archive and `checksums.txt` (`gh attestation verify --repo
  --signer-workflow`). Only after all layers pass does the verifier extract
  the binary, compare `pose version` against the tag and run
  `install → doctor --json → check --strict` on a fresh repository.
- **Controlled rebuild (R3):** the verifier clones the tag source and
  rebuilds the platform binary with the pipeline's flags (`CGO_ENABLED=0`,
  `-trimpath`, pinned `mod_timestamp`, same ldflags). A bit-identical digest
  is reported as MATCH; a difference is reported with its explained
  nondeterministic inputs (toolchain revision, buildid) and explicitly
  classified as a reproducibility delta, not an authenticity failure.
  GoReleaser builds now set `-trimpath` and commit-pinned `mod_timestamp` to
  minimize those deltas.
- **Evidence:** the verification report is a retained CI artifact
  (400 days); the procedure is public and consumers can run the same script.
- **Coverage:** the verifier exercises its native target (linux/amd64)
  natively; other targets are covered by digest/signature/provenance
  verification, with emulation documented as future work rather than
  claimed.

## Consequences

- Positive: packaging, signing, provenance and compatibility failures are
  caught by a consumer-shaped environment that shares nothing with the
  producer.
- Positive: reproducibility is measured and reported honestly instead of
  promised.
- Trade-off: verification depends on GitHub availability (release download,
  attestation store); acceptable for a GitHub-distributed product.
- Residual: only the verifier's native platform executes the binary; and a
  compromised repository could alter verifier and producer together —
  mitigated partially by the published, stable verification procedure that
  third parties can run independently.
