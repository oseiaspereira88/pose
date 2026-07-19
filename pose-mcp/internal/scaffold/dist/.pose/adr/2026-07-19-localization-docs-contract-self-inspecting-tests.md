# ADR: Localization and docs contract — self-inspecting tests over hand-maintained lists

## Status
Accepted (2026-07-19) — spec `pose-localization-docs-contract`

## Context

Two independent locale/docs gaps existed before this spec, undetected by
any test: `.pose/templates/knowledge.md` and `.pose/templates/doc-audit-report.md`
— the **English default** scaffold templates — were actually written
entirely in Portuguese, and neither had a `locales/pt-BR/` counterpart
(so a `pt-BR` install would get English behavior for exactly the two
templates that already happened to read as Portuguese by accident). The
existing scaffold parity test
(`TestEditorialDefaultsAreEnglishAndPtBROverlayIsComplete`) already
existed and already caught this class of bug — for `.pose/workflows/`,
`.pose/rules/`, `.agents/skills/` — but its `prefixes` list simply never
included `.pose/templates/`. Root-causing the fix also surfaced a second,
structural issue: `install.go`'s locale overlay used two different path
conventions — `locales/<locale>/templates/<name>` for templates (prefix
stripped) versus `locales/<locale>/.pose/workflows/...` for everything
else (prefix kept) — which is exactly the kind of asymmetry that makes a
"just add templates to the prefix list" fix insufficient on its own.

Alternatives considered:

1. **Special-case the test's path mapping for `.pose/templates/`** (strip
   the prefix only for that one directory) instead of touching
   `install.go`. Cheaper, but preserves the very asymmetry that let the
   bug exist, and every future locale-overlaid directory would need the
   same judgment call about which convention it follows.
2. **A hand-maintained list of "documented commands" and "valid
   subcommands"** to test R1 (every documented command runs or parses).
   Fast to write, but a second copy of `cli.go`'s dispatch table that
   silently drifts the same way the docs it's meant to protect could
   drift — defeats the purpose.
3. **Fix the root asymmetry in `install.go` (one convention:
   `locales/<locale>/<same-path-as-default>`, no exceptions), extend the
   existing parity test's prefix list, and make new contract tests derive
   their "ground truth" from the running source itself** (`cli.go`'s own
   switch statement for valid commands) rather than a parallel
   hand-maintained list.

## Decision

Option 3.

- **`install.go`'s locale overlay is now one loop, one convention:**
  `.pose/templates`, `.pose/workflows`, `.pose/rules`, `.agents/skills` are
  all copied via the same `copyTreeInto(dist, "locales/"+locale+"/"+dir,
  ...)` call — `locales/pt-BR/.pose/templates/*.md` replaces the old
  `locales/pt-BR/templates/*.md` layout. The two missing translations
  (`knowledge.md`, `doc-audit-report.md`) were written, and the English
  defaults were rewritten in actual English.
- **`TestEditorialDefaultsAreEnglishAndPtBROverlayIsComplete`'s prefix
  list grew one entry** (`.pose/templates/`) — the exact, minimal change
  that would have caught the original bug, now guarding it and every
  future template addition permanently.
- **R1 (documented commands run or parse) is tested by reading `cli.go`'s
  own switch statement at test time** (`dispatchedCommands` in
  `internal/cli/localization_docs_test.go`) rather than duplicating the
  command list — a renamed or removed command fails the moment its
  self-inspection regex sees the switch change, with zero double-
  maintenance. Every `pose <word>` mention across `README.md` and
  `docs-site/docs/*.md` is checked against that live set.
- **R3 (Diátaxis classification + version applicability) is a visible,
  grep-checked line** (`**Doc type:** <kind> · **Applies to:** POSE ≥
  0.9.0`) added right after every page's H1 — visible to a reader
  navigating the docs site, not just metadata. Twelve pages were
  classified: quickstart → Tutorial; ci/monorepo-recipes/package-channels
  → How-to; cli/mcp/frontmatter → Reference; the rest → Explanation.
- **The security scan reuses `pose-agent-skills-conformance`'s existing
  patterns** (`unsafeSkillPatterns`, `secretLikePatterns`) against docs
  content instead of writing new ones, plus one new check specific to
  docs (`sudo` in a copyable example — POSE's own design principle is
  that no command needs elevated privileges).
- **Compatibility (stable anchors) is satisfied by construction, not a
  new mechanism:** every edit in this spec inserts a line after an
  existing H1 rather than renaming or moving any heading or page, so no
  anchor changed and no redirect is needed.

## Consequences

- Positive: the exact bug class that motivated this spec (a locale
  overlay directory silently excluded from parity checking) cannot recur
  silently — any new overlay directory needs a one-line addition to
  `prefixes`, and forgetting it is now the only way to reintroduce this
  gap, not a structural inevitability.
- Positive: `TestDocumentedCommandsAreRecognizedByTheCLI` and
  `dispatchedCommands` cost nothing to keep in sync — they read the
  source of truth directly, so this spec adds zero new places for the
  command list to drift.
- Negative: `mkdocs build --strict` (the non-functional "strict links and
  deterministic snippets" requirement) is not executable in this sandbox
  (no `pip`/`mkdocs` available) — already wired in `.github/workflows/docs.yml`
  on every PR touching `docs-site/**`; verification deferred to that CI
  run, consistent with how sandbox-unavailable infrastructure was handled
  in `pose-package-manager-distribution` and `pose-slsa-provenance`.
- Neutral: the Diátaxis marker line is manually classified per page, not
  derived from file location or frontmatter — acceptable since there are
  only 12 pages and misclassification is now itself a testable, visible
  fact (`TestDocsAreDiataxisClassifiedWithVersionApplicability`) rather
  than a silent editorial choice.
