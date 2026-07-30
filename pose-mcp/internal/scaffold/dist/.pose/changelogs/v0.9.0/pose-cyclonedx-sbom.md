---
spec: pose-cyclonedx-sbom
category: added
breaking: false
refs:
---

Each release archive now ships a CycloneDX SBOM generated from the exact
packaged artifact — component versions, hashes and detected licenses — named
`<archive>.cdx.json` and signed with the release. The release gate validates
the schema and fails when a direct production dependency is missing from the
inventory.
