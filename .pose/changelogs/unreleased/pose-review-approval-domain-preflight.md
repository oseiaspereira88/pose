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
