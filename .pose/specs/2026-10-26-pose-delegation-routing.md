---
slug: pose-delegation-routing
status: in-progress
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegation-surface, pose-delegated-review-run-provenance
priority: 2
components: pose-mcp
task_type: feature
delivers: capability:delegation-routing
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

- modified: .pose/specs/2026-10-26-pose-delegation-routing.md
- created: .pose/starts/pose-delegation-routing.json
- created: .pose/changelogs/unreleased/pose-delegation-routing.md
- created: pose-mcp/internal/pose/delegation_identity.go
- created: pose-mcp/internal/pose/delegation_routing.go
- created: pose-mcp/internal/pose/delegation_routing_test.go
- created: pose-mcp/internal/pose/delegation_routing_review_test.go
- modified: pose-mcp/internal/pose/delegation_policy.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/review_run_provenance_test.go
- modified: pose-mcp/internal/pose/review_assurance.go
- modified: pose-mcp/internal/pose/remediation_lineage_test.go
- modified: pose-mcp/internal/pose/main_test.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_test.go
- created: pose-mcp/internal/cli/delegate_routing_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_dispatch.go
- modified: pose-mcp/internal/mcpserver/review_run_provenance_test.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/cli/review_run_provenance_test.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/review-and-trust.md

### Delivery targets

- capability:delegation-routing module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### API/contract changes

Bundle payload field `implementation`; policy keys `routing.prefer`, `routing.on_unavailable`, `routing.lead_models`, `routing.max_wait`, `agents.<name>.probe`; run record fields.

## 4. Tasks

- [x] Implementation identity at seal, detection and override
- [x] Legacy reading for earlier bundles
- [x] Availability and `agents` states
- [x] Resolver with preference and fallback steps
- [x] Routing record and achieved independence, disclosed at verify and closeout
- [x] First independent review: changes requested (B01–B08, F01–F05), all addressed; B04 by correcting D5, whose claim that a probe never stales a recorded run was wrong
- [x] Second independent review: B01–B08 and F01–F05 accepted as fixed (B04 by the corrected D5, B08 by both layers); changes requested for B09, with F06 and F07, all addressed with regression tests that fail on ca7aab5
- [ ] Third independent review and closeout

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

### Decision D3
- Basis: R1, D1
- Minimal option: put the identity in the payload like any other field
- Selected option: seal it in the payload, but never prepare it, and match a sealed bundle with a preparation by their content once the preparation takes the sealed identity
- Rationale: every freshness check compares a preparation with a sealed bundle; an identity in the prepared payload would make every bundle stale for any session of another vendor, and an identity outside the payload could be edited after the fact
- Consequences: resealing content a bundle already holds returns the bundle that governs it, with its identity, so a reviewer closing out never becomes the implementation; only a declared `--lead` naming another identity than the governing bundle's seals anew, and that seal names the bundle whose identity it replaces, so it governs whatever the seal instants or the ids say and a historical identity declared again is a new seal; a run compared with a replaced identity stops governing, and a run routed against an identity its bundle does not record does not start
- Falsifier: a bundle reported stale, or a reseal minting a bundle, when only the sealing session changed

### Decision D4
- Basis: R2, R11
- Minimal option: fall back to the policy's `provenance.implementation` when the session is not detected
- Selected option: record vendor `unknown`, warn at seal, and refuse a run-backed attestation against it
- Rationale: R4 makes the policy legacy for later bundles; reading it when detection fails would bring back the per-scope edit D1 removes, silently
- Consequences: sealing from a terminal or CI without `--lead` gives a bundle routing treats as of unknown vendor and attested review cannot use; the seal says so
- Falsifier: an attested different-actor review whose implementation vendor no seal recorded

### Decision D5
- Basis: R6, R7
- Minimal option: no default `max_wait`, so `wait` applies only when configured
- Selected option: default `max_wait` 5h, bounded at 24h; a probe is an argv with a 10s default and 60s bound, no placeholder, outside the adapter the review path verifies
- Rationale: `wait` is a default step, and a default that never applies is a default in name only; the vendors' usage windows are a few hours; a probe in the adapter would change the adapter digest a run's provenance binds
- Consequences: an instance that wants no waiting removes `wait` from `on_unavailable` or sets `max_wait` to 0; adding a probe is still an edit of `.pose/policy`, and like every policy edit it changes the policy baseline, so runs recorded under the earlier policy stop governing as they do for any other policy change — a probe is not exempt from that rule
- Falsifier: a run that waited longer than `max_wait`, or a run's provenance failing on its adapter digest because a probe was added

### Decision D6
- Basis: R10
- Minimal option: implement the task identity inside `pose delegate task`
- Selected option: deliver the session-identity determination R2 and R3 describe as the one function the seal uses, and leave its call from `task` to `pose-delegation-task`, whose purpose this spec does not run
- Rationale: `task` is still refused by the surface; a second determination would drift from the seal's
- Consequences: R10 is satisfied by the shared detector and its tests; `pose-delegation-task` calls it with the launching session's `--lead`
- Falsifier: a task run whose implementation identity is determined differently from a seal in the same session

## 6. Validation

### Strategy

R10 is delivered as the session-identity determination the seal uses (D6); its caller in `pose delegate task` is deferred to `pose-delegation-task`, and the trace records it as such rather than as an executed task path. Fixture sessions with each vendor's markers seal bundles with the expected identity; `--lead` overrides and is labelled; a fake agent set with one vendor limited and one CLI missing routes through each fallback step in order; `wait` starts after a recorded reset and is skipped beyond `max_wait`; a same-model run is disclosed and refused as `different-actor`; an earlier bundle still reads the policy identity.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `routing` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
