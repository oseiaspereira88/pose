---
slug: pose-security-validation-release-api-auth
status: in-progress
created_at: 2026-10-14
completed_at:
depends_on:
priority: 1
components: ci
surface: minimal
task_type: bugfix
changelog: none
delivers: governance:security-validation-release-api-auth
---

# Spec: Authenticate release listing from Security validation

## 1. Intent

### Goal

Pass the existing workflow token to the Security validation step that indirectly runs the install-and-upgrade journey.

### Business value

Security run 38504224761 on PR #171 failed while listing published releases with HTTP 403. The direct journey in CI run 38504224767 passed. The journey already authenticates when GITHUB_TOKEN or GH_TOKEN is available; the Security validation caller supplied neither.

### Constraints

Preserve the journey, anonymous local fallback, required validation severity, action pins and existing permissions. Do not implement any B25/B26/B37 behavior or change the documentary consolidation matrix. Use the prior decision in spec:pose-ci-avoids-anonymous-rate-limits; no new architectural decision or credential is required. Search of knowledge found no additional incident record needed for this narrow configuration correction.

## 2. Requirements

- R1: The Deterministic validation to SARIF step shall supply GITHUB_TOKEN from github.token so its registered journey check lists releases with authentication.
- R2: The step shall retain POSE_RELEASE_HISTORY_AVAILABLE=true, the existing validation command and required journey; workflow permissions and all other steps shall remain unchanged.
- R3: Retain evidence of the failed job, compare step configuration to the authenticated direct CI caller, validate the diff and workflow contracts, and confirm the corrected journey on GitHub before claiming runtime verification.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-14-pose-security-validation-release-api-auth.md
- created: .pose/starts/pose-security-validation-release-api-auth.json
- created: .pose/reports/2026-10-14-security-validation-release-api-auth.md
- modified: .github/workflows/security.yml

### Delivery targets

- governance:security-validation-release-api-auth module:. profile:release-governance entrypoint:.github/workflows/security.yml

### Approach

Declare the token only in the validation step env alongside the existing history promise. The validation runner inherits os.Environ, and the Go journey test inherits it into Bash. Reuse the existing integration test rather than weakening or skipping it. Reverting the single environment addition rolls back the correction. HTTP 403 is observed; shared anonymous rate-limit exhaustion is the likely cause, but the failed log does not preserve headers/body proving the precise GitHub reason.

## 5. Decisions

No new structural decision. Apply the existing authenticated API listing contract to its indirect Security caller.

## 6. Validation

### Strategy

Compare YAML before/after: only the selected step gains GITHUB_TOKEN and permissions, commands, pins and history declaration are preserved. Run the existing WorkflowSecurity/history tests where available, lint-spec, artifact-check and check --strict. Confirm TestJourneysInstallAndUpgradeFromThePublishedRelease via Security validation-findings on the corrected PR. Do not introduce a duplicate test for a reversible environment binding.

### Requirement trace

Record executed evidence and remaining CI verification in the linked report; keep the spec in-progress until review and native closeout.

## 7. Final Report

### Delivered scope

Correction prepared; local structural evidence and GitHub verification are recorded in [the report](../reports/2026-10-14-security-validation-release-api-auth.md). Runtime verification, review and closeout remain pending.

### Residual risks

Authenticated requests may still fail on service outage or exhausted token limits. The fix addresses the absent credential, without converting network failures to success. Existing Security job permissions include contents:read and security-events:write; no permission is added.

### Follow-ups

After merging this isolated correction, update the documentary PR #171 against main and rerun its failed validation on the corrected workflow.
