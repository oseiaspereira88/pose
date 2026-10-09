---
slug: pose-canonical-positioning
status: in-progress
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-release-recovery-verification
priority: 1
components: docs
delivers: governance:canonical-positioning
---

# Spec: Canonical positioning and SDD authority model

## 1. Intent

### Goal
Fix the two statements POSE makes about itself that are inaccurate: that it is
a governance layer, and that other SDD frameworks are complementary to it at
runtime.

### Business value
The README opens with "POSE is the free, Apache-2.0 governance core". A reader
arriving from Spec Kit, OpenSpec or Kiro concludes that they keep their SDD
framework and add POSE on top. The product does not work that way: `pose init`,
`new-spec`, `suggest`, `lint-spec --ready-check`, `validate --strict` and the
closeout flow are a complete specification-and-delivery lifecycle. POSE owns
requirement IDs, status, dependencies, readiness and the definition of done
once adopted.

The README also states that these products "can be complementary". Two base
SDD frameworks over one lifecycle produce competing authorities for exactly
the artifacts POSE gates on. The importers that already exist (`pose import
spec-kit`, `pose import openspec`) encode the real model — migration, not
coexistence — and no public surface says so. Someone who follows the current
README into a dual-framework setup will hit contradictions POSE cannot resolve.

### Constraints
- The comparison must stay factual and verifiable. The site's own content
  policy forbids unverifiable claims about other products, and a launch that
  breaks that policy costs more credibility than the positioning gains.
- Honesty about fit is retained: POSE carries more lifecycle structure than a
  lightweight planning folder, and readers for whom that is wrong should be
  able to tell.

### Non-goals
- A feature-by-feature competitive table. Teach the category first.
- Deprecating the importers. They become more prominent, not less.

---

## 2. Requirements

### Functional
- R1: The canonical description shall be "an open-source Spec-Driven
  Development framework for governed agentic software delivery", used
  identically in the README, the docs home, the repository description and the
  site.
- R2: The lifecycle `discover → specify → route → execute → prove → close →
  learn` shall appear as the product's conceptual signature on every surface
  that explains what POSE does, with no competing five-verb or four-verb
  variant left in place.
- R3: The interoperability section shall state the actual model: migration
  interoperability with other base SDD frameworks, single authoritative
  lifecycle after adoption — and name the artifacts whose ownership would
  otherwise be contested (requirement IDs, status, dependencies, readiness,
  definition of done, closeout, knowledge).
- R4: Agent-neutrality and SDD-framework-neutrality shall be distinguished
  explicitly. POSE is neutral about who executes; it is not neutral about who
  owns the lifecycle.
- R5: The "free core / scale path" framing shall be replaced. POSE is a
  complete product in its domain, not a deliberately limited tier of Harne8.

### Compatibility
- No CLI or artifact-format change. This spec changes what is claimed, not
  what is built.

---

## 3. Technical Plan

### Affected areas
- `README.md`, `README.pt-BR.md`
- `docs-site/docs/index.md`, `docs-site/docs/concepts.md`
- GitHub repository description and topics

### Artifacts
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/index.md
- modified: docs-site/docs/concepts.md
- modified: docs-site/docs/analytics.md
- modified: docs-site/docs/architecture.md
- modified: docs-site/docs/capability-assessment.md
- modified: docs-site/docs/ci.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/frontmatter.md
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/monorepo-recipes.md
- modified: docs-site/docs/package-channels.md
- modified: docs-site/docs/product-roadmaps.md
- modified: docs-site/docs/quickstart.md

Backfilled on 2026-09-29: `82726a3` (docs: keep the Diátaxis applicability line evergreen), `3ed4cc0` (docs: reposition POSE as a governed SDD framework) carries this spec's trailer and also changed the 11 paths declared last. They are claimed so the Git change set reconciles; the commit bundled other work, so a claim here records provenance, not that this spec designed the change.

### Delivery targets
- governance:canonical-positioning module:. profile:release-governance entrypoint:README.md

### Technical risks
- Overclaiming the category invites "this is just Spec Kit with extra steps".
  The defense is the lifecycle after implementation — readiness, evidence,
  follow-up disposition, recurrence — which is demonstrable, not rhetorical.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Canonical description and lifecycle across surfaces (R1, R2)
- [x] Increment 2: Rewrite the interoperability section (R3, R4)
- [x] Increment 3: Replace the free-core framing (R5)

---

## 5. Decisions

### Decision 1
- Date: 2026-09-06
- Context: "governance framework" versus "SDD framework" as the entry category.
- Decision: lead with SDD, differentiate with governed delivery.
- Rationale: "governance framework" is a category the reader must already
  believe in, and it describes half the product. "SDD framework" is a category
  with existing search demand and existing competitors, which makes POSE
  findable and comparable. The differentiation then does real work: SDD that
  does not end when the agent starts writing code.
- Consequences: POSE is compared against Spec Kit and OpenSpec directly. That
  is the correct comparison, and it is only survivable because the post-
  implementation lifecycle genuinely exists.

---

## 6. Validation

### Deterministic checks

#### Lint
- Command: `pose docs-check`
- Expected: exit 0

#### Security / Contract
- Command: `pose public-claims --strict`
- Expected: exit 0 — no surface contradicts the canonical description

### Requirement trace

- R1 [deferred-integration: xref:proj.harne8/spec:harne8-pose-open-integrations-reconciliation] <README.md, README.pt-BR.md (translated) and docs-site/docs/index.md carry the canonical description verbatim, as `.pose/public/claims.json` declares it, and `pose public-claims --strict` passes; the harne8.com site belongs to the Harne8 authority, and the GitHub repository description is an operation on the hosting account recorded as a follow-up>
- R2 [satisfied] check:public-claims evidence:manual <README.md, README.pt-BR.md, docs-site/docs/index.md and docs-site/docs/concepts.md show `discover → specify → route → execute → prove → close → learn`; the six-stage loop of the concepts page is now presented as the artifacts each part of that lifecycle leaves, not a competing variant>
- R3 [satisfied] check:public-claims evidence:manual <README "POSE and other SDD frameworks" states migration interoperability and a single lifecycle after adoption, and "Agent-neutral, lifecycle-authoritative" names requirement IDs, status, dependencies, readiness, definition of done, closeout and knowledge>
- R4 [satisfied] check:public-claims evidence:manual <README states that agent-neutral is not SDD-framework-neutral: neutral about who executes, not about who owns the lifecycle>
- R5 [satisfied] check:public-claims evidence:manual <README "POSE and Harne8" presents POSE as complete within its domain, with no paywall; the table that read "Start with POSE / Scale with Harne8" now reads "POSE governs a repository / Harne8 governs a portfolio", in both languages>

---

## 7. Final Report

### Delivered scope

The canonical description appears verbatim in the README and the docs home, and in the pt-BR README in translation; the lifecycle signature is on every surface in this repository that explains POSE, including the concepts page, whose six-stage loop no longer competes with it; the interoperability and agent-neutrality sections state the model; the Harne8 section and its table describe two scopes rather than a scale path. `pose docs-check` has no manifest in this repository, so `pose public-claims --strict` is the deterministic check, and it passes.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Set the GitHub repository description to the canonical description ("an open-source Spec-Driven Development framework for governed agentic software delivery"); it currently reads "A modern governance framework for agentic software engineering" (owner:@pose-maintainers crit:low review:2026-10-23)
