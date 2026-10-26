# Review, judgment and trust

**Doc type:** How-to & explanation · **Applies to:** POSE v7.4.1 · **Reviewed:** 2026-10-20

A sealed bundle fixes the subject and the contracts that judge it. An
append-only attestation records the reviewer's conclusions. The review plan
says which evidence, judgments and reviewer separation are owed. Signing
proves a binding to a trusted key; the reviewer still answers for the conclusion.

## Prepare the subject

```bash
pose review-plan spec:customer-export --json --explain
pose validate --strict
pose review bundle spec:customer-export --seal
pose review auto-attest spec:customer-export --reviewer agent:reviewer-a
```

Use the returned bundle id for subsequent operations. `auto-attest` prepares
mechanical answers from sealed evidence and reports pending judgments. It
cannot turn a successful test into an engineering conclusion. Evidence must
have an accepted class and belong to the component the plan names.

The opt-in engineering-judgment, high-criticality and structural-materiality
profiles add obligations according to scope. Observed material structural facts
can require mappings to requirements, assumptions or decisions (R/A/D), or an
explicit disposition with rationale. Invalidated assumptions and withdrawn
decisions cannot serve as active causal bases under adopted causal closeout.
The engine checks the relation; the reviewer judges whether it is convincing.

A policy change cannot authorize its own weakened review. The protected
baseline restores stronger obligations from the resolved base where supported;
an unresolved base is disclosed as unprotected. Read the plan's reasons and
coverage before recording a conclusion.

## Delegate a review

```bash
pose review brief <bundle-id> --kind review
pose delegate review <bundle-id> --dry-run
# After configuring and inspecting the agent:
pose delegate review <bundle-id> --label <name>
pose delegate ls <name>
```

The brief is generated from the bundle. Implementer notes are labelled
separately. Dispatch runs a configured command in a disposable worktree at the
sealed commit and retains its outcome under `.pose/review-runs/`. File changes,
timeouts, budget overruns and execution failures are failed runs. A completed
run does not create an attestation or approve the spec. The agent is a local
command, not a host security sandbox.

The bundle records who implemented it when it is sealed: the vendor detected
from the sealing agent's session, or declared with `--seal --lead
<vendor>[/<model>]`, and labelled as detected or declared. Seal from the
implementing session; resealing the same content later, from a reviewer's
session, keeps the identity the bundle already recorded. Routing then prefers
an agent of another vendor, and every run records the independence it
achieved — another vendor, another model, or the same model, which never
satisfies a review that requires a different actor. A model nobody declared is
unknown and never counts as a difference.

Read the returned work, complete every judgment, then use `pose review attest`
and the [CLI reference](cli.md#component-aware-and-convergent-review). Finish
with `pose review verify`, `pose review-check` and `pose closeout-check` for
the scope. Changed semantic inputs require a fresh bundle and current evidence.

## Sign reviews without a hosted service

A native Ed25519 issuer stores its private key outside the repository. Inspect
the issuer and pin only the authority it should exercise:

```bash
pose issuer init local-review
pose issuer pin local-review --attestations --apply
pose issuer list
```

A key usable by agents on the account should not be granted human authority.
When verified human authority is required, configure a separately protected
issuer and an explicit authority grant. Native and external issuers coexist;
pinning or rotating one does not remove the others.

`pose review attest … --sign <issuer>` signs the attestation. `--authority`
adds the claim used by verified identity. The claim binds the project and the
verifier audience separately, the bundle digest, reviewer and required
executions. Review independence, identity assurance and attribution are
separate disclosures:

| Question | What the record establishes |
|---|---|
| Who prepared, concluded, confirmed or applied? | Explicit attribution, with confirmation bound to the exact conclusions. Authorizing an operation differs from adopting its conclusions. |
| Was another actor or execution required? | The review plan's independence floor. |
| Is identity declared or verified? | A declared label, or an authenticated claim under configured trust. |
| Is cognitive independence proven? | It remains not observable. |

The MCP `pose_review_attest` writer accepts a trusted signed envelope; it does
not provide an unsigned shortcut. See [MCP](mcp.md) for preview and apply.

## Adopt signing with existing history

Preview `pose adopt signed-attestations` before applying it. From v7.3.0,
adoption can seal existing unsigned attestations into a signed legacy ledger
under `.pose/review-ledgers/`. Their bytes remain unchanged. New or altered
attestations must satisfy signing requirements. Use `--issuer <name>` when
more than one local issuer is eligible. The ledger preserves attributable
history; it does not retrospectively establish verified human authorship.

## Development-only hardening

Pending changes strengthen shared approval preflight, proof disclosure,
historical trust, source reuse and incomplete-source handling. Additional
Git/dispatch snapshot work is still `in-progress` at this review. These are
separate from the behavior shipped in v7.4.1; see [development status](evolution.md#on-the-development-branch-not-in-v741)
before depending on their flags, schemas or refusal codes.
