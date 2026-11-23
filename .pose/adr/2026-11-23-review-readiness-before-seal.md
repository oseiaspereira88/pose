# ADR: Check review readiness before the seal and inside the run

## Status

Accepted on 2026-11-23 for `pose-review-readiness-before-seal`.

## Context

A delegated review round costs a seal, a run of about forty minutes and a conclusion. In the 23 rounds of `pose-validate-carries-unchanged-checks`, about seven were lost to refusals that existed before the reviewer started: `pose close` refused an approved scope for an undeclared delivery; the conclusion preflight refused drafts for a decision with no requirement basis, prefixed finding ids, prior findings outside the open ones, and criterion evidence outside its component. The reviewer could not see those rules applied to its own draft until the run had ended.

## Decision

Move each check to the earliest point that has its inputs, reusing the same predicate:

- The seal of a new spec bundle refuses the declaration findings the delivery index holds at error severity for the scope (`undeclared-delivery` and the artifact finding `undeclared`), which `pose close` reports. Evidence findings stay at close. When a required criterion asks for structural mappings, it also refuses a decision or assumption that reaches no requirement or constraint, the namespace the preflight checks mappings against.
- `pose review draft-check` runs the conclusion's draft parser, the round accounting and the attestation content checks over a draft, writing nothing. The content checks are the part of the bundle-attestation validation that does not depend on the run's signature or identity, extracted as one function both paths call.
- A sealed checkout holds neither the bundle nor the run ledger, so the dispatch writes the bundle and the round's open findings to the run's private directory and names the file in `POSE_REVIEW_DRAFT_CONTEXT`. It also names its own executable in `POSE_REVIEW_CLI`, because the reviewer's shell may resolve another installed `pose`. `--draft -` reads standard input for read-only sandboxes. A run id selects its bundle.
- The brief of the new template `review-brief/8` asks the reviewer to run the check. Template texts are immutable, so `/7` stays for the runs briefed with it.

## Consequences

A scope that would fail at close now fails at seal, before any review cost. A reviewer can converge its draft inside one run. The draft check is advisory: the conclusion step stays the gate and still runs every check, including the signature and identity checks the draft check cannot run.

Residual limits:
- L1: the draft check judges against the context the dispatch wrote. A reviewer that edits that file only misleads itself, because the conclusion re-reads the sealed bundle and the ledger.
- L2: the seal reads the delivery finding from the committed index. A stale index can miss a finding that a later `pose index` would add. `pose close` still refuses it, so the cost is a round, never a wrong closure.

## Alternatives considered

- A per-instance cap on review rounds would stop the loop by approving unreviewed work.
- Running the full `pose close` gate at seal would refuse scopes for evidence that only exists after the review.
