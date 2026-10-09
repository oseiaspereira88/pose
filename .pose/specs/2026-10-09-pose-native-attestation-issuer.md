---
slug: pose-native-attestation-issuer
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: feature
changelog:
delivers: capability:native-attestation-issuer
---

# Spec: Native attestation issuer, coexisting with external issuers

## 1. Intent

### Goal

Let any project that installs only POSE sign review attestations and reviewer authority claims, so `signed-attestations` and `verified-identity` can be adopted without Harne8, while an external issuer such as the Harne8 Conductor stays trusted alongside the native one.

### Business value

Found while adopting 7.1.0 on 2026-10-09: the engine verifies Ed25519 attestation envelopes from issuers pinned as `<issuer>#sha256:<key-digest>` in `trusted_attestation_issuers` and `human_authority_issuers`, but POSE has no way to create an issuer key, compute its pin or sign an attestation. The only emitter is Harne8's Conductor, which signs after an authenticated person confirms through its endpoint. Harne8 had to defer both capabilities, and a repository without Harne8 can never adopt them. Action answers already have a native path (`pose identity add` and `ssh-keygen -Y sign`); review attestations do not.

The maintainer's direction: every project can have native issuers and the Harne8 Conductor issuer active at the same time, not one excluding the other. A developer working with the repository and POSE alone signs natively; a developer also using Harne8 signs through the Conductor; both signatures count.

### Constraints

