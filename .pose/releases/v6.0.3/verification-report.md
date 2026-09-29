# POSE independent release verification

- release: v6.0.3
- verifier: clean environment, no producer caches or credentials

## Authentication before execution
- PASS: sha256 checksums verified for all downloaded archives
- PASS: Sigstore signatures + CycloneDX SBOMs (pinned identity)
- PASS: SLSA provenance (pose_6.0.3_linux_amd64.tar.gz: digest + repo + signer workflow)
- PASS: SLSA provenance (checksums.txt)

## Inspection and execution (only after verification)
- PASS: binary reports 6.0.3 (matches v6.0.3)
- PASS: install → doctor --json → check --strict on a fresh repository

## Reference extension (consumer-side)
- PASS: reference extension verifies against its published signature
- PASS: reference extension installs after verification
- PASS: a tampered extension is rejected

## Controlled rebuild (reproducibility)
- MATCH: independent rebuild is bit-identical (sha256 a1e7f082bc762e968806d29169c241fd6fdca3ee85e419bc5197647e4455df87)

Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution.
