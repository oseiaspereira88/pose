# ADR: Sealed review bundles and separate attestations

## Status
Accepted (2026-08-13) — implemented by spec `pose-review-bundle-convergence`
Amended (2026-09-08) by spec `pose-review-subject-unclassified-removals` — see Amendments
Amended (2026-09-08) by spec `pose-attestation-evidence-must-be-in-the-bundle` — see Amendments
Amended (2026-09-09) by spec `pose-evidence-scoped-to-component` — see Amendments
Amended (2026-09-09) by spec `pose-component-evidence-is-not-inherited-upward` — see Amendments
Amended (2026-09-09) by spec `pose-bundles-seal-the-contracts-that-govern-them` — see Amendments
Amended (2026-09-10) by spec `pose-bundle-findings-take-the-contract-the-legacy-path-had` — see Amendments
Amended (2026-09-10) by spec `pose-reuse-is-sealed-signing-stays-live` — see Amendments
Amended (2026-09-10) by spec `pose-report-coverage-that-rests-on-inference` — see Amendments

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
entire Markdown bytes. Unknown attributed paths fail sealing, except removals
(amended 2026-09-08). Consumed policy,
profile, rule and index slices remain governed even when their containing files
are generally generated or shared.

Derive the bundle ID from SHA-256 of byte-canonical JSON. Store the envelope
append-only under `.pose/review-bundles/`. Use canonical patch and sorted
tree/content-manifest digests as stable implementation identity; retain
base/head and provider merge SHAs as advisory provenance rather than the sole
verification key.

The sealed payload also names the governance contracts in force when it was
sealed, so verification judges the bundle by the rules that existed then rather
than by a date in today's policy (amended 2026-09-09), and the two closeout
gates that depend on policy — whether reservations may close a scope, and which
risk severities may be accepted (amended 2026-09-10).

Record review approval as a separate immutable attestation referencing the
exact bundle ID and digest. A criterion recorded `passed` must cite evidence the
bundle contains, of a class the criterion asks for (amended 2026-09-08), from a
component it answers for — which a component containing it does, and a directory
inside it does not (amended 2026-09-09). Coverage that rests on the containing
module rather than on a result naming the target is reported, not refused
(amended 2026-09-10). Creating, importing, superseding or verifying an
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
- Trade-off: sealing fails on unclassified attributed paths that still exist,
  creating explicit maintenance when new artifact classes are introduced. An
  unclassified removal is admitted as `removed` instead (amended 2026-09-08).
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

## Amendments

### 2026-09-08 — an unclassified removal is admitted as `removed`

Spec `pose-review-subject-unclassified-removals`.

The accepted decision made every unclassified attributed path fail sealing. It
was written when classification always had content behind it: a path the shape
rules do not recognise is a class POSE has not been taught, and refusing to seal
is how that stays visible instead of being reviewed as nothing.

A removal is the case that premise does not cover. The path is gone, so there is
no content to classify and none to read; the reviewable fact is the deletion,
and the deletion is already in the subject. Refusing the whole bundle over it
spends the entire review to tell the reviewer nothing they cannot see — the
third time in a week an unrecognised path blocked everything, after gitlinks in
1.7.11 and their precedence in 1.8.0. The previous two were fixed by teaching
the classifier the case; this is the first where the classification genuinely
does not exist.

The revision is bounded on purpose:

- Only `action: removed` is admitted. Creations and modifications of an
  unclassified path still fail sealing, and both directions are asserted so the
  change cannot be read as a general relaxation.
- The entry is **included** in the subject as class `removed`, never excluded.
  Excluding it would make the gate pass by hiding a real change, which is the
  failure this subsystem exists to prevent.
- `removed` joins `documentation` and `governance` in the subject slice every
  criterion digests, including criteria that are not subject-sensitive. Its
  category is unknown and no content survives to show it belonged to neither, so
  a deletion must invalidate reuse rather than let a verdict issued before it
  stand over it.

This narrows the compatibility contract on the subject-class registry: `removed`
is a new public class, and consumers pinned to the previous closed set will see
it. Nothing else in the decision changes — the payload shape, digest algorithm,
attestation separation and lifecycle gates are untouched.

### 2026-09-08 — a passed criterion must be supported by the bundle

Spec `pose-attestation-evidence-must-be-in-the-bundle`.

The accepted decision separated approval from the subject and said verification
"verifies the attestation", without stating what that verification owes the
subject. In practice it owed nothing: every check was on the shape of the
attestation — that required criteria appeared, that dispositions were spelled
correctly, that `not-applicable` carried a rationale — and none on whether a
`passed` was supported. A criterion could cite evidence absent from the bundle,
evidence of a class it never asked for, or nothing at all, and verify.

