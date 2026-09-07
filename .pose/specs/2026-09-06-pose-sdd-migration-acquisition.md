---
slug: pose-sdd-migration-acquisition
status: in-progress
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-canonical-positioning
priority: 3
components: docs, cli
delivers:
---

# Spec: SDD migration as an acquisition path

## 1. Intent

### Goal
Turn the existing Spec Kit and OpenSpec importers from a documented capability
into the shortest path a user of another SDD framework has into POSE.

### Business value
`pose import spec-kit` and `pose import openspec` already exist. They are
described as integrations. They are actually the answer to the strongest
objection POSE faces — "I already have specs written elsewhere" — and to the
strongest acquisition query POSE can rank for: someone searching for an
alternative to a framework they have outgrown.

Presented as an integration, the importer implies coexistence, which
contradicts the authority model. Presented as migration, it states the model
correctly and removes the switching cost in the same breath.

### Constraints
- Claims about other frameworks must be factual and verifiable, per the site's
  content policy. Describe what POSE imports, not what a competitor lacks.
- The importers must be exercised against real fixture inputs, not described.

### Non-goals
- Supporting concurrent operation of two base SDD frameworks. That is the
  model this spec exists to state clearly.

---

## 2. Requirements

### Functional
- R1: A migration guide shall exist per supported source framework, covering
  dry-run inspection, import, review of what was imported, and the first
  governed loop on the imported work.
- R2: Each guide shall state explicitly what transfers, what does not, and
  what requires a human decision — a migration path that hides its losses
  produces an angry user at the first gate.
- R3: The authority model shall be stated in each guide: migration
  interoperability, single authoritative lifecycle after adoption.
- R4: Each documented migration shall be exercised by a test against a fixture
  repository representative of the source framework's layout.
- R5: The guides shall be reachable from the landing page and the README, not
  only from the documentation tree.

### Compatibility
- No importer behavior change. If a gap is found while writing the guides, it
  becomes a follow-up rather than scope creep here.

---

## 3. Technical Plan

### Affected areas
- `docs-site/docs/` — migration guides
- `tests/import/` — fixture-based migration tests

### Artifacts
- created: docs-site/docs/migrate/spec-kit.md
- modified: docs-site/mkdocs.yml
- modified: README.md
- modified: .github/workflows/ci.yml
- created: docs-site/docs/migrate/openspec.md
- created: tests/import/fixtures/
- created: tests/import/migration-guides.sh

### Technical risks
- Writing the guides will surface importer gaps. Recording them as follow-ups
  keeps this spec honest; silently documenting around them does not.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Fixture repositories per source framework (R4)
- [x] Increment 2: Spec Kit guide, including what does not transfer (R1, R2, R3)
- [x] Increment 3: OpenSpec guide (R1, R2, R3)
- [x] Increment 4: Executable migration tests (R4)
- [ ] Increment 5: Surface the guides from landing and README (R5)

---

## 6. Validation

### Deterministic checks

#### Test
- Command: `bash tests/import/migration-guides.sh`
- Scope: each documented migration against its fixture
- Expected: exit 0; the imported work reaches a first governed loop

### Execution log
- Date: 2026-09-07
- Environment: local Linux, Go 1.26.5
- Notes: the test caught a documentation defect before publication. The Spec
  Kit guide originally claimed `--dry-run` prints the rendered spec including
  its Import Provenance section; it actually emits a structured
  `import.spec` / `import.curation` / `import.summary` summary. The guide was
  corrected to show the real output and to tell readers to sanity-check the
  `requirements=` count. That is the whole reason R4 exists.

### Results summary
- Successes: R1, R2, R3, R4 verified; R5 partially (README done).
- Failures: none.
- Warnings: see Known gaps.

### Requirement trace
- R1 [satisfied] <docs-site/docs/migrate/spec-kit.md, docs-site/docs/migrate/openspec.md — dry run, import, curation and first governed loop>
- R2 [satisfied] <"What does not transfer" in both guides, derived from import.go rather than assumed; check:migration-guides>
- R3 [satisfied] <"POSE becomes the lifecycle authority" in spec-kit.md, referenced from openspec.md>
- R4 [satisfied] <tests/import/migration-guides.sh against tests/import/fixtures/; check:ci-migration-guides>
- R5 [deferred-integration: spec:harne8-pose-launch-surfaces] <README links added; the landing-page link belongs to the site repository's spec>

### Known gaps
- R5 is half done. The README points at both guides; the landing page does not
  yet, because `site/pose.html` lives in the harne8 repository and is governed
  by `harne8-pose-launch-surfaces`.
- The guides describe two importers. Any third source framework added later
  needs its own guide and fixture, and nothing detects a new importer shipping
  without them.

---

## 7. Final Report

### Follow-ups

- [open]
