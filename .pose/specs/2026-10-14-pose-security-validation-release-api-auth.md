---
slug: pose-security-validation-release-api-auth
status: done
created_at: 2026-10-14
completed_at: 2026-11-03
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

Compare YAML before/after: only the selected step gains `GITHUB_TOKEN`; preserve permissions, commands, pins and the history promise. Reuse the existing WorkflowSecurity/history tests where available and the applicable local POSE gates. The maintainer explicitly waived remote GitHub CI confirmation for this closeout on 2026-11-03; keep that limitation explicit and do not treat it as evidence of runner runtime. Do not introduce a duplicate test for a reversible environment binding.

### Requirement trace

- R1 [satisfied] The current Security step binds `github.token`; review of the validation, journey-test and shell inheritance path is recorded in report:.pose/reports/2026-10-14-security-validation-release-api-auth.md; the original binding is commit:2255621cc82fa348ee2419b892296b96ea7b276e.
- R2 [satisfied] Removing only the selected step's token binding from the original workflow change makes the complete parsed YAML documents equal; the command, permissions, pins, other steps and history promise are preserved in report:.pose/reports/2026-10-14-security-validation-release-api-auth.md; commit:2255621cc82fa348ee2419b892296b96ea7b276e.
- R3 [waived: only the corrected GitHub runner confirmation was waived by the maintainer's express instruction on 2026-11-03] The incident, same-head direct CI control, local configuration comparison and validation evidence remain recorded in report:.pose/reports/2026-10-14-security-validation-release-api-auth.md. This waiver does not prove successful runtime on a GitHub runner.

## 7. Final Report

### Delivered scope

The configuration correction is present and its local structural evidence is recorded in [the report](../reports/2026-10-14-security-validation-release-api-auth.md). The maintainer waived corrected GitHub runner confirmation on 2026-11-03; runtime verification on that runner is not claimed. Keep this spec `in-progress` until the principal completes the remaining review and native closeout gates.

### Residual risks

Authenticated requests may still fail on service outage or exhausted token limits. The fix addresses the absent credential, without converting network failures to success. Existing Security job permissions include contents:read and security-events:write; no permission is added. No corrected GitHub runner result was checked in this closeout, so the local binding and inheritance evidence does not establish token issuance or successful network runtime there.

### Follow-ups

- [wont-do: the maintainer's express 2026-11-03 instruction waives the remote rerun; the PR update is obsolete because PR #171 was merged in commit 655940a47d5622080dd213f1247b638704be6c43; no rerun or corrected remote result is claimed] Update documentary PR #171 against main and rerun its failed validation on the corrected workflow. Keep the historical incident and local evidence in the report.
