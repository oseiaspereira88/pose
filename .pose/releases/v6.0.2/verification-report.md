# POSE independent release verification

- release: v6.0.2
- verifier: clean environment, no producer caches or credentials

## Authentication before execution
- PASS: sha256 checksums verified for all downloaded archives
- PASS: Sigstore signatures + CycloneDX SBOMs (pinned identity)
- PASS: SLSA provenance (pose_6.0.2_linux_amd64.tar.gz: digest + repo + signer workflow)
- PASS: SLSA provenance (checksums.txt)

## Inspection and execution (only after verification)
- PASS: binary reports 6.0.2 (matches v6.0.2)
- PASS: install → doctor --json → check --strict on a fresh repository

## Reference extension (consumer-side)
- PASS: reference extension verifies against its published signature
- PASS: reference extension installs after verification
- PASS: a tampered extension is rejected

## Controlled rebuild (reproducibility)
- MATCH: independent rebuild is bit-identical (sha256 da3f60507f6a1833f292d6f31f6d94b31187be640ebfabe790d8721913cf0d0f)

Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution.
