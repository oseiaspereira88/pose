# ADR: Sealed review bundles and separate attestations

## Status
Accepted (2026-08-13) — implemented by spec `pose-review-bundle-convergence`

## Context

POSE currently binds immutable review attempts to `scope_digest` and, under
component-aware policy, `plan_digest`. The implementation computes a spec scope
digest from the complete Markdown body. Closing a scope then appends or changes
execution logs, requirement evidence, final reporting, lifecycle metadata,
state refreshes and provenance records. Some of those changes alter the object
that was just approved or extend the Git range used by validation. The result
is a non-convergent loop: approval can make its own validation or freshness
inputs stale.

The planning reproduction adds one new delivery target to a draft spec. A clean
`HEAD` snapshot passes `pose check --strict`, while the worktree then reports 92
stale/missing integration-evidence errors across unrelated historical specs.
The current plan also renders 28 per-path mapping warnings for one component.
This confirms that invalidation and human guidance are currently broader than
the consumed subject, not merely that one closeout happened to be difficult.

The same implementation showed a second boundary problem. A Git provider may
review a transient merge/squash object that is not a durable fetchable ref,
while POSE needs a stable offline identity for the reviewed content. Harne8's
Conductor is the appropriate place for reviewer assignment, durable workflow,
retries and remediation, but POSE must remain independently usable and capable
of deciding whether a scope can close.

Options considered:

1. Add more exclusions to the existing full-body `scope_digest` — rejected
   because path/field exceptions do not define a stable semantic review subject
   and will regress as new closeout artifacts appear.
2. Freeze the entire repository from review start through closeout — rejected
   because recording the approval, lifecycle result and derived state is a
   legitimate part of closeout and should not require an artificial commit
   dance.
3. Move review validity entirely to Harne8/Conductor — rejected because it
   makes an online service the authority for an otherwise offline,
   provider-neutral project contract.
4. Seal a canonical semantic bundle, record approval as a separate attestation
   and let an optional orchestrator operate on those contracts — selected
   because the reviewed subject reaches a fixed point without losing offline
   verification or durable orchestration.

## Decision

Introduce a versioned immutable `ReviewBundle` as the only new review subject
after explicit policy adoption. Its canonical payload contains the semantic
scope projection, attributed implementation subject, effective review-plan
inputs, required validation identities and hierarchical child bundle digests.
It does not contain lifecycle transitions, reviewer identity, attestation data,
rendered operational reports or derived state that the closeout itself creates.

Classify attributed inputs through a closed, explainable registry. Parse the
spec and roadmap into typed semantic projections instead of hashing their
entire Markdown bytes. Unknown attributed paths fail sealing. Consumed policy,
profile, rule and index slices remain governed even when their containing files
are generally generated or shared.

Derive the bundle ID from SHA-256 of byte-canonical JSON. Store the envelope
append-only under `.pose/review-bundles/`. Use canonical patch and sorted
tree/content-manifest digests as stable implementation identity; retain
base/head and provider merge SHAs as advisory provenance rather than the sole
verification key.

Record review approval as a separate immutable attestation referencing the
exact bundle ID and digest. Creating, importing, superseding or verifying an
attestation never changes the bundle. `review-check`, `closeout-check` and
`pose close` verify the attestation and then perform their own lifecycle and
bookkeeping gates without adding new inputs to the approved subject.

When governed input changes, seal a superseding bundle and derive a typed delta.
Permit targeted rereview only by explicit policy: a prior criterion disposition
may be reused when its criterion contract digest, governed input slice,
evidence identities and independence requirement are byte-identical. POSE, not
the orchestrator, verifies reuse and projects one complete final disposition.

Derive invalidation from explicit consumed-input edges, never from every
delivery target that happens to share a module. A new target cannot redefine the
historical subject of an unrelated closed spec. Keep detailed diagnostics in
canonical JSON, but group repeated path diagnostics and phase human tool
guidance so rigor does not become one warning or checklist row per artifact.

Keep orchestration outside the core. POSE owns bundle preparation, sealing,
export, import, verification and closeout authority. Harne8/Conductor may own
assignment, retries, findings, remediation and attestation production through
the same public JSON schemas. Local/manual attestations remain supported
offline. Signed external envelopes are an optional trust-policy layer and may
not weaken local independence or evidence requirements.

Adopt the bundle schema explicitly and preserve schema-v1/v2 artifacts and
behavior for non-adopters. Keep `pose review record` as a compatibility entry
point. Do not promote bundle-backed component-aware review from preview until
the source and installed binaries prove one-cycle convergence, derived-only
non-staleness, semantic-change staleness, synthetic-provider provenance and
offline closeout.

## Consequences

- Positive: review gains a stable subject; recording approval and closing a
  scope cannot invalidate the approval by construction.
- Positive: false staleness becomes testable through an explicit semantic
  include/exclude contract instead of accidental full-file hashing.
- Positive: patch/tree identities survive transient provider merge refs.
- Positive: Conductor can implement a humane durable rereview loop without
  becoming the sole review authority.
- Positive: narrow remediation can reuse exactly unaffected criteria while
  POSE verifies the complete final decision.
- Positive: new delivery targets do not force historical evidence regeneration
  for unrelated closed work.
- Positive: grouped diagnostics and phase-aware tools keep the review contract
  complete without exposing users to mechanically duplicated work.
- Trade-off: canonical payloads, classification, schemas and digest algorithms
  become public compatibility contracts that require golden fixtures and
  schema-versioned evolution.
- Trade-off: sealing fails on unclassified attributed paths, creating explicit
  maintenance when new artifact classes are introduced.
- Trade-off: immutable bundles add repository artifacts, although deterministic
  IDs and idempotent writes prevent duplicate logical state.
- Trade-off: signed external envelopes add trust-policy complexity; they remain
  optional so the base workflow stays offline.
- Neutral: validation, follow-up, surface and lifecycle gates still apply; they
  are separated from the review digest, not waived.

## Review triggers

Revisit this decision if the semantic projection causes material false-negative
freshness, if patch/tree identities are insufficient for a supported provider,
if targeted reuse cannot be bounded safely, if signed envelope verification
requires a new trust root, or if a future online-only product mode proposes
moving closeout authority out of POSE.
