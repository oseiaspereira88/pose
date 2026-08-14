# POSE independent release verification

- release: v1.2.0
- verifier: clean environment, no producer caches or credentials

## Authentication before execution
- PASS: sha256 checksums verified for all downloaded archives
- PASS: Sigstore signatures + CycloneDX SBOMs (pinned identity)
- PASS: SLSA provenance (pose_1.2.0_linux_amd64.tar.gz: digest + repo + signer workflow)
- PASS: SLSA provenance (checksums.txt)

## Inspection and execution (only after verification)
- PASS: binary reports 1.2.0 (matches v1.2.0)
- PASS: install → doctor --json → check --strict on a fresh repository

## Reference extension (consumer-side)
- PASS: reference extension verifies against its published signature
- PASS: reference extension installs after verification
- PASS: a tampered extension is rejected

## Controlled rebuild (reproducibility)
- DIFFERENCE (explained inputs follow): released fd9a483e77e8bd0e0918c906291e9cd3c6dadb391efbd0738d847555ccdfbabb, rebuilt d2fd18a9e31083cfe74a5ca6d80d17227d4c8ce90a0d230aeb71051c4427d3e7
  - toolchain: verifier Go go1.26.5-X:nodwarf5 vs release pipeline Go (see release run)
  - known nondeterministic inputs: Go toolchain revision and buildid; mod timestamp is pinned to commit (1786700766) and paths are trimmed
  - a digest mismatch here is a reproducibility delta, not an authenticity failure: authenticity is established by the layers above

Result: VERIFIED — signature, provenance, checksum and SBOM checked before execution.
