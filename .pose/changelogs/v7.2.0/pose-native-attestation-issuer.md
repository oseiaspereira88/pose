---
spec: pose-native-attestation-issuer
category: added
breaking: false
refs:
---

A project with POSE alone can now require signed review attestations and verified reviewer identity. `pose issuer init` creates an issuer key outside the project, `pose issuer pin` trusts it in review policy, and `pose review attest … --sign <issuer> [--authority …]` signs the attestation and the reviewer's authority claim. Native issuers coexist with external ones such as a Harne8 Conductor: any pinned issuer's signature counts, and pinning or rotating one never removes another.
