---
slug: pose-standalone-dogfood
status: draft
created_at: 2026-07-18
completed_at:
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
- [ ] Confirm baseline and fixtures against [DORA platform engineering](https://dora.dev/capabilities/platform-engineering/).

### Implementation
- [ ] Define minimum artifact ownership and review rules for this repository. ([reference](https://dora.dev/capabilities/platform-engineering/))
- [ ] Run this portfolio through check, index and readiness gates. ([reference](https://scorecard.dev/))
- [ ] Add CI retention and quarterly housekeeping evidence without claiming past history. ([reference](https://dora.dev/capabilities/platform-engineering/))

### Validation
- [ ] Run `pose check --strict` and retain the result artifact. ([reference](https://dora.dev/capabilities/platform-engineering/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://scorecard.dev/))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [DORA platform engineering](https://dora.dev/capabilities/platform-engineering/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-standalone-dogfood --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Process theater emerges if artifacts are created after implementation or never reviewed.
- Follow-ups: none until implementation starts.