- A private key never enters the repository, a log or command output; it lives outside the project (default under the user's config directory) with owner-only permissions.
- Trust is pinned explicitly in review policy; a self-declared issuer or key is never trusted alone (the current envelope rule).
- The envelope format the engine verifies today is unchanged, so Conductor envelopes keep verifying.
- MCP does not sign with a key on the agent's behalf: it returns the canonical bytes to sign and accepts the signed envelope, as `pose_review_attest` does now.

### Non-goals

Hosted key management, hardware-key enforcement for issuers, and changing how the Conductor signs.

## 2. Requirements

### Functional

- R1: `pose issuer init <name>` shall create an Ed25519 issuer key outside the project with owner-only permissions, print only its public pin, and refuse to overwrite an existing key.
- R2: `pose issuer pin <name|public-key> [--attestations] [--human-authority] [--apply]` shall add `<name>#sha256:<digest>` to the chosen policy lists as a preview first, keeping every pin already there.
- R3: `pose review attest <scope> ... --sign <issuer>` shall complete the attestation as recording would, sign its canonical bytes with the issuer key and record the envelope, which `pose review verify` accepts under `require_signed_attestations`.
- R4: The native issuer shall sign a reviewer authority claim the same way, so `verified-identity` reads a human reviewer from a native claim.
- R5: When native and external pins coexist in the same policy lists, an envelope from any pinned issuer shall count and an envelope from an unpinned issuer shall be refused; adopting or rotating one issuer shall never remove another's pin.
- R6: The `signed-attestations` and `verified-identity` prerequisite messages in `pose adopt` and `pose setup` shall name `pose issuer init` and `pose issuer pin` as the native way to satisfy them.
- R7: `pose issuer rotate <name>` shall create a new key and pin it next to the old one, leaving the old pin until the operator removes it, so attestations signed before the rotation keep verifying.

### Non-functional

- Signing and verification stay offline: no network call.

### Security

- Private key files are 0600 and their directory 0700; commands refuse a key file with wider permissions.
- No command prints, logs or embeds private key material; tests assert it.

### Compatibility

- Existing envelopes, pins and the Conductor emitter keep working unchanged.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/cli` (new `issuer` command, `review attest --sign`), `pose-mcp/internal/pose` (issuer key store, signing beside `VerifyReviewAttestationEnvelope`, authority-claim signing), capability prerequisite messages, docs.

### Artifacts

- created: .pose/specs/2026-10-09-pose-native-attestation-issuer.md
- created: .pose/starts/pose-native-attestation-issuer.json
- created: .pose/changelogs/unreleased/pose-native-attestation-issuer.md
- created: pose-mcp/internal/pose/issuer_key.go
- created: pose-mcp/internal/pose/issuer_key_test.go
- created: pose-mcp/internal/pose/issuer_key_links_unix.go
- created: pose-mcp/internal/pose/issuer_key_links_windows.go
- created: pose-mcp/internal/cli/issuer.go
- created: pose-mcp/internal/cli/issuer_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: composition-contract.json
- modified: docs-site/docs/cli.md

### Delivery targets

- capability:native-attestation-issuer module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Key handling mistakes are security defects; the permission checks and no-print assertions are part of the gate.

## 4. Tasks

### Implementation
- [x] Issuer key store and `pose issuer init|pin|rotate`
- [x] `review attest --sign` and native reviewer authority claims
- [x] Prerequisite messages and docs

### Validation
- [x] Fresh-instance journey with only a native issuer, then with native and external pins together

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: the verifier already accepted any issuer pinned in `trusted_attestation_issuers` and `human_authority_issuers`; the only emitter was the Harne8 Conductor.
- Options considered: (a) a native emitter of the same envelope, pinned like any other issuer; (b) a separate native proof type with its own verifier.
- Decision: (a).
- Rationale: one proof and one verifier keep native and external issuers interchangeable, which is what lets them coexist (the maintainer's direction), and leave Conductor envelopes untouched.
- Consequences: a native signature is held to exactly the checks an external one is; `--sign` verifies the envelope before the preflight and records it through `RecordReviewAttestationEnvelope`.

### Decision D2
- Date: 2026-10-09
- Context: an authority claim names one audience and one project, the ones review policy declares.
- Decision: `pose issuer pin` sets `authority_audience` and `authority_project` only when empty, defaulting both to the project id from `.pose/project.json` for `--human-authority`, and refuses to replace a different value.
- Rationale: every issuer's claims answer to the same audience; replacing it to suit a new issuer would invalidate the others' claims.

## 6. Validation

### Strategy

Unit tests for key creation, permissions, pin computation and signing; an integration test that adopts `signed-attestations` in a fresh instance with only a native issuer, closes a spec with a natively signed attestation, then adds a second (external-style) pin and verifies envelopes from both.

### Execution log

2026-10-09: the CLI journey test first failed on the implementation itself: `--sign` ran the preflight on the attestation before the envelope was attached, and the preflight refused it as unsigned under `signed-attestations`; the envelope is now verified first. `go test ./...` in pose-mcp passes; the composition contract gained `POSE_ISSUER_HOME`.

2026-10-09, independent review (agent:gpt-6.1-sol, Codex): changes required, high severity. `POSE_ISSUER_HOME` accepted a directory inside the project and `issuer init` wrote the private key there (for example `<project>/.pose/issuers/`), one `git add -A` from publication, against R1. Every key operation now takes the project root and refuses a key directory inside it, symlinks resolved, before anything is written; `TestNativeIssuerKeyIsRefusedInsideTheProject` covers the direct path, a symlink into the project, and signing from such a directory.

2026-10-09, second independent review (agent:gpt-6.1-sol): two more defects. High: the key directory was checked but the `<issuer>.key` file could be a symlink to a key inside the project, and loading followed it. Medium: two rotations in the same second archived to the same file and `rename` silently replaced the earlier retired key. The key file is now read with `Lstat` and refused when it is a symlink or has another hard link (Unix; Windows has no link count), and a retired key is archived with a hard link that fails instead of replacing, numbered when the name is taken. `TestNativeIssuerKeyFileMustNotBeALink` and `TestRotationsInTheSameSecondKeepEveryRetiredKey` failed on 7932776b and pass now.

2026-10-09, third independent review (agent:gpt-6.1-sol): one low-severity defect. A valid issuer name containing `.retired-` (for example `ops.retired-backup`) was read as a retired key by `issuer list` and disappeared from it. Listing now reads the issuer from the key file and treats only `<issuer>.key` as current; `TestIssuerNameContainingRetiredIsListed` failed on 24b54a69 and passes on 5a590598.

### Requirement trace

- R1 [satisfied] capability:native-attestation-issuer evidence:unit test:TestNativeIssuerKeyIsPrivateAndNeverOverwritten test:TestNativeIssuerKeyIsRefusedInsideTheProject test:TestNativeIssuerKeyFileMustNotBeALink
- R2 [satisfied] capability:native-attestation-issuer evidence:unit test:TestNativeAndExternalIssuersCoexist
- R3 [satisfied] capability:native-attestation-issuer evidence:integration test:TestNativeIssuerJourneyWithPOSEAlone
- R4 [satisfied] capability:native-attestation-issuer evidence:unit test:TestNativeAuthorityClaimSatisfiesVerifiedIdentity
- R5 [satisfied] capability:native-attestation-issuer evidence:unit test:TestNativeAndExternalIssuersCoexist
- R6 [satisfied] capability:native-attestation-issuer evidence:integration test:TestNativeIssuerJourneyWithPOSEAlone
- R7 [satisfied] capability:native-attestation-issuer evidence:unit test:TestRotatedIssuerKeepsEarlierSignaturesValid test:TestRotationsInTheSameSecondKeepEveryRetiredKey test:TestIssuerNameContainingRetiredIsListed

## 7. Final Report

### Delivered scope

`pose issuer init|pin|list|rotate` and `pose review attest --sign <issuer> [--authority …]`: a project with POSE alone signs review attestations and reviewer authority claims, and native and external issuers coexist in the same policy lists.

### Residual risks

- The issuer key is protected by file permissions, not a passphrase or hardware; a compromised account can sign as the issuer until its pin is removed.
- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
