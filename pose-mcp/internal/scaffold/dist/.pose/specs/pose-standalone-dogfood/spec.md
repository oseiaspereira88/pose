---
slug: pose-standalone-dogfood
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on:
priority: 2
---

# Spec: Standalone product dogfooding

## 1. Intent

### Goal
govern POSE's standalone delivery with product-owned specs, roadmaps, reports and recurrence evidence.
### Business value
Turns an empty installed instance into proof that the freemium product works on its own repository.
### Constraints
- Do not fabricate historic evidence or copy Harne8-only governance state.
### Non-goals
- Move Harne8 product architecture into the standalone repository.

## 2. Requirements

### Functional
- R1: Every non-trivial POSE product change shall have one owned spec and at most one active roadmap membership.
- R2: CI shall retain structural, validation and history evidence produced by the standalone instance.
- R3: A quarterly audit shall identify stale specs, roadmaps, knowledge and follow-ups.

### Non-functional
- Keep governance overhead proportional to change risk.

### Security
- Exclude restricted context, tokens and CI secrets from reports and history.

### Compatibility
- Dogfooding shall use a released CLI or an explicitly identified development build.

## 3. Technical Plan

### Affected areas
- `.pose/`, contributor workflow and CI.

### API/contract changes
- Contribution and release gates require product-owned POSE evidence.

### Data/storage changes
- Begin append-only history at adoption time; never backfill invented events.

### Technical risks
- Process theater emerges if artifacts are created after implementation or never reviewed.

### Primary references
- [DORA platform engineering](https://dora.dev/capabilities/platform-engineering/)
- [OpenSSF Scorecard](https://scorecard.dev/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [DORA platform engineering](https://dora.dev/capabilities/platform-engineering/): the installed instance had planning artifacts (7 roadmaps, 35 specs) but empty knowledge/reports, no ownership metadata, no CI gates and template-only `AGENTS.md` (kept as installer template by design).

### Implementation
- [x] Define minimum artifact ownership and review rules for this repository: "Dogfooding governance" section in `CONTRIBUTING.md`; module ownership in `.pose/indexes/module-metadata.json` (`pose-mcp`, `mcp-enforce` → `@pose-maintainers`, criticality high). ([reference](https://dora.dev/capabilities/platform-engineering/))
- [x] Run this portfolio through check, index and readiness gates: `pose check --strict`, `pose index`, ready-checks for the contract-baseline specs. ([reference](https://scorecard.dev/))
- [x] Add CI retention and quarterly housekeeping evidence without claiming past history: `governance` job in `.github/workflows/ci.yml` (structural + history gates with a `-dev`-identified build, evidence uploaded as artifact) and the scheduled `.github/workflows/governance-audit.yml` (quarterly audit of follow-ups, knowledge, history and stats; report retained 400 days). History begins at adoption — no backfill. ([reference](https://dora.dev/capabilities/platform-engineering/))

### Validation
- [x] Run `pose check --strict` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://dora.dev/capabilities/platform-engineering/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://scorecard.dev/))

## 5. Decisions

- Root `AGENTS.md`/`POSE.md` stay as installer templates (`{{PROJECT_NAME}}` placeholders): they are embedded into the binary and template-substituted at `pose install` time, so repository-specific governance lives in `CONTRIBUTING.md` instead of the template files. Structural or public contract changes still require an ADR; none was needed here (versioning policy is owned by ADR `2026-07-19-authoritative-release-version-source`).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-standalone-dogfood --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `pose check --strict` — SUCCESS.
- `pose index` — SUCCESS (module metadata propagated to `repo-map.json`/`packages.json`).
- `pose lint-spec pose-standalone-dogfood --ready-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (first governed evidence record in `.pose/reports/`).
- `pose knowledge-check`, `pose followups --all`, `pose stats` — SUCCESS (audit command set proven locally before scheduling).

## 7. Final Report

### Delivered scope

Ownership and review rules for the standalone instance (`CONTRIBUTING.md`,
`module-metadata.json`); CI `governance` job retaining structural/history
evidence produced by an explicitly identified development build; scheduled
quarterly governance audit with durable artifacts; first real validation
evidence recorded at adoption time (no fabricated history). The embedded
scaffold now excludes `.pose/reports/` (and IDE noise) — evidence is instance
state, and embedding it made every `pose validate --report` run drift the
embed parity guard it had just been tested by.

### Residual risks

- Process theater remains possible if audit findings are never dispositioned —
  mitigated by the audit failing on gate errors and by the CONTRIBUTING rule
  that findings become issues or specs; the quarterly cadence has not yet
  completed its first cycle.

### Follow-ups

- [open] Review the first quarterly audit run (2026-10-01) and disposition its findings. (owner:@pose-maintainers crit:medium review:2026-10-08)
- [covered: pose-ossf-security-baseline] Extend CI evidence with OpenSSF Scorecard and supply-chain checks.
