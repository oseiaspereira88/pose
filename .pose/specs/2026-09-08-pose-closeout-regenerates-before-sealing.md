---
slug: pose-closeout-regenerates-before-sealing
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
delivers:
---

# Spec: The closeout says to regenerate evidence before sealing

## 1. Intent

### Goal
Put the order a closeout has to follow — regenerate, index, seal, attest, commit
— in the skill every instance receives.

### Business value
The engine reads exactly one validation results file: the path
`.pose/policy/delivery.json` names in `results_path`. Nothing else under
`.pose/results/` is read by anything. A bundle seals whatever that file holds
when it is sealed.

Neither the skill nor the workflow said so. In one adopting repository the path
named a file created for a spec closed weeks earlier, and every sealed bundle in
that repository — seven of them — carried the same two results from an unrelated
component, whatever the spec covered. Every attestation that cited its own
spec's real check cited something the bundle could not contain. It was not
carelessness by whoever attested; the evidence was unreachable, and nothing said
where to reach it.

Skipping the regeneration does not fail. That is what makes it worth documenting
rather than assuming: the bundle seals silently, the attestation records, and
the discrepancy only surfaces when something finally compares the two — which
POSE only began doing in 2.0.0.

The order matters as much as the step. Committing the result before sealing
moves the head and invalidates that result's provenance for a scope still open,
so the sequence has to end with the commit.

### Constraints
- Both locales, and the embedded scaffold, or an instance receives one and not
  the others.

### Non-goals
- Enforcing the order. A check that a bundle's evidence is current would be
  better than a paragraph, and is a larger change than restoring the paragraph
  that was missing.

---

## 2. Requirements

### Functional
- R1: The shipped closeout skill shall state the sequence and the reason the
  order matters.
- R2: It shall say the engine reads one file, and that skipping the step fails
  silently rather than loudly.
- R3: Both locales and the embedded scaffold shall carry it.

### Non-functional
- The scaffold parity test passes.

---

## 3. Technical Plan

### Affected areas
- `.agents/skills/pose-spec-closeout/SKILL.md` and its pt-BR counterpart

### Artifacts
- created: .pose/specs/2026-09-08-pose-closeout-regenerates-before-sealing.md
- renamed: .pose/changelogs/unreleased/pose-closeout-regenerates-before-sealing.md -> .pose/changelogs/v3.0.0/pose-closeout-regenerates-before-sealing.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md

### Technical risks
- Documentation does not enforce anything. The failure it describes is silent,
  so an operator who does not read the step gets the same outcome as before.

---

## 4. Tasks

### Implementation
- [x] Increment 1: State the sequence in both locales (R1, R2, R3)

### Validation
- [x] Renumbering checked and the scaffold regenerated

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: this text first existed as an instance-local edit in the adopting
  repository, which `pose update` replaced on the next run — `.agents/skills` is
  machinery, delivered whole-file.
- Decision: it belongs here.
- Rationale: it is not a fact about that repository. The engine reads one
  results file in every instance, and the order is the same everywhere. The
  update that wiped the local copy also delivered a note contributed upstream
  earlier, which is the same mechanism proving where the content should live.

---

## 6. Validation

### Strategy
Check that the step lands in the right place, that the list renumbers, and that
the scaffold copies match.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./internal/scaffold/...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: the step is inserted before the review pass and the following items
  renumber from 4, verified by reading the rendered list in both files.
  `go generate ./internal/scaffold` mirrored both into the embedded scaffold,
  whose parity test fails if they diverge.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <step 3 of the closeout checklist gives the two commands and states that the commit moves the head and invalidates provenance for an open scope>
- R2 [satisfied] <the same step says the engine reads one file and nothing else under .pose/results/, and that skipping does not fail>
- R3 [satisfied] <both SKILL.md files and both embedded copies changed; the scaffold parity test covers the mirror>

### Known gaps
- Nothing checks that a bundle's evidence was regenerated for the scope being
  sealed. Documentation is the weakest form of this fix.

---

## 7. Final Report

### Follow-ups

- [done] Warn when a bundle seals evidence whose run predates the change set it approves. Delivered by `pose-seal-names-carried-forward-evidence`, which names such evidence at seal time rather than leaving it silent. (owner:unowned crit:medium review:2026-11-08)
