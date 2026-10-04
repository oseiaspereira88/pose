# Obligations, agency and readiness

**Doc type:** Explanation &nbsp;·&nbsp; **Applies to:** POSE 6.x (current stable)

POSE answers one operational question for a delivery: **what is still owed,
by whom, from which source, restricting which phase of which scope, and what
continues meanwhile?** This page describes the model behind that answer. The
decisions are recorded in the ADR
`obligations-are-projected-action-requests-are-persisted`.

## Obligations are projected, not stored

Many subsystems already know that something is still owed: readiness knows a
prerequisite is not done, review knows a judgment criterion has no answer,
closeout knows evidence is stale. An **obligation** is the common shape those
facts are projected into. It is a read model:

- the source stays authoritative — `depends_on`, the sealed review plan, the
  evidence index, an ActionRequest;
- the projection holds no "resolved" state of its own; when the source
  changes, the obligation turns satisfied or disappears;
- reading obligations never writes anything.

Only facts no other source holds are persisted. Today that is one kind: a
material request to a person or an external system (an **ActionRequest**).

## The record

Every obligation carries (schema `pose-mcp/schemas/v1/obligation.schema.json`):

| Field | Meaning |
|---|---|
| `id` | `obl-<16 hex>`, derived from project, producer, source node, rule and a producer discriminator. Wording, order and time never change it. |
| `source` | Producer and qualified source reference, with a pointer back into the producer's own output. |
| `category` | `dependency`, `judgment`, `evidence`, `reconciliation`, `remediation`, `release`, `actor-action` or `residual-debt`. |
| `reason_code`, `condition`, `rule` | Why it exists, what satisfies it, which rule requires it. |
| `recipient` | A principal, a role, or `unassigned` — never guessed. |
| `targets`, `effects` | The qualified nodes it concerns, and per phase (`start`, `execution`, `review`, `closeout`, `release`) whether it `block`s or is `advisory`, over which scope. |
| `satisfaction` | `pending`, `satisfied`, `waived`, `cancelled` or `invalidated`, as the producing domain defines it. |
| `knowledge` | `known` or `unknown`: whether the engine had current data to answer. |
| `waiting` | What the condition depends on: an artifact, an actor, an external system, the running execution, nothing, or unknown. |
| `observation` | Source revision, freshness and coverage, with limitations. |
| `message` | A rendering for people. Nothing may parse it. |

Four questions stay on separate fields because they have different answers.
`unknown` is never a resolution: an obligation the engine could not observe
stays `pending` and keeps restricting its phase.

## Qualified references

`R4` is local to one spec. Every target is qualified by project and artifact:
`xref:<project>/spec:<slug>#requirement:R4`. Two specs, or two projects, with
the same slug and requirement id never collide.

## Legacy blockers

A subsystem that still reports a blocker only as text is projected as
`coverage: legacy-opaque`: its origin and the limitation are kept, and it
gains no invented actor, target or unblock condition.

## Action requests

An **ActionRequest** is the one obligation POSE persists, because nothing else
records it: a material request to a person or an external system.

```bash
pose action open --origin spec:storage --kind decision \
  --question "Will the next delivery keep reading schema v1?" \
  --option preserve-v1="Keep the v1 reader" --option break-v1="Consumers migrate" \
  --recommend preserve-v1 --recipient-role maintainer --requested-by agent:impl \
  --target requirement:R4 --effect execution:block --effect closeout:block \
  --subject requirement:R4 --apply
```

Open one only when a different answer would materially change execution,
scope, authority, risk acceptance, closeout or publication, and no authorization
already given covers it. Everything else is conversation.

A resolution is bound and guarded:

- it names the request digest it answers — a changed question, option, target
  or effect is a new request, and a changed subject invalidates an earlier answer;
- it names the revision it read and carries an idempotency key — a retry is a
  no-op, a second answer against the same revision is refused;
- it comes from an actor holding the recipient role in `.pose/policy/actions.json`;
  under `identity_assurance: verified` it also carries a signed claim from a
  trusted issuer bound to project, audience, digest, principal and answer.

**Answered is not satisfied.** A declined approval, a rejected acceptance or a
failed external operation is recorded, and the condition stays unmet.

## Reading it: Attention

`pose state --attention` (and the MCP tool `pose_obligations`) read the
projection. The answer starts with coverage: if any producer failed or is not
integrated yet, it says `INCOMPLETE` before anything else, because an empty
group is only meaningful when every producer was read. Then it lists what
needs a person or role (`--actor` narrows this), what restricts each phase,
and residual debt, which never restricts a phase by default.

```bash
pose state --attention --scope spec:storage-refactor
pose state --attention --actor reviewer --json
```

Attention is not a gate. When governed effects are adopted, the write points
of start, close and release revalidate the obligations of their own phase.

## What this model does not do

It does not schedule work, assign workers, deliver notifications or compute a
quality or efficiency score. Scheduling and presentation belong to a control
plane such as Harne8, which consumes these contracts.
