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
