# ADR: Obligations are projected; material action requests are persisted

## Status

Accepted — 2026-10-04, with the versioned read model of `pose-obligation-contract`
(`pose-mcp/internal/pose/obligation.go`, `pose-mcp/schemas/v1/obligation.schema.json`).
Proposed the same day as the planning decision for the `record-truthfulness`,
`agency-readiness`, `governance-efficiency` and `epistemic-lifecycle` roadmaps.

The invariants are enforced by `ValidateObligation`: a stable id from the
logical source, qualified node references, separate satisfaction / knowledge /
waiting / effect axes, `unknown` never carrying a resolution, and a
legacy-opaque blocker carrying no invented actor or target.

## Context

The third consolidated analysis of POSE at `392aaa5a`
([report](../reports/2026-10-03-pose-consolidated-analysis.md), backlog in
[JSON](../reports/2026-10-03-pose-consolidated-backlog.json)) found that the
engine already discovered "something is still owed" many times, each with its
own shape: `WaitingOn` in readiness, `ReviewAttestationPendency` in review,
`Blockers []string` and `NextAction string` in closeout, `Obligations` and
`Reconciliation` in atomic start, `RefreshPending` in state, `ReviewPending` in
docs review and `OpenObligations` in spec transfer (findings F05, F08). An agent
has to rebuild causal state from several calls and from prose.

It also found that a material request to a person or external system — choose
between two compatibility strategies, accept a residual risk, confirm a defined
publication, perform an operation the agent does not control — has no
authoritative home. Follow-ups are post-delivery debt; `depends_on` is a
spec-level precondition; `status: blocked` is one bit with three conflicting
meanings across the skill, readiness and adoption metrics (F06, F07).

Finally, the 6.3.0 cycle produced sixteen attestations under `human:oseias`
whose conclusions the agent wrote and applied (F01). The records are
schema-valid; the reading "a person reviewed this" is not supported by them.
Identity, authorship, conclusion, confirmation, application and assurance are
different facts.

## Decision

D01. Obligations are a **read model**. They are derived from the sources that
already own them; the aggregator never stores a mutable "resolved" state of its
own. Resolving the source changes the projection. New persistence is reserved
for facts that no other source holds.

D02. **ActionRequest** is the authoritative source for a material request to an
actor. It persists the question, options and consequences, recipient role,
required authority, qualified targets, per-phase effect, request digest and an
append-only journal of answers, cancellations, supersessions and
invalidations. An ordinary question without a governed consequence is not an
ActionRequest.

D03. The persisted lifecycle stays small. Blocking is an **effect on a phase
and a scope**, computed from obligations; it is not a lifecycle status.
`blocked` keeps being read for compatibility until a major release migrates its
technical uses (spec transfer staging) and its metric meaning.

D04. Satisfaction, knowledge, waiting and blocking are separate axes. `unknown`
never means satisfied, cancelled or "no blocker".

D05. Local nodes (`R4`, `D2`, `A1`) are qualified by artifact and project
(`xref:<project>/spec:<slug>` + node kind + id). They are never global IDs.

D06. Operational answers carry a snapshot: project, authority context
revision, source revision, policy digest, generation time and per-producer
coverage (`current`, `stale`, `unavailable`, `unsupported`). A failed producer
is visible coverage, never zero obligations.

D07. Typed diagnostics (reason code + structured refs + satisfaction condition)
are born in the producing subsystem. Normalising prose with regular expressions
is rejected; a legacy adapter may expose an opaque blocker with its origin and a
limited-coverage flag, and may not invent an actor, target or unblock condition.

D08. Review and resolution records distinguish `prepared_by`, `concluded_by`,
`confirmed_by` and `applied_by`, plus the assurance level actually verified.
Roles may coincide and may be absent; an absent role stays absent and is never
copied from `reviewer`. A declared `confirmed_by: human:x` written by an agent
is a declaration, not observed confirmation.

D09. Sealed history keeps its meaning. New contracts are prospective and
versioned; old bundles are judged by what they sealed.

D10. The engine never attests cognitive independence or architectural
adequacy. It reports observed separation (actor, execution, credential) and
leaves adequacy to judgment.

D11. `pose state` / `pose_project_state` and the existing closeout machinery
evolve; one domain implementation serves CLI and MCP. No top-level command per
kind of pending item.

D12. Evidence and criteria are reused only when the material inputs of the
observation or criterion are equivalent and the policy allows it; each reuse
cites its origin and why it still holds.

D13. When the capability is adopted, governed effects are enforced at the
domain write points (Store, CLI, MCP). Attention is not a gate; a pending item
shown only in a UI is not enforcement.

D14. Scheduling, workers, leases, queues, retries and notification delivery
stay outside the POSE core. Harne8 consumes the contracts and decides when to
present a request.

D15. Costs and outcomes are measured as separate dimensions. No aggregate
quality or efficiency score.

D16. Authorizations already given are part of the context. The engine and the
workflows do not produce a confirmation only to fill the model, and do not
record an authorization the engine did not observe.

Rejected alternatives:

- **Persist one obligation per pending item.** Duplicates derivable state and
  creates a second ledger to reconcile with specs, reviews and follow-ups.
- **A `waiting-*` family of lifecycle statuses.** Mixes phase, cause,
  knowledge and satisfaction on one axis and multiplies transitions.
- **Reuse follow-ups for pre-delivery requests.** Follow-ups are debt that
  survived a delivery; reading them as blockers would re-introduce the drift
  the analysis measured (123 open items that are not the implementation
  backlog).
- **`available_work` from the absence of blockers.** Absence of a known
  relation does not prove independence; the projection reports what it knows
  and the limits of its coverage.
- **A new `pose next` command as the first deliverable.** State/Attention with
  structured next steps delivers the value without growing the CLI.

## Consequences

The engine gains one versioned read-model schema, one new persisted journal
(ActionRequest) and typed diagnostics in the producers. Existing fields
(`Ready bool`, `Blockers []string`, `NextAction string`) keep their meaning
during migration; new meaning lives in new fields.

Adapters are added incrementally; each answer lists the producers it did not
consult. The first vertical slice covers spec dependencies, review judgment,
closeout and ActionRequest; release queues, docs reviews, capability triggers
and findings follow with explicit coverage.

Enforcement is opt-in by capability and gated by a pilot with a stop/go record
(`pose-agency-readiness-pilot`). An instance that does not adopt keeps the
documented legacy behaviour; an engine update never adds a gate silently.

Every new gate in this program is reviewed against its cheapest formal
satisfaction (the adversarial corpus in `pose-mechanization-adversarial-corpus`).