Three closeouts in an adopting repository were approved that way, one against a
bundle sealing zero evidence. The separation of attestation from bundle is what
makes this decision sound; it is also what left the two free to disagree, and
nothing was closing the loop.

The revision states the missing obligation:

- A criterion recorded `passed` must cite a reference present in the bundle's
  sealed evidence, and where the criterion declares evidence classes, the cited
  evidence must be of one of them. `not-applicable` and `finding` are unchanged:
  the first already carries a reviewer's rationale in place of evidence, and the
  second records a problem rather than a clearance.
- `pose review auto-attest` no longer synthesises a reference. Where the scope
  is expected to carry validation evidence it refuses; where it is not, it
  records the criterion `not-applicable` with a rationale.
- A criterion demanding an evidence class no registered check may emit blocks
  the plan. Dropping the class — the treatment tools receive — is wrong for a
  criterion, because one left with no class accepts any sealed evidence: the
  demand does not weaken, it disappears.

**Compatibility.** This fails attestations that already verify. They are
unsupported and were reported as sound, so the failure is the correction, not a
regression; the remedy is a superseding attestation, which this decision's
append-only model already provides, and no edit path is added. Plan digests
change wherever a profile is reconciled, and therefore bundle digests, which is
the ordinary consequence of a governed input changing and what supersession
exists for.

**The vocabulary underneath.** Two lists govern evidence classes and they
disagree in both directions: `ValidEvidenceClasses`, what a check may emit, and
`reviewEvidenceClassCatalog`, what a profile may demand. Six of nineteen classes
appear in both. Ten a profile may demand — including `test`, `contract`,
`validation` and `observability` — can never be produced; three a check may emit
can never be demanded. The profiles shipped with POSE are reconciled to the
intersection here, and the plan now blocks on the rest. Unifying the two lists
is not done, and is recorded as a follow-up.

### 2026-09-09 — evidence answers for the component it was asked about

Spec `pose-evidence-scoped-to-component`.

The 2026-09-08 amendment tied a passed criterion to evidence the bundle
contains, of a class the criterion demands. It did not tie it to a component,
and recorded that as a known gap: a backend criterion could be satisfied by a
frontend sibling's integration result, which is real, demanded, and silent about
the thing the criterion is about.

Closing it required a change to the payload this decision defines, which is why
it is recorded here rather than left to the spec.

**The canonical payload gains `plan.selected_profiles`**, recording which
components each profile was selected for. A criterion names the profiles it came
from; nothing mapped a profile to what it matched, so the sealed subject could
not decide component scope even in principle. The alternative — re-reading the
selection from the current policy at verification time — was rejected: it would
judge an immutable bundle by today's configuration, which is the property this
decision exists to establish. This engine had already made that mistake once,
resolving a profile ref against `policy.Profiles` rather than against what the
attempt recorded.

The rule is bounded:

- Only a criterion demanding an evidence class is scoped. Without one the plan
  made no claim about what the evidence shows, and narrowing by module would
  invent a constraint the plan never stated.
- A criterion governed by any base profile answers for every component and is
  never narrowed. Only one governed solely by overlays is constrained, to what
  those overlays matched.
- A tool is scoped by its own `component`, which the plan already recorded.
- `moduleMatchesTarget` decides coverage, so a repository-root result satisfies
  any scope. That is the engine's single definition of module coverage, kept
  rather than forked here.

**Compatibility.** A bundle sealed by this release does not digest the same as
one sealed before it, because the payload carries a field it did not. That is
the ordinary consequence of a governed input changing and is what supersession
exists for. The dated adoption exemption covers these checks as it covers the
rest of the evidence-support rules, so a completed closeout recorded before an
instance received the contract keeps its approval.

### 2026-09-09 — component evidence answers downward, not upward

Spec `pose-component-evidence-is-not-inherited-upward`.

**What changed.** A result registered for a module answers for a target or
criterion inside that module. It no longer answers for one that contains it.

**Why.** `moduleMatchesTarget` accepted a path prefix in either direction, so a
result from `site/api` satisfied a criterion about `site`. It covers one
directory of the component and was accepted as covering all of it, which is the
same shape as the sibling case the previous amendment closed: real evidence, of
the demanded class, silent about most of what it was asked about.

The other direction is not the same claim. A module-wide run — `go test ./...`,
`npm test` — does exercise its subtree, and running checks once at the module
root is how nearly every project is laid out. Refusing it would not tighten the
gate; it would make the ordinary layout unsatisfiable and push projects to
declare per-directory checks that run the same command.

