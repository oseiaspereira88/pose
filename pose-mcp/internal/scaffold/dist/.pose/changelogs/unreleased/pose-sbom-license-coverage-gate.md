---
spec: pose-sbom-license-coverage-gate
category: added
breaking: false
refs:
---

The artifact-identity gate now fails a release whose SBOM resolves licenses for
fewer than 75% of its components, reporting the observed ratio. License
resolution depends on a syft setting and a populated module cache, either of
which can stop working without any step failing — which is how four releases
shipped an SBOM advertised as carrying licenses and carrying exactly one. The
floor is overridable through `SBOM_MIN_LICENSE_PCT`.
