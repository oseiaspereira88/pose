---
slug: pose-delegation-routing
status: draft
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegation-surface, pose-delegated-review-run-provenance
priority: 2
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Delegated runs are routed by the implementation they judge

## 1. Intent

### Goal

Choose the agent of a delegated run from the identity of whoever produced the subject, fall back in a declared order when the preferred agent is unavailable, and record the independence each run achieved.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md). Today `--via` is mandatory and the implementation identity is a static declaration in `reviewers.json`, with a fixed execution. When the leading agent changes from one scope to the next, "the other vendor" has no answer, and the reviewer is compared with the policy file instead of with whoever implemented. A vendor out of quota is discovered only by a failed run.

### Constraints

Keep the 2026-10-09 rule: another vendor or another model satisfies `different-actor`; another vendor is preferred, not required. The implementation identity is a declaration of the sealing session, signed when signed attestations are adopted, never a vendor receipt.

### Non-goals

Continuing a stopped run with another agent (`pose-delegation-handoff`); relaxing `different-actor` when the capability is off (`pose-delegated-review-capability`).

## 2. Requirements

### Functional

- R1: `pose review bundle --seal` shall record the implementation identity — principal, vendor, model, execution digest and its source (`detected` or `declared`) — in the bundle payload.
- R2: The vendor shall be detected from the sealing session's markers (the variables the surface spec removes from children), the model from `--lead <vendor>/<model>` or the policy's `routing.lead_models`, and an undetermined model shall be recorded as `unknown`, which can never establish a model difference.
- R3: `--lead <vendor>[/<model>]` shall override detection and be recorded as `declared`.
- R4: For bundles sealed before R1, the engine shall keep reading `provenance.implementation` from the policy; for later bundles it shall ignore it and warn when it is present.
- R5: `--agent auto` shall order candidates by `routing.prefer` (default `opposite-vendor-of-implementation`, in the policy's declared agent order), then by the steps of `routing.on_unavailable` (default `same-vendor-other-model`, `wait`, `same-model`).
- R6: An agent shall be unavailable when its CLI is not on the path, when its last run recorded a usage limit whose reset instant is in the future, or when its optional `probe` command fails within its bounded timeout; `pose delegate agents` shall show `available`, `limited until <instant>` or `missing`.
- R7: The `wait` step shall apply only when every earlier candidate is limited with a known reset within `routing.max_wait`; the run shall then start after the reset without polling, and otherwise the next step shall apply.
- R8: Each run shall record the routing decision — candidates considered, the reason each was skipped, the step chosen — and `independence_achieved`: `different-vendor`, `different-model` or `same-model`.
- R9: An explicit `--agent <name>` shall bypass routing, still record `independence_achieved`, and warn on stderr when the agent shares the implementation's vendor.
- R10: For `pose delegate task`, the implementation shall be the launching session's identity, determined as in R2 and R3.
- R11: A `review` run with `same-model` independence shall never satisfy `different-actor`; `pose review verify` and the closeout shall disclose the achieved independence of the run behind each attestation.

### Non-functional

- Availability is read from local records and the path; only an explicit `probe` runs anything.

### Security

- The execution digest binds the session reference without storing it; the reference itself stays private as continuity requires.
- A declared identity is labelled as such everywhere it is disclosed.

### Compatibility

- Bundles sealed before R1 verify as they do today.

## 3. Technical Plan

### Affected areas

Bundle sealing and payload schema, routing resolver, availability reader, run record (`routing`, `independence_achieved`), assurance disclosure, `pose delegate agents`.

### Artifacts

Implementation artifacts are declared when the spec starts.

### API/contract changes

Bundle payload field `implementation`; policy keys `routing.prefer`, `routing.on_unavailable`, `routing.lead_models`, `routing.max_wait`, `agents.<name>.probe`; run record fields.

## 4. Tasks

- [ ] Implementation identity at seal, detection and override
- [ ] Legacy reading for earlier bundles
- [ ] Availability and `agents` states
- [ ] Resolver with preference and fallback steps
- [ ] Routing record and achieved independence, disclosed at verify and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R4
- Minimal option: keep `provenance.implementation` and edit it per scope
- Selected option: record the implementation in the sealed bundle
- Rationale: the party to compare with is whoever produced the sealed subject; a policy file edited per scope is a manual step and a race between scopes
- Consequences: the bundle payload schema grows; the policy key becomes legacy
- Falsifier: two bundles sealed by different vendors that resolve the same implementation identity

### Decision D2
- Basis: R2
- Minimal option: infer the model from the vendor's default
- Selected option: record `unknown` unless declared or configured
- Rationale: a guessed model would let a same-model run claim a model difference
- Consequences: instances declare `routing.lead_models` to get model-based independence
- Falsifier: a run credited with a model difference whose implementation model was never declared

## 6. Validation

### Strategy

Fixture sessions with each vendor's markers seal bundles with the expected identity; `--lead` overrides and is labelled; a fake agent set with one vendor limited and one CLI missing routes through each fallback step in order; `wait` starts after a recorded reset and is skipped beyond `max_wait`; a same-model run is disclosed and refused as `different-actor`; an earlier bundle still reads the policy identity.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `routing` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
