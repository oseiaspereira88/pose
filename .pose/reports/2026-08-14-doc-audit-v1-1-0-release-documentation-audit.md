# Doc Audit Report — 2026-08-14

## Scope

Audit public English and Brazilian Portuguese entry points, the MkDocs source
and generated site, the structured capability assessment, MCP catalog claims,
portfolio counts, release installation examples and CI/release guidance after
delivery of component-aware review plans and convergent sealed review bundles.
Prepare documentation for the `v1.1.0` candidate; exclude tag creation,
provider publication and independent verification of unpublished assets.

## Findings

- High: the public CLI reference did not document `review-plan`, bundle
  preparation/sealing, separate attestations, verification or closeout checks.
- High: release-facing docs and metadata still identified the product as
  `1.0.x`/`1.0.0`, which blocked `pose release plan --version v1.1.0`.
- Medium: MCP claims stated 48 tools (45 POSE + 3 reporters), while the golden
  catalog contains 50 tools (47 POSE + 3 reporters).
- Medium: README and roadmap pages reported stale repository/portfolio counts;
  the canonical state is 91 completed specs, 8 completed roadmaps and 39
  roadmap-bound specs.
- Medium: the capability assessment still reported 9 skills and stale review
  evidence; the distributed catalog has 11 skills in each locale.
- Medium: the CI guide linked to a nonexistent `docs/RELEASE.md`.
- Medium: full validation traversed the local `.qwen/worktrees/` cache and
  treated a nested checkout as release scope.

## Fixes applied

- Added a CLI reference section for component-aware and convergent review,
  including opt-in boundaries, dry-run/apply semantics and the fixed-point
  `prepare → validate → seal → attest → verify → close` sequence.
- Updated README, pt-BR parity, MkDocs applicability headers, architecture,
  concepts, MCP, CI, capability and roadmap narratives for the `v1.1.0`
  candidate.
- Reconciled public counts against repository artifacts and the MCP golden
  fixture: 91 specs, 8 roadmaps, 11 skills and 50 MCP tools.
- Updated the structured capability source with current evidence and preserved
  the published `v1.0.0` references only where they describe historical facts.
- Replaced the broken release-doc link with the maintained CLI release
  lifecycle and regenerated the checked-in MkDocs output.
- Advanced the authoritative engine/registry compatibility surfaces to
  `1.1.0` and added the authenticated `1.0.0 → 1.1.0` upgrade pair.
- Excluded `.qwen/` from version control and validation-module discovery, with
  a regression test proving nested local worktrees cannot become modules.

## Validation evidence

- `PYTHONPATH=/tmp/pose-v110-mkdocs python3 -m mkdocs build --strict -f docs-site/mkdocs.yml`: success with MkDocs 1.6.1.
- `/tmp/pose-v110 assess`: success, 16 mechanisms.
- `/tmp/pose-v110 skills-check`: success, 22 locale entries, no errors or warnings.
- `go test ./internal/version/... ./internal/mcpserver -run 'Catalog|Initialize|ToolsList' -count=1`: success (isolated Go cache; local HTTP test listener authorized outside the sandbox).
- `go test ./internal/scaffold -count=1`: success.
- `/tmp/pose-v110 check --strict`: success.
- `GOCACHE=/tmp/pose-v110-go-cache /tmp/pose-v110 validate --strict`: success
  after the local-worktree exclusion; all four repository modules and the
  three permanent delivery/review checks passed.
- `/tmp/pose-v110 release plan --version v1.1.0 --json`: success, 2 fragments, no blockers, non-breaking minor recommendation.

## Residual risks

- Docs governance remains intentionally opt-in in this source repository:
  `.pose/docs.json` is absent, so `pose docs-check` reports that no manifest is
  configured rather than evaluating a declared inventory.
- The MkDocs build emits the upstream Material warning about the future MkDocs
  2.0 ecosystem; the pinned MkDocs 1.6.1 strict build succeeds.
- `v1.1.0` is a prepared candidate until an immutable tag, provider publication
  and independent artifact verification are recorded. Public examples point to
  `v1.1.0` as part of the release snapshot and become consumable after publish.

## Follow-ups

- [open] Publish and independently verify `v1.1.0` only after the reviewed
  prepared snapshot is committed. (owner:@pose-maintainers crit:high review:2026-08-15)
- [open] Decide whether the source repository should adopt `.pose/docs.json`;
  do not create it implicitly as part of a release cut. (owner:@pose-maintainers crit:low review:2026-09-13)
