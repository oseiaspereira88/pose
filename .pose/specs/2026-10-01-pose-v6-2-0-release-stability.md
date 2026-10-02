---
slug: pose-v6-2-0-release-stability
status: in-progress
created_at: 2026-10-01
completed_at:
delivers: surface:release-stability, capability:release-stability
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
---

# Spec: Stabilize the next POSE release

## 1. Intent

### Goal

Resolve the two defects reported against 6.1.0 and make the clean-host package
channel smoke inspect an installed POSE instance.

### Business value

Contributor instructions must not disagree about authorization, a root
configuration example must be reviewable, and the Windows package check must
measure the installed binary rather than the source checkout's state.

### Constraints

Preserve explicit user approval before staging and submitting contributions.
Only classify the exact root `.env.example`; keep `.env` unclassified.
Do not suppress `doctor` errors from the fresh instance.

### Non-goals

The community-launch roadmap and unrelated follow-ups remain separate scopes.

## 2. Requirements

### Functional

- R1: When contributor mode is enabled, generated AGENTS.md and POSE.md in both
  locales shall require user confirmation before local staging and separately
  before upstream submission.
- R2: A spec-attributed root `.env.example` shall be sealed as governance with
  its content digest, while `.env` and nested examples remain unclassified
  without an explicit component mapping.
- R3: The Homebrew and WinGet package-channel jobs shall install POSE into a
  fresh temporary repository and run `pose doctor --json` there after installing
  the published binary.

### Non-functional

- The three regressions shall have deterministic tests.
- The package-channel gate shall keep failing when the fresh instance fails
  `pose doctor`.

### Security

- The bundle shall retain the bytes of `.env.example` for security review.
- Staged contribution text shall continue to exclude secrets and private data.

### Compatibility

- No schema or public CLI command changes.

## 3. Technical Plan

### Affected areas

- Contributor-mode templates, help and public documentation.
- Review-bundle subject classification.
- Package-channel GitHub Actions workflow.

### Artifacts

- created: .pose/specs/2026-10-01-pose-v6-2-0-release-stability.md

- modified: pose-mcp/internal/cli/contribute.go
- modified: pose-mcp/internal/cli/contribute_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/concepts.md
- modified: .github/workflows/package-channels.yml
- created: pose-mcp/internal/version/package_channel_smoke_test.go
- created: .pose/changelogs/unreleased/pose-v6-2-0-release-stability.md

### API/contract changes

The generated contributor instructions and review-bundle path classification
are corrected to their existing governed contracts.

### Data/storage changes

None.

### Technical risks

The Windows runner cannot be reproduced on this Linux host. A structural
workflow regression test covers the smoke boundary; the next tagged workflow
run must provide platform evidence.

### Delivery targets

- surface:release-stability module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- capability:release-stability module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

These targets identify contributor guidance and immutable review classification.
Package smoke has structural coverage; native execution remains deferred.

## 4. Tasks

### Implementation

- [x] Align contributor instructions, CLI output and public docs (R1).
- [x] Classify and test the exact root `.env.example` (R2).
- [x] Move both package-channel doctor checks to fresh installations (R3).

### Validation

- [x] Run targeted regression tests and `go test ./...`.
- [x] Run `go vet ./...` and `pose validate --strict --module pose-mcp`.
- [x] Transfer native macOS/Windows verification to the dedicated deferred spec
  at the user's request; execution is skipped, not passed.

## 5. Decisions

### Decision 1

- Date: 2026-10-01
- Context: the earlier contributor-mode protocol said drafts were automatic,
  while the later covered-followup spec requires consent for both staging and
  submission.
- Decision: follow the later explicit-adjudication contract in generated text.
- Consequences: local staging requires a user decision; upstream submission
  still requires its own decision.

### Decision 2

- Date: 2026-10-01
- Context: WinGet installed the 6.1.0 archive successfully, then `doctor`
  failed on `state.integrity` in the checked-out POSE source tree.
- Decision: run package smoke on a newly installed temporary project, as the
  independent installer verification already does.
- Consequences: the smoke checks the published binary in an instance it creates.

## 6. Validation

### Strategy

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Generated EN/PT manuals agree | `go test ./internal/cli -run TestContributorDocsRequireConsentForStagingAndSubmission -count=1` | Both require separate consent |
| Root example seals with bytes | `go test ./internal/pose -run TestReviewBundleSealsRootEnvExampleAsGovernance -count=1` | Sealed governance entry and digest |
| Secret file stays unclassified | `go test ./internal/pose -run TestReviewBundleDoesNotGeneralizeEnvExampleClassification -count=1` | Both negative paths rejected |
| Package smoke uses a fresh instance | `go test ./internal/version -run TestPackageChannelSmokeUsesFreshInstance -count=1` | Both OS legs install, then doctor inside it |
| Module regression | `go test ./...` and `go vet ./...` | All packages pass |
| Deferred native round | See `spec:pose-package-channels-deferred-native-verification` | Skipped now; future manual jobs must both pass |

### Execution log

- 2026-10-01: R1 and R2 targeted regressions, `go test ./...`, `go vet ./...`
  and `pose validate --tolerant --module pose-mcp --report` passed (43/43
  validation steps). R3 implementation followed: structural regression passed;
  a newly built Linux binary installed a temporary Git instance and
  `pose doctor --json` returned zero errors. The final `go test ./...`,
  `go vet ./...` and `pose validate --strict --module pose-mcp` passed
  (43/43). The macOS/Windows tagged runner evidence remains pending.

### Requirement trace

- R1 [satisfied] surface:release-stability evidence:integration check:delivery-integration test:TestContributorDocsRequireConsentForStagingAndSubmission
- R2 [satisfied] capability:release-stability evidence:integration check:delivery-integration test:TestReviewBundleSealsRootEnvExampleAsGovernance test:TestReviewBundleDoesNotGeneralizeEnvExampleClassification
- R3 [satisfied] test:TestPackageChannelSmokeUsesFreshInstance test:TestPackageChannelVerificationIsManualOnly

### Known gaps

The full macOS/Windows package smoke is skipped/deferred by explicit user
direction on 2026-10-01. It belongs to
`spec:pose-package-channels-deferred-native-verification`, is scheduled only
after implementation, and does not block other specs or release publication.

## 7. Final Report

### Delivered scope

Contributor consent text, root example classification and installed-instance
package smoke are implemented and covered by deterministic regressions.

### Residual risks

Native behavior remains unverified until the dedicated manual round runs.
Structural Linux checks do not establish macOS/Windows success.

### Follow-ups

- [spawned: pose-package-channels-deferred-native-verification] Run the native
  package-channel round only after implementation finishes and retain both OS
  job results. Explicitly deferred by the user; nonblocking for all other work.
