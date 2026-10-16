---
spec: pose-review-approval-domain-preflight
category: security
breaking: false
---

The shared review-attestation writer now refuses an approving attestation the
verifier would reject before writing anything, with one typed refusal for the
Store, `pose review attest` (preview, apply, sign and envelope import),
`pose review auto-attest` and the MCP `pose_review_attest` tool. It also refuses
the approval of a superseded or stale bundle. A trusted signature over invalid
criteria still verifies identity but is never recorded as an approval;
legitimate negative decisions remain recordable, and persisted history is not
rewritten. Classified for a minor release.

`pose review attest` no longer judges `--criterion` and `--tool` answers in its
own flag parser: a semantic defect (an unknown or duplicate criterion, a finding
the attestation does not record, a not-applicable without a reason, an unknown
or omitted required tool) now returns the domain's refusal with every blocker
at once (exit 1) instead of the first parser error (exit 2), and a
`changes-requested` or `rejected` decision no longer needs the approval's
judgment criteria or required tools.

An explicit bundle id given to `pose review attest` is now attested as named:
the approval of a superseded bundle is refused by the domain with every other
blocker, and a negative decision about it is recorded, as the Store and a
signed import already did.

A `--mapping` disposed as missing-evidence, not-applicable or accepted-risk
without a rationale now reaches the domain too, which refuses it for an
approval and records it in a negative. `pose review auto-attest` prepares the
bundle as named and, for a complete draft, its preview reports the same
refusal the apply would return.