**Options considered.**

1. Leave both directions. Rejected: partial coverage accepted as complete is the
   defect this ADR's previous amendment exists to prevent.
2. Refuse both prefixes and require an exact module match. Rejected: it breaks
   the layout where a module runs its checks once, and POSE would be inferring a
   stricter claim than the result actually makes in the other direction too.
3. Keep downward, refuse upward. Selected.

**Consequences.**

- A component whose evidence sits only in a directory inside it now has no
  evidence. That is the correct reading and it is a new refusal: such a scope
  stops closing until a check is registered for the component, or the target is
  declared at the directory that actually has one.
- The repository-root rule is untouched. A single-module project is its own
  component, and refusing that would break every repository that never split.
- Nothing in this repository exercised either prefix direction: its delivery
  targets carry module `.` or `pose-mcp` and its results carry `pose-mcp`, so
  matching is by equality or by the root rule. The change was verified against
  fixtures, and `surface-check` produces the same 524 `validated-by` edges
  before and after.

### 2026-09-09 — a bundle names the contracts that govern it

Spec `pose-bundles-seal-the-contracts-that-govern-them`.

**What changed.** The sealed payload gains `governing_contracts`: the ids of the
governance contracts this engine knew when the bundle was sealed. Verification
asks the bundle whether a contract governed it. The adoption date in the policy
is consulted only for a bundle that carries no such list.

**Why.** The exemption that keeps a finished scope's approval when a contract
arrives after it compared the review's timestamp against
`contract_adoptions` in the live policy. That date is editable and re-read on
every verification, so moving it forward retroactively exempts more historical
closeouts, and an immutable bundle was judged by today's configuration. It is the
same objection this ADR already accepted for `selected_profiles` one amendment
earlier, applied to the other thing verification still read live.

Sealing the contract set also removes the need for a new dated marker per
contract. A bundle sealed by an engine that did not know a contract never lists
it and is never retroactively held to it — a property a date cannot have,
because a date can be edited afterwards and a sealed digest cannot.

**Options considered.**

1. Keep the dated exemption. Rejected: the defect is that an editable value
   decides a historical verdict.
2. Stamp the contracts in the envelope rather than the payload, so digests do not
   change. Rejected: what is outside the digest is not sealed, and therefore
   editable — the defect in a new place.
3. Derive the governing contracts from the bundle's own content — `selected_profiles`
   implies component-aware, classed citations imply evidence-vocabulary. Rejected
   as the primary mechanism: it infers what should be declared, and it does not
   generalise to a contract that leaves no observable trace. It is what makes the
   483 already-sealed bundles readable, and that is what the dated fallback does
   instead, explicitly.
4. Seal the contract ids in the payload, with the dated rule demoted to the
   reading of an unstamped bundle. Selected.

**Consequences.**

- A bundle sealed by this release does not digest the same as one sealed before
  it, exactly as the previous amendment's field change.
- The 483 bundles already sealed carry no list and keep the dated reading. They
  cannot be back-filled: rewriting an append-only immutable artifact is what this
  ADR exists to prevent, and re-sealing would break the attestations bound to
  their digests.
- The dated exemption is therefore not removed. It is demoted: it governs only
  bundles predating the field, and the legacy attempt path, which has no bundle
  at all. It can be deleted when no unstamped bundle is reachable, and not
  before.
- A contract added to the registry governs every bundle sealed after it, with no
  date to stamp. What is sealed is asserted to be the registry, so a contract in
  one and not the other cannot happen silently.
- Scope: this amendment covers the contracts. Verification still reads
  `allow_criterion_reuse`, `require_signed_attestations`, `reviewer_independence`,
  `accepted_risk_severities`, `allow_approved_with_reservations` and
  `component_aware` from the live policy, and each needs its own answer —
  `require_signed_attestations` most likely *should* be live, since tightening a
  signing requirement is not something an old bundle should escape. Recorded as a
  follow-up rather than settled here.

### 2026-09-10 — the bundle path enforces the finding contract

Spec `pose-bundle-findings-take-the-contract-the-legacy-path-had`.

**What changed.** Attestation validation against a sealed bundle now applies the
same finding contract the legacy attempt path always applied, and the sealed
payload gains `gates`: the accepted risk severities and the reservations flag in
force when the bundle was sealed.

**Why.** This was not a design decision that had been made and could be revisited.
`validateBundleAttestationWith` blocked a finding only when its disposition was
`open` or `changes-requested`. Everything else passed with no blocker: a
`critical` accepted risk with no owner, no rationale and no review date; a
disposition the engine does not know; a finding with neither severity nor
action. `accepted_risk_severities` and `allow_approved_with_reservations` were
not consulted at all on this path.

