---
slug: pose-tool-dispositions-must-be-supported
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-attestation-evidence-must-be-in-the-bundle
priority: 0
components: pose-mcp
delivers:
---

# Spec: A tool disposition must be supported by the bundle too

## 1. Intent

### Goal
Give the tool half of an attestation the guarantee the criteria half received:
a disposition that claims a tool passed must cite evidence the sealed bundle
contains, and `auto-attest` must stop inventing that evidence.

### Business value
`pose-attestation-evidence-must-be-in-the-bundle` closed this on criteria and
recorded the other half as a known gap: tool dispositions were still unchecked
against the bundle, and `auto-attest` still wrote `validation:auto-attest`.

An attestation has two halves and the release notes claim one property about
both — that `auto-attest` no longer synthesises a reference, and that what an
attestation asserts is supported by the subject it approves. Shipping with the
tool half fabricating makes that claim half true, in a change whose entire
argument is that a claim must be checkable.

The tool path already compared the cited class against what the tool asks for.
What it never did was ask whether the reference exists: a disposition could name
a class the tool demands and an id that appears nowhere, and verify.

### Constraints
- Only a tool that declares evidence classes is held to the sealed set. Tools
  that declare none cite a different kind of reference — `artifact-check` and
  `review-check` are POSE commands, and what supports their disposition is that
  the command ran — which the bundle does not carry and is not meant to.
- The legacy attempt path has no bundle to check against and must keep working.

### Non-goals
- Verifying that a tool actually ran. `check:<tool>` records the claim; nothing
  here proves the process executed, which is the same trust the command has
  always had.

---

## 2. Requirements

### Functional
- R1: A tool disposition recorded `passed` or `failed`, for a tool declaring
  evidence classes, shall be rejected when its evidence is absent from the
  sealed bundle.
- R2: `auto-attest` shall not synthesise tool evidence. Where the scope is
  expected to carry validation evidence it shall refuse, naming the tool and the
  missing class; where it is not, it shall defer the tool with a rationale.
- R3: A tool declaring no evidence class shall cite its own run, not an
  unrelated sealed result.
- R4: A tool gated on `delivery-target-declared`, in a scope that has none,
  shall be deferrable rather than required to have passed.

### Non-functional
- The existing review suite passes, with its fixtures corrected where they were
  fabricating.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go` — tool coverage evaluation
- `pose-mcp/internal/pose/review_bundle.go` — auto-attest's tool half

### Artifacts
- created: .pose/specs/2026-09-08-pose-tool-dispositions-must-be-supported.md
- renamed: .pose/changelogs/unreleased/pose-tool-dispositions-must-be-supported.md -> .pose/changelogs/v2.0.0/pose-tool-dispositions-must-be-supported.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go

### Technical risks
- R4 widens what a required tool may be. A tool that genuinely should have run
  is now deferrable whenever the scope declares no delivery target, which is a
  property of the scope rather than of the reviewer's diligence. Without it,
  removing the fabrication makes a documentation-only closeout impossible, so
  the alternative is worse: the engine would demand a disposition only a
  fabricated one could satisfy.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Reject a tool disposition the bundle does not support (R1)
- [x] Increment 2: Stop auto-attest inventing tool evidence (R2, R3)
- [x] Increment 3: Let an unmeetable precondition defer a required tool (R4)

### Validation
- [x] Each half asserted, and each assertion shown to fail without its change

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: what a tool that declares no evidence class should cite. The previous
  code reached for the first sealed result, whatever it was.
- Decision: `check:<tool-id>`.
- Rationale: `artifact-check` and `review-check` do not report a validation
  result, so no sealed evidence says anything about whether they ran. Citing an
  unrelated result was worse than citing nothing: it looked like support. Naming
  the tool is the true statement — this disposition rests on the tool having run
  — and it is the form the CLI already uses by hand.

### Decision 2
- Date: 2026-09-08
- Context: the first implementation marked such tools `not-used`, and every
  documentation-only closeout broke: a required tool recorded `not-used` is a
  blocker whatever the reason.
- Decision: `deferred`, and treat an unmet `delivery-target-declared`
  precondition the way `review-complete` already is.
- Rationale: the engine already models a required tool that cannot run yet. A
  tool gated on a delivery target, in a scope that has none, is the same
  situation — it is not that the reviewer skipped it, it is that there is
  nothing for it to run against. Requiring it to have passed there is requiring
  a fabricated disposition, which is what this spec removes.

### Decision 3
- Date: 2026-09-08
- Context: the first version of the auto-attest test removed `integration`
  results and asserted a refusal. It passed with the tool path unfixed, because
  a *criterion* also demands `integration` and refused first.
- Decision: demand `reachability`, which no criterion in the fixture asks for.
- Rationale: an assertion that passes whether or not the code under test is
  present measures nothing. This is the sixth fixture in this session to confirm
  a case adjacent to the one intended; the pattern is now reliable enough to
  check for deliberately.

---

## 6. Validation

### Strategy
Assert each half against a bundle that reproduces it, and disable each in turn.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass. Disabling the presence check leaves
  `blockers = ""` where the test wants the absent tool evidence named; restoring
  the `<class>:auto-attest` fabrication fails the refusal test on "auto-attest
  invented tool evidence for a class the bundle does not seal". The tool half of
  `approvedBundleAttestation` was still citing a fixed `integration:bundle-test`
  that appeared in no bundle — the criteria half of the same helper had been
  corrected one spec earlier and the tool half was left.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <reviewToolEvidenceBlocker takes the sealed set and rejects a ref absent from it, for tools declaring classes only; TestToolDispositionMustCiteEvidenceTheBundleSeals asserts a supported attestation is accepted first, then that naming an unsealed id of the right class is blocked>
- R2 [satisfied] <AutoAttestReviewBundle refuses when requiresEvidence and defers otherwise; TestAutoAttestDoesNotInventToolEvidence demands reachability, which no criterion asks for, so only the tool path can produce the refusal>
- R3 [satisfied] <a tool with no declared class cites check:<tool-id>; the doc-only closeout tests exercise it>
- R4 [satisfied] <evaluateReviewToolCoverage treats delivery-target-declared as deferrable when the scope has no delivery target; TestReviewBundleSealAndCloseoutForDocOnlySpecWithNoDeliveryTargets passes without fabrication>

### Known gaps
- Tool evidence is matched by reference and class, not by component, exactly as
  criteria are. The same follow-up covers both.
- `check:<tool-id>` records that a disposition rests on the tool having run,
  and nothing verifies the run happened.

---

## 7. Final Report

### Follow-ups

- [done] Match tool evidence to the tool's component, alongside the same work for criteria. Delivered by `pose-evidence-scoped-to-component`: `reviewToolEvidenceBlocker` refuses a component-scoped tool citing evidence from outside that component, the same check the criteria path got. Verified against the code, not the record. (owner:unowned crit:high review:2026-11-08)
