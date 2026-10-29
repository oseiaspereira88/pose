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

## Adopt delegated review

Run `pose init` in a new project for an interactive delegation offer (default
yes). Acceptance detects Codex and Claude CLIs on PATH without starting them,
creates read-only agent profiles, and provisions a project-specific native
executor issuer outside the repository. Its pin grants attestation signing,
never human authority. With no supported CLI, a refusal or a non-interactive
init, delegation is explicitly off and no agent policy or key is created.
An existing instance is preserved; `pose update` only suggests adoption.

For an existing project, configure `.pose/policy/delegation.json`, including
review-capable agents with explicit executor principals, executor-attested
provenance and `conclusions.issuer`. Pin that exact local issuer for
attestations; bind the authority project to `.pose/project.json` and configure
its audience. Preview and apply:

```bash
pose adopt delegated-review
pose adopt delegated-review --apply
pose delegate agents --json
```

The B17 signed-run-proof barrier remains live: an old prefix-only approval
can stop satisfying current verification after adoption; its record is preserved.

With delegation on, a new plan requires a different actor. Automatic routing
that cannot find another vendor or model opens a maintainer action request;
mandatory-human review also escalates. Disable explicitly with
`pose adopt delegated-review --off --apply` to allow new different-actor
plans to use same-actor-separate-execution, disclosed as
`delegated-review-disabled` in the plan, attestation, verification and closeout.
Mandatory-human and a stronger protected baseline remain stronger. Absence of
the policy key retains legacy behavior; decline/defer record configuration
decisions without changing that policy. Disabling preserves pins and history.

For an agent review of the change that disables an adopted baseline, the
sealed plan retains `delegated_run_proof_required`. Independent routing and
automatic verified conclusion remain available for that protected review only;
unrelated off-capability work stays off. A reviewer prefix alone cannot approve
the disabling change.

If init saves the policy or key but cannot activate, retain those artifacts,
inspect the publication state, finish pinning the issuer and retry adoption.
Inspect a durability-unconfirmed publication before retrying.

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
run alone does not approve the spec. With delegation adopted and its issuer
configured, the engine signs and records the completed run's verified typed
conclusion automatically. An incomplete or invalid conclusion is refused with
its reasons; a rejecting conclusion is recorded and blocks approval until the
subject changes. The agent is a local command, not a host security sandbox.

The bundle records who implemented it when it is sealed: the vendor detected
from the sealing agent's session, or declared with `--seal --lead
<vendor>[/<model>]`, and labelled as detected or declared. Seal from the
implementing session; resealing the same content later, from a reviewer's
session, keeps the identity the bundle already recorded. Routing then prefers
an agent of another vendor, and every run records the independence it
achieved — another vendor, another model, or the same model, which never
satisfies a review that requires a different actor. A model nobody declared is
unknown and never counts as a difference.

Read the returned work and the engine's conclusion. Follow its `next_actions`;
answer a rejection with a change, current evidence and a new sealed bundle.
Preserve the rejection rather than rerunning the unchanged subject for a more
favorable verdict. Finish with `pose review verify`, `pose review-check` and
`pose closeout-check` for the scope. Use `pose delegate task <prompt-file>` for
general work; a task never supplies a review or delivery evidence. See the
[CLI reference](cli.md#component-aware-and-convergent-review) for explicit
attestations when automatic conclusions do not apply.

## Request a delegated review any runner can fulfil

A project can ask for a delegated review, adjudication or smoke run as an
action request, and let whichever runner it uses — a developer's machine, a
CI job or Harne8's Conductor — fulfil it through the same contract:

```bash
pose review request spec:<slug> --kind delegated-review [--purpose review|adjudicate|smoke] [--agents <name,...>] --apply
# Locally: run the request's brief with an accepted agent; with
# delegated_review and conclusions.issuer set, the engine concludes the run
# and the request is fulfilled.
pose delegate review spec:<slug> --request <act-id>
# From any runner: submit the completed, concluded run.
pose action fulfil <act-id> --run <run-id> --runner agent:<runner> --idempotency-key <key> --expected-revision <n> --channel ci --apply
```

The request binds the project, the worktree it was asked from, the scope,
the sealed bundle, the brief, the notes, the policies, the agents it accepts
and the destination of the conclusion — an attestation for a review, an
adjudication record of the spec for an adjudication, smoke evidence for a
smoke run. No actor answers it. A run fulfils it only when the engine reads,
of that run: the request's brief and bundle, an accepted agent, a completed
run whose containment completed, a signed proof that verifies, an intact
attempt ledger, and the conclusion the engine recorded at the request's
destination. A rejecting run fulfils the request — the review was performed —
and nothing is approved: approval stays the attestation's. A superseded
bundle or a policy change invalidates the request, even once fulfilled; it is
asked again with `pose review request ... --supersedes <act-id>`, which retires it.

The proof of a run verifies in any checkout of the project: the worktree it
ran in is recorded, not required of the reader — the project must be
declared in `.pose/project.json`, so every checkout is the same project. A
CI job therefore runs the
requested review in its own checkout — with `delegated_review` and
`conclusions.issuer` set and the issuer key on the runner, the engine
concludes the run and fulfils the request there — and commits what it wrote
under `.pose/`, the request's journal included:

```yaml
# .github/workflows/delegated-review.yml (example)
on:
  workflow_dispatch:
    inputs:
      slug: {required: true}
      request: {required: true}
      agent: {required: true}
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: pose delegate review spec:${{ inputs.slug }} --request ${{ inputs.request }} --agent ${{ inputs.agent }}
      - run: pose action show ${{ inputs.request }} | grep -q 'action.state=fulfilled'
      - run: git add .pose && git commit -m "delegated review of ${{ inputs.slug }}" && git push
```

A runner that concludes a run itself — `pose review conclude <run> --issuer
<name> --apply` — submits it with `pose action fulfil`, from a file:

```json
{"request_id": "act-…", "request_digest": "sha256:…", "expected_revision": 1,
 "idempotency_key": "conductor-…", "run_id": "rrn-…", "runner": "agent:conductor", "channel": "conductor"}
```

`pose action fulfil --envelope <file> --apply` reads it, and the MCP tool
`pose_action_fulfil` takes the same object as `envelope` (or the fields
`id`, `run`, `runner`, `request_digest`, `expected_revision`,
`idempotency_key`, recorded on the `mcp` channel). The runner is an `agent:`
principal — a fulfilment records no person — and chooses neither the
authority nor the destination; a retry with the same key and run is a
no-op.

A person is asked only by exception, each time as a decision request to the
maintainer role with the question, its options and their consequences: a
scope whose plan reserves its review for a person, an attempt budget spent
without the review, delegated reviewers that disagree on one bundle, and a
business decision a reviewer raised in its draft (`business_decision`).

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
