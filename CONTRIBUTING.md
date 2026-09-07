# Contributing to POSE

POSE is developed by dogfooding: every non-trivial change to POSE is governed
by POSE itself — a spec with acceptance criteria, deterministic validation and
a formal closeout. Contributions follow the same path.

That is more ceremony than most projects ask for, and it is worth being blunt
about which parts apply to you. A typo fix needs none of it. A change to engine
behaviour needs most of it. The sections below are ordered so you can stop
reading at the level your change actually reaches.

## Start here: reproduce the gates locally

Before anything else, confirm you can run what CI runs. POSE gates its own
repository strictly, and finding out what failed from a CI log on someone
else's schedule is the most common reason an outside contribution stalls.

```bash
git clone https://github.com/oseiaspereira88/pose && cd pose
bash scripts/verify.sh --fast
```

That builds the development binary and runs the Go tests plus the structural,
skills, history and public-claims gates. Drop `--fast` to also run the
installer end-to-end and the negative artifact-identity gate, which is what CI
does — do that before opening a pull request.

If `scripts/verify.sh` passes and CI still fails on your branch, that is a bug
in the script, not in your change. Please report it: the script is supposed to
be the complete local mirror of the gates.

## Commit trailers: `POSE-Spec:`

Every commit that implements, changes or tests an artifact declared in a spec
must carry the trailer:

```
POSE-Spec: <spec-slug>
```

This is not bookkeeping. Without it, `pose artifact-check` and `pose close`
cannot attribute Git change sets to the spec's `### Artifacts` section, and the
delivery contract fails with "no Git change sets are attributed". A run of
specs was once closed without it and left 75 unresolvable gate errors behind —
the trailer is cheap at commit time and expensive to reconstruct afterwards.

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

## What is yours and what is the maintainer's

So a pull request does not stall on a gate you had no way to satisfy:

| You | The maintainer |
|---|---|
| The spec, its `R<N>` acceptance criteria and its Technical Plan | Roadmap membership, if the change belongs on one |
| The implementation and its tests | The review attestation and `pose close` |
| `bash scripts/verify.sh` passing locally | Release cut, changelog assembly and tagging |
| `POSE-Spec:` trailers on your commits | Anything requiring repository or org credentials |

You are not expected to record a review of your own work. POSE's review policy
sets `reviewer_independence: same-actor-separate-execution` precisely so that
the author's own approval never closes the loop.

## Pull request expectations

- `bash scripts/verify.sh` passes locally.
- `pose check --strict` and `pose lint-spec <your-spec> --strict` pass.
- Native engine changes come with Go tests under `pose-mcp/internal/`.
- Docs changes keep `AGENTS.md`/`POSE.md` references valid (`pose check`
  verifies them).
- One cohesive change per PR; follow-ups you discover go into the spec's
  Final Report with a disposition, not into scope creep.
- Diffs to `internal/mcpserver/testdata/tool-catalog.golden.json` are public
  API changes: review them as such. Removals or incompatible schema changes
  additionally require an ADR and a release note (ADR
  `2026-07-19-mcp-tool-catalog-is-a-release-gated-contract`).

## Style

- Go: stdlib-first, `gofmt`/`go vet` clean, no network calls in gates.
- Documentation: imperative, concrete, no aspirational claims — a statement of
  delivery requires verifiable gate evidence (see
  `.pose/rules/delivery-evidence.md`).

## Code of conduct

Be professional and assume good faith. Disagreements are resolved with
evidence (reproducible commands, specs, ADRs), not volume.
