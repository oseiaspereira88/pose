# Contributing to POSE

POSE is developed by dogfooding: every non-trivial change to POSE is governed
by POSE itself — a spec with acceptance criteria, deterministic validation and
a formal closeout. Contributions follow the same path.

## Proposing a change

1. **Open an issue first** for anything beyond a typo fix. Describe the problem
   (not the solution) and, if you can, the evidence: a failing gate, a
   confusing workflow step, a gap against the documented contract.
2. **Non-trivial changes get a spec.** Run `pose new-spec <slug>` in your
   fork and fill Intent, Requirements (acceptance criteria with stable
   `- R<N>:` IDs) and Technical Plan. The spec travels with the PR — it is the
   PR description's source of truth.
3. **Architectural decisions get an ADR** (`pose new-adr "<title>"`): new
   contracts, changed frontmatter semantics, new gate behavior.

## Dogfooding governance (spec `pose-standalone-dogfood`)

The standalone repository is itself a governed POSE instance. The minimum
ownership and review rules are:

- **One spec, one roadmap.** Every non-trivial product change has exactly one
  owned spec in `.pose/specs/` and at most one active roadmap membership in
  `.pose/roadmaps/` (`pose check --strict` enforces exclusivity).
- **Owned modules.** `.pose/indexes/module-metadata.json` names an owner for
  every module; changes to a module follow its validation profile via
  `pose validate --strict --module <path> --report`.
- **Evidence is append-only.** Validation reports and JSONL history under
  `.pose/reports/` start at adoption time and are never backfilled or edited.
  CI re-runs the structural gate and retains the evidence produced by the
  build as workflow artifacts.
- **Identified builds only.** Gates run either a released `pose` binary or a
  development build compiled from the tree — development builds always report
  the explicit `-dev` version suffix and never impersonate a release
  (ADR `2026-07-19-authoritative-release-version-source`).
- **Quarterly audit.** The scheduled `governance-audit` workflow (also
  manually dispatchable) runs the structural gate, the open follow-up backlog,
  the knowledge overdue gate and outcome stats every quarter, and publishes
  the result as an artifact. Stale specs, roadmaps, knowledge or follow-ups
  found by the audit become issues or specs — silence is not a disposition.
- **No secrets in evidence.** Reports, history and audit artifacts must not
  contain tokens, restricted knowledge content or CI credentials.

## Pull request expectations

- `pose check --strict` and `pose lint-spec <your-spec> --strict` pass.
- Native engine changes come with Go tests under `pose-mcp/internal/`.
- Docs changes keep `AGENTS.md`/`POSE.md` references valid (`pose check`
  verifies them).
- One cohesive change per PR; follow-ups you discover go into the spec's
  Final Report with a disposition, not into scope creep.

## Style

- Go: stdlib-first, `gofmt`/`go vet` clean, no network calls in gates.
- Documentation: imperative, concrete, no aspirational claims — a statement of
  delivery requires verifiable gate evidence (see
  `.pose/rules/delivery-evidence.md`).

## Code of conduct

Be professional and assume good faith. Disagreements are resolved with
evidence (reproducible commands, specs, ADRs), not volume.
