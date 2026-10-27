---
type: handoff
slug: delegated-rounds-legacy-status
owner: @oseiaspereira88
sensitivity: public-internal
created_at: 2026-10-27
last_reviewed_at: 2026-10-27
expires_at: 2026-11-26
source_refs:
  spec: pose-delegated-review-round-protocol
  workflow: review
  commands:
    - pose review status --json
    - pose delegate ls --json
    - go -C pose-mcp test ./internal/cli -run TestRunContinuityInterruptedResumeAndAccounting -count=1
  external_sources: []  # [{url: "", accessed_at: "YYYY-MM-DD"}]
---

# Handoff: legacy status round disclosure

## Context

Independent round 4 of spec pose-delegated-review-round-protocol reviewed fa0c589aac678fe548e3725f745ba6628cef3e8f. All earlier blockers are corrected. Reviewer agent:independent-gpt-6.1-sol-review is the maintainer-authorized declared Codex/OpenAI identity, another vendor than Claude/Anthropic implementation. This instance uses unsigned attestations. Last reviewed on 2026-10-27.

## Current state

f-r4-01 is low severity: the deprecated pose review status path omits round/thread fields when constructing interrupted summaries in pose-mcp/internal/pose/review_run_continuity.go:611; its text renderer in internal/cli/review_run_continuity.go:116 also omits the round. The current pose delegate ls path and persisted run contain the correct round, and verify/approval do not consume this legacy presentation.

The independent composed-handoff fixture produced a native interrupted run with Round=1. Before handing it off, pose review status --json returned an interrupted_runs row with no round field, while the recorded run had Round=1. The same fixture then handed off and continued the handoff-owned session successfully in round 2. Local reproduction and output: .review-work/round4/handoff-probe.txt and handoff-probe.log (deliberately untracked).

## Next checks

Reproduce with the existing run-continuity fixture or a fake agent that captures a synthetic session and exits with declared limit code 75. Compare the persisted run, pose delegate ls --json, and pose review status --json before resume. Add regression coverage that the legacy summary/text reports the round and relevant lineage, or explicitly document its narrower legacy projection before retirement. Run the continuity, delegation and round suites. Review by 2026-11-26 or before removing the deprecated alias.

## Risks

Users of the deprecated status command miss round context; the deprecation hint already points to pose delegate ls. This does not turn partial output into a decision, alter the ledger, or bypass approval. Acceptance is limited to this low-severity presentation gap and its existing replacement.

## Next owner

@oseiaspereira88; low criticality; review by 2026-11-26.

## References

Spec: .pose/specs/2026-10-18-pose-delegated-review-round-protocol.md. Review: .review-work/round4/REPORT.md (local). Native attestation rva-78b867ebfada5dcb on bundle rvb-aaec9b9e16165671 and the spec follow-up record f-r4-01.
