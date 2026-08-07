---
spec: pose-sbom-license-inventory
category: fixed
breaking: false
refs:
---

Published CycloneDX SBOMs now carry component licenses. Scanning the packaged
binary recovers module names and versions but no license texts, so releases
shipped an SBOM with a license for one component out of 27 while the
documentation advertised an SBOM'd release. Reading the module cache the build
already populates takes that to 24 of 27; the remaining three are the project's
own modules and the binary, covered by the repository LICENSE.
