# POSE independent release verification

- release: v1.2.1
- verifier: clean environment, no producer caches or credentials

## Authentication before execution
- PASS: sha256 checksums verified for all downloaded archives
- PASS: Sigstore signatures + CycloneDX SBOMs (pinned identity)
- PASS: SLSA provenance (pose_1.2.1_linux_amd64.tar.gz: digest + repo + signer workflow)
- PASS: SLSA provenance (checksums.txt)

## Inspection and execution (only after verification)
- PASS: binary reports 1.2.1 (matches v1.2.1)
- PASS: install → doctor --json → check --strict on a fresh repository

## Reference extension (consumer-side)
- PASS: reference extension verifies against its published signature
- PASS: reference extension installs after verification
- PASS: a tampered extension is rejected

## Controlled rebuild (reproducibility)
- MATCH: independent rebuild is bit-identical (sha256 06297faf57f61e9d4a1896629ec2d23f6d8a6ac504e5a42acd9777f92597028a)

Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution.
