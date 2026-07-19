---
spec: pose-reproducible-release-verification
category: added
breaking: false
refs:
---

Every published release is now re-verified by an independent workflow in a
clean environment: signature, provenance, checksum and SBOM are authenticated
before the binary is ever executed, followed by a functional smoke test and a
controlled rebuild that reports reproducibility honestly — bit-identical
matches or explained differences. The same procedure is runnable by any
consumer.
