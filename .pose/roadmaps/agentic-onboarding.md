---
slug: agentic-onboarding
status: active        # draft | active | done | abandoned
created_at: 2026-08-15
depends_on:          # prerequisite roadmaps, inline list: other-roadmap-a, other-roadmap-b
---

# Roadmap: agentic-onboarding

**Outcome:** onboarding a brownfield repository with `pose init`/`pose
install` produces an `AGENTS.md` populated from the repository's own
existing documentation, a validation module catalog free of duplicate
scanning logic, a path for the stack's rule extensions to actually reach a
non-English instance, and a way for the doctor's "install this extension"
advisory to become a single runnable command instead of a manual step.

This roadmap closes out
[#21](https://github.com/oseiaspereira88/pose/issues/21) (the one GitHub
issue from the v1.3.0 triage still open) and continues three follow-ups
`agentic-onboarding`'s predecessor roadmap
([`adaptive-instance-provisioning`](adaptive-instance-provisioning.md))
deliberately deferred rather than folded in:

- Issue #21 listed four problems. v1.3.0 closed two (hardcoded domain rules,
  empty module metadata). The remaining two — unfilled `AGENTS.md`
  placeholders, and pre-existing project documentation never being read —
  are one problem in practice: nothing parses `README.md`/`CLAUDE.md` into
  `AGENTS.md`'s `## Project context` section on a brownfield install.
  `adaptive-instance-provisioning`'s own roadmap text named this
  "evaluated and deliberately excluded... tracked separately", because
  summarizing prose carries a different risk profile than deterministic
  file-presence detection. `context-extraction` below is that separate
  tracking.
- `pose-stack-detection-consolidation`'s follow-up: `scanModules`/
  `discovery.go` and `discoverValidationModules` are two module scanners
  doing overlapping work; only the latter got reused for module-metadata
  seeding in v1.3.0. Consolidating is deferred tech debt, not urgent, but
  real.
- `pose-adaptive-rule-delivery`'s follow-up: the doctor's rule-extension
  recommendation is advisory-only because `pose extension install` has no
  way to resolve an extension ID to a real package — it only accepts a
  local directory. `pose-extension-catalog-lifecycle` (2026-07-19) already
  specified "a signed catalog shall support discovery" (R3), but the
  current `cmdExtensionInstall` implementation never grew a catalog/URL
  resolution path — R3 shipped as `pose extension list`/`verify` over an
  already-installed catalog, not discovery of new packages. This milestone
  is the gap between that spec's stated intent and what the CLI actually
  does today.
- `pose-domain-rule-extension-migration`'s follow-up: `pose-rule-backend-go`
  and `pose-rule-frontend-react` ship English-only content, so a pt-BR
  instance installing either gets a locale-inconsistent result — the same
  gap `pose-rule-kubernetes` already had before this roadmap, now
  reproduced by the two new extensions.

Ordering: `context-extraction` is self-contained and ships first regardless
of the rest, the same way `index-hygiene` did in the predecessor roadmap.
`scanner-consolidation` and `locale-parity` are both independent and can run
in parallel with anything else here. `extension-catalog-resolution` carries
the most technical risk — it introduces a network-facing resolution step
into a mechanism that has been local-only since it shipped — and should
start with confirming the trust/signing model extends cleanly to a
resolved-then-fetched package before writing implementation code; it is
sequenced last for that reason, not because anything else blocks it.

## Milestone: context-extraction
- after:
- target_start:
- target_due:
- specs: pose-onboarding-context-extraction

**Exit gate:** `pose init`/`pose install` against a brownfield repository
with a non-trivial `README.md` (and `CLAUDE.md` when present) produces an
`AGENTS.md` whose `## Project context` section is populated from that
content instead of the current placeholder comments; a repository with
neither file falls back to today's placeholder behavior unchanged.

## Milestone: scanner-consolidation
- after:
- target_start:
- target_due:
- specs: pose-validation-scanner-consolidation

**Exit gate:** `discoverValidationModules` is the only module scanner
`pose install`/`pose init`/`pose validate` use; `scanModules`/
`discovery.go`'s overlapping logic is either removed or reduced to a thin
call into the consolidated scanner. Cloudflare Workers (`wrangler.json(c)`)
is a recognized entry in `validation-matrix.json`'s `stacks` catalog.

## Milestone: locale-parity
- after:
- target_start:
- target_due:
- specs: pose-rule-extension-locale-parity

**Exit gate:** a pt-BR instance installing `pose-rule-backend-go` or
`pose-rule-frontend-react` receives pt-BR content, matching the parity
`pose-locale-coverage-contract` already enforces for core machinery.

## Milestone: extension-catalog-resolution
- after:
- target_start:
- target_due:
- specs: pose-extension-catalog-resolution

**Exit gate:** `pose extension install <extension-id>` (no local directory
required) resolves the id to a real, signature-verified package through a
defined catalog source, so `pose doctor`'s rule-extension recommendation
can be satisfied by one runnable command instead of the operator sourcing
the package manually.