So a project that adopted review bundles — the mechanism this ADR introduced to
make review *more* rigorous — silently lost a gate it had before, and two policy
settings stopped taking effect without anything saying so.

**Options considered.**

1. Read the two settings live at verification. Rejected for the reason the
   previous amendment gives: a flag flipped today would approve a closeout
   recorded years ago.
2. Seal the whole contract, structural parts included. Rejected: a finding
   without a severity is incomplete under any policy, and sealing it would imply
   a project could configure it away.
3. Seal the two policy-dependent gates, apply the rest unconditionally.
   Selected.

**Consequences.**

- The sealed payload changes again, so a bundle sealed by this release does not
  digest the same as one sealed before it.
- A bundle carrying no `gates` reads as reservations refused and no severity
  accepted, which is precisely what this path did before — so no bundle already
  sealed changes verdict. Measured rather than assumed: the 469 attestations
  recorded in this repository carry no findings at all, and none is
  approved-with-reservations.
- The structural half applies to every bundle, old ones included. It is not
  configuration, and grandfathering it would be preserving a defect rather than
  a contract.

### 2026-09-10 — reuse is sealed, the signing requirement is not

Spec `pose-reuse-is-sealed-signing-stays-live`.

**What changed.** `allow_criterion_reuse` joins the sealed gates.
`require_signed_attestations` stays read from the live policy, deliberately, and
is now the only gate a sealed bundle is judged by that is not sealed with it.

**Why.** The two look alike and point opposite ways.

Reuse carries a prior criterion disposition into this attestation. Whether that
was permitted is a property of the review that happened, so reading it live lets
a project enabling reuse today retroactively legitimise an attestation that
reused a criterion when its own policy forbade it.

The signing requirement is a bar, not a permission. A project that starts
requiring signed attestations is raising it, and a bundle sealed before that must
not be permanently exempt — an exemption is precisely what a hurry, or an
attacker, would reach for. Sealing it would mean the new requirement only ever
applies to future work, which is the opposite of what adopting it means.

**Options considered.**

1. Seal both, for consistency. Rejected: consistency here would freeze a security
   requirement at the moment it was weakest.
2. Read both live, for consistency. Rejected: it reopens the retroactive
   legitimisation this ADR has now closed three times.
3. Seal reuse, keep signing live. Selected.

**Consequences.**

- Every gate a sealed bundle is judged by is now sealed with it except one, and
  that exception is stated at the call site rather than left to be inferred.
- A bundle sealed before the gates existed carries no reuse permission, so an
  attestation reusing a criterion against it is refused. That is the
  conservative reading and matches what an instance with reuse disabled already
  got.

### 2026-09-10 — coverage that rests on an inference is reported

Spec `pose-report-coverage-that-rests-on-inference`.

**What changed.** A delivery target whose only passing evidence for a class comes
from a module containing it now carries an `inferred-coverage` finding. Nothing
is refused.

**Why.** The 2026-09-09 amendment settled the direction: a result from a
containing module answers for a target inside it, because a module-wide run does
exercise its subtree. Usually. A run configured to skip a directory inside it
does not, and nothing in the result says which — that is the one part of the
rule POSE cannot check, and it was left as a known gap.

Closing it properly means a check declaring what it walked, which is a change to
the validation result contract and a migration for every project. Making the
inference visible costs nothing, breaks nothing, and produces the evidence that
decision needs: how many targets, in real repositories, actually rest on it. In
this one the answer is none — its targets and results name the same module, so
the rule matches by equality or by the root.

**Options considered.**

1. Have a check declare the subtree it walked, in the validation matrix.
   Deferred: it replaces the engine's inference with the matrix author's
   assertion, which is better because someone is accountable, but it is still a
   claim rather than the run — and it is a contract change designed against no
   observed case.
2. Derive the subtree from the check's arguments. Rejected: it reads `./...` and
   says nothing about `npm test`. Parsing a command line is inference with
   another name.
3. Have the check report what it walked. Rejected as the first step: it is the
   only option that observes rather than asserts, and it requires every tool to
   cooperate. Most do not.
4. Report the inference and keep the rule. Selected.

**Consequences.**

- A repository laid out with checks at the module root and targets inside it
  will see one finding per target and class. That is the intended visibility and
  it is a warning, so nothing stops closing.
- The finding fires only when *no* result names the target's own module: a
  target with its own check is not resting on the inference.
- Nothing changes in this repository, which is the point of measuring before
  designing the contract that would.

