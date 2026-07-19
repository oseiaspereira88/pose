# ADR: Brownfield reference kits — checked-in fixtures, git-native rollback

## Status
Accepted (2026-07-19) — spec `pose-brownfield-reference-kits`

## Context

`pose install` and `pose import spec-kit|openspec` already existed and
were already well tested at the engine level (`import_test.go` covers
symlink rejection, collision, byte limits, rollback-on-write-failure).
What was missing was the adopter-facing artifact this spec asks for:
"executable adoption kits" proving the real, end-to-end journey — direct
adoption, Spec Kit import, OpenSpec import/reconciliation — from first
contact through a blocking gate, including what rollback looks like and
what mapping loss actually shows up as.

Alternatives considered:

1. **Narrative-only docs** (a guide describing the journey without a real
   fixture or test). Fastest to write, but exactly the risk the spec's own
   Technical risk calls out: "idealized examples conceal real migration
   costs" — a hand-written narrative silently drifts from what the CLI
   actually does the next time either changes.
2. **A new `pose adopt`/`pose uninstall` command pair** to give kits a
   single scripted entry/rollback point. Rejected: POSE already has no
   destructive or irreversible install-side mutation (install never
   touches instance content; import writes one new, self-contained
   `.pose/specs/<slug>/` directory) — a dedicated uninstall command would
   be new surface solving a problem that plain `git clean`/`git revert`
   already solves, for every one of these kits, by construction.
3. **Real, checked-in fixture repositories under `examples/`, driven by
   Go tests that call the exact same `Main`/`cmdInstall` entry points the
   CLI itself uses**, asserting preservation (byte-for-byte pre-existing
   content), surfaced curation warnings, DoR readiness, and rollback
   safety (`git status --porcelain` shows zero modification to anything
   pre-existing).

## Decision

Option 3.

- **Three kits under `examples/brownfield-kits/`**, each `fixture/` a
  small, real, intentionally-imperfect brownfield repo:
  `direct-adoption` (a bare Go module, no spec system),
  `spec-kit-import` (a real spec-kit feature missing `plan.md`),
  `openspec-import` (a real OpenSpec change missing `design.md`). The
  missing companion files are deliberate, not an oversight — they're what
  makes `TestBrownfieldSpecKitImportKit`/`TestBrownfieldOpenSpecImportKit`
  able to assert a genuine curation warning surfaces (`plan.md not found`
  / `design.md not found`) instead of exercising only the clean-input
  happy path a synthetic, idealized fixture would give.
- **`examples/` joins the scaffold's exclusion list** (`gen/main.go` and
  `scaffold_test.go`, alongside the pre-existing `tests`/`docs-site`
  exclusions) — these are dev-only reference material, not something
  `pose install` should copy into every adopter's repository.
- **CI execution is three Go tests** in
  `internal/cli/brownfield_kits_test.go`, not a new shell script: they
  reuse `repoRootForTest()` (already used by the skills-check dogfood
  test) to locate the real fixture on disk, copy it into a fresh git repo,
  and drive it through `Main`/`cmdInstall` exactly as the CLI does. This
  keeps the "kit" and the "test of the kit" in the same language and
  process as the rest of the CLI's own test suite, with no new tooling.
- **Rollback is documented and proven, not built:** every kit test's final
  assertion is that `git status --porcelain` shows only untracked (`??`)
  new paths — nothing pre-existing carries a modification marker. Each
  kit's README states the rollback story as "plain `git clean -fdx` /
  `git revert`" rather than inventing an uninstall command.
- **Readiness and curation warnings are documented as orthogonal**, a real
  and slightly non-obvious finding: `pose lint-spec --ready-check` only
  requires Intent/Requirements/Technical Plan to be non-placeholder text —
  the importer's own "curate this" fallback prose satisfies that
  structurally, so a freshly-imported spec with open curation follow-ups
  still reports `spec.ready=true`. Both kit READMEs and the top-level
  `examples/brownfield-kits/README.md` state this explicitly so an adopter
  doesn't mistake DoR-readiness for "curation is done."

## Consequences

- Positive: the kits cannot silently drift from real CLI behavior — a
  future change to `pose import`'s warning text, slug derivation, or
  install's file set fails `TestBrownfieldDirectAdoptionKit` /
  `TestBrownfieldSpecKitImportKit` / `TestBrownfieldOpenSpecImportKit`
  immediately, the same as any other regression in `internal/cli`.
- Positive: zero new mutation surface — every command a kit runs already
  existed and was already covered at the unit level; this spec adds
  integration-level, fixture-driven proof on top, not new engine code.
- Negative: three fixture repositories add a small amount of checked-in
  content to maintain (README + go.mod/spec-kit/openspec files); judged
  worth it since they are the literal proof artifact for R1–R3, not
  incidental.
- Neutral: `examples/` uses the same two-line exclusion pattern `tests`
  already established in the scaffold generator/drift-test pair — adding
  a fourth reference-material directory later is the same one-line change
  in both places.
