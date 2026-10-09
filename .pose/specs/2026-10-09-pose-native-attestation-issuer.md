---
slug: pose-native-attestation-issuer
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: feature
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

### Technical risks

- Key handling mistakes are security defects; the permission checks and no-print assertions are part of the gate.

## 4. Tasks

### Implementation
- [ ] Issuer key store and `pose issuer init|pin|rotate`
- [ ] `review attest --sign` and native reviewer authority claims
- [ ] Prerequisite messages and docs

### Validation
- [ ] Fresh-instance journey with only a native issuer, then with native and external pins together

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

Unit tests for key creation, permissions, pin computation and signing; an integration test that adopts `signed-attestations` in a fresh instance with only a native issuer, closes a spec with a natively signed attestation, then adds a second (external-style) pin and verifies envelopes from both.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 from the POSE 7.1.0 adoption in Harne8, pose-dist, audio-relay and storageclose.

### Residual risks

None yet.

### Follow-ups
