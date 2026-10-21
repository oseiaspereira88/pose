---
slug: pose-ci-security-refresh
status: in-progress
created_at: 2026-10-21
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
---

# Spec: Refresh the CI security baseline

## 1. Intent

### Goal
Restore the security checks observed on PR 195 without weakening detection.

### Constraints
Keep action SHA pins, full-history scanning, and all vulnerability gates enabled.

## 2. Requirements

### Functional
- R1: Every Go workflow shall use Go 1.26.10, correcting the standard-library GO-2026-6809 finding observed with Go 1.26.9.
- R2: Secret scanning shall exempt only the exact historical B28 file-digest finding and continue rejecting other findings.
- R3: The review bundle shall seal the exact root .gitleaksignore bytes as governance and continue rejecting unknown or nested security metadata paths.

- R4: Public documentation release-line claims shall be registered and signature instructions shall pin the exact release-workflow identity and issuer.

## 3. Technical Plan

### Affected areas
CI run 39562619357 reproduced both failures. The vulnerability scanner identifies crypto/x509 in Go 1.26.9, fixed in 1.26.10. Gitleaks identifies a SHA-256 value for json_duplicate_keys.go in the B28 review file manifest, commit 11e05be1d054cefee35651d0f13fcbb66ff066e0, line 576. It is a source-file digest, not an authentication value. Pin an ignore fingerprint to that commit, path, rule and line. The native closeout preview additionally reproduces an unclassified-subject blocker for .gitleaksignore. Extend the existing exact root security-metadata list, with byte-sealing and negative classification regression coverage. No related security knowledge artifact was found. Rollback reverts the patch and restores the two gate failures.

### Artifacts
- modified: .github/workflows/ci.yml
- modified: .github/workflows/governance-audit.yml
- modified: .github/workflows/package-channels.yml
- modified: .github/workflows/release.yml
- modified: .github/workflows/security.yml
- modified: .github/workflows/verify-release.yml
- modified: .gitleaksignore
- modified: .pose/public/claims.json
- modified: pose-mcp/internal/version/contract_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- created: .pose/specs/2026-10-21-pose-ci-security-refresh.md

### API/contract changes
None. Compiler patch, exact false-positive classification, root security metadata inclusion and documentation contract reconciliation only. No transport or CLI change.

### Technical risks
Go patch availability and upstream vulnerability database changes can independently affect CI. The fingerprint does not exempt a path or a whole commit.

## 6. Validation

### Strategy
Run workflow security contracts, vulnerability scans for both Go modules under Go 1.26.10, and the exact CI gitleaks version over complete history. Validate the exception with a modified line fingerprint and a synthetic secret in the same file path, both of which must remain detectable. Existing scanning is the regression oracle; no implementation-mirroring unit test is needed.

### Deterministic checks
- Command: go test ./internal/version -run WorkflowSecurity -count=1
- Expected: existing security contract passes.
- Command: GOTOOLCHAIN=go1.26.10 go run golang.org/x/vuln/cmd/govulncheck@v1.4.0 ./...
- Expected: no affected symbols in either module.
- Command: go run github.com/zricethezav/gitleaks/v8@v8.21.2 git --no-banner --redact --config ../.gitleaks.toml ..
- Expected: full history passes; negative controls remain detected.

### Execution log
- 2026-10-21: Reproduced the remote vulnerability finding and the single historical source-file digest match locally. Assessment reused unchanged pose-mcp content.
- Go 1.26.10: govulncheck v1.4.0 returned no vulnerabilities in both pose-mcp and mcp-enforce. WorkflowSecurity tests passed.
- Gitleaks v8.21.2: full history passed after the exact exception. Changing its line from 576 to 577 exposed the original finding; a synthetic token in the same report path was also detected. An initial control using an extra ignore file was inconclusive because the scanner also loads the repository ignore file; the repeated control mutated and restored that file directly.
- git diff --check passed. Remote CI at 1105883 passed both security scanners and workflow contracts; the documentation merge exposed the public-claims registration and exact-identity test inconsistencies, now corrected. A separate static review of 23a6290 approved its narrow scope (report:.pose/reports/2026-10-21-ci-security-review-r1.md); later changes require a new review.
- R3 regression first failed with the unclassified .gitleaksignore blocker, then passed after the exact-list fix, including nested/unknown rejection controls. /var/tmp isolated fixtures avoid unrelated external /tmp Git metadata.
- Targeted ArtifactIdentityContract, WorkflowSecurity and complete docs-release-line registration tests passed after reconciliation.
- Separate static review R2 approved the cumulative R1-R4 scope, with no actionable findings (report:.pose/reports/2026-10-21-ci-security-review-r2.md). This is a separate gpt-6.1-sol execution of the same vendor/model, with declared identity and correlated-model limits, no human confirmation and no cryptographic identity claim. It did not run tests or native gates.
- Execution details and environmental retry limits: report:.pose/reports/2026-10-21-ci-security-execution.md.
- Final native validation is recorded at the canonical policy results_path (.pose/results/delivery-validation.json); sealing must use a fresh passing result for the committed source. Integration assessment and tech-debt scan were rerun; pre-existing heuristic consumer gaps and incomplete legacy debt-spec coverage remain disclosed.

### Requirement trace
- R1 [satisfied] test:pose-mcp/internal/version/workflow_security_test.go <Go 1.26.10 vulnerability scans passed in both modules>
- R2 [satisfied] <Full-history scan and both negative controls passed; exception is fingerprint scoped>
- R3 [satisfied] test:pose-mcp/internal/pose/review_bundle_test.go
- R4 [satisfied] test:pose-mcp/internal/version/contract_test.go test:pose-mcp/internal/cli/release_line_claims_test.go

## 7. Final Report

### Delivered scope
Updated six Go workflows to 1.26.10, added one exact historical fingerprint, classified its root metadata for sealed review, and reconciled documentation claim and signature-identity contracts. The native closeout records final validation separately from the static review. No adoption, merge or production-efficacy claim is made.

### Residual risks
No credential finding is waived; new findings remain blocking.

### Follow-ups
- [open] Monitor the final remote CI result and merge decision; local closeout does not publish or adopt this engine revision. (owner:@pose-maintainers crit:medium review:2026-10-22)
