---
spec: pose-attestation-evidence-must-be-in-the-bundle
category: fixed
breaking: true
refs:
---

A criterion recorded `passed` must now cite evidence the sealed bundle actually
contains, of a class the criterion asks for. Nothing checked this before: a
`passed` could reference evidence absent from the bundle, evidence of the wrong
class, or nothing at all, and the attestation verified.

`pose review auto-attest` no longer synthesises a reference. Where the scope is
expected to carry validation evidence it refuses, naming the criterion, the
missing class and the way out; where it is not — a documentation-only spec with
no delivery target — it records the criterion `not-applicable` with a rationale
naming what is absent.

Criteria now go through the same evidence-vocabulary filter as tools: one
demanding a class no registered check may emit has that class dropped from the
plan with a warning, instead of planning a gate that only a fabricated
disposition could pass.

**This fails attestations that already verify.** They are unsupported and were
reported as sound; the remedy is a superseding attestation, which the
append-only model already provides. Plan digests change where an unproducible
class was dropped, so a bundle sealed before this does not match one sealed
after.
