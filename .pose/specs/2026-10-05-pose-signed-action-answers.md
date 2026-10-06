---
slug: pose-signed-action-answers
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: standard
delivers: capability:signed-action-answers
---

# Spec: A principal proves an answer with a key it registered, with or without Harne8

## 1. Intent

### Goal

Let a project verify who answered an action request without any external service: each principal registers SSH public keys in `.pose/policy/actions.json`, an answer carries an SSH signature (SSHSIG) over a canonical statement bound to the project, the request version, the principal and the answer, and POSE verifies it natively. Harne8's issuer claims stay accepted as a second emitter of the same proof.

### Business value

Decision 7 of the 7.0.0 consolidation: POSE must be governable standalone and with Harne8. Until now the only verified answer was an Ed25519 claim from a trusted issuer — something only Harne8 emits — so a standalone project could record a `human:` principal only as a declaration. A git identity names someone but proves nothing (anyone sets `user.email`); a signature by a key the project registered does, and a FIDO security key (`ed25519-sk`) adds proof that a person touched the device. Part of roadmap pose-v7-onboarding-and-consolidation (milestone identity).

### Constraints

No new dependency: SSHSIG is parsed and verified with the standard library. Signing uses the user's own `ssh-keygen -Y sign`; POSE never reads a private key. Declared answers keep working where the policy asks for nothing more. An answer recorded as verified stays verified when its key is later removed: removal stops future answers, it does not rewrite history.

### Non-goals

Certificates (`ssh-keygen -s`), RSA/ECDSA keys, GPG, and key distribution beyond the project's own policy file. The MCP resolution tool is spec pose-mcp-action-resolve-signed-only.

## 2. Requirements

### Functional

- R1: `.pose/policy/actions.json` shall accept `keys`, mapping a principal to the SSH public keys it signs with (`ssh-ed25519` or `sk-ssh-ed25519@openssh.com`, in authorized_keys form, with the date added), and `require_presence`; a malformed or unsupported key shall make the policy unreadable rather than be ignored.
- R2: `pose identity add [<principal>] --key <file|line> [--role <role>] [--apply]` shall register a key for a principal (suggesting `human:<name>` from the git identity when none is given, and saying that the suggestion is a name, not a proof), optionally grant a role, preview unless `--apply`, refuse a key already registered to another principal, and say whether the key proves presence; `pose identity list [--json]` shall show principals, roles and keys by fingerprint; `pose identity remove <principal> [--fingerprint <SHA256:…>] [--apply]` shall remove keys.
- R3: `pose action statement <act-id> --actor <principal> --answer <answer>` shall print the canonical statement for the current request version; `pose action resolve … --sign <private-key>` shall sign it with `ssh-keygen -Y sign -n pose-action-answer`, and `--signature <file>` shall attach a signature made elsewhere.
- R4: An answer carrying a signature shall be recorded as verified only when the signature is a valid SSHSIG in namespace `pose-action-answer`, over the statement for this project, request id, request digest, actor and answer, by a key registered to the actor; any mismatch shall refuse the answer.
- R5: Under `identity_assurance: verified`, an answer shall carry either such a signature or a trusted issuer claim; a security-key signature shall record whether presence (and verification) was asserted, and under `require_presence` a `human:` answer without asserted presence shall be refused.
- R6: `pose doctor` shall warn when a `human:` principal holding a role under verified assurance has no registered key, or has only keys that cannot prove presence (an error under `require_presence`).
- R7: The recorded event shall keep the statement, signature and key, so the answer can be re-verified from the journal alone.

### Non-functional

- Verification is offline and deterministic.

### Security

- The statement binds project, request id, request digest, actor, answer and idempotency key, so a signature cannot be replayed onto another request, version, project or answer.
- POSE never reads, stores or asks for a private key; `--sign` hands the key path to `ssh-keygen`.
- A key registered to one principal cannot be registered to another.

### Compatibility

- Policies without `keys` read exactly as before; journals gain an optional `ssh_signature` field on answers.

## 3. Technical Plan

### Affected areas

Action policy, resolution verification, a native SSHSIG verifier, the `pose identity` command, `pose action statement`/`resolve --sign|--signature`, doctor.

