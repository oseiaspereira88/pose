---
slug: adaptive-instance-provisioning
status: draft        # draft | active | done | abandoned
created_at: 2026-08-15
depends_on:          # prerequisite roadmaps, inline list: other-roadmap-a, other-roadmap-b
---

# Roadmap: Adaptive instance provisioning

**Outcome:** a `pose init`/`pose install` run reflects the target
repository's actual stack and topology, never the engine's own development
footprint or a one-size-fits-all Go/React default.

This roadmap originates from four issues surfaced in a single dogfooding
session — [#21](https://github.com/oseiaspereira88/pose/issues/21),
[#22](https://github.com/oseiaspereira88/pose/issues/22),
[#23](https://github.com/oseiaspereira88/pose/issues/23) and
[#24](https://github.com/oseiaspereira88/pose/issues/24) — each independently
pointing at the same root cause from a different angle: onboarding a
brownfield or non-Go/React repository still leaves traces of POSE's own
development environment behind (leaked `pose-mcp`/`mcp-enforce` module
entries, an embedded Go/React rule pair with no relevance to the target
stack, unpopulated module metadata, redundant monorepo validation) rather
than adapting to what is actually there.

The implementation order is intentional. `index-hygiene` is a self-contained
bugfix and ships first regardless of everything else. `rule-extensionization`
must resolve its compatibility ADR and land before `adaptive-delivery` can
exist, because "install the right rule for this stack" only becomes a
well-defined problem once the target rule is an installable extension rather
than a file baked into every install. `stack-detection` is independent and
can run in parallel with `rule-extensionization`. `monorepo-advisory` has no
dependency on the rest of this roadmap and may land at any point in the
sequence. Reading and summarizing a repository's existing prose
documentation (README/CLAUDE.md) into `AGENTS.md` was evaluated and
deliberately excluded from this roadmap: it carries a materially different
risk profile (summarization, not deterministic file-presence detection) than
every milestone below and is tracked separately rather than diluting this
roadmap's exit criteria.

## Milestone: index-hygiene
- after:
- target_start:
- target_due:
- specs: pose-scaffold-index-template-neutralization

**Exit gate:** a fresh `pose install` into an empty repository produces
`.pose/indexes/module-metadata.json` and `.pose/indexes/validation-matrix.json`
with zero references to `pose-mcp`, `mcp-enforce`, `docs-site` or
`@pose-maintainers`; a regression test enforces this against the embedded
scaffold dist on every build.

## Milestone: rule-extensionization
- after:
- target_start:
- target_due:
- specs: pose-domain-rule-extension-migration

**Exit gate:** `backend-go.md` and `frontend-react.md` are no longer embedded
in core machinery; they install exclusively through the same signed,
transactional extension mechanism already proven by `pose-rule-kubernetes`.
An ADR fixes the compatibility strategy for already-installed instances
before this milestone's spec starts implementation — resolving that decision
is a precondition, not a parallel task.

## Milestone: stack-detection
- after:
- target_start:
- target_due:
- specs: pose-stack-detection-consolidation

**Exit gate:** stack detection is consolidated onto a single canonical
scanner; a fresh `pose install`/`pose init --wizard` run seeds
`.pose/indexes/module-metadata.json` from what it actually discovers instead
of shipping a static seed, and resolves `AGENTS.md`'s mechanical placeholders
(project name, detected stack) without requiring manual edits first.

## Milestone: adaptive-delivery
- after: rule-extensionization, stack-detection
- target_start:
- target_due:
- specs: pose-adaptive-rule-delivery

**Exit gate:** the stack detected during install/onboarding resolves to a
concrete rule-extension install decision (auto-install the matched baseline
or prompt for confirmation) — no repository receives a domain rule for a
stack it does not use, and no repository is left without one for a stack
POSE recognizes.

## Milestone: monorepo-advisory
- after:
- target_start:
- target_due:
- specs: pose-monorepo-validation-advisory

**Exit gate:** `pose doctor` recognizes the signature of redundant
root-plus-child validation execution in npm/pnpm/Yarn/Cargo workspace
layouts and advises declaring the existing `moduleOverrides.<path>.
replaceDefaultChecks` mechanism — it never silently skips a module on its
own. `pose validate --root-only` and `--workspace <path>` ship as documented
aliases of the existing `--module` selector.

## Risk controls

- No milestone in this roadmap may silently change validation-evidence
  provenance for an existing check without an explicit, machine-readable
  reason recorded — auto-skip-by-inference is out of scope everywhere in
  this roadmap, not only in `monorepo-advisory`.
- `rule-extensionization` must not ship until its compatibility ADR
  explicitly states what happens to already-installed instances on their
  next `pose update` — silently dropping a rule file an agent depends on is
  a regression, not a cleanup.
- Keep `index-hygiene`'s regression test as a durable structural guard, not
  a one-time fix — the same leak class already recurred once
  (`pose-scaffold-self-referential-policy-fix`, issue #17) before this
  roadmap's issue #22.
