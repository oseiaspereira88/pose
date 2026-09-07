---
slug: pose-community-contribution-surfaces
status: in-progress
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-readme-evaluation-path
priority: 3
components: docs
delivers: governance:community-contribution-surfaces
---

# Spec: Community contribution surfaces

## 1. Intent

### Goal
Make the repository legible to someone who wants to contribute but has no
context, so that an outsider can find a real task they could finish in a
morning.

### Business value
The launch's target is not stars. It is three people running POSE a second
time in repositories the maintainer does not control, and at least one
external contribution. Both require the repository to look like a project that
accepts outside work.

Today it does not, through no fault of its content: 140 specs and a governance
model of this depth read, from outside, as a system one must fully understand
before touching. The barrier is not the code. It is that no path exists from
"I noticed something" to "I opened a scoped change".

### Constraints
- No fabricated `good first issue` labels on work that is neither good nor
  first. An issue that turns out to require reading 115 specs costs more trust
  than it buys.
- No manufactured discussion threads. Maintainer-opened questions are honest;
  simulated conversation is not.

### Non-goals
- Building a community program. This is the minimum surface that makes
  contribution possible.

---

## 2. Requirements

### Functional
- R1: `CONTRIBUTING.md` shall describe how POSE's own governance applies to an
  external contribution — which artifacts a change is expected to carry, and
  which are maintainer responsibilities — so a contributor is not surprised by
  a gate after opening a pull request.
- R2: A set of genuinely scoped, self-contained issues shall exist, each
  stating the expected change surface and how to verify it locally.
- R3: Discussion categories shall exist including an RFC category, signalling
  that decisions are still open to influence.
- R4: The initial discussion threads shall be maintainer questions with real
  uncertainty behind them, not announcements phrased as questions.
- R5: A contributor shall be able to run the same validation CI runs, from a
  documented single command, without maintainer access.

### Compatibility
- No change to the governance model itself.

---

## 3. Technical Plan

### Affected areas
- `CONTRIBUTING.md`
- `.github/ISSUE_TEMPLATE/`
- GitHub Discussions configuration

### Artifacts
- modified: CONTRIBUTING.md
- created: .github/ISSUE_TEMPLATE/scoped_task.yml
- created: scripts/verify.sh

### Delivery targets
- governance:community-contribution-surfaces module:. profile:release-governance entrypoint:CONTRIBUTING.md

### Technical risks
- POSE gates its own repository strictly. An external contributor who cannot
  reproduce the gates locally will abandon the change. R5 is the load-bearing
  requirement here, not R2.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Contributor-facing governance expectations (R1)
- [x] Increment 2: One documented local command reproducing CI (R5)
- [ ] Increment 3: Scoped issues with verification instructions (R2)
- [ ] Increment 4: Discussion categories including RFC (R3, R4)

---

## 6. Validation

### Deterministic checks

#### Test
- Command: the single documented contributor validation command
- Scope: run from a fresh clone with no maintainer credentials
- Expected: exit 0, and its result matches what CI reports

### Execution log
- Date: 2026-09-07
- Environment: local Linux, Go 1.26.5
- Notes: `scripts/verify.sh --fast` was exercised in both directions — green on
  a clean tree, and exit 1 naming "Public claims gate" with every other gate
  still run to completion when drift was injected into a surface. A
  contributor script that stops at the first failure forces one round trip per
  problem.

### Results summary
- Successes: R1 and R5 delivered and verified.
- Failures: none.
- Warnings: R2, R3 and R4 need actions on the GitHub repository itself —
  see Known gaps.

### Requirement trace
- R1 [satisfied] <CONTRIBUTING.md: "Start here", the `POSE-Spec:` trailer section, and the ownership table separating contributor duties from maintainer ones>
- R2 [deferred-integration: spec:pose-community-contribution-surfaces] <the issue template exists; the issues themselves are an action on the public repository>
- R3 [deferred-integration: spec:pose-community-contribution-surfaces] <Discussions categories are repository settings>
- R4 [deferred-integration: spec:pose-community-contribution-surfaces] <same>
- R5 [satisfied] <scripts/verify.sh; check:verify-sh-green, check:verify-sh-failure-path>

### Known gaps
- R2, R3 and R4 are deliberately not executed here. Opening public issues and
  creating Discussion categories are outward-facing actions on the project's
  public repository, and doing them unprompted would publish content under the
  maintainer's name. The template and the local reproduction path — the parts
  that are code — are done; the publishing step is the maintainer's.
- R5 carries an upkeep hazard: `scripts/verify.sh` mirrors `ci.yml` by hand.
  A gate added to CI and not to the script produces false local confidence,
  which is worse than having no script. The header says so; nothing enforces
  it.

---

## 7. Final Report

### Follow-ups

- [open]
