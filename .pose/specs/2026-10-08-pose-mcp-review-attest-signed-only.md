---
slug: pose-mcp-review-attest-signed-only
status: in-progress
created_at: 2026-10-08
completed_at:
supersedes:
depends_on: pose-review-attribution-roles, pose-mcp-action-resolve-signed-only
remediates:
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:mcp-signed-review-attestation
---

# Spec: An agent records a review confirmation over MCP only inside a trusted issuer's envelope

## 1. Intent

### Goal

Add `pose_review_attest`, the MCP way to record a review attestation, accepting only an attestation signed by a trusted issuer (`ReviewAttestationEnvelope`), the same proof `pose review attest --envelope` already verifies. Its preview completes a draft attestation for a sealed bundle (schema, bundle digest, attested time, attestation id, confirmation digest) and returns the exact bytes the issuer signs, recording nothing.

### Business value

A person confirming an agent's review conclusions (`confirmation_mode: adopted-conclusions` or `authorized-operation`) is only provable today from the CLI, by importing an envelope file. A trusted interaction channel, such as the Harne8 issuer of `xref:proj.harne8/spec:harne8-action-request-confirmation-channel` (its R4), has no way to deliver that proof over MCP. The envelope already binds the whole attestation, including `confirmed_by`, `confirmation_mode` and `confirmation_digest`, and the authority claim inside it names the human; this spec only opens the write path, the way `pose_action_resolve` did for action requests.

### Constraints

The same verification and recording as `pose review attest --envelope`: no new claim or envelope format, no second assurance mode. No declared attestation over MCP under any policy. The server never holds a key.

### Non-goals

Signing inside the MCP server; changing what makes a confirmation verified (spec pose-review-attribution-roles); attribution supplements over MCP.

## 2. Requirements

### Functional

- R1: When called without `apply`, `pose_review_attest` shall complete the given draft attestation for its sealed bundle with the values recording would set (schema version, bundle digest, attested time, attestation id) and, when it names a confirming principal, its confirmation digest, and return the completed attestation, its canonical signing bytes and the blockers `pose review verify` would report, recording nothing.
- R2: When called with `apply`, it shall record an attestation only from a `ReviewAttestationEnvelope` that the CLI's envelope import verifies, and refuse a call without an envelope whatever the policy's assurance.
- R3: An envelope that does not verify (untrusted issuer or key, another bundle, content changed after signing, a confirmation bound to other content) shall be refused with the reason, and nothing recorded.
- R4: A recorded attestation whose authority claim names a `human:` principal equal to `confirmed_by`, signed by an issuer the review policy trusts, shall be disclosed with confirmation assurance `verified`; the same attestation with an agent principal or without the claim shall be disclosed as `declared`.
- R5: The tool catalog, `pose_review_bundle` and the manuals shall name the tool and its proof requirement.
- R6: A read-only `pose_review_prepare` shall return, for a sealed bundle and a reviewer principal, the auto-attest draft and its pending judgment criteria, writing nothing, so a channel can present a draft for a person to complete and confirm.

### Non-functional

- None.

### Security

- The MCP server holds no key and accepts no declared attestation, so an agent cannot record a confirmation the issuer did not sign, and a confirmation signed for one conclusion cannot be replayed onto another.

### Compatibility

- Additive tool; the CLI and the recorded format are unchanged.

## 3. Technical Plan

### Affected areas

Review attestation preparation (shared by recording and preview), MCP tool dispatch, definitions and catalog, manuals.

### Artifacts

- created: .pose/specs/2026-10-08-pose-mcp-review-attest-signed-only.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/mcpserver/server_test.go
- created: pose-mcp/internal/mcpserver/review_attest_test.go
- created: pose-mcp/internal/pose/review_attest_signed_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/mcp.md
- created: .pose/changelogs/unreleased/pose-mcp-review-attest-signed-only.md

### Delivery targets

- capability:mcp-signed-review-attestation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

- New governance-write MCP tool `pose_review_attest` and read-only `pose_review_prepare`.

### Data/storage changes

- None.

### Technical risks

- The preview and recording must normalize the attestation identically, or a signed preview would not verify; one shared preparation function removes the drift.

## 5. Decisions

### Decision D1
- Date: 2026-10-08
- Context: Harne8's trusted confirmation channel needs to record a person's confirmation of review conclusions with proof. The engine already verifies an issuer-signed `ReviewAttestationEnvelope` over the whole attestation, and discloses a confirmation as verified when the authority claim names the confirming human; only the MCP write path is missing.
- Options considered: (a) expose the existing envelope import over MCP; (b) add a dedicated confirmation claim bound to the confirmation digest; (c) let the channel run `pose review attest --envelope` through the CLI.
- Decision: (a).
- Rationale: one proof form, already bound to the conclusions through `confirmation_digest`; (b) would be a second claim format for what the envelope already proves, and (c) makes a remote service write into a checkout through a path no decision approved. On 2026-10-08 the maintainer chose an engine tool over (c); (a) over (b) is this spec's proposal, open to review.
- Consequences: Harne8's R4 becomes deliverable once its pin contains this tool.

## 6. Validation

### Strategy

The tool driven over HTTP JSON-RPC on a fixture with a sealed bundle and a trusted test issuer: the preview's signing bytes, a call without an envelope refused, an untrusted issuer and a tampered confirmation refused, a valid envelope recorded and disclosed as a verified confirmation, and an agent principal disclosed as declared.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/mcpserver -run 'ReviewAttest|Catalog' && go test ./internal/pose -run 'ReviewAttestation|Attribution'`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestSignedPreviewIsWhatTheEnvelopeRecordsAndTheConfirmationIsVerified test:TestThePreviewBindsOnlyWhatItIsGiven test:TestToolsCall_ReviewAttest_PreviewNeedsADraftForASealedBundle
- R2 [satisfied] test:TestToolsCall_ReviewAttest_RefusesAnAttestationWithoutAnEnvelope test:TestSignedPreviewIsWhatTheEnvelopeRecordsAndTheConfirmationIsVerified
- R3 [satisfied] test:TestAnEnvelopeOverOtherContentIsRefused test:TestToolsCall_ReviewAttest_RefusesAnUntrustedEnvelope
- R4 [satisfied] test:TestSignedPreviewIsWhatTheEnvelopeRecordsAndTheConfirmationIsVerified test:TestAnAgentConfirmationStaysDeclared
- R5 [satisfied] test:TestCatalogMatchesGolden test:TestCatalogDocsConformance test:TestToolsList
- R6 [satisfied] test:TestToolsCall_ReviewPrepare_IsReadOnlyAndNeedsABundleAndAReviewer test:TestCatalogMatchesGolden

### Known gaps

- The signed positive path runs in package `pose`, where the sealed-bundle fixture lives; over JSON-RPC the tool is driven through its refusals and preview errors.

## 7. Final Report

### Delivered scope

`pose_review_attest` (governance-write): a preview that completes a draft attestation as recording would, binding the confirmation digest when it names `confirmed_by`, and returns the exact signing bytes with the blockers verify would report; with `apply`, recording only from a trusted issuer's envelope, the same verification and record path as `pose review attest --envelope` (now shared through `RecordReviewAttestationEnvelope`). Recording and preview share `completeReviewAttestation`, so a signed preview records under the id it was signed with.

### Residual risks

### Follow-ups
