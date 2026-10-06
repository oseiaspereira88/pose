---
slug: pose-legacy-contract-cutoffs
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: 
priority: 0
components: pose-mcp
delivers: surface:legacy-contract-cutoffs
task_type: bugfix
---

# Spec: Read the legacy adoption cutoffs of every registered review contract

## 1. Intent

### Goal

Make `explicit_judgment_adopted_at` and `structural_causality_adopted_at` reach the
typed review policy and the `ContractAdoptedAt` fallback, with the same precedence rules
as the three older contracts.

### Business value

The repository policy declares both dates, the registry maps them to contracts, and yet
`ReviewPolicy` has no field for them and the legacy fallback only knows three contracts;
`pose doctor` in the clean quickstart record reports them as unread keys. A configured
cutoff that produces no effect misleads anyone reading the history. Bundles already
sealed are not affected: they are judged by the contracts they stamped.

### Constraints

The `contract_adoptions` map keeps precedence, including an explicit empty value. Sealed
bundles keep their stamped contracts; editing a date later must not re-evaluate them.
The policy digest of a policy that does not set the new fields must not change.

Program source: backlog items POSE-01 (P0, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F03; sources
E04, E08, E18, E30). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Rewriting historical attestations; adopting opt-in capabilities (atomic start, contract
nodes, causality closeout); changing the registry's contract list.

### Anti-mechanization guardrail

Corrigir o dado efetivo; não desativar um gate para fazer o histórico passar.

## 2. Requirements

### Functional

- R1: When a policy declares `explicit_judgment_adopted_at` or `structural_causality_adopted_at` and no `contract_adoptions` entry for that contract, `ContractAdoptedAt` shall return the declared date.
- R2: When the map and a legacy key coexist, the map shall prevail, including an explicit empty map value, and `pose doctor` shall report the shadowed legacy key.
- R3: A bundle whose `governing_contracts` stamp is present shall be judged by that stamp regardless of a later edit to either date.
- R4: A table-driven test shall walk every registry entry and fail when a legacy key the registry names is not consumed by the typed reader or the fallback.
- R5: `pose doctor` shall distinguish absent, explicitly empty, map-declared and legacy-declared adoption for every registered contract.

### Non-functional

- No new policy schema version; the change is a reader fix.

### Security

- An unparsable date is an error with the key named, never a silent absence that weakens a gate.

### Compatibility

- Policies without the two keys serialize and digest exactly as before; old binaries keep reading the policy.

## 3. Technical Plan

### Affected areas

Review policy reader, contract registry, legacy adoption fallback and doctor
diagnostics.

### Artifacts

- created: .pose/specs/2026-10-04-pose-legacy-contract-cutoffs.md
- modified: pose-mcp/internal/pose/review_closeout.go
- created: pose-mcp/internal/pose/review_contract_legacy_test.go
- modified: pose-mcp/internal/cli/doctor.go
- created: pose-mcp/internal/cli/doctor_legacy_contract_cutoffs_test.go
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-legacy-contract-cutoffs.md -> .pose/changelogs/v7.0.0/pose-legacy-contract-cutoffs.md

Reconciled against the tree at activation.

### Delivery targets

- surface:legacy-contract-cutoffs module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A naive fix could reclassify historical bundles that were legitimately grandfathered; the stamped-bundle test guards it.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-legacy-contract-cutoffs`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Reproduce first: a fixture policy with only the two legacy keys returns an empty
adoption date today (test fails before the fix). Then the compatibility matrix
map/legacy/absent/empty/conflict for every registry contract, plus the stamped-bundle
invariance test.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./...`
- Scope: engine packages touched by this spec
- Expected: pass, including the new negative tests

#### Lint
- Command: `cd pose-mcp && go vet ./...`
- Scope: pose-mcp
- Expected: no findings

#### Security / Contract
- Command: `pose lint-spec pose-legacy-contract-cutoffs --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:legacy-contract-cutoffs evidence:integration test:TestContractAdoptedAtReadsEveryRegisteredLegacyField check:legacy-contract-cutoffs-integration
- R2 [satisfied] test:TestContractAdoptionMapPrevailsOverLegacyIncludingExplicitEmpty check:legacy-contract-cutoffs-integration test:TestDoctorReportsAShadowedLegacyCutoff
- R3 [satisfied] test:TestLegacyKeyCutoffsReachTheDatedRuleButNeverAStampedBundle check:legacy-contract-cutoffs-integration
- R4 [satisfied] test:TestContractAdoptedAtReadsEveryRegisteredLegacyField check:legacy-contract-cutoffs-integration
- R5 [satisfied] test:TestContractAdoptionSourceDistinguishesEveryDeclaration check:legacy-contract-cutoffs-integration test:TestDoctorReportsLegacyContractCutoffSources

## 7. Final Report

### Delivered scope

Delivers reading of every registered legacy adoption key with map precedence and doctor reporting. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
