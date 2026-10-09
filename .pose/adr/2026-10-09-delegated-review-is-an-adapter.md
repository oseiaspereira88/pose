---
title: A delegated reviewer is an adapter that turns a sealed bundle into a draft
status: accepted
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

## Decision

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
- **The engine records a verified run, not an actor.** A run produces a
  run record under `.pose/review-runs/` and the reviewer's conclusion. When
  the run passes the engine's checks — brief generated from the sealed
  bundle, disposable copy left unchanged, run record intact and signed,
  vendor or model different from the implementation's, conclusion complete
  and accepted by the same preflight as any attestation, no earlier
  rejection on an unchanged bundle — the engine records the attestation
  itself. An approval closes without anyone; a rejection is recorded and
  blocks until the code changes.
- **People by exception, through policy.** A person is asked only when
  review policy says so: criteria marked as requiring a person (risk
  acceptance, public contracts, or whatever the project chooses), reviewers
  in conflict, a reviewer asking for a business decision, or the run budget
  exhausted without approval. Each case is an action request with the
  question ready, answered as a confirmation or approval, never as manual
  steps.
- **Independence as evidence.** The run record names adapter, vendor and
  model, binds the prompt and output digests, and can be signed by a native
  issuer; `different-actor` is read from that record, not from a reviewer
  prefix.
- **Immutable attempts.** Every run is kept, rejections included; another run
  on an unchanged bundle needs a recorded reason.
- **Opt-in.** A catalog capability, `delegated-review`, off by default.

## Maintainer's decisions (2026-10-09)

1. **Who records:** the engine records the conclusion of a run that passes
   its checks; a person is involved only where review policy escalates. The
   maintainer rejected "the maintainer applies" as a manual step that runs
   against POSE's aim of autonomous development, and the implementer or the
   reviewer recording on their own as unverified.
2. **Agent independence:** an agent run satisfies `different-actor` when its
   run record shows a vendor or model different from the implementation's.
3. **Vendor:** a different vendor is preferred, not required.
4. **Scope:** reviews, independent adjudications and smoke runs from the
   first delivery, through the same contract.

## Consequences

- The prefix convention `agent:independent-` is retired for instances that
  adopt `delegated-review`; until then it stays, documented as a declaration.
- Personal launchers such as `secondary-agent` become examples of an adapter
  rather than the mechanism.
- Harne8 and a project using POSE alone run the same contract.
