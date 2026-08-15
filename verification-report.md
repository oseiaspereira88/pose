# POSE independent release verification

- release: v1.2.2
- verifier: clean environment, no producer caches or credentials

## Authentication before execution
- PASS: sha256 checksums verified for all downloaded archives
- PASS: Sigstore signatures + CycloneDX SBOMs (pinned identity)
- PASS: SLSA provenance (pose_1.2.2_linux_amd64.tar.gz: digest + repo + signer workflow)
- PASS: SLSA provenance (checksums.txt)

## Inspection and execution (only after verification)
- PASS: binary reports 1.2.2 (matches v1.2.2)
- PASS: install → doctor --json → check --strict on a fresh repository

## Reference extension (consumer-side)
- PASS: reference extension verifies against its published signature
- PASS: reference extension installs after verification
- PASS: a tampered extension is rejected

## Controlled rebuild (reproducibility)
- MATCH: independent rebuild is bit-identical (sha256 03a016ab1c1658b56042918894730793a1b90e0948285b3bb8ef8c48f5392ae8)

Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution.
