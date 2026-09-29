---
slug: public-claims-reads-release-lines
status: in-progress
created_at: 2026-09-29
completed_at:
supersedes:
depends_on: pose-public-claims-contract
priority: 1
components: pose-mcp, docs
task_type: bugfix
delivers: surface:public-claims-release-lines
---

# Spec: public-claims reads a release line

## 1. Intent

### Goal

The docs site states the release line it applies to, and the public-claims gate
fails when that line is not the released one.

### Business value

Every docs page opened with "Applies to: POSE 5.x (current stable)" through
6.0.0 to 6.0.4. `pose public-claims --strict` passed the whole time: its prose
pattern required `\d+\.\d+`, so "5.x" was not a claim to it, and eleven of the
fifteen pages were not declared surfaces anyway. A reader landing on any page
was told the current stable line was one major behind. It surfaced during the
closeout review of `pose-docs-and-manuals-match-5-0-2`, whose R1 had written the
banner correctly for 5.x.

### Constraints

Historical mentions of old releases stay legal on pages that are not declared
surfaces, and on declared pages when not phrased as "POSE <version>".

### Non-goals

Removing the banner. It answers a real question — which release a page
describes — and now it is checked.

## 2. Requirements

- R1: `POSE N.x` is a version claim whose version is the major `N`, compared
  with the released version at that precision.
- R2: A `current-only` surface naming a previous major fails, naming the line
  and the release; an evergreen surface naming any line fails.
- R3: Every docs-site page carrying an "Applies to: POSE" banner is a declared
  `current-only` surface, enforced by a test over this repository's contract.
- R4: The docs pages state the current line, 6.x, and the repository passes
  `pose public-claims --strict`.

## 3. Technical Plan

One pattern joins `publicVersionPatterns`, capturing the major.
`versionMatchesRelease` already compares at the claim's precision, so "6"
matches 6.0.4 and "5" does not. The contract gains the eleven undeclared pages
and moves the four evergreen ones to `current-only`, since their banner is a
deliberate line claim. One historical sentence on `product-roadmaps.md` is
reworded from "POSE 1.0.0 added" to "The v1.0.0 release added", the form the
next sentence on that page already uses.

### Artifacts

- created: .pose/specs/2026-09-29-public-claims-reads-release-lines.md
- created: .pose/changelogs/unreleased/public-claims-reads-release-lines.md
- created: pose-mcp/internal/cli/release_line_claims_test.go
- modified: pose-mcp/internal/cli/publicclaims.go
- modified: .pose/public/claims.json
- modified: .pose/indexes/validation-matrix.json
- modified: docs-site/docs/analytics.md
- modified: docs-site/docs/architecture.md
- modified: docs-site/docs/capability-assessment.md
- modified: docs-site/docs/ci.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/concepts.md
- modified: docs-site/docs/frontmatter.md
- modified: docs-site/docs/index.md
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/migrate/openspec.md
- modified: docs-site/docs/migrate/spec-kit.md
- modified: docs-site/docs/monorepo-recipes.md
- modified: docs-site/docs/package-channels.md
- modified: docs-site/docs/product-roadmaps.md
- modified: docs-site/docs/quickstart.md

### Delivery targets

- surface:public-claims-release-lines module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

The gate now reads claims it ignored, so a repository whose declared surfaces
name an old line starts failing; that is the defect being reported, not a
regression. Reverting restores the blind spot.

## 4. Tasks

- [x] Confirm the gate passes over the stale banners.
- [x] Pin release-line claims and the banner-surface rule in tests that fail without the fix.
- [x] Read `POSE N.x`, declare every bannered page and move the banners to 6.x.
- [ ] Run the checks, obtain review and close.

## 5. Decisions

### Decision D1

- Status: active
- Declare the bannered pages `current-only` rather than `none`. `none` would
  forbid the banner, and the banner is useful. The cost is that the four pages
  that were `none` may now also name the current minor; if one does, the gate
  fails as soon as that minor is superseded, so the drift is still caught.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Release-line claims | `go test ./internal/cli -run 'PublicClaimsReadsAMajorReleaseLine\|EveryDocsPageThatNamesAReleaseLineIsADeclaredSurface' -count=1` (from pose-mcp) | current line passes; previous line and evergreen line fail; every bannered page declared |
| Repository | `pose public-claims --strict` | 17 surfaces, 0 errors |

### Execution log

2026-09-29: with the pages at 5.x and the release at 6.0.4,
`pose public-claims --strict` reported SUCCESS. Both new tests failed before the
fix: the gate passed a previous major, and fifteen bannered pages were either
undeclared or declared `none`.

2026-09-29, implemented. Both tests pass, the whole `internal/cli` package
passes, and `pose public-claims --strict` reports 17 surfaces and 0 errors. The
first run after declaring the pages caught "POSE 1.0.0 added" on
`product-roadmaps.md`, a historical sentence, reworded as described above.

### Requirement trace

- R1 [satisfied] surface:public-claims-release-lines evidence:integration check:public-claims-release-line-integration test:TestPublicClaimsReadsAMajorReleaseLine — 1.x passes against 1.7.10
- R2 [satisfied] surface:public-claims-release-lines evidence:integration check:public-claims-release-line-integration test:TestPublicClaimsReadsAMajorReleaseLine — 0.x fails naming version 0 and 1.7.10; an evergreen surface naming 1.x fails
- R3 [satisfied] surface:public-claims-release-lines evidence:integration check:public-claims-release-line-integration test:TestEveryDocsPageThatNamesAReleaseLineIsADeclaredSurface — all fifteen bannered pages are current-only surfaces
- R4 [satisfied] surface:public-claims-release-lines evidence:integration check:public-claims-release-line-integration test:TestEveryDocsPageThatNamesAReleaseLineIsADeclaredSurface — pages read 6.x and the repository gate passes with 0 errors

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

A page without the banner can still describe an old release in prose; only
declared surfaces and the recognised claim shapes are checked.

### Follow-ups

None.
