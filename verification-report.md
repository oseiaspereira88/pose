# POSE independent release verification

- release: v1.4.0
- verifier: clean environment, no producer caches or credentials

## Authentication before execution
- PASS: sha256 checksums verified for all downloaded archives
- PASS: Sigstore signatures + CycloneDX SBOMs (pinned identity)
- PASS: SLSA provenance (pose_1.4.0_linux_amd64.tar.gz: digest + repo + signer workflow)
- PASS: SLSA provenance (checksums.txt)

## Inspection and execution (only after verification)
- PASS: binary reports 1.4.0 (matches v1.4.0)
- PASS: install → doctor --json → check --strict on a fresh repository

## Reference extension (consumer-side)
- PASS: reference extension verifies against its published signature
- PASS: reference extension installs after verification
- PASS: a tampered extension is rejected

## Controlled rebuild (reproducibility)
- MATCH: independent rebuild is bit-identical (sha256 85bf79a2cf26a5d69ce7c91f8b8beabc9451088ea699a3092e0c14097cdb42f0)

Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution.
