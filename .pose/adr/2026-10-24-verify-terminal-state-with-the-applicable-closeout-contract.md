# ADR: Verify terminal state using the applicable closeout contract

## Status
Accepted (2026-10-24), spec `pose-closeout-terminal-verification`.

## Context
PlanCloseout currently certifies every step and terminal=true from lifecycle done, while closeout-check can refuse the same scope. B12 already distinguishes operation result from effect. Existing review verification preserves sealed historical contracts and adoption cutoffs; reconstructing a current bundle for done scopes would impose retrospective requirements. The amendment prerequisite is now done and owns its one journal/schema/gate.

## Decision
Use GetCloseoutState as the shared applicable terminal verifier for done plans, checks and apply/resume. It retains ReviewCheck's historical/cutoff policy and action restrictions and consumes the existing InspectAmendmentHistory gate. Report verification basis and historical limitations explicitly, without inventing a receipt. Missing required review or unreadable context cannot certify terminal.

A done plan records lifecycle_done separately, runs context/verify checks, and marks historical execution steps `not-run`, never done by status alone. Transition is already recorded by lifecycle, not executed again. `not-run` is a new non-executable step state; NextCloseoutStep skips it. A blocked done plan offers explicit repair/review next actions and cannot auto-regenerate evidence, seal, attest, reopen or mutate checkpoints. Apply/resume returns B12 already-satisfied only after successful verification and reports the current readonly verification in data, with empty steps_run/effects (preserving B12: no operation step ran); failed verification returns blocked with zero effects. Plain close also verifies an already-done spec and performs no transition. No new top-level command or automatic policy adoption.

In-progress plans consume the same amendment assessment before sealing; guarded close already rechecks it under the lifecycle lock. Do not duplicate seeding, taxonomy or logs. Do not introduce new retrospective trace/delivery requirements for done legacy scopes: verification is explicitly bounded to the applicable persisted contract and current opted-in history/restrictions.

## Consequences
Breaking correctness fix for consumers interpreting done as terminal or assuming all steps in a done plan were executed. Additive lifecycle/verification metadata discloses the basis and limits. Old sealed contracts remain governed by existing verification rules, not reconstructed under current overlays. Explicit no-op protects baseline/start/journal/checkpoint bytes. Rejected alternatives: raw-status shortcut fabricates checks; rerunning the entire pipeline on done would rewrite history and impose new policy; separate transport verifiers would drift again. No concurrency mechanism or lock order is added.