### Artifacts

- created: .pose/specs/2026-10-05-pose-signed-action-answers.md
- created: .pose/starts/pose-signed-action-answers.json
- created: pose-mcp/internal/pose/sshsig.go
- created: pose-mcp/internal/pose/sshsig_test.go
- created: pose-mcp/internal/pose/principal_keys.go
- created: pose-mcp/internal/pose/principal_keys_test.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/action_request.go
- created: pose-mcp/internal/cli/identity.go
- created: pose-mcp/internal/cli/identity_test.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/action_resolve.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- renamed: .pose/changelogs/unreleased/pose-signed-action-answers.md -> .pose/changelogs/v7.0.0/pose-signed-action-answers.md

### Delivery targets

- capability:signed-action-answers module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A hand-written verifier could accept what OpenSSH rejects; tests cover tampered statements, wrong namespace, wrong key, unregistered key and truncated blobs, and an interop test verifies a signature made by the real `ssh-keygen` when it is installed.

## 4. Tasks

### Implementation
- [x] Native SSHSIG verifier with in-process signers for plain and security keys, and an OpenSSH interop test
- [x] Principal keys in the action policy, the signed statement and verification in the resolution path
- [x] `pose identity`, `pose action statement`, `resolve --sign|--signature`, re-verification in `action show`
- [x] `actions.keys` in doctor

### Validation
- [x] Run the deterministic checks below and retain results

## 5. Decisions

- D1: The git identity only suggests a name; the proof is a signature by a key the project registered. Rationale: `user.email` is set by anyone, a key is held.
- D2: A registered key's signature and a trusted issuer's claim are two emitters of one proof, accepted alike under verified assurance. Rationale: POSE works standalone and with Harne8 without a second assurance mode.
- D3: Humans are warned, not refused, when their keys cannot prove presence; `require_presence` makes it a refusal. Rationale: decision 7 open choice, taken as recommended.

## 6. Validation

### Strategy

Signatures produced in-process for plain and security keys (with and without presence), and by `ssh-keygen` when available, verified through the resolution path; the identity command driven end to end.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run 'SSHSig|PrincipalKey|SignedAnswer|Identity'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestPrincipalKeyPolicyRefusesBadKeysAndSharedKeys check:signed-action-answers-integration
- R2 [satisfied] test:TestIdentityAddSuggestsTheGitNameAndRegistersTheKey check:signed-action-answers-integration
- R3 [satisfied] test:TestSignedAnswerThroughTheCLIIsVerifiedAndReverified check:signed-action-answers-integration
- R4 [satisfied] test:TestSignedAnswerByARegisteredKeyIsVerifiedWithoutAnIssuer test:TestSignedAnswerRefusesEveryMismatch test:TestSSHSigVerifiesPlainAndSecurityKeys test:TestSSHSigRefusesWhatOpenSSHWouldRefuse test:TestSSHSigInteroperatesWithSSHKeygen check:signed-action-answers-integration
- R5 [satisfied] test:TestSignedAnswerRequirePresenceRefusesAnUntouchedKey test:TestSignedAnswerThroughTheCLIIsVerifiedAndReverified check:signed-action-answers-integration
- R6 [satisfied] test:TestDoctorReportsHumanRolesThatCannotProveAnAnswer check:signed-action-answers-integration
- R7 [satisfied] test:TestSignedAnswerByARegisteredKeyIsVerifiedWithoutAnIssuer test:TestSignedAnswerThroughTheCLIIsVerifiedAndReverified check:signed-action-answers-integration

### Known gaps

- The CLI tests and the interop test need `ssh-keygen`; they skip without it. The in-process tests cover the verifier, presence and every mismatch without it.

## 7. Final Report

### Delivered scope

Native SSHSIG verification (Ed25519 and FIDO Ed25519, presence and verification flags), principal keys in the action policy, `pose identity add|list|remove`, `pose action statement`, `resolve --sign|--signature`, re-verification in `action show`, `actions.keys` in doctor. Interop with OpenSSH 9.6 confirmed in both directions.

### Residual risks

None beyond the technical risk.

### Follow-ups
