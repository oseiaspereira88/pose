---
spec: pose-slsa-provenance
category: security
breaking: false
refs:
---

Every release archive and the checksum manifest now carries SLSA v1 build
provenance attested from the release pipeline — verifiable with
`gh attestation verify`, rejecting modified artifacts, wrong repositories and
untrusted builders. The claim is explicitly SLSA Build L2, with the L3
isolation gap documented rather than implied away.
