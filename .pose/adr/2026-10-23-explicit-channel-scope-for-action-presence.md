# ADR: Explicit channel scope for action presence

## Status
Accepted — prospective implementation decision by Codex under the user's
2026-10-23 B03 authorization. This is not human confirmation or policy adoption.

## Context

B03 exposes a mismatch between signed-action-answers R5/D3 (human presence)
and schema 1's delivered SSHSIG-only enforcement. An issuer claim and an SSH
signature are alternative identity proofs under foundation D2; they do not
establish equivalent presence. Declared unsigned human answers also bypass the
current check. B01 verifies recorded proof and B04 prohibits retrospective
policy or historical authorization claims.

Separate same-vendor Codex design advice, actual run
20261023T180459-codex-3011272, recommended explicit versioning and migration.
The advice is not a sealed delivery approval. OpenSSH's
[security-key protocol](https://raw.githubusercontent.com/openssh/openssh-portable/master/PROTOCOL.u2f)
binds authenticator flags to the signed message. This authenticates the
registered key's assertion; it does not attest hardware provenance or identify
the person touching a device. Synthetic fixtures do not demonstrate real touch.

## Decision

Add action-policy schema 2. Keep require_presence false by default. When true,
require presence_scope to be exactly universal or sshsig; reject unsupported
scopes and reject this new setting in schema 1. Older engines reject schema 2.
Expose schema 1's interpretation as legacy-sshsig-only. Refuse NEW human answers
with migration-required when schema 1 enables presence, until the operator
explicitly selects schema 2 and a scope. Do not silently choose either scope.

Enforce admission centrally after verifying every supplied identity proof and
before appending an answer, under declared or verified identity assurance.
Universal scope requires an authenticated SK user-present assertion for every
human answer. Issuer-only claims have no supported presence proof; refuse them
explicitly. Missing proof and valid proof lacking presence have distinct
diagnostics from invalid signatures and unsupported issuer presence. SSHSIG
scope applies the same requirement whenever an SSH signature is supplied; an
issuer-only answer is explicitly outside that scope and establishes no presence.
Human requirements do not require touch from agents or label agent presence human.

Keep the existing signed statement and issuer claim contracts. A valid SK
signature may accompany a valid issuer claim; neither invalid proof can fall
back to the other. The project-local SSH audience is the signed project plus
pose-action-answer namespace. The issuer audience/project realm is the current
pinned review policy. Both proofs bind the same actor, request ID/digest,
answer and reason; the request digest includes its logical project. The SSH
statement additionally binds idempotency; journal revision and idempotency
checks prevent claim reuse from publishing a second answer. Do not claim the
issuer separately signs idempotency or that these two audience names are equal.

Reverify historical proof with the existing B01/B04 rules and expose presence
as a derived observation from authenticated SK flags. Treat historical presence
requirements as unrecorded, never as the current policy or an editable receipt.
Do not introduce an unsigned historical compliance label. Keep declared legacy,
issuer time precision, current issuer revocation and SSH key-rotation semantics.

Render policy and validate it through the shared decoder before publication.
Preserve scope/version and unknown non-security metadata through policy edits.
Expose effective policy in previews as not-evaluated, and distinguish identity
verification from presence in human/JSON CLI, MCP and doctor.

## Consequences

Classify migration-required refusals as a breaking public behavior change,
requiring a major release; mark the unreleased fragment breaking:true. This
work merges the engine contract, not a release/tag or Harne8 policy adoption.
Schema-1 policies with presence disabled and existing historical proof remain
compatible. Operators explicitly migrate enabled policies; no automatic update
selects their security scope. Unknown scopes fail closed before writes.

Reject silently retaining the legacy bypass: it leaves scope ambiguous. Reject
silently imposing universal scope on schema 1: it adopts an operator policy
without an explicit migration. Reject a scope field ignored by older engines,
issuer/Portal authentication as touch, and a new presence claim without an
authenticated protocol. Existing review-policy semantics and completed foundation
specs are preserved; this ADR records their reconciliation without rewriting them.
