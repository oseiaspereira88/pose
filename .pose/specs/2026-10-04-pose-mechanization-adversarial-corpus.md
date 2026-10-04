---
slug: pose-mechanization-adversarial-corpus
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-review-assurance-disclosure, pose-obligation-contract
priority: 1
components: pose-mcp
delivers: capability:adversarial-corpus
changelog: none
task_type: feature
---

# Spec: Maintain an adversarial corpus against formal compliance without value

## 1. Intent

### Goal

Keep a deterministic corpus of real and synthetic cases where a gate can be satisfied in
form without its intended value, and test the engine against it in every wave.

### Business value

Each anti-mechanization mechanism has a mechanized version of itself: human label
without confirmation, unknown omitted, every doubt as blocker, minimal option as ritual.
Schemas do not catch these; behaviour tests do.

### Constraints

The corpus belongs to governance development; it does not add a checklist to adopters'
features.

Program source: backlog items POSE-28 (P1, wave 1) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F01, F02, F05, F06, F08, F11; sources
E01, E07, E16, E21). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

An LLM judge in the core; an automatic quality score.

### Anti-mechanization guardrail

Este corpus pertence ao desenvolvimento da governança; não cria checklist obrigatório em
cada feature do adotante.

## 2. Requirements

### Functional

- R1: The corpus shall include the 6.3.0 `human:oseias` attestations and fail if any render reads as human review while the schema stays valid.
- R2: A case with an unavailable producer shall fail if Attention or obligations report a complete empty answer.
- R3: A previously authorized instruction and a trivial change shall fail the corpus if they produce an artificial ActionRequest or a new mandatory step.
- R4: Cases shall exercise refusals and recovery paths, not only the JSON produced by the same writer.
- R5: Each new gate added by this program shall add its cheapest-formal-satisfaction case to the corpus before its spec closes; the review template gains that question for governance changes.

### Non-functional

- Runs inside `go test ./...` and in the validation matrix.

### Security

- None specific beyond the shared constraints.

### Compatibility

- None specific beyond the shared constraints.

## 3. Technical Plan

### Affected areas

Test corpus under pose testdata, review template for governance changes.

### Artifacts

- created: .pose/specs/2026-10-04-pose-mechanization-adversarial-corpus.md
- created: pose-mcp/internal/cli/adversarial_corpus_test.go
- created: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: .pose/indexes/validation-matrix.json

Reconciled against the tree at activation.

### Delivery targets

- capability:adversarial-corpus module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

The review template planned in the draft (`.pose/templates/review.md`) is the
parsed legacy review-attempt format, so the cheapest-formal-satisfaction
question went into the review skill instead, where governance changes are
reviewed.

### Technical risks

- Corpus rot; each case states the invariant it protects and the spec that owns it.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-mechanization-adversarial-corpus`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Prove each case fails against the tree with the defect (gate must fail before it
passes), then passes after the owning spec lands.

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
- Command: `pose lint-spec pose-mechanization-adversarial-corpus --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
