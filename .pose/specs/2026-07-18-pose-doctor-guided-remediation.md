---
slug: pose-doctor-guided-remediation
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-public-install-contract
priority: 27
---

# Spec: Doctor-guided remediation

## 1. Intent

### Goal
turn diagnosable failures into safe actionable remediation.
### Business value
Reduces first-run abandonment and support load without hiding failures.
### Constraints
- Default to advice or dry-run and require explicit apply for mutation.
### Non-goals
- Edit arbitrary files or install system dependencies silently.

## 2. Requirements

### Functional
- R1: Every finding shall have stable code, severity, evidence and remediation.
- R2: Machine output shall distinguish detectable, fixable and externally blocked.
- R3: Safe fixes shall support preview, confirmation, idempotency and recheck.

### Non-functional
- Keep diagnosis fast, offline and platform-aware.

### Security
- Never print secrets, elevate privileges or bypass TLS.

### Compatibility
- Preserve JSON fields or version the doctor schema.

## 3. Technical Plan

### Affected areas
- Doctor, CLI UX, docs anchors, installer and fixtures.

### API/contract changes
- Define diagnostic codes and opt-in fix action schema.

### Data/storage changes
- Local remediation logs exclude sensitive values.

### Technical risks
- Overconfident fixes can damage custom setups.

### Primary references
- [Diátaxis](https://diataxis.fr/)
- [JSON Schema](https://json-schema.org/specification)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Diátaxis](https://diataxis.fr/).

### Implementation
- [ ] Inventory failure modes and define stable remediation codes. ([reference](https://diataxis.fr/))
- [ ] Implement dry-run fixes for confined reversible conditions. ([reference](https://json-schema.org/specification))
- [ ] Test clean, degraded, blocked and secret-redaction scenarios. ([reference](https://diataxis.fr/))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Doctor|Remediation|Redact'` and retain evidence. ([reference](https://diataxis.fr/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://json-schema.org/specification))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-doctor-guided-remediation-confined-fix-registry.md` (Accepted): a small explicit `doctorFixRegistry` (3 entries: `hooks.pre-commit`, `mcp.config`, `skills.symlinks`), each reusing an already-tested confined action; classification (fixable/detectable/blocked) derived centrally from that registry plus a fixed blocked-list, so existing `add()` call sites needed no change; JSON schema additive only (`doctor_schema_version`, `evidence`, `remediation_class`, `fix_code`, an optional `fix` object) — `check`/`level`/`message`/`hint` and top-level `findings`/`errors`/`warnings` unchanged. Rejected: a generic "re-run install" auto-fix (violates the confined/reversible constraint); free-form fix logic without a registry (no structural guarantee of confinement). `schema.version` deliberately excluded from the registry — `pose upgrade` remains the explicit, separately-governed path.

## 6. Validation

**Strategy:** validate the classification function directly (table test), the redaction helper, JSON-contract backward compatibility, and the full fix lifecycle (preview non-mutation, apply+recheck, idempotent reapply, `--only` scoping, invalid-code/`--yes`-without-`--fix` usage errors) against a real installed fixture.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'Doctor|Classify|RedactSecretShaped' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-doctor-guided-remediation --ready-check`.

### Requirement trace
- R1 [satisfied] every finding carries `check`/`level`/`evidence`/(`hint` when non-ok); check:test (TestDoctorFindingsHaveEvidenceAndRemediationClass)
- R2 [satisfied] `remediation_class` is one of n/a/fixable/detectable/blocked, with `fix_code` present exactly when fixable; check:test (TestClassifyFinding, TestDoctorFindingsHaveEvidenceAndRemediationClass)
- R3 [satisfied] preview is non-mutating, apply changes only the targeted files then rechecks and reports per-finding success, reapply is idempotent ("nothing fixable"), `--only` scopes to one check; check:test (TestDoctorFixPreviewIsNonMutating, TestDoctorFixApplyAppliesAndRechecksAndIsIdempotent, TestDoctorFixOnlyScopesToOneCheck, TestDoctorFixSkillsSymlinks)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'Doctor|Classify|RedactSecretShaped' -v -count=1` — SUCCESS (10 tests, including the pre-existing `TestDoctorHealthyAndBrokenInstance` unmodified — proving JSON backward compatibility).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-doctor-guided-remediation --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Security (never print secrets): `redactSecretShapedContent` applied to every message/evidence/hint/fix-error string; TestRedactSecretShapedContent proves a fake AWS-key-shaped fixture never survives verbatim. Doctor makes no TLS or privilege-elevation calls, so those two clauses are vacuously satisfied by the diagnostic pass's existing design (no network, no `sudo`/setuid anywhere in the CLI).
- Non-functional (fast, offline, platform-aware): every check and every fix action is filesystem/`exec.LookPath`-only — no network — consistent with the rest of the native CLI.

## 7. Final Report

- Delivered scope: `pose doctor` findings now carry a stable code, evidence and remediation class (fixable/detectable/blocked), versioned additively via `doctor_schema_version`; `pose doctor --fix` (preview) / `--fix --yes` (apply + recheck) / `--only <check>` covers three confined, reversible, idempotent repairs (pre-commit hook, `.mcp.json`, `.claude/skills` symlinks) by reusing existing, already-tested code paths (`cmdHooks`, `configureMCP`, a newly-shared `recreateClaudeSkillSymlinks`); defensive secret redaction applied to all doctor output.
- Residual risk: overconfident fixes could still damage a custom setup if a future registry entry is added carelessly — mitigated structurally by keeping the registry small and requiring every entry to reuse an already-tested, confined, idempotent action (documented as the pattern in the ADR) rather than writing new bespoke mutation logic per check; `schema.version` (the most common real finding) intentionally stays manual (`pose upgrade`), not silently foldable into `--fix`.
- Follow-ups: none — all three requirements are satisfied with executed evidence and no sandbox-unavailable gap (unlike the release-pipeline specs, `pose doctor` needs no network or external release infrastructure to test end to end).
