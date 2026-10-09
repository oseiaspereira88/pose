---
slug: delegated-review
status: draft
created_at: 2026-10-09
depends_on:
---

# Roadmap: Delegated review

**Outcome:** any project can hand a sealed review to another agent — Codex,
Claude Code or a runner such as Harne8's Conductor — and get an independent
review whose prompt the implementer did not write, whose run is recorded and
cannot be quietly retried away, and whose independence is evidence rather
than a reviewer prefix.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex
through a personal launcher (`secondary-agent`). It found three real defects
in three rounds, and exposed what the engine lacks; ADR
`2026-10-09-delegated-review-is-an-adapter` records the analysis, the
proposed contract and four decisions that are the maintainer's.

The order is deliberate. Nothing starts before the ADR is accepted. The
brief ships first because it removes the implementer-written prompt on its
own, even with today's launcher. Dispatch replaces the launcher. Provenance
and the attempt ledger make the result trustworthy, and only then does the
capability become adoptable and runnable by Harne8 or CI.

## Milestone: contract
- after:
- target_start:
- target_due:
- specs: pose-delegated-review-contract

**Exit gate:** the ADR is accepted with the four open decisions answered by
the maintainer, and every later spec's requirements are reconciled with
those answers.

## Milestone: brief
- after: contract
- target_start:
- target_due:
- specs: pose-delegated-review-brief

**Exit gate:** `pose review brief <bundle>` produces the same bytes for the
same sealed bundle, carries no implementer text outside a labelled notes
section, and a review run from it finds a seeded defect in a fixture.

## Milestone: dispatch
- after: brief
- target_start:
- target_due:
- specs: pose-delegated-review-dispatch

**Exit gate:** `pose review dispatch` runs a configured adapter on a
disposable worktree at the sealed commit, leaves the repository untouched,
and records a run with transcript and draft; a run that edits files or
exceeds its budget is recorded as failed, not as a review.

## Milestone: trust
- after: dispatch
- target_start:
- target_due:
- specs: pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger

**Exit gate:** `different-actor` is satisfied from a run record (signed by a
pinned native issuer when signed attestations are adopted) and no longer
from the `agent:independent-` prefix in adopting instances; a rejection
stays visible, and rerunning an unchanged bundle without a reason is
refused.

## Milestone: adoption
- after: trust
- target_start:
- target_due:
- specs: pose-delegated-review-capability, pose-delegated-review-action-fulfillment

**Exit gate:** `delegated-review` is a catalog capability, off by default,
documented with Codex and Claude Code adapter examples; a "review by another
actor" action request is fulfilled by a local adapter and by a non-local
runner (CI or Harne8 Conductor) through the same contract.
