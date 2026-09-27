---
slug: pose-governance-gate-activation
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on:
priority: 3
components: governance, assessment
delivers: governance:gate-activation
---

# Spec: Turn the tolerated governance warnings into gates that bite

## 1. Intent

### Goal
Complete the adoption of governance mechanisms POSE already ships but still runs
in warning mode, and fix the one evidence-identity defect that makes a governed
claim silently retarget.

### Business value
Each of these gates was built, tested and then deliberately left tolerant until
the repository could satisfy it. That was the right call at build time and is
now the reason none of them can fail: a legacy spec without a requirement trace,
an overdue follow-up, an unrecorded amendment baseline — all pass today. A gate
that cannot fail is documentation, not governance, and this repository is the
reference instance others copy.

The assessment item is different in kind and is the reason this spec is not
purely procedural: debt marker ids are positional (`DEBT-001`), so a spec citing
one has evidence that silently points at a different marker as soon as an
earlier one is added. `pose-assessment-engine-precision` fixed exactly this
defect for integration gaps and left debt ids untouched.

### Constraints
- Flipping a gate to strict must come after the repository satisfies it, never
  before. Each requirement pairs the flip with the cleanup that earns it.
- Existing debt citations must keep resolving across the id change, or the
  change trades one silent retarget for a loud break.

### Non-goals
- Adding new governance mechanisms. Everything here already exists.
- Revisiting whether any of these gates should exist.

---

## 2. Requirements

### Functional
- R1: Debt marker ids shall derive from file and marker identity rather than
  scan position, so a citation cannot retarget when an earlier marker appears.
- R2: The ten pre-contract specs shall gain requirement-trace sections or be
  archived, and the legacy done-without-trace warning shall then become an
  error.
- R3: Amendment baselines shall be recorded for active-roadmap specs as they
  enter execution, making the amendment gate effective beyond fixtures.
- R4: `--fail-overdue` shall be adopted in the quarterly governance audit.
- R5: The feature and bugfix workflows and skills shall cite knowledge refs, so
  citation becomes routine rather than exceptional.
- R6: The first quarterly audit run and the first real recurrence intervention
  shall be reviewed and their verdicts recorded.
- R7: `--sarif` shall be adopted in the CI security surface once code-scanning
  upload accepts validation results.

### Non-functional
- No gate flips without the repository first passing it locally in strict mode.

### Security
- R7 publishes validation findings to code scanning; confirm no finding body
  carries a path or value outside the repository before enabling upload.

### Compatibility
- R1 changes the shape of debt ids once. Citations must be migrated in the same
  change, exactly as `gap_id` was.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/techdebt.go` — id derivation (R1).
- The ten pre-contract specs — trace sections or archival (R2).
- `.pose/workflows/` and `.agents/skills/` — knowledge citation (R5).
- `.github/workflows/security.yml` — SARIF upload (R7).

### Artifacts
- modified: .pose/specs/pose-governance-gate-activation/spec.md
- modified: pose-mcp/internal/pose/techdebt.go
- modified: pose-mcp/internal/pose/techdebt_coverage_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/trace_lint_test.go
- modified: .github/workflows/governance-audit.yml
- modified: .github/workflows/security.yml
- modified: .pose/workflows/feature.md
- modified: .pose/workflows/bugfix.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-bugfix/SKILL.md

### Delivery targets
- governance:gate-activation module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- Debt marker ids change shape once (R1), the same one-time break
  `pose-assessment-engine-precision` accepted for `gap_id`.

### Technical risks
- R2 is the largest: ten legacy specs is real archaeology, and archiving one is
  a judgement call about whether its requirements still describe the system.
- Flipping a gate in the same change that cleans up for it makes the diff hard
  to review. Prefer cleanup first, flip second, in separate commits.

---

## 4. Tasks

### Planning
- [x] Enumerate the pre-contract specs and decide trace-or-archive for each —
      there are eleven, not ten, and all eleven earned a trace
- [x] Confirm which debt citations exist today, so R1's migration is bounded

### Implementation
- [x] R1: derive debt ids from file and marker identity
- [x] R2: cleanup then flip the legacy trace warning to an error
- [x] R3: record amendment baselines for the specs in execution
- [x] R4: adopt --fail-overdue in the quarterly audit
- [x] R5: cite knowledge refs from the workflows and skills, both locales
- [ ] R6: review the first audit run and the first recurrence intervention
- [x] R7: adopt --sarif in the security surface

### Validation
- [x] Each flipped gate demonstrated failing on a deliberately broken fixture
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A gate flip is only proven by watching it fail: for each, construct the
violation it is supposed to catch and confirm a non-zero exit, then confirm the
repository itself passes. R1 is proven by adding a marker earlier in the scan
and asserting existing citations still resolve.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/pose ./internal/cli -run "Debt|Trace|Amendment|Followup" -count=1`
- Scope: debt id identity and the flipped gates
- Expected: ok

#### Security / Contract
- Command: `pose check --strict` and `pose lint-spec --all --strict`
- Scope: the repository under the newly strict gates
- Expected: SUCCESS

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.18.2-dev.
- Notes: the spec said ten pre-contract specs; there are eleven. It also assumed
  they might be old enough that retrofitting a trace would be dishonest. They
  are from 2026-07-18/21 — three weeks — and every one describes behaviour that
  still exists, several of them confirmed against published artifacts earlier
  today. So all eleven earned a real trace and none was archived.

