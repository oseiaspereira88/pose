---
spec: pose-release-signing
category: security
breaking: false
refs:
---

Every release artifact — archives, SBOMs and the checksum manifest — is now
signed with keyless Sigstore signing and ships an offline-verifiable bundle.
Verification instructions pin the exact workflow identity and OIDC issuer,
and the release pipeline refuses to succeed on unsigned artifacts or an
identity mismatch. No long-lived signing keys exist.
