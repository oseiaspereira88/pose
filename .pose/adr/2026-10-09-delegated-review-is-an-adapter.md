---
title: A delegated reviewer is an adapter that turns a sealed bundle into a draft
status: proposed
date: 2026-10-09
decision_type: architecture
---

# A delegated reviewer is an adapter that turns a sealed bundle into a draft

## Context

On 2026-10-09 the pose-dist overlays started requiring a different-actor
reviewer for pose-mcp specs, and the review of `pose-native-attestation-issuer`
was delegated to another vendor's agent (Codex, gpt-6.1-sol) through a
personal launcher script (`secondary-agent`). It found three real defects in
three rounds and stopped instead of fixing them. The arrangement also showed
what is missing from POSE:

- the implementer wrote the reviewer's prompt, so the reviewed party steered
  the review;
- under declared assurance, `different-actor` is satisfied by the reviewer
  prefix `agent:independent-`, which anyone can type — the engine's own
  comment says so (`review_closeout.go`);
- nothing recorded the reviewer runs, so a rejection could be dropped by
  running the review again or switching models;
- sandboxing, memory and `.git` access were solved by hand per run;
- the launcher depends on two vendor CLIs and lives outside the engine.

## Options considered

1. **Documentation only**: a skill and a prompt template. Nothing above is
   enforced.
2. **An engine adapter**: POSE generates the review brief from the sealed
   bundle, runs a configured reviewer command on a disposable copy, and
   records the run and a draft; it never records the attestation itself.
3. **An action request only**: the engine opens a "review by another actor"
   request and any runner fulfils it. Leaves brief, run record and
   independence evidence undefined.
4. **An MCP tool the implementing agent calls**: the shortest path to an
   agent looping until it is approved.

## Decision (proposed)

Option 2, with option 3 as the way Harne8's Conductor or CI runs the same
contract, and option 1 as the documentation of both:

- **Brief from the bundle.** The review request is generated from the sealed
  bundle (scope, diff, pending criteria, structural facts, evidence),
  deterministic and versioned. Implementer notes are allowed and labelled as
  the implementer's.
- **Adapter, not dependency.** Reviewer commands are project policy
  (`.pose/policy/reviewers.json`); POSE ships examples for Codex and Claude
  Code and depends on neither.
- **Disposable copy.** The reviewer runs on a worktree at the sealed commit;
  it may run tests there and cannot change the repository.
- **Draft, never record.** A run produces a draft attestation and a run
  record under `.pose/review-runs/`; recording is a separate step.
- **Independence as evidence.** The run record names adapter, vendor and
  model, binds the prompt and output digests, and can be signed by a native
  issuer; `different-actor` is read from that record, not from a reviewer
  prefix.
- **Immutable attempts.** Every run is kept, rejections included; another run
  on an unchanged bundle needs a recorded reason.
- **Opt-in.** A catalog capability, `delegated-review`, off by default.

## Open decisions (the maintainer's)

1. Who records the attestation: the reviewer run itself, the maintainer, or
   the implementer applying the reviewer's draft unchanged.
2. Whether an agent run can satisfy `different-actor` when vendor or model
   differ, or only a person can and agent runs stay advisory drafts.
3. Whether a different vendor is required or preferred.
4. Initial scope: spec reviews only, or also adjudications and smoke runs.

Until they are answered the specs of roadmap `delegated-review` assume: the
maintainer or a configured policy applies; an agent run satisfies
`different-actor` only with a run record whose vendor/model differs from the
implementation's; a different vendor is preferred, not required; spec
reviews first.

## Consequences

- The prefix convention `agent:independent-` is retired for instances that
  adopt `delegated-review`; until then it stays, documented as a declaration.
- Personal launchers such as `secondary-agent` become examples of an adapter
  rather than the mechanism.
- Harne8 and a project using POSE alone run the same contract.
