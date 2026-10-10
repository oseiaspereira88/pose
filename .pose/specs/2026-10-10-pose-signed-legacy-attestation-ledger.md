---
slug: pose-signed-legacy-attestation-ledger
status: in-progress
created_at: 2026-10-10
completed_at:
supersedes:
depends_on: pose-native-attestation-issuer, pose-reuse-is-sealed-signing-stays-live
priority: 0
components: pose-mcp
task_type: feature
changelog:
delivers: capability:signed-legacy-attestation-ledger
---

# Spec: A signed ledger lets a project with history require signed attestations

## 1. Intent

### Goal

Let a project that already has unsigned review attestations adopt `signed-attestations` without re-reviewing its history and without exempting anything by date: a trusted issuer signs, once and before adoption, the exact set of attestations that exist, and the verifier accepts an unsigned attestation only when it is in that set, unchanged.

### Business value

On 2026-10-10 Harne8 tried to adopt `signed-attestations` with a native issuer from 7.2.0, and `check --strict` failed 49 closed specs. Signing is deliberately the one review gate that stays live for old bundles (`pose-reuse-is-sealed-signing-stays-live`): a date cutoff would let an attestation forged later on an old bundle pass. Attestations are immutable and content-addressed, so they cannot be signed after the fact. The maintainer chose a signed legacy ledger over a cutoff, re-reviewing the 49, or not adopting.

### Constraints

- No exemption by date: an attestation recorded after the ledger, or changed after it, is not covered.
- The ledger is signed by an issuer pinned in `trusted_attestation_issuers`, so it carries the same trust a signed attestation does.
- Existing signed attestations and envelopes are unaffected.

## 2. Requirements

### Functional

- R1: In a project with unsigned attestations, `pose adopt signed-attestations` shall show in its preview how many will be sealed and by which issuer, and with `--apply` write `.pose/review-ledgers/legacy-<issuer>-<timestamp>.json` — project, issuer, public key, sealing time and, per attestation, its id and full content digest, signed with the issuer's key — before turning the requirement on. The issuer is the only local key pinned for attestations, or `--issuer <name>`.
- R2: No ledger shall be sealed once review policy requires signed attestations, and none when there is no unsigned attestation; a project without history adopts with no ledger.
- R3: Under `require_signed_attestations`, an attestation without an envelope shall be accepted only when a ledger signed by a pinned attestation issuer, for this project, lists its id with the same full content digest; otherwise it shall be refused as today.
- R4: An attestation recorded after the ledger, an attestation whose content changed, a ledger signed by an unpinned key, a tampered ledger and a ledger for another project shall each be refused.
- R5: Adoption, directly or through a configuration-review answer, shall be refused when unsigned attestations exist and no local issuer pinned for attestations can seal them (naming `pose issuer init` and `pin`), or when several could and `--issuer` was not given.
- R6: `pose issuer init` shall suggest pinning a new key for attestations only, and say that `--human-authority` belongs to a key a person alone controls.

### Security

- The ledger names each attestation by its full content digest, not the 16-hex attestation id alone.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-signed-legacy-attestation-ledger.md
- created: .pose/starts/pose-signed-legacy-attestation-ledger.json
- created: .pose/changelogs/unreleased/pose-signed-legacy-attestation-ledger.md
- created: pose-mcp/internal/pose/legacy_ledger.go
- created: pose-mcp/internal/pose/legacy_ledger_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/configuration_review.go
- modified: pose-mcp/internal/cli/issuer.go
- modified: pose-mcp/internal/cli/issuer_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: docs-site/docs/cli.md

### Delivery targets

- capability:signed-legacy-attestation-ledger module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

### Implementation
- [x] Ledger format, sealing and signature verification
- [x] Verifier accepts ledger-covered unsigned attestations
- [x] Adopt prerequisite and `issuer init` guidance
- [x] Docs and changelog

### Validation
- [x] Journey: an instance with history seals, adopts and passes `check --strict`; a later unsigned attestation is refused

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: adopting signed attestations re-judged 49 closed specs in Harne8.
- Options considered: (a) signed legacy ledger; (b) date cutoff; (c) re-review every closed spec with signed attestations; (d) not adopting in projects with history.
- Decision: (a), chosen by the maintainer.
- Rationale: only (a) keeps the property `pose-reuse-is-sealed-signing-stays-live` protects — nothing recorded later is exempt — while making adoption possible.

### Decision D2
- Date: 2026-10-10
- Context: the first implementation added `pose issuer seal-legacy`, a second contract an agent had to learn and run before `pose adopt`.
- Decision: the maintainer asked for one contract; sealing became an effect of `pose adopt signed-attestations`, planned in its preview and applied before the policy is written, in both the direct and the configuration-review paths.
- Rationale: the ledger exists only to make that adoption possible, so it belongs to it; and sealing only inside the adoption also makes "never after adoption" structural.

## 6. Validation

### Strategy

Unit tests for each refusal in R4 and for sealing rules; a CLI journey for R1-R5; every test fails on the engine before the change.

### Execution log

2026-10-10, measured on a disposable Harne8 worktree at 092c49e0 with the candidate CLI and explicit project roots: adopting `signed-attestations` failed 49 review closeouts; `pose adopt signed-attestations` now previews sealing 193 unsigned attestations with `harne8-agents`, and after `--apply` `check --strict` reports 0 review-closeout errors. `TestLegacyLedgerKeepsSealedAttestationsValid` fails without the verifier change.

### Requirement trace

- R1 [satisfied] capability:signed-legacy-attestation-ledger evidence:integration test:TestAdoptingSignedAttestationsSealsHistory
- R2 [satisfied] capability:signed-legacy-attestation-ledger evidence:unit test:TestLegacyLedgerIsSealedOnlyBeforeAdoption
- R3 [satisfied] capability:signed-legacy-attestation-ledger evidence:unit test:TestLegacyLedgerKeepsSealedAttestationsValid
- R4 [satisfied] capability:signed-legacy-attestation-ledger evidence:unit test:TestLegacyLedgerRefusesEverythingOutsideIt
- R5 [satisfied] capability:signed-legacy-attestation-ledger evidence:unit test:TestAdoptingSignedAttestationsSealsTheHistory test:TestAdoptingSignedAttestationsSealsHistory
- R6 [satisfied] capability:signed-legacy-attestation-ledger evidence:integration test:TestNativeIssuerJourneyWithPOSEAlone

## 7. Final Report

### Delivered scope

`pose adopt signed-attestations` seals a project's unsigned history into a signed legacy ledger and the verifier accepts only what the ledger names, unchanged; `pose issuer init` no longer suggests human authority for a shared key.

### Residual risks

- Whoever holds the sealing issuer key at adoption time could include an attestation forged just before adopting; that is the same trust a signed attestation already places in the key.

### Follow-ups