### Results summary
- Successes: R1, R2, R3, R4, R5, R7, each with the gate demonstrated failing
  where a gate was flipped
- Failures: none
- Deferred: R6 (calendar-bound)

### Requirement trace
- R1 [satisfied] governance:gate-activation evidence:integration check:delivery-integration test:TestTechDebtIDIsDerivedFromMarkerIdentityNotScanPosition — debt ids now derive from file, marker and line text; `DEBT-001` became `DEBT-05b998d4` and no longer moves when a marker is added earlier in the scan
- R2 [satisfied] test:TestTraceDoneWithoutTraceSectionFails check:lint-spec-all — all eleven pre-contract specs gained a trace, then the warning became an error; proven by removing `pose-version-contract`'s trace and watching the lint fail, then restoring it
- R3 [satisfied] check:amend-baseline — baselines recorded for the three specs currently in execution; the gate had zero baselines before, so it could only ever pass
- R4 [satisfied] check:governance-audit — the quarterly audit now runs `followups --all --fail-overdue`; the repository passes it today (overdue=0), which is what earns the flip
- R5 [satisfied] report:.pose/workflows/feature.md report:.pose/workflows/bugfix.md — both workflows and both skills now name the exact `knowledge:<slug>` form the counter recognises, in English and pt-BR
- R6 [waived: calendar-bound, nothing to review yet] — the first quarterly audit is dated 2026-10-01 and the first recurrence intervention needs recurrence-check to flag something; neither can be reviewed before it happens
- R7 [satisfied] check:security-workflow — `validate --tolerant --sarif` publishes deterministic validation findings to code scanning; inspected the generated SARIF first and confirmed it carries only repository-relative URIs and no absolute path

### Findings

**F1 — the gate count was wrong in the spec (severity: low).**
Eleven specs lacked a trace, not ten. Whoever wrote the item counted from a
stale reading. Worth noting because the same off-by-one would have left one spec
silently exempt if the cleanup had been driven by the number instead of by the
lint output.

**F2 — one retrofitted trace could not honestly say "satisfied" (severity: medium).**
`pose-cyclonedx-sbom` R1 requires an SBOM "with versions, hashes and known
licenses". The published SBOMs carry versions and hashes but almost no licenses,
which I confirmed earlier today. Its trace records `[waived]` with that reason
rather than claiming satisfaction. `pose-release-signing` R3 is waived for the
same kind of reason: CI signs and verifies, but no run has ever failed on an
unsigned artifact, so the rejection path is asserted rather than demonstrated.
Retrofitting traces surfaced two requirements that were never actually met.

### Known gaps
- R6 is calendar-bound: the first quarterly audit is dated 2026-10-01.
- Debt citations in `.pose/reviews/` still name `DEBT-001`. Reviews are
  immutable by contract, so they were not rewritten; the id changed shape once,
  exactly as `gap_id` did, and the changelog says so.

---

## 7. Final Report

### Delivered scope
Six of seven gates activated. Debt ids are content-derived, so a citation cannot
retarget. Eleven pre-contract specs gained real traces and the legacy warning is
now an error. Amendment baselines exist for the first time. The quarterly audit
can fail on overdue follow-ups. Both workflows and both skills, in both locales,
name the citation form the counter actually recognises. Validation findings
reach code scanning as SARIF.

### Files and modules changed
- pose-mcp/internal/pose/techdebt.go and its test
- pose-mcp/internal/cli/lintspec.go and trace_lint_test.go
- eleven pre-contract specs (requirement traces)
- .github/workflows/governance-audit.yml, .github/workflows/security.yml
- .pose/workflows/{feature,bugfix}.md and .agents/skills/pose-{feature,bugfix}, both locales

### Validation executed
- Command: `go -C pose-mcp test ./... -count=1`
- Result: PASS

### Residual risks
- The retrofitted traces are as good as the evidence I could verify today. Where
  I could not verify, they say waived and why — but a reader should treat a
  three-week-old retrofit as weaker than a trace written at closeout.
- R7 publishes to code scanning on every security run. If validation output ever
  starts carrying user-supplied text, the SARIF inspection should be repeated.

### Follow-ups

- [done] R2's archaeology found the opposite of what was feared: the eleven specs are three weeks old and all describe live behaviour, so every one earned a real trace and none was archived. (owner:@pose-maintainers crit:medium review:2026-10-23)
- [open] R6: review the first quarterly audit run (2026-10-01) and the first real recurrence intervention (`validate-native`, registered 2026-08-14; knowledge:escalation-validate-native), then record both verdicts. The intervention window ends 2026-09-28 and the audit cannot be reviewed before it runs. (owner:@pose-maintainers crit:medium review:2026-10-08)
- [done] Both delivered rather than amended: licenses now resolve in the SBOM (pose-sbom-license-inventory) and the signing rejection path is exercised on every CI run (tests/release/verify-negative.sh). F2: `pose-cyclonedx-sbom` R1 and `pose-release-signing` R3 were waived in their retrofitted traces because the behaviour was never actually delivered — licenses are absent from the SBOM and the signing rejection path has never fired. Decide whether to deliver them or amend the requirements. (owner:@pose-maintainers crit:medium review:2026-09-18)
