---
slug: supply-chain-trust
status: done
created_at: 2026-07-18
depends_on:
---

# Roadmap: Supply-chain trust

**Portfolio order:** 2 of 7
**Outcome:** let adopters verify who built POSE, what it contains and whether the release pipeline resisted common compromise paths.

This work starts as soon as the version contract is stable. Checksums alone detect corruption but do not establish release identity or build provenance. Use SLSA, CycloneDX, Sigstore and OpenSSF Scorecard as the benchmark set.

## Milestone: security-baseline
- after:
- target_start: 2026-08-03
- target_due: 2026-08-14
- specs: pose-ossf-security-baseline

**Exit gate:** minimum-permission CI, static analysis, dependency review and secret scanning are blocking.

## Milestone: artifact-identity
- after: security-baseline
- target_start: 2026-08-17
- target_due: 2026-08-28
- specs: pose-release-signing, pose-cyclonedx-sbom

**Exit gate:** every release archive has verifiable identity and an inventory tied to its digest.

## Milestone: attested-release
- after: artifact-identity
- target_start: 2026-09-01
- target_due: 2026-09-18
- specs: pose-slsa-provenance, pose-reproducible-release-verification

**Exit gate:** a clean consumer environment verifies signature, provenance, checksum and SBOM before execution.

## Risk controls

- Pin CI permissions and third-party actions to the minimum required scope.
- Keep signing identity outside repository secrets when keyless signing applies.
- Publish failed verification evidence without exposing sensitive workflow data.
